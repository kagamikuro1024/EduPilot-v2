package assignment_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/assignment"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

// rrow chèn bài nộp MATCHED (phiên bản v, nhận lúc `at`) + kết quả hiện hành với trạng thái chỉ định.
type rrow struct {
	sub, res uuid.UUID
}

func (e *env) rrow(a assignment.Assignment, stu uuid.UUID, v int, at time.Time, runState, status, crit string, flags []string) rrow {
	e.t.Helper()
	var r rrow
	e.one(&r.sub, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at, received_at, extracted_text)
		values ($1, $2, $3, 'IN_APP', $4, 'k', 'a.pdf', 1, 'application/pdf', 'x', 'MATCHED', 'IN_APP', $5, $5, 'Nội dung bài làm.') returning id`, e.course, a.ID, stu, v, at)
	if runState == "" {
		return r
	}
	pub := "null"
	if status == "PUBLISHED" {
		pub = "now()"
	}
	if flags == nil {
		flags = []string{}
	}
	var total any
	if runState == "DONE" {
		total = 10
	}
	e.one(&r.res, `insert into grading_results (course_id, submission_id, run_state, status, criteria, total, ai_total_before_late, ai_scores, flags, published_at)
		values ($1, $2, $3, $4, $5::jsonb, $6, $6, '{"c1":"4.00","c2":"6.00"}'::jsonb, $7, `+pub+`) returning id`, e.course, r.sub, runState, status, crit, total, flags)
	return r
}

func (e *env) student() uuid.UUID {
	s := e.user("STUDENT")
	e.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, e.course, s)
	return s
}

func (e *env) queue(f assignment.QueueFilter) assignment.QueuePage {
	e.t.Helper()
	p, err := e.svc.Queue(e.ctx, e.course, f)
	require.NoError(e.t, err)
	return p
}

func resultIDs(p assignment.QueuePage) []uuid.UUID {
	var out []uuid.UUID
	for _, it := range p.Items {
		if it.Result != nil {
			out = append(out, it.Result.ID)
		}
	}
	return out
}

// TestQueueDefaultFilter — AC1: mặc định chỉ MATCHED + DRAFT (đã chấm xong); lọc status khác lấy đúng nhóm; bài thay phiên bản không hiện.
func TestQueueDefaultFilter(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	now := time.Now().UTC()
	draft := e.rrow(a, e.student(), 1, now, "DONE", "DRAFT", crit10, nil)
	reviewed := e.rrow(a, e.student(), 1, now.Add(time.Second), "DONE", "REVIEWED", crit10, nil)
	running := e.rrow(a, e.student(), 1, now.Add(2*time.Second), "RUNNING", "DRAFT", `[]`, nil)
	published := e.rrow(a, e.student(), 1, now.Add(3*time.Second), "DONE", "PUBLISHED", crit10, nil)
	failed := e.rrow(a, e.student(), 1, now.Add(4*time.Second), "FAILED", "DRAFT", `[]`, nil)
	none := e.rrow(a, e.student(), 1, now.Add(5*time.Second), "", "", "", nil)
	old := e.student()
	e.rrow(a, old, 1, now.Add(6*time.Second), "DONE", "DRAFT", crit10, nil)
	e.rrow(a, old, 2, now.Add(7*time.Second), "", "", "", nil) // bản mới hơn → bản 1 không hiện

	require.Equal(t, []uuid.UUID{draft.res}, resultIDs(e.queue(assignment.QueueFilter{})))
	for status, want := range map[string]uuid.UUID{"REVIEWED": reviewed.res, "RUNNING": running.res, "PUBLISHED": published.res, "FAILED": failed.res} {
		require.Equal(t, []uuid.UUID{want}, resultIDs(e.queue(assignment.QueueFilter{Status: status})), status)
	}
	p := e.queue(assignment.QueueFilter{Status: "NONE"})
	ids := map[uuid.UUID]bool{}
	for _, it := range p.Items {
		ids[it.ID] = true
		require.Nil(t, it.Result)
	}
	require.True(t, ids[none.sub])
	require.Len(t, ids, 2, "bài chưa chấm: none và bản 2 của sinh viên thay phiên bản")
	_, err := e.svc.Queue(e.ctx, e.course, assignment.QueueFilter{Status: "NOPE"})
	require.Equal(t, apierr.ValidationFailed, code(err))
}

// TestQueueNeedsReviewFirst — AC1: bài có cờ lên đầu dù nộp sau.
func TestQueueNeedsReviewFirst(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	now := time.Now().UTC()
	plain := e.rrow(a, e.student(), 1, now, "DONE", "DRAFT", crit10, nil)
	flagged := e.rrow(a, e.student(), 1, now.Add(time.Hour), "DONE", "DRAFT", crit10, []string{"NEEDS_REVIEW"})
	harmless := e.rrow(a, e.student(), 1, now.Add(2*time.Hour), "DONE", "DRAFT", crit10, []string{"SCORE_ADJUSTED"})
	got := resultIDs(e.queue(assignment.QueueFilter{}))
	require.Equal(t, []uuid.UUID{flagged.res, plain.res, harmless.res}, got)
	only := true
	require.Equal(t, []uuid.UUID{flagged.res}, resultIDs(e.queue(assignment.QueueFilter{NeedsReview: &only})))
	require.Equal(t, []uuid.UUID{flagged.res}, resultIDs(e.queue(assignment.QueueFilter{Flag: "NEEDS_REVIEW"})))
}

// TestQueueCursorPagination — AC1: keyset qua ranh giới cờ / không cờ, không trùng, không sót.
func TestQueueCursorPagination(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	now := time.Now().UTC()
	want := map[uuid.UUID]bool{}
	for i := range 25 {
		var flags []string
		if i%3 == 0 {
			flags = []string{"OFF_TOPIC"}
		}
		r := e.rrow(a, e.student(), 1, now.Add(time.Duration(i)*time.Second), "DONE", "DRAFT", crit10, flags)
		want[r.res] = true
	}
	seen := map[uuid.UUID]bool{}
	var order []bool
	cursor := ""
	for range 5 {
		p := e.queue(assignment.QueueFilter{Limit: 10, Cursor: cursor})
		for _, it := range p.Items {
			require.False(t, seen[it.Result.ID], "trùng dòng giữa hai trang")
			seen[it.Result.ID] = true
			order = append(order, it.Result.NeedsReview)
		}
		if p.NextCursor == nil {
			break
		}
		cursor = *p.NextCursor
	}
	require.Equal(t, want, seen)
	for i := 1; i < len(order); i++ {
		require.False(t, !order[i-1] && order[i], "cờ phải đứng trước không cờ xuyên các trang")
	}
	_, err := e.svc.Queue(e.ctx, e.course, assignment.QueueFilter{Cursor: "@@"})
	require.Equal(t, apierr.ValidationFailed, code(err))
}

// TestNeedsReviewGeneratedColumn — AC1: needs_review là cột sinh từ flags (không ghi trực tiếp).
func TestNeedsReviewGeneratedColumn(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	now := time.Now().UTC()
	yes := e.rrow(a, e.student(), 1, now, "DONE", "DRAFT", crit10, []string{"INJECTION_SUSPECTED"})
	no := e.rrow(a, e.student(), 1, now, "DONE", "DRAFT", crit10, []string{"SCORE_ADJUSTED"})
	var ny, nn bool
	e.one(&ny, `select needs_review from grading_results where id = $1`, yes.res)
	e.one(&nn, `select needs_review from grading_results where id = $1`, no.res)
	require.True(t, ny)
	require.False(t, nn)
	_, err := e.pool.Exec(e.ctx, `update grading_results set needs_review = true where id = $1`, no.res)
	require.Error(t, err)
}

func (e *env) detail(res uuid.UUID) assignment.ResultView {
	e.t.Helper()
	v, err := e.svc.ResultDetail(e.ctx, e.course, res)
	require.NoError(e.t, err)
	return v
}

func (e *env) edit(res uuid.UUID, in assignment.EditIn) (assignment.ResultView, error) {
	return e.svc.EditResult(e.ctx, e.act, e.course, res, in)
}

func sc(s string) *assignment.Num { n := assignment.Num(s); return &n }

func crit(id, score string) assignment.CriterionEdit {
	return assignment.CriterionEdit{ID: id, Score: sc(score)}
}

func (e *env) draft(flags ...string) (assignment.Assignment, rrow) {
	e.t.Helper()
	a := e.create("BT03")
	return a, e.rrow(a, e.student(), 1, time.Now().UTC().Add(-time.Hour), "DONE", "DRAFT", crit10, flags)
}

// TestEditScoreRange — AC4: ngoài [0; max] → 422 SCORE_OUT_OF_RANGE (max của tiêu chí).
func TestEditScoreRange(t *testing.T) {
	e := newEnv(t)
	_, r := e.draft()
	v := e.detail(r.res).Result.Version
	for _, bad := range []string{"4.25", "-1", "abc", "11"} {
		_, err := e.edit(r.res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit("c1", bad)}})
		require.Equal(t, apierr.ScoreOutOfRange, code(err), bad)
	}
	_, err := e.edit(r.res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit("zz", "1")}})
	require.Equal(t, apierr.CriterionUnknown, code(err))
}

// TestEditScoreStep — AC4: bội 0,25; "2,25" (dấu phẩy) và 2.25 (số JSON) đều nhận.
func TestEditScoreStep(t *testing.T) {
	e := newEnv(t)
	_, r := e.draft()
	v := e.detail(r.res).Result.Version
	_, err := e.edit(r.res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit("c1", "1.1")}})
	require.Equal(t, apierr.ScoreStep, code(err))
	out, err := e.edit(r.res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit("c1", "2,25")}})
	require.NoError(t, err)
	require.Equal(t, "2.25", *out.Criteria[0].Score)
	var in assignment.EditIn
	require.NoError(t, json.Unmarshal([]byte(`{"version":2,"criteria":[{"id":"c1","score":3.5}]}`), &in))
	require.Equal(t, assignment.Num("3.5"), *in.Criteria[0].Score)
}

// TestEditScorePrecision — AC4: quá 2 chữ số thập phân → 422 SCORE_PRECISION (kiểm trước bước).
func TestEditScorePrecision(t *testing.T) {
	e := newEnv(t)
	_, r := e.draft()
	v := e.detail(r.res).Result.Version
	_, err := e.edit(r.res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit("c1", "2.255")}})
	require.Equal(t, apierr.ScorePrecision, code(err))
}

// TestEditRecomputesTotal — AC4: tổng tính lại bằng decimal; tiêu chí đổi mang edited; REVIEWED quay về DRAFT.
func TestEditRecomputesTotal(t *testing.T) {
	e := newEnv(t)
	_, r := e.draft()
	v := e.detail(r.res).Result.Version
	out, err := e.edit(r.res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit("c1", "3.25"), crit("c2", "4.5")}})
	require.NoError(t, err)
	require.Equal(t, "7.75", *out.Result.Total)
	require.True(t, out.Criteria[0].Edited)
	require.Equal(t, v+1, out.Result.Version)
	rv, err := e.svc.Review(e.ctx, e.act, e.course, r.res, out.Result.Version)
	require.NoError(t, err)
	require.Equal(t, "REVIEWED", rv.Result.Status)
	back, err := e.edit(r.res, assignment.EditIn{Version: rv.Result.Version, Criteria: []assignment.CriterionEdit{crit("c2", "5")}})
	require.NoError(t, err)
	require.Equal(t, "DRAFT", back.Result.Status, "sửa bài đã duyệt phải duyệt lại")
	require.Equal(t, "8.25", *back.Result.Total)
}

// TestEditKeepsAiScores — AC4: ai_scores và ai_total_before_late không đổi; có audit_log trước / sau.
func TestEditKeepsAiScores(t *testing.T) {
	e := newEnv(t)
	_, r := e.draft()
	v := e.detail(r.res).Result.Version
	_, err := e.edit(r.res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit("c1", "1")}})
	require.NoError(t, err)
	var ai string
	var aiTotal decimal.Decimal
	e.one(&ai, `select ai_scores::text from grading_results where id = $1`, r.res)
	e.one(&aiTotal, `select ai_total_before_late from grading_results where id = $1`, r.res)
	require.JSONEq(t, `{"c1":"4.00","c2":"6.00"}`, ai)
	require.True(t, aiTotal.Equal(decimal.NewFromInt(10)))
	var n int
	e.one(&n, `select count(*)::int from audit_log where entity = 'grading_results' and entity_id = $1 and action = 'grading_result.edit' and before is not null and after is not null`, r.res.String())
	require.Equal(t, 1, n)
}

// TestEditVersionConflict — AC4: sai version → 409 kèm giá trị của người kia.
func TestEditVersionConflict(t *testing.T) {
	e := newEnv(t)
	_, r := e.draft()
	v := e.detail(r.res).Result.Version
	_, err := e.edit(r.res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit("c1", "1")}})
	require.NoError(t, err)
	_, err = e.edit(r.res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit("c1", "2")}})
	require.Equal(t, apierr.VersionConflict, code(err))
	ae := err.(*apierr.Error) //nolint:errorlint // apierr.Error trả thẳng
	cur := ae.Details.(map[string]any)["current"].(assignment.ResultView)
	require.Equal(t, "1.00", *cur.Criteria[0].Score)
}

// TestEditPublishedRejected — AC4: đã PUBLISHED → 409 RESULT_PUBLISHED (sửa chỉ qua phúc khảo).
func TestEditPublishedRejected(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	r := e.rrow(a, e.student(), 1, time.Now().UTC(), "DONE", "PUBLISHED", crit10, nil)
	_, err := e.edit(r.res, assignment.EditIn{Version: 1, Criteria: []assignment.CriterionEdit{crit("c1", "1")}})
	require.Equal(t, apierr.ResultPublished, code(err))
	_, err = e.svc.Review(e.ctx, e.act, e.course, r.res, 1)
	require.Equal(t, apierr.ResultPublished, code(err))
}

// TestLatePenaltyEditableByStaff — AC4: nhân viên sửa khoản trừ muộn → late_penalty_manual, tổng tính lại; đổi hạn về sau không tính đè.
func TestLatePenaltyEditableByStaff(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	r := e.rrow(a, e.student(), 1, time.Now().UTC().Add(-time.Hour), "DONE", "DRAFT", crit10, nil)
	v := e.detail(r.res).Result.Version
	out, err := e.edit(r.res, assignment.EditIn{Version: v, LatePenalty: sc("1,5")})
	require.NoError(t, err)
	require.True(t, out.Late.Manual)
	require.Equal(t, "1.50", out.Late.Penalty)
	require.Equal(t, "8.50", *out.Result.Total)
	_, err = e.edit(r.res, assignment.EditIn{Version: out.Result.Version, LatePenalty: sc("11")})
	require.Equal(t, apierr.ScoreOutOfRange, code(err))
	require.NoError(t, e.svc.RecomputeLate(e.ctx, e.course, a.ID))
	again := e.detail(r.res)
	require.Equal(t, "1.50", again.Late.Penalty, "khoản trừ sửa tay không bị tính đè")
}

func (e *env) review(res uuid.UUID, v int32) (assignment.ResultView, error) {
	return e.svc.Review(e.ctx, e.act, e.course, res, v)
}

// TestReviewSetsReviewed — AC6: DRAFT → REVIEWED, reviewed_by/at, audit_log.
func TestReviewSetsReviewed(t *testing.T) {
	e := newEnv(t)
	_, r := e.draft()
	v := e.detail(r.res).Result.Version
	out, err := e.review(r.res, v)
	require.NoError(t, err)
	require.Equal(t, "REVIEWED", out.Result.Status)
	require.NotNil(t, out.Result.ReviewedAt)
	var by uuid.UUID
	e.one(&by, `select reviewed_by from grading_results where id = $1`, r.res)
	require.Equal(t, e.teacher, by)
	var n int
	e.one(&n, `select count(*)::int from audit_log where entity = 'grading_results' and entity_id = $1 and action = 'grading_result.review'`, r.res.String())
	require.Equal(t, 1, n)
}

// TestReviewNeedsScores — AC6: bài FAILED chưa có điểm → 422 SCORES_REQUIRED; nhập đủ điểm tay thì duyệt được.
func TestReviewNeedsScores(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	r := e.rrow(a, e.student(), 1, time.Now().UTC(), "FAILED", "DRAFT", `[{"id":"c1","score":null},{"id":"c2","score":null}]`, nil)
	v := e.detail(r.res).Result.Version
	_, err := e.review(r.res, v)
	require.Equal(t, apierr.ScoresRequired, code(err))
	out, err := e.edit(r.res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit("c1", "3"), crit("c2", "5")}})
	require.NoError(t, err)
	require.Equal(t, "DONE", out.Result.RunState, "nhập tay đủ điểm đưa bài FAILED về DONE")
	require.Equal(t, "8.00", *out.Result.Total)
	rv, err := e.review(r.res, out.Result.Version)
	require.NoError(t, err)
	require.Equal(t, "REVIEWED", rv.Result.Status)
}

// TestReviewKeepsFlagsHistory — AC6: duyệt bài còn cờ được phép; cờ giữ lại trong lịch sử.
func TestReviewKeepsFlagsHistory(t *testing.T) {
	e := newEnv(t)
	_, r := e.draft("NEEDS_REVIEW", "INJECTION_SUSPECTED")
	v := e.detail(r.res).Result.Version
	out, err := e.review(r.res, v)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"NEEDS_REVIEW", "INJECTION_SUSPECTED"}, out.Flags)
	var nr bool
	e.one(&nr, `select needs_review from grading_results where id = $1`, r.res)
	require.True(t, nr)
	only := true
	require.Empty(t, e.queue(assignment.QueueFilter{NeedsReview: &only}).Items, "đã duyệt thì không còn trong lọc mặc định Chưa duyệt")
}

// TestReviewIdempotent — AC6: duyệt lại bài đã REVIEWED → 200 như cũ, không đổi version.
func TestReviewIdempotent(t *testing.T) {
	e := newEnv(t)
	_, r := e.draft()
	v := e.detail(r.res).Result.Version
	a1, err := e.review(r.res, v)
	require.NoError(t, err)
	a2, err := e.review(r.res, v)
	require.NoError(t, err)
	require.Equal(t, a1.Result.Version, a2.Result.Version)
	_, err = e.review(r.res, v+50)
	require.NoError(t, err, "đã REVIEWED: idempotent bất kể version")
}

// TestReviewWhileRunningRejected — AC7: QUEUED / RUNNING → 409 NOT_READY cho cả PUT và review.
func TestReviewWhileRunningRejected(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	for _, st := range []string{"QUEUED", "RUNNING"} {
		r := e.rrow(a, e.student(), 1, time.Now().UTC(), st, "DRAFT", `[]`, nil)
		v := e.detail(r.res).Result.Version
		_, err := e.review(r.res, v)
		require.Equal(t, apierr.NotReady, code(err), st)
		_, err = e.edit(r.res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit("c1", "1")}})
		require.Equal(t, apierr.NotReady, code(err), st)
	}
}

// TestReviewSupersededRejected — AC7: sinh viên đã nộp bản mới → 409 SUPERSEDED, bản cũ không duyệt / sửa được.
func TestReviewSupersededRejected(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	stu := e.student()
	r := e.rrow(a, stu, 1, time.Now().UTC(), "DONE", "DRAFT", crit10, nil)
	e.rrow(a, stu, 2, time.Now().UTC().Add(time.Minute), "", "", "", nil)
	v := e.detail(r.res).Result.Version
	require.True(t, e.detail(r.res).Submission.Superseded)
	_, err := e.review(r.res, v)
	require.Equal(t, apierr.Superseded, code(err))
	_, err = e.edit(r.res, assignment.EditIn{Version: v, Criteria: []assignment.CriterionEdit{crit("c1", "1")}})
	require.Equal(t, apierr.Superseded, code(err))
}

var _ = fmt.Sprint

// TestResultDetailBySubmissionID — màn /grading/[submissionId]: id bài nộp trả kết quả hiện hành (round cao nhất); bài chưa chấm → 404.
func TestResultDetailBySubmissionID(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	r := e.rrow(a, e.student(), 1, time.Now().UTC(), "DONE", "DRAFT", crit10, nil)
	e.exec(`insert into grading_results (course_id, submission_id, round, run_state, status, criteria, total) values ($1, $2, 2, 'DONE', 'DRAFT', $3::jsonb, 9)`, e.course, r.sub, crit10)
	v, err := e.svc.ResultDetail(e.ctx, e.course, r.sub)
	require.NoError(t, err)
	require.EqualValues(t, 2, v.Result.Round)
	none := e.rrow(a, e.student(), 1, time.Now().UTC(), "", "", "", nil)
	_, err = e.svc.ResultDetail(e.ctx, e.course, none.sub)
	require.Equal(t, apierr.NotFound, code(err))
}
