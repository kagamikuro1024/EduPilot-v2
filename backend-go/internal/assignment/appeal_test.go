package assignment_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/assignment"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
)

const (
	goodReason = "Em nghĩ ví dụ của em đã đủ ý, nhờ thầy xem lại."
	goodResp   = "Thầy đã xem lại bài của em."
)

func (e *env) stu(id uuid.UUID) assignment.Actor { return assignment.Actor{UserID: id, Student: true} }

// published trả bài đã CÔNG BỐ (4 + 6) và sinh viên của nó.
func (e *env) published(a assignment.Assignment) (rrow, uuid.UUID) {
	e.t.Helper()
	e.exec(`update assignments set published_at = coalesce(published_at, now()) where id = $1`, a.ID)
	r := e.reviewed(a)
	_, err := e.publish1(r.res)
	require.NoError(e.t, err)
	return r, e.studentOf(r)
}

func (e *env) file(stu uuid.UUID, a assignment.Assignment, ids ...string) (assignment.AppealCreated, error) {
	return e.svc.FileAppeal(e.ctx, e.stu(stu), e.course, a.ID, assignment.AppealIn{CriteriaIDs: ids, Reason: goodReason})
}

func (e *env) filed(a assignment.Assignment) (uuid.UUID, rrow, uuid.UUID) {
	e.t.Helper()
	r, stu := e.published(a)
	out, err := e.file(stu, a, "c2")
	require.NoError(e.t, err)
	return out.Appeal.ID, r, stu
}

func (e *env) decide(id uuid.UUID, in assignment.DecideIn) (assignment.DecideOut, error) {
	if in.Version == 0 {
		in.Version = 1
	}
	if in.Response == "" {
		in.Response = goodResp
	}
	return e.svc.DecideAppeal(e.ctx, e.act, e.course, id, in)
}

func adj(c2 string) assignment.DecideIn {
	return assignment.DecideIn{Decision: "ADJUSTED", CriteriaScores: []assignment.ScoreEntry{{ID: "c2", Score: assignment.Num(c2)}}}
}

// TestAppealCreate — AC1: ghi OPEN với score_before, tiêu chí đã chọn, outbox; không ghi escalation_tickets.
func TestAppealCreate(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	id, _, _ := e.filed(a)
	var st, before string
	var ids []string
	e.one(&st, `select status from grade_appeals where id = $1`, id)
	e.one(&before, `select score_before::text from grade_appeals where id = $1`, id)
	e.one(&ids, `select criteria_ids from grade_appeals where id = $1`, id)
	require.Equal(t, "OPEN", st)
	require.Equal(t, "10.00", before)
	require.Equal(t, []string{"c2"}, ids)
	var n int
	e.one(&n, `select count(*)::int from outbox where topic = $1 and payload->>'appeal_id' = $2`, assignment.TopicAppealCreated, id.String())
	require.Equal(t, 1, n)
}

// TestAppealWritesNoEscalationRows — phúc khảo bài tập không đi qua escalation_tickets (kể cả khi quyết định).
func TestAppealWritesNoEscalationRows(t *testing.T) {
	e := newEnv(t)
	id, _, _ := e.filed(e.create("BT03"))
	_, err := e.decide(id, assignment.DecideIn{Decision: "UPHELD"})
	require.NoError(t, err)
	var n int
	e.one(&n, `select count(*)::int from escalation_tickets where course_id = $1`, e.course)
	require.Zero(t, n)
}

// publishedOpen trả bài tập đã công bố + sinh viên có kết quả PUBLISHED.
func (e *env) publishedOpen(code string) (assignment.Assignment, uuid.UUID) {
	a := e.create(code)
	_, stu := e.published(a)
	return a, stu
}

// TestAppealCreatesNoTicket — AC1: gửi phúc khảo không ghi escalation_tickets.
func TestAppealCreatesNoTicket(t *testing.T) {
	e := newEnv(t)
	a, stu := e.publishedOpen("BT03")
	_, err := e.file(stu, a, "c1")
	require.NoError(t, err)
	var n int
	e.one(&n, `select count(*)::int from escalation_tickets where course_id = $1`, e.course)
	require.Zero(t, n)
}

// TestAppealNotPublished — AC2: chưa công bố (REVIEWED, kể cả đã nộp) → 404, không lộ nháp.
func TestAppealNotPublished(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	e.exec(`update assignments set published_at = now() where id = $1`, a.ID)
	r := e.reviewed(a)
	_, err := e.file(e.studentOf(r), a, "c1")
	require.Equal(t, apierr_NotFound, code(err))
}

// TestAppealUnknownCriterion — AC2: tiêu chí ngoài rubric → 422; không chọn tiêu chí nào → 422; không tạo dòng.
func TestAppealUnknownCriterion(t *testing.T) {
	e := newEnv(t)
	a, stu := e.publishedOpen("BT03")
	_, err := e.file(stu, a, "zz")
	require.Equal(t, "CRITERION_UNKNOWN", code(err))
	_, err = e.file(stu, a, "c1", "zz")
	require.Equal(t, "CRITERION_UNKNOWN", code(err))
	_, err = e.file(stu, a)
	require.Equal(t, "VALIDATION_FAILED", code(err))
	var n int
	e.one(&n, `select count(*)::int from grade_appeals where course_id = $1`, e.course)
	require.Zero(t, n)
}

// TestAppealCriteriaSubset — AC1: tập con của rubric (kể cả cả hai tiêu chí; trùng id gộp lại).
func TestAppealCriteriaSubset(t *testing.T) {
	e := newEnv(t)
	a, stu := e.publishedOpen("BT03")
	out, err := e.file(stu, a, "c2", "c1", "c1")
	require.NoError(t, err)
	var ids []string
	e.one(&ids, `select criteria_ids from grade_appeals where id = $1`, out.Appeal.ID)
	require.Equal(t, []string{"c2", "c1"}, ids)
}

// TestAppealReasonBounds — AC1: 10–1.000 ký tự sau khi cắt khoảng trắng và loại ký tự điều khiển.
func TestAppealReasonBounds(t *testing.T) {
	e := newEnv(t)
	a, stu := e.publishedOpen("BT03")
	seq := 10
	for reason, want := range map[string]string{
		strings.Repeat("a", 9):       "REASON_LENGTH",
		"   \x00\x01 chín chữ  \t  ": "REASON_LENGTH",
		strings.Repeat("a", 1001):    "REASON_LENGTH",
		"":                           "REASON_LENGTH",
		strings.Repeat("ạ", 1000):    "",
		"  " + strings.Repeat("b", 10) + "  \x00": "", // đúng 10 sau khi cắt
	} {
		if want == "" { // thành công chỉ một lần mỗi sinh viên: mỗi ca đúng dùng một sinh viên mới
			seq++
			b := e.create(fmt.Sprintf("BT%02d", seq))
			_, s2 := e.published(b)
			_, err := e.svc.FileAppeal(e.ctx, e.stu(s2), e.course, b.ID, assignment.AppealIn{CriteriaIDs: []string{"c1"}, Reason: reason})
			require.NoError(t, err, reason)
			continue
		}
		_, err := e.svc.FileAppeal(e.ctx, e.stu(stu), e.course, a.ID, assignment.AppealIn{CriteriaIDs: []string{"c1"}, Reason: reason})
		require.Equal(t, want, code(err), reason)
	}
	var stored string
	e.one(&stored, `select reason from grade_appeals where reason like 'bbbbbbbbbb%'`)
	require.Equal(t, strings.Repeat("b", 10), stored)
}

// TestAppealAlreadyFiled — AC2: một phúc khảo mỗi bài.
func TestAppealAlreadyFiled(t *testing.T) {
	e := newEnv(t)
	a, stu := e.publishedOpen("BT03")
	_, err := e.file(stu, a, "c1")
	require.NoError(t, err)
	_, err = e.file(stu, a, "c2")
	require.Equal(t, "APPEAL_ALREADY_FILED", code(err))
}

// TestAppealDisabled — AC2: appeal_days = 0 → 409 APPEAL_DISABLED.
func TestAppealDisabled(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	e.exec(`update assignments set appeal_days = 0 where id = $1`, a.ID)
	_, stu := e.published(a)
	_, err := e.file(stu, a, "c1")
	require.Equal(t, "APPEAL_DISABLED", code(err))
}

const apierr_NotFound = "ASSIGNMENT_NOT_FOUND" //nolint:revive,staticcheck // mã lỗi cho dễ đọc

func (e *env) publishedAt() (t time.Time) {
	e.one(&t, `select published_at from grading_results where status = 'PUBLISHED' and course_id = $1 order by published_at limit 1`, e.course)
	return t
}

// TestAppealWindowClosed — AC2: quá hạn → 409 APPEAL_WINDOW_CLOSED kèm closes_at; không tạo dòng.
func TestAppealWindowClosed(t *testing.T) {
	e := newEnv(t)
	a, stu := e.publishedOpen("BT03")
	closes := e.publishedAt().Add(7 * 24 * time.Hour)
	e.svc.Now = func() time.Time { return closes.Add(time.Hour) }
	_, err := e.file(stu, a, "c1")
	require.Equal(t, "APPEAL_WINDOW_CLOSED", code(err))
	ae := err.(*apierr.Error) //nolint:errorlint // trả thẳng
	require.Equal(t, closes, ae.Details.(map[string]any)["closes_at"])
	var n int
	e.one(&n, `select count(*)::int from grade_appeals where course_id = $1`, e.course)
	require.Zero(t, n)
}

// TestAppealWindowBoundaryTable — AC2: hạn tính theo đồng hồ MÁY CHỦ: đúng hạn còn nhận, +1 phút đóng, −1 phút nhận.
func TestAppealWindowBoundaryTable(t *testing.T) {
	for name, c := range map[string]struct {
		off  time.Duration
		want string
	}{"trước hạn 1 phút": {-time.Minute, ""}, "đúng hạn": {0, ""}, "quá hạn 1 phút": {time.Minute, "APPEAL_WINDOW_CLOSED"}} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			a, stu := e.publishedOpen("BT03")
			closes := e.publishedAt().Add(7 * 24 * time.Hour)
			e.svc.Now = func() time.Time { return closes.Add(c.off) }
			_, err := e.file(stu, a, "c1")
			require.Equal(t, c.want, code(err))
		})
	}
}

// TestAppealOwnOnly — AC10: sinh viên khác không có kết quả công bố của bài → 404, không tạo dòng.
func TestAppealOwnOnly(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	r, stu := e.published(a)
	other := e.students[0]
	if other == stu {
		other = e.students[1]
	}
	_, err := e.file(other, a, "c1")
	require.Equal(t, apierr_NotFound, code(err))
	var n int
	e.one(&n, `select count(*)::int from grade_appeals where result_id = $1`, r.res)
	require.Zero(t, n)
}

// TestAppealFileDoubleClick — bấm đúp: đúng một dòng; lần còn lại 409 ALREADY_FILED.
func TestAppealFileDoubleClick(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	_, stu := e.published(a)
	var wg sync.WaitGroup
	codes := make([]string, 5)
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.file(stu, a, "c1")
			codes[i] = code(err)
		}()
	}
	wg.Wait()
	ok, dup := 0, 0
	for _, c := range codes {
		switch c {
		case "":
			ok++
		case "APPEAL_ALREADY_FILED":
			dup++
		}
	}
	require.Equal(t, 1, ok)
	require.Equal(t, 4, dup)
}

// TestAppealDecideUpheld — AC5: giữ nguyên: điểm không đổi, phản hồi + audit + outbox, đóng ngay.
func TestAppealDecideUpheld(t *testing.T) {
	e := newEnv(t)
	id, r, _ := e.filed(e.create("BT03"))
	v0 := e.version(r.res)
	out, err := e.decide(id, assignment.DecideIn{Decision: "UPHELD"})
	require.NoError(t, err)
	require.Equal(t, "UPHELD", out.Status)
	require.Nil(t, out.ScoreAfter)
	var tot decimal.Decimal
	var v int
	e.one(&tot, `select total from grading_results where id = $1`, r.res)
	e.one(&v, `select version from grading_results where id = $1`, r.res)
	require.True(t, tot.Equal(decimal.NewFromInt(10)))
	require.Equal(t, v0, v)
	var n int
	e.one(&n, `select count(*)::int from audit_log where entity = 'grade_appeals' and entity_id = $1 and action = 'appeal.decide'`, id.String())
	require.Equal(t, 1, n)
	e.one(&n, `select count(*)::int from outbox where topic = $1 and payload->>'appeal_id' = $2`, assignment.TopicAppealDecided, id.String())
	require.Equal(t, 1, n)
}

// TestAdjustedUpdatesResultInPlace — AC4: ADJUSTED sửa tiêu chí + tổng trên kết quả đã công bố (version + 1), giữ published_at / ai_scores / PUBLISHED; score_after ghi; cột điểm tính từ kết quả mới.
func TestAdjustedUpdatesResultInPlace(t *testing.T) {
	e := newEnv(t)
	id, r, stu := e.filed(e.create("BT03"))
	var pub time.Time
	e.one(&pub, `select published_at from grading_results where id = $1`, r.res)
	v0 := e.version(r.res)
	out, err := e.decide(id, adj("5.5"))
	require.NoError(t, err)
	require.Equal(t, "9.50", *out.ScoreAfter)
	var tot decimal.Decimal
	var st string
	var v int
	var pub2 time.Time
	e.one(&tot, `select total from grading_results where id = $1`, r.res)
	e.one(&st, `select status from grading_results where id = $1`, r.res)
	e.one(&v, `select version from grading_results where id = $1`, r.res)
	e.one(&pub2, `select published_at from grading_results where id = $1`, r.res)
	require.True(t, tot.Equal(decimal.RequireFromString("9.5")))
	require.Equal(t, "PUBLISHED", st)
	require.Equal(t, v0+1, v)
	require.True(t, pub.Equal(pub2))
	var after, before string
	e.one(&after, `select score_after::text from grade_appeals where id = $1`, id)
	e.one(&before, `select score_before::text from grade_appeals where id = $1`, id)
	require.Equal(t, "9.50", after)
	require.Equal(t, "10.00", before)
	m, err := e.svc.Mine(e.ctx, e.stu(stu), e.course, mustAsg(e, r))
	require.NoError(t, err)
	require.Equal(t, "ADJUSTED", m.Result.Appeal.State)
	require.Equal(t, "9.50", m.Result.Total)
}

func (e *env) version(res uuid.UUID) (v int) {
	e.one(&v, `select version from grading_results where id = $1`, res)
	return v
}

func mustAsg(e *env, r rrow) (a uuid.UUID) {
	e.one(&a, `select assignment_id from submissions where id = $1`, r.sub)
	return a
}

// TestAppealResponseRequired — AC3: phản hồi 10–2.000 ký tự bắt buộc cho cả hai quyết định.
func TestAppealResponseRequired(t *testing.T) {
	e := newEnv(t)
	id, _, _ := e.filed(e.create("BT03"))
	for _, resp := range []string{"ngắn", "          ", strings.Repeat("a", 2001)} {
		for _, d := range []assignment.DecideIn{{Decision: "UPHELD"}, adj("5")} {
			_, err := e.svc.DecideAppeal(e.ctx, e.act, e.course, id, assignment.DecideIn{Decision: d.Decision, CriteriaScores: d.CriteriaScores, Response: resp, Version: 1})
			require.Equal(t, "RESPONSE_REQUIRED", code(err), resp)
		}
	}
	var st string
	e.one(&st, `select status from grade_appeals where id = $1`, id)
	require.Equal(t, "OPEN", st)
}

// TestAppealAdjustedNeedsChange — AC3: ADJUSTED không đổi tiêu chí nào 422 NO_CHANGE; điểm vượt max / lệch bước 422; kết quả không đổi.
func TestAppealAdjustedNeedsChange(t *testing.T) {
	e := newEnv(t)
	id, r, _ := e.filed(e.create("BT03"))
	v0 := e.version(r.res)
	_, err := e.decide(id, assignment.DecideIn{Decision: "ADJUSTED"})
	require.Equal(t, "NO_CHANGE", code(err))
	_, err = e.decide(id, adj("6"))
	require.Equal(t, "NO_CHANGE", code(err))
	_, err = e.decide(id, adj("7"))
	require.Equal(t, "SCORE_OUT_OF_RANGE", code(err))
	_, err = e.decide(id, adj("5.1"))
	require.Equal(t, "SCORE_STEP", code(err))
	require.Equal(t, v0, e.version(r.res))
	_, err = e.decide(id, assignment.DecideIn{Decision: "UPHELD", Version: 9})
	require.Equal(t, "VERSION_CONFLICT", code(err))
}

// TestAppealClosedOnDecide — AC3: quyết xong là đóng: không quyết lại (409 APPEAL_ALREADY_DECIDED), sinh viên thấy trạng thái + phản hồi.
func TestAppealClosedOnDecide(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	id, r, stu := e.filed(a)
	_, err := e.decide(id, assignment.DecideIn{Decision: "UPHELD"})
	require.NoError(t, err)
	_, err = e.decide(id, assignment.DecideIn{Decision: "UPHELD", Version: 2})
	require.Equal(t, "APPEAL_ALREADY_DECIDED", code(err))
	m, err := e.svc.Mine(e.ctx, e.stu(stu), e.course, mustAsg(e, r))
	require.NoError(t, err)
	require.Equal(t, "UPHELD", m.Result.Appeal.State)
	require.Equal(t, goodResp, *m.Result.Appeal.Response)
}

// TestAppealDecideAdjusted — AC3: ADJUSTED trả score_after và phản hồi; nhân viên thấy điểm trước / sau.
func TestAppealDecideAdjusted(t *testing.T) {
	e := newEnv(t)
	id, _, _ := e.filed(e.create("BT03"))
	out, err := e.decide(id, adj("5.5"))
	require.NoError(t, err)
	require.Equal(t, "ADJUSTED", out.Status)
	v, err := e.svc.GetAppeal(e.ctx, e.course, id)
	require.NoError(t, err)
	require.Equal(t, "10.00", v.ScoreBefore)
	require.Equal(t, "9.50", *v.ScoreAfter)
	require.Equal(t, goodResp, *v.Response)
}

// TestAppealDecideConcurrent — hai giảng viên quyết cùng lúc: đúng một thắng, bên còn lại 409; kết quả chỉ đổi một lần.
func TestAppealDecideConcurrent(t *testing.T) {
	e := newEnv(t)
	id, r, _ := e.filed(e.create("BT03"))
	v0 := e.version(r.res)
	var wg sync.WaitGroup
	codes := make([]string, 4)
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.decide(id, adj("5"))
			codes[i] = code(err)
		}()
	}
	wg.Wait()
	ok := 0
	for _, c := range codes {
		if c == "" {
			ok++
		} else {
			require.Contains(t, []string{"APPEAL_ALREADY_DECIDED", "VERSION_CONFLICT"}, c)
		}
	}
	require.Equal(t, 1, ok)
	require.Equal(t, v0+1, e.version(r.res))
}

// TestAppealDecideWhenGradebookLocked — AC8: sổ điểm LOCKED không chặn quyết định phúc khảo.
func TestAppealDecideWhenGradebookLocked(t *testing.T) {
	e := newEnv(t)
	id, _, _ := e.filed(e.create("BT03"))
	e.exec(`insert into gradebooks (course_id, status, snapshot_version) values ($1, 'LOCKED', 1) on conflict (course_id) do update set status = 'LOCKED', snapshot_version = 1`, e.course)
	_, err := e.decide(id, adj("5"))
	require.NoError(t, err)
}

// TestStatsAppealRates — tỉ lệ phúc khảo / sửa điểm của bài.
func TestStatsAppealRates(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	id1, _, _ := e.filedOn(a)
	id2, _, _ := e.filedOn(a)
	_, err := e.decide(id1, adj("5"))
	require.NoError(t, err)
	_, err = e.decide(id2, assignment.DecideIn{Decision: "UPHELD"})
	require.NoError(t, err)
	s, err := e.svc.Stats(e.ctx, e.course, a.ID)
	require.NoError(t, err)
	require.Equal(t, 2, s.Appealed)
	require.Equal(t, 1, s.Adjusted)
	require.True(t, decimal.RequireFromString(*s.AppealSuccessRate).Equal(decimal.RequireFromString("0.5")))
}

// TestStatsNullWhenZero — AC8: chưa có phúc khảo → tỉ lệ null (không chia cho 0).
func TestStatsNullWhenZero(t *testing.T) {
	e := newEnv(t)
	a, _ := e.publishedOpen("BT03")
	s, err := e.svc.Stats(e.ctx, e.course, a.ID)
	require.NoError(t, err)
	require.Zero(t, s.Appealed)
	require.Nil(t, s.AppealSuccessRate)
	_, err = e.svc.FileAppeal(e.ctx, e.stu(e.studentOfPublished()), e.course, a.ID, assignment.AppealIn{CriteriaIDs: []string{"c1"}, Reason: goodReason})
	require.NoError(t, err)
	s, err = e.svc.Stats(e.ctx, e.course, a.ID)
	require.NoError(t, err)
	require.Equal(t, 1, s.Appealed)
	require.Zero(t, s.Adjusted)
	require.NotNil(t, s.AppealRate)
	require.True(t, decimal.RequireFromString(*s.AppealSuccessRate).IsZero(), "adjusted ÷ appealed = 0, không phải null")
}

func (e *env) studentOfPublished() (u uuid.UUID) {
	e.one(&u, `select s.student_id from grading_results r join submissions s on s.id = r.submission_id where r.course_id = $1 and r.status = 'PUBLISHED' limit 1`, e.course)
	return u
}

// filedOn gửi phúc khảo trên bài a cho một sinh viên mới của lớp.
func (e *env) filedOn(a assignment.Assignment) (uuid.UUID, rrow, uuid.UUID) {
	e.t.Helper()
	stu := e.user("STUDENT")
	e.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, e.course, stu)
	e.exec(`update assignments set published_at = coalesce(published_at, now()) where id = $1`, a.ID)
	r := e.rrow(a, stu, 1, time.Now().UTC().Add(-time.Hour), "DONE", "REVIEWED", crit10, nil)
	_, err := e.publish1(r.res)
	require.NoError(e.t, err)
	out, err := e.file(stu, a, "c2")
	require.NoError(e.t, err)
	return out.Appeal.ID, r, stu
}

// decidedWorld: một phúc khảo đã gửi + quyết (ADJUSTED 10 → 9), sinh viên có email đã xác minh.
func (e *env) decidedWorld() (id, stu uuid.UUID, msg func(topic string) outbox.Message) {
	e.t.Helper()
	a := e.create("BT03")
	id, _, stu = e.filed(a)
	e.exec(`update users set email_verified_at = now() where id = $1`, stu)
	msg = func(topic string) outbox.Message {
		var payload []byte
		e.one(&payload, `select payload from outbox where topic = $1 and payload->>'appeal_id' = $2`, topic, id.String())
		return outbox.Message{ID: uuid.New(), Topic: topic, Payload: payload}
	}
	_, err := e.decide(id, adj("5"))
	require.NoError(e.t, err)
	return id, stu, msg
}

// TestAppealReplyNotifiesStudent — AC6: chuông APPEAL_REPLIED + thư tới ĐÚNG sinh viên, khử trùng khi chạy lại.
func TestAppealReplyNotifiesStudent(t *testing.T) {
	e := newEnv(t)
	id, stu, msg := e.decidedWorld()
	for range 2 {
		require.NoError(t, e.svc.HandleAppealDecided(e.ctx, msg(assignment.TopicAppealDecided)))
	}
	var n int
	e.one(&n, `select count(*)::int from notifications where user_id = $1 and type = 'APPEAL_REPLIED'`, stu)
	require.Equal(t, 1, n)
	e.one(&n, `select count(*)::int from notifications where type = 'APPEAL_REPLIED' and user_id <> $1`, stu)
	require.Zero(t, n)
	e.one(&n, `select count(*)::int from mail_outbox where template = 'appeal_reply' and dedupe_key = $1`, "appeal_reply:"+id.String())
	require.Equal(t, 1, n)
}

// TestAppealReplyMailContent — AC6: thư có quyết định, điểm cũ → mới, phản hồi NGUYÊN VĂN; không nhắc AI.
func TestAppealReplyMailContent(t *testing.T) {
	e := newEnv(t)
	_, _, msg := e.decidedWorld()
	require.NoError(t, e.svc.HandleAppealDecided(e.ctx, msg(assignment.TopicAppealDecided)))
	sender := &capSender{}
	e.sendAll(e.mailHandler(sender))
	require.Len(t, sender.sent, 1)
	body := sender.sent[0].Text
	require.Contains(t, body, "Đã sửa điểm")
	require.Contains(t, body, "10,00 → 9,00")
	require.Contains(t, body, goodResp)
	require.NotContains(t, strings.ToLower(body), " ai ")
}

// TestAppealBellAggregatedForTeacher — AC6: chuông gộp cho TEACHER ("2 yêu cầu xem lại mới"), một dòng; TA không nhận; hết phúc khảo OPEN thì dòng gộp được dọn.
func TestAppealBellAggregatedForTeacher(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	var ids []uuid.UUID
	for range 2 {
		id, _, _ := e.filedOn(a)
		ids = append(ids, id)
	}
	for _, id := range ids {
		var payload []byte
		e.one(&payload, `select payload from outbox where topic = $1 and payload->>'appeal_id' = $2`, assignment.TopicAppealCreated, id.String())
		require.NoError(t, e.svc.HandleAppealCreated(e.ctx, outbox.Message{ID: uuid.New(), Topic: assignment.TopicAppealCreated, Payload: payload}))
	}
	var n int
	var title string
	e.one(&n, `select count(*)::int from notifications where user_id = $1 and type = 'APPEAL_NEW'`, e.teacher)
	e.one(&title, `select title from notifications where user_id = $1 and type = 'APPEAL_NEW'`, e.teacher)
	require.Equal(t, 1, n)
	require.Equal(t, "2 yêu cầu xem lại mới", title)
	e.one(&n, `select count(*)::int from notifications where type = 'APPEAL_NEW' and user_id <> $1`, e.teacher)
	require.Zero(t, n)
}

// llmSpy đếm lời gọi LLM có cấu trúc: đường phúc khảo phải để số này bằng 0.
type llmSpy struct{ calls atomic.Int32 }

func (s *llmSpy) Structured(context.Context, llm.Request, json.RawMessage) (json.RawMessage, error) {
	s.calls.Add(1)
	return nil, errors.New("không được gọi LLM")
}

// TestAppealPathNeverCallsLLM — AC5: gửi, xem, quyết định, thông báo đều không gọi LLM; không chấm lại tự động (không có job grading mới).
func TestAppealPathNeverCallsLLM(t *testing.T) {
	e := newEnv(t)
	spy := &llmSpy{}
	e.svc.LLM = spy
	a := e.create("BT03")
	id, r, stu := e.filed(a)
	_, err := e.svc.GetAppeal(e.ctx, e.course, id)
	require.NoError(t, err)
	_, err = e.svc.ListAppeals(e.ctx, e.course, nil, "", httpx.PageParams{Limit: 50})
	require.NoError(t, err)
	_, err = e.svc.ResultDetail(e.ctx, e.course, r.res)
	require.NoError(t, err)
	_, err = e.decide(id, adj("5"))
	require.NoError(t, err)
	var payload []byte
	e.one(&payload, `select payload from outbox where topic = $1 and payload->>'appeal_id' = $2`, assignment.TopicAppealDecided, id.String())
	require.NoError(t, e.svc.HandleAppealDecided(e.ctx, outbox.Message{ID: uuid.New(), Topic: assignment.TopicAppealDecided, Payload: payload}))
	_, err = e.svc.Mine(e.ctx, e.stu(stu), e.course, a.ID)
	require.NoError(t, err)
	require.Zero(t, spy.calls.Load())
	var n int
	e.one(&n, `select count(*)::int from grading_results where submission_id = $1`, r.sub)
	require.Equal(t, 1, n, "không chấm lại tự động")
}

// TestAppealNotInAgentTools — AC5: không tool agent nào đọc phúc khảo.
func TestAppealNotInAgentTools(t *testing.T) {
	reg := agent.DefaultPrivateRegistry(nil, nil, nil, nil, nil)
	for _, n := range reg.Names() {
		require.NotContains(t, strings.ToLower(n), "appeal")
	}
	for _, tl := range reg.Tools() {
		require.NotContains(t, strings.ToLower(tl.ArgsType().String()), "appeal")
	}
}

// TestAppealReasonNotIndexed — AC5: lý do của sinh viên không vào đoạn tài liệu (RAG), tin nhắn chat, hay bài Threads.
func TestAppealReasonNotIndexed(t *testing.T) {
	e := newEnv(t)
	a, stu := e.publishedOpen("BT03")
	secret := "LYDO-" + uuid.NewString()
	_, err := e.svc.FileAppeal(e.ctx, e.stu(stu), e.course, a.ID, assignment.AppealIn{CriteriaIDs: []string{"c1"}, Reason: secret + " em xin xem lại"})
	require.NoError(t, err)
	for _, q := range []string{`select count(*)::int from content_chunks where text like '%' || $1 || '%'`, `select count(*)::int from chat_messages where content like '%' || $1 || '%'`,
		`select count(*)::int from outbox where payload::text like '%' || $1 || '%'`} {
		var n int
		e.one(&n, q, secret)
		require.Zero(t, n, q)
	}
}

// post gọi một route POST với khoá idempotency CỐ ĐỊNH (gửi lại cùng khoá).
func (h *httpEnv) post(path string, uid uuid.UUID, role auth.Role, key, body string) (int, string) {
	h.t.Helper()
	tok, err := auth.NewIssuer(asgSecret, time.Hour, clock.Real{}).Issue(uid.String(), role, "")
	require.NoError(h.t, err)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Idempotency-Key", key)
	w := httptest.NewRecorder()
	h.h.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

// TestAppealIdempotent — AC1: gửi lại cùng Idempotency-Key → cùng phản hồi, đúng một phúc khảo + một outbox.
func TestAppealIdempotent(t *testing.T) {
	h := newHTTPEnv(t)
	a, stu := h.publishedOpen("BT03")
	path := "/api/v1/courses/" + h.course.String() + "/assignments/" + a.ID.String() + "/appeal"
	body := `{"criteria_ids":["c1"],"reason":"` + goodReason + `"}`
	key := uuid.NewString()
	c1, b1 := h.post(path, stu, auth.RoleStudent, key, body)
	c2, b2 := h.post(path, stu, auth.RoleStudent, key, body)
	require.Equal(t, http.StatusCreated, c1)
	require.Equal(t, c1, c2)
	require.Equal(t, b1, b2)
	var n int
	h.one(&n, `select count(*)::int from grade_appeals where course_id = $1`, h.course)
	require.Equal(t, 1, n)
	h.one(&n, `select count(*)::int from outbox where topic = $1`, assignment.TopicAppealCreated)
	require.Equal(t, 1, n)
}

// TestAppealDecideIdempotent — AC3: gửi lại cùng khoá → cùng phản hồi, kết quả chỉ đổi một lần.
func TestAppealDecideIdempotent(t *testing.T) {
	h := newHTTPEnv(t)
	id, r, _ := h.filed(h.create("BT03"))
	v0 := h.version(r.res)
	path := "/api/v1/courses/" + h.course.String() + "/appeals/" + id.String() + "/decide"
	body := `{"decision":"ADJUSTED","response":"` + goodResp + `","version":1,"criteria_scores":[{"id":"c2","score":"5"}]}`
	key := uuid.NewString()
	c1, b1 := h.post(path, h.teacher, auth.RoleTeacher, key, body)
	c2, b2 := h.post(path, h.teacher, auth.RoleTeacher, key, body)
	require.Equal(t, http.StatusOK, c1)
	require.Equal(t, c1, c2)
	require.Equal(t, b1, b2)
	require.Equal(t, v0+1, h.version(r.res))
}

// TestAppealsRBACMatrix — AC10: gửi chỉ STUDENT; đọc TA + TEACHER; quyết định chỉ TEACHER; ADMIN và lớp khác 403.
func TestAppealsRBACMatrix(t *testing.T) {
	h := newHTTPEnv(t)
	a := h.create("BT03")
	id, _, stu := h.filed(a)
	root := "/api/v1/courses/" + h.course.String()
	type c struct {
		id   uuid.UUID
		role auth.Role
	}
	staffNo := map[string]c{"sinh viên": {stu, auth.RoleStudent}, "admin": {h.adm, auth.RoleAdmin}, "giảng viên lớp khác": {h.other, auth.RoleTeacher}}
	for name, who := range staffNo {
		require.Equal(t, http.StatusForbidden, h.call(http.MethodGet, root+"/appeals", who.id, who.role, ""), name)
		require.Equal(t, http.StatusForbidden, h.call(http.MethodGet, root+"/appeals/"+id.String(), who.id, who.role, ""), name)
	}
	for name, who := range map[string]c{"ta": {h.ta, auth.RoleTA}, "giảng viên": {h.teacher, auth.RoleTeacher}} {
		require.Equal(t, http.StatusOK, h.call(http.MethodGet, root+"/appeals", who.id, who.role, ""), name)
		require.Equal(t, http.StatusOK, h.call(http.MethodGet, root+"/appeals/"+id.String(), who.id, who.role, ""), name)
	}
	body := `{"decision":"UPHELD","response":"` + goodResp + `","version":1}`
	for name, who := range map[string]c{"ta": {h.ta, auth.RoleTA}, "sinh viên": {stu, auth.RoleStudent}, "admin": {h.adm, auth.RoleAdmin}, "giảng viên lớp khác": {h.other, auth.RoleTeacher}} {
		require.Equal(t, http.StatusForbidden, h.call(http.MethodPost, root+"/appeals/"+id.String()+"/decide", who.id, who.role, body), name)
	}
	file := root + "/assignments/" + a.ID.String() + "/appeal"
	fb := `{"criteria_ids":["c1"],"reason":"` + goodReason + `"}`
	for name, who := range map[string]c{"ta": {h.ta, auth.RoleTA}, "giảng viên": {h.teacher, auth.RoleTeacher}, "admin": {h.adm, auth.RoleAdmin}} {
		require.Equal(t, http.StatusForbidden, h.call(http.MethodPost, file, who.id, who.role, fb), name)
	}
	require.Equal(t, http.StatusOK, h.call(http.MethodPost, root+"/appeals/"+id.String()+"/decide", h.teacher, auth.RoleTeacher, body))
}
