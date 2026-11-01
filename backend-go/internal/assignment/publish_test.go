package assignment_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/assignment"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
)

func (e *env) reviewed(a assignment.Assignment) rrow {
	e.t.Helper()
	return e.rrow(a, e.student(), 1, time.Now().UTC().Add(-time.Hour), "DONE", "REVIEWED", crit10, nil)
}

func (e *env) publish1(ids ...uuid.UUID) (assignment.ResultsPublishOut, error) {
	return e.svc.PublishResults(e.ctx, e.act, e.course, assignment.PublishIn{ResultIDs: ids})
}

func (e *env) status(res uuid.UUID) (s string) {
	e.one(&s, `select status from grading_results where id = $1`, res)
	return s
}

func detailIDs(t *testing.T, err error) []uuid.UUID {
	t.Helper()
	ae := err.(*apierr.Error) //nolint:errorlint // apierr.Error trả thẳng
	return ae.Details.(map[string]any)["ids"].([]uuid.UUID)
}

// TestPublishBulkOnlyReviewed — AC1: công bố đúng các bài REVIEWED; độ lệch + audit + outbox mỗi bài; không ghi grade_entries.
func TestPublishBulkOnlyReviewed(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	r1, r2 := e.reviewed(a), e.reviewed(a)
	out, err := e.publish1(r1.res, r2.res)
	require.NoError(t, err)
	require.Equal(t, 2, out.Published)
	for _, r := range []rrow{r1, r2} {
		require.Equal(t, "PUBLISHED", e.status(r.res))
		var by uuid.UUID
		var at time.Time
		e.one(&by, `select published_by from grading_results where id = $1`, r.res)
		e.one(&at, `select published_at from grading_results where id = $1`, r.res)
		require.Equal(t, e.teacher, by)
	}
	var n int
	e.one(&n, `select count(*)::int from outbox where topic = $1 and payload->>'result_id' in ($2, $3)`, assignment.TopicResultPublished, r1.res.String(), r2.res.String())
	require.Equal(t, 2, n)
	e.one(&n, `select count(*)::int from audit_log where action = 'grading_result.publish' and entity_id in ($1, $2)`, r1.res.String(), r2.res.String())
	require.Equal(t, 2, n)
	e.one(&n, `select count(*)::int from grade_entries where course_id = $1`, e.course)
	require.Zero(t, n, "công bố không ghi grade_entries (TLR-14 A)")
}

// TestPublishAllOrNothingExplicit — AC1: một bài không REVIEWED trong danh sách tường minh → 422 NOT_REVIEWED {ids}, không công bố gì.
func TestPublishAllOrNothingExplicit(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	ok := e.reviewed(a)
	draft := e.rrow(a, e.student(), 1, time.Now().UTC().Add(-time.Hour), "DONE", "DRAFT", crit10, nil)
	_, err := e.publish1(ok.res, draft.res)
	require.Equal(t, apierr.NotReviewed, code(err))
	require.Equal(t, []uuid.UUID{draft.res}, detailIDs(t, err))
	require.Equal(t, "REVIEWED", e.status(ok.res), "không công bố gì")
	_, err = e.publish1(uuid.New())
	require.Equal(t, apierr.NotReviewed, code(err), "id lạ cũng là chưa duyệt")
}

// TestPublishScopeSkips — AC1: scope ALL_REVIEWED bỏ qua bài chưa duyệt và trả {published, skipped}.
func TestPublishScopeSkips(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	e.reviewed(a)
	e.reviewed(a)
	draft := e.rrow(a, e.student(), 1, time.Now().UTC().Add(-time.Hour), "DONE", "DRAFT", crit10, nil)
	_ = draft
	out, err := e.svc.PublishResults(e.ctx, e.act, e.course, assignment.PublishIn{AssignmentID: &a.ID, Scope: "ALL_REVIEWED"})
	require.NoError(t, err)
	require.Equal(t, 2, out.Published)
	again, err := e.svc.PublishResults(e.ctx, e.act, e.course, assignment.PublishIn{AssignmentID: &a.ID, Scope: "ALL_REVIEWED"})
	require.NoError(t, err)
	require.Zero(t, again.Published, "lần hai không có bài REVIEWED nào")
	_, err = e.svc.PublishResults(e.ctx, e.act, e.course, assignment.PublishIn{ResultIDs: []uuid.UUID{draft.res}, Scope: "ALL_REVIEWED"})
	require.Equal(t, apierr.ValidationFailed, code(err))
	_, err = e.svc.PublishResults(e.ctx, e.act, e.course, assignment.PublishIn{})
	require.Equal(t, apierr.ValidationFailed, code(err))
}

// TestPublishIdempotent — AC1: công bố lại cùng danh sách → 409 ALREADY_PUBLISHED, không đổi gì, không thêm outbox / audit (Idempotency-Key phát lại cùng phản hồi ở middleware).
func TestPublishIdempotent(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	r := e.reviewed(a)
	_, err := e.publish1(r.res)
	require.NoError(t, err)
	_, err = e.publish1(r.res)
	require.Equal(t, apierr.AlreadyPublished, code(err))
	var n int
	e.one(&n, `select count(*)::int from outbox where topic = $1 and payload->>'result_id' = $2`, assignment.TopicResultPublished, r.res.String())
	require.Equal(t, 1, n)
}

// TestPublishOnlyFromHandler — AC1: không có đường nào ngoài PublishMark (câu SQL duy nhất) đặt status = 'PUBLISHED' ở mã sản xuất (SQL nguồn, script seed, cmd).
func TestPublishOnlyFromHandler(t *testing.T) {
	setsPublished := func(src string) bool {
		for _, stmt := range strings.Split(src, ";") {
			low := strings.ToLower(stmt)
			i := strings.Index(low, "update grading_results")
			if i < 0 {
				continue
			}
			set := low[i:]
			if w := strings.Index(set, "where"); w >= 0 {
				set = set[:w]
			}
			if regexp.MustCompile(`status\s*=\s*'published'`).MatchString(set) {
				return true
			}
		}
		return false
	}
	var hits []string
	scan := func(root string, exts ...string) {
		require.NoError(t, filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || strings.HasSuffix(p, "_test.go") || strings.Contains(p, "/migrations/") || strings.HasSuffix(p, ".sql.go") || strings.Contains(p, "node_modules") {
				return err
			}
			for _, x := range exts {
				if strings.HasSuffix(p, x) {
					b, rerr := os.ReadFile(p)
					require.NoError(t, rerr)
					if setsPublished(string(b)) {
						hits = append(hits, p)
					}
				}
			}
			return nil
		}))
	}
	scan("../store/queries", ".sql")
	scan("../../cmd", ".go")
	scan("../", ".go")
	scan("../../../scripts", ".mjs", ".js", ".sh", ".sql")
	require.Equal(t, []string{"../store/queries/review.sql"}, hits, "PublishMark phải là nơi duy nhất")
}

// TestPublishOneTx — AC1: một giao dịch — bài thứ hai trong lô bị LATE_RECOMPUTED thì bài thứ nhất không được công bố.
func TestPublishOneTx(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	good := e.reviewed(a)
	late := e.rrow(a, e.student(), 1, time.Now().UTC().Add(-time.Hour), "DONE", "REVIEWED", crit10, nil)
	e.exec(`update grading_results set total = 9 where id = $1`, late.res) // tổng đã duyệt lệch với tính lại (hạn đổi sau khi duyệt)
	_, err := e.publish1(good.res, late.res)
	require.Equal(t, apierr.LateRecomputed, code(err))
	require.Equal(t, []uuid.UUID{late.res}, detailIDs(t, err))
	require.Equal(t, "REVIEWED", e.status(good.res))
	out, err := e.svc.PublishResults(e.ctx, e.act, e.course, assignment.PublishIn{AssignmentID: &a.ID, Scope: "ALL_REVIEWED"})
	require.NoError(t, err)
	require.Equal(t, 1, out.Published, "scope: bài lệch bị bỏ qua, bài tốt vẫn công bố")
	require.Equal(t, 1, out.Skipped)
}

// TestPublishConcurrentOnce — AC8: hai giảng viên cùng công bố một bài → công bố một lần; người sau ALREADY_PUBLISHED.
func TestPublishConcurrentOnce(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	r := e.reviewed(a)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var ok, dup int
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.publish1(r.res)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case code(err) == apierr.AlreadyPublished:
				dup++
			default:
				t.Errorf("lỗi lạ: %v", err)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, 1, ok)
	require.Equal(t, 5, dup)
	var n int
	e.one(&n, `select count(*)::int from outbox where topic = $1 and payload->>'result_id' = $2`, assignment.TopicResultPublished, r.res.String())
	require.Equal(t, 1, n)
}

// TestPublishVersionConflict — AC8: bài bị sửa sau khi chọn (version lệch) → 409 VERSION_CONFLICT cho đúng bài đó, cả lô không công bố.
func TestPublishVersionConflict(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	r1, r2 := e.reviewed(a), e.reviewed(a)
	var v1 int32
	e.one(&v1, `select version from grading_results where id = $1`, r1.res)
	_, err := e.svc.PublishResults(e.ctx, e.act, e.course, assignment.PublishIn{ResultIDs: []uuid.UUID{r1.res, r2.res}, Versions: map[string]int32{r1.res.String(): v1 + 3}})
	require.Equal(t, apierr.VersionConflict, code(err))
	require.Equal(t, []uuid.UUID{r1.res}, detailIDs(t, err))
	require.Equal(t, "REVIEWED", e.status(r2.res))
	_, err = e.svc.PublishResults(e.ctx, e.act, e.course, assignment.PublishIn{ResultIDs: []uuid.UUID{r1.res, r2.res}, Versions: map[string]int32{r1.res.String(): v1}})
	require.NoError(t, err)
}

// TestDeviationStoredAtPublish — AC5: deviation = |tổng cuối trước trừ muộn − AI|, teacher_changed_criteria = tiêu chí đổi điểm.
func TestDeviationStoredAtPublish(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	r := e.rrow(a, e.student(), 1, time.Now().UTC().Add(-time.Hour), "DONE", "DRAFT", crit10, nil)
	v := e.detail(r.res).Result.Version
	out, err := e.edit(r.res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit("c1", "3"), crit("c2", "5.5")}})
	require.NoError(t, err)
	rv, err := e.review(r.res, out.Result.Version)
	require.NoError(t, err)
	_ = rv
	_, err = e.publish1(r.res)
	require.NoError(t, err)
	var dev decimal.Decimal
	var changed []string
	e.one(&dev, `select deviation from grading_results where id = $1`, r.res)
	e.one(&changed, `select teacher_changed_criteria from grading_results where id = $1`, r.res)
	require.True(t, dev.Equal(decimal.RequireFromString("1.5")), dev.String())
	require.ElementsMatch(t, []string{"c1", "c2"}, changed)
}

// TestDeviationDemoStudentB — AC5 (DEMO, sinh viên B): AI nháp 7,00, giảng viên nâng tiêu chí 2 từ 1,00 lên 2,00 → deviation 1,00, teacher_changed_criteria = [c2].
func TestDeviationDemoStudentB(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	sub := e.rrow(a, e.student(), 1, time.Now().UTC().Add(-time.Hour), "", "", "", nil)
	var res uuid.UUID
	e.one(&res, `insert into grading_results (course_id, submission_id, run_state, status, criteria, total, ai_total_before_late, ai_scores)
		values ($1, $2, 'DONE', 'DRAFT', '[{"id":"c1","score":"4.00"},{"id":"c2","score":"1.00"}]'::jsonb, 5, 5, '{"c1":"4.00","c2":"1.00"}'::jsonb) returning id`, e.course, sub.sub)
	v := e.detail(res).Result.Version
	out, err := e.edit(res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit2("c2", "2")}})
	require.NoError(t, err)
	_, err = e.review(res, out.Result.Version)
	require.NoError(t, err)
	_, err = e.publish1(res)
	require.NoError(t, err)
	var dev decimal.Decimal
	var changed []string
	e.one(&dev, `select deviation from grading_results where id = $1`, res)
	e.one(&changed, `select teacher_changed_criteria from grading_results where id = $1`, res)
	require.Equal(t, "1.00", dev.StringFixed(2))
	require.Equal(t, []string{"c2"}, changed)
}

func crit2(id, score string) assignment.CriterionEdit { return crit(id, score) }

// TestStatsEndpoint — AC5: số liệu bài; tỷ lệ null khi mẫu số 0.
func TestStatsEndpoint(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	empty, err := e.svc.Stats(e.ctx, e.course, a.ID)
	require.NoError(t, err)
	require.Nil(t, empty.MeanAbsDeviation)
	require.Nil(t, empty.AppealRate)
	r1, r2 := e.reviewed(a), e.reviewed(a)
	e.rrow(a, e.student(), 1, time.Now().UTC().Add(-time.Hour), "DONE", "DRAFT", crit10, nil)
	_, err = e.publish1(r1.res, r2.res)
	require.NoError(t, err)
	st, err := e.svc.Stats(e.ctx, e.course, a.ID)
	require.NoError(t, err)
	require.Equal(t, 3, st.Submitted)
	require.Equal(t, 3, st.Graded)
	require.Equal(t, 2, st.Reviewed)
	require.Equal(t, 2, st.Published)
	require.Equal(t, "0.00", *st.MeanAbsDeviation)
	require.Equal(t, "0.0000", *st.TeacherChangedRate)
	_, err = e.svc.Stats(e.ctx, e.course, uuid.New())
	require.Equal(t, apierr.NotFound, code(err))
	raw, _ := json.Marshal(st)
	require.Contains(t, string(raw), `"appeal_success_rate":null`)
}

// TestPublishOutboxReplayNoDuplicate — AC8: xử lý lại outbox `assignment.result_published` không gửi đôi chuông / thư.
func TestPublishOutboxReplayNoDuplicate(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	r := e.reviewed(a)
	e.exec(`update users set email_verified_at = now() where id = $1`, e.studentOf(r))
	_, err := e.publish1(r.res)
	require.NoError(t, err)
	msg := e.publishedMsg(r.res)
	for range 3 {
		require.NoError(t, e.svc.HandleResultPublished(e.ctx, msg))
	}
	var n int
	e.one(&n, `select count(*)::int from notifications where user_id = $1 and type = 'GRADE_PUBLISHED'`, e.studentOf(r))
	require.Equal(t, 1, n)
	e.one(&n, `select count(*)::int from mail_outbox where template = 'assignment_feedback' and dedupe_key = $1`, "assignment_feedback:"+r.res.String())
	require.Equal(t, 1, n)
}

func (e *env) studentOf(r rrow) (u uuid.UUID) {
	e.one(&u, `select student_id from submissions where id = $1`, r.sub)
	return u
}

func (e *env) publishedMsg(res uuid.UUID) outbox.Message {
	var payload []byte
	e.one(&payload, `select payload from outbox where topic = $1 and payload->>'result_id' = $2`, assignment.TopicResultPublished, res.String())
	return outbox.Message{ID: uuid.New(), Topic: assignment.TopicResultPublished, Payload: payload}
}

// TestMinePublishedFields — AC6: sinh viên thấy điểm tổng / max, từng tiêu chí (điểm / max, nhận xét, trích dẫn đã xác minh), khoản trừ muộn, nhận xét chung, nhãn chung; KHÔNG thấy điểm AI, cờ, độ tin cậy, hai lượt, người duyệt, đáp án, guidance.
func TestMinePublishedFields(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	e.publish(a)
	r := e.reviewed(a)
	stu := e.studentOf(r)
	e.exec(`update grading_results set overall_feedback = 'Nhận xét chung của giảng viên.', flags = '{NEEDS_REVIEW}', confidence = 0.42, late_penalty = 0.5, late_days = 1, late_penalty_manual = true, total = 9.5,
		pass_detail = '{"c1":{"pass1":"CANARY-PASS1"}}'::jsonb,
		criteria = '[{"id":"c1","score":"4.00","comment":"Đúng.","evidence":[{"quote":"đoạn trích của em","start":0,"end":5}]},{"id":"c2","score":"6.00","comment":"Tốt.","evidence":[]}]'::jsonb where id = $1`, r.res)
	before, err := e.svc.Mine(e.ctx, assignment.Actor{UserID: stu, Student: true}, e.course, a.ID)
	require.NoError(t, err)
	require.Nil(t, before.Result, "chưa công bố: không có kết quả")
	require.Equal(t, assignment.StateGrading, before.State)
	_, err = e.publish1(r.res)
	require.NoError(t, err)
	out, err := e.svc.Mine(e.ctx, assignment.Actor{UserID: stu, Student: true}, e.course, a.ID)
	require.NoError(t, err)
	require.Equal(t, assignment.StateGraded, out.State)
	require.NotNil(t, out.Result)
	require.Equal(t, "9.50", out.Result.Total)
	require.Equal(t, "10.00", out.Result.Max)
	require.Equal(t, "Nhận xét chung của giảng viên.", out.Result.OverallFeedback)
	require.Equal(t, assignment.ResultLabel, out.Result.Label)
	require.Equal(t, &assignment.MineLate{Days: 1, Penalty: "0.50"}, out.Result.Late)
	require.Len(t, out.Result.Criteria, 2)
	require.Equal(t, assignment.MineCriterion{Name: "Khái niệm", Score: "4.00", Max: "4.00", Comment: "Đúng.", Evidence: []string{"đoạn trích của em"}}, out.Result.Criteria[0])
	require.Equal(t, "NONE", out.Result.Appeal.State)
	raw, _ := json.Marshal(out)
	for _, bad := range []string{"CANARY", "NEEDS_REVIEW", "confidence", "ai_scores", "reference", "guidance", "deviation", "reviewed_by", "pass", "flags", "run_state"} {
		require.NotContains(t, string(raw), bad)
	}
	// sinh viên khác không thấy kết quả này
	other, err := e.svc.Mine(e.ctx, assignment.Actor{UserID: e.student(), Student: true}, e.course, a.ID)
	require.NoError(t, err)
	require.Nil(t, other.Result)
}
