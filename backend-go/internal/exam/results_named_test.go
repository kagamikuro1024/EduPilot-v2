package exam_test

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/today"
)

// Các test dưới đây là các ca có tên cố định của AC US-PE-08 (mỗi ca một ý, nhỏ); các ca tổng hợp ở `grade_test.go`.

func routeNo(n int) exam.Route {
	for _, rt := range exam.Routes() {
		if rt.No == n {
			return rt
		}
	}
	panic(fmt.Sprintf("không có thao tác #%d", n))
}

// onlyTeacher: route chỉ Giảng viên ACTIVE đi qua (TA, sinh viên, người ngoài lớp, Admin → 403).
func onlyTeacher(t *testing.T, no int) {
	t.Helper()
	rt := routeNo(no)
	for _, c := range []struct {
		jwt auth.Role
		m   auth.Membership
	}{
		{auth.RoleTA, auth.Membership{Found: true, Role: auth.RoleTA, Status: "ACTIVE"}},
		{auth.RoleStudent, auth.Membership{Found: true, Role: auth.RoleStudent, Status: "ACTIVE"}},
		{auth.RoleTeacher, auth.Membership{}},
		{auth.RoleAdmin, auth.Membership{Found: true, Role: auth.RoleTeacher, Status: "ACTIVE"}},
	} {
		code, _ := serve(t, rt, c.jwt, c.m)
		require.Equal(t, http.StatusForbidden, code, fmt.Sprintf("#%d · %s", no, c.jwt))
	}
	code, _ := serve(t, rt, auth.RoleTeacher, auth.Membership{Found: true, Role: auth.RoleTeacher, Status: "ACTIVE"})
	require.Equal(t, http.StatusNoContent, code)
}

// ---- AC1 ---------------------------------------------------------------------------------------------------------------

// TestCloseBlocksNewWrites — AC1: từ lúc CLOSED không bắt đầu / lưu / nộp mới (409 EXAM_NOT_OPEN / ATTEMPT_CLOSED…).
func TestCloseBlocksNewWrites(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e, vs, svs, clk := r.closingExam(1)
	r.exec(`update exams set publish_hold = true where id=$1`, e.ID)
	clk.Advance(11 * time.Minute)
	r.tick(clk)
	svc := r.at(clk)
	_, _, err := svc.StartAttempt(t.Context(), r.student("ACTIVE"), r.course, e.ID, uuid.New())
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "EXAM_NOT_OPEN", code)
	_, err = svc.SaveAnswers(t.Context(), svs[0], r.course, e.ID, vs[0].Attempt.ID, uuid.New(), exam.AnswersIn{Items: []exam.AnswerIn{pick(vs[0].Items[0], 0)}})
	st, _ = apiStatus(t, err)
	require.Equal(t, 409, st)
	_, err = svc.SubmitAttempt(t.Context(), svs[0], r.course, e.ID, vs[0].Attempt.ID, uuid.New())
	st, _ = apiStatus(t, err)
	require.Equal(t, 409, st)
}

// TestClosedOutboxOnce — AC1: `exam.closed` đúng một lần dù tick chạy nhiều lần / song song.
func TestClosedOutboxOnce(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	d := r.scheduled("đóng một lần")
	r.exec(`update exams set publish_hold = true where id=$1`, d.ID)
	clk := clock.NewFake(time.Now().UTC().Add(48 * time.Hour))
	for range 3 {
		r.tick(clk)
	}
	require.Equal(t, "CLOSED", r.examStatus(d.ID))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.closed' and payload->>'exam_id'=$1`, d.ID.String()))
}

// ---- AC2 / AC3: chấm ---------------------------------------------------------------------------------------------------

// TestGradeMCQSync — AC2: bài chỉ trắc nghiệm chấm NGAY lúc nộp (không đợi tick / máy chấm).
func TestGradeMCQSync(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("trắc nghiệm", false, r.mcqSet(2)...)
	v, _ := r.start(e, sv, tab)
	_, err := r.save(v, sv, tab, answerCorrect(v.Items[0]))
	require.NoError(t, err)
	_, err = r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
	require.NoError(t, err)
	st, _, sc, _ := r.attemptRow(v.Attempt.ID)
	require.Equal(t, "GRADED", st)
	require.Equal(t, "5.00", *sc)
}

// TestGradeCodeWaitsForJudge — AC2: có bản nộp code QUEUED → GRADING; chỉ khi `DONE` mới GRADED.
func TestGradeCodeWaitsForJudge(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := m.finish(true, true)
	require.Equal(t, "GRADING", r.attemptStatus(m.v.Attempt.ID))
	r.exec(`update code_submissions set status='RUNNING', lease_until = now() + interval '1 minute' where id=$1`, sub)
	require.NoError(t, r.svc.OnSubmissionDone(t.Context(), r.course, m.v.Attempt.ID))
	require.Equal(t, "GRADING", r.attemptStatus(m.v.Attempt.ID), "RUNNING chưa xong")
	m.judged(sub, "AC")
	require.NoError(t, r.svc.OnSubmissionDone(t.Context(), r.course, m.v.Attempt.ID))
	require.Equal(t, "GRADED", r.attemptStatus(m.v.Attempt.ID))
}

// TestNoSubmissionNoDraftZero — AC2: câu code không có bản nộp và không có nháp → earned 0 (không phải lỗi), chấm xong ngay.
func TestNoSubmissionNoDraftZero(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	st, _, sc, _ := r.attemptRow(m.v.Attempt.ID)
	require.Equal(t, "GRADED", st)
	require.Equal(t, "5.00", *sc, "MCQ 1/1 + code 0/1")
	require.Equal(t, 1, r.count(`select count(*) from exam_attempts a, jsonb_array_elements(a.breakdown) b where a.id=$1 and b->>'earned' = '0'`, m.v.Attempt.ID))
}

// TestComputeScoreStores — AC3: `breakdown` lưu `earned` ĐỦ chữ số (không làm tròn trung gian), `max`, ảnh chụp code; `auto_score` làm tròn một lần từ tổng chưa làm tròn.
func TestComputeScoreStores(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := m.finish(true, true)
	m.judged(sub, "WA") // passed 2/3
	require.NoError(t, r.svc.OnSubmissionDone(t.Context(), r.course, m.v.Attempt.ID))
	var raw []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select breakdown from exam_attempts where id=$1`, m.v.Attempt.ID).Scan(&raw))
	var bd []map[string]any
	require.NoError(t, json.Unmarshal(raw, &bd))
	require.Equal(t, "0.6666666666666667", bd[1]["earned"])
	require.Equal(t, "1.00", bd[1]["max"])
	require.NotNil(t, bd[1]["code"])
	_, _, sc, _ := r.attemptRow(m.v.Attempt.ID)
	require.Equal(t, "8.33", *sc)
}

// TestRecomputeIdempotent — AC3: tính lại cùng dữ liệu → cùng kết quả, không thông báo, lượt không rời GRADED.
func TestRecomputeIdempotent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := m.finish(true, true)
	m.judged(sub, "WA")
	require.NoError(t, r.svc.OnSubmissionDone(t.Context(), r.course, m.v.Attempt.ID))
	_, _, before, _ := r.attemptRow(m.v.Attempt.ID)
	for range 3 {
		ok, err := r.svc.RecomputeAttempt(t.Context(), r.course, m.v.Attempt.ID)
		require.NoError(t, err)
		require.True(t, ok)
	}
	st, _, after, _ := r.attemptRow(m.v.Attempt.ID)
	require.Equal(t, "GRADED", st)
	require.Equal(t, *before, *after)
	require.Zero(t, r.count(`select count(*) from outbox where topic='exam.regraded'`))
}

// regradeEnv: bài đã chấm + công bố, bộ test đổi, `regrade` đã chạy (bản nộp về QUEUED). Trả bản nộp.
func (r *rig) regradeEnv(m *mixEnv) uuid.UUID {
	r.t.Helper()
	sub := m.finish(true, true)
	m.judged(sub, "WA")
	require.NoError(r.t, r.svc.OnSubmissionDone(r.t.Context(), r.course, m.v.Attempt.ID))
	r.published(m)
	r.exec(`update code_problems set tests_version = tests_version + 1 where question_id = (select question_id from exam_items where id=$1)`, m.codeI)
	jid, err := r.svc.Regrade(r.t.Context(), r.teacher, r.course, m.e.ID, exam.RegradeIn{Scope: "all", Reason: "test đổi"})
	require.NoError(r.t, err)
	require.Equal(r.t, "SUCCEEDED", runJob(r.t, r, newRunner(r.t, r, &exam.Worker{}), jid).Status)
	return sub
}

// TestRegradeKeepsOldScoreUntilDone — AC3/AC15: trong lúc chấm lại lượt GIỮ GRADED + điểm cũ.
func TestRegradeKeepsOldScoreUntilDone(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	r.regradeEnv(m)
	st, _, sc, _ := r.attemptRow(m.v.Attempt.ID)
	require.Equal(t, "GRADED", st)
	require.Equal(t, "8.33", *sc)
}

// TestRegradingFlagClearsWhenAllDone — AC3: `regrading` gỡ khi MỌI bản nộp xong và mọi lượt đã tính lại (không phải khi việc xếp hàng xong).
func TestRegradingFlagClearsWhenAllDone(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := r.regradeEnv(m)
	r.ageExam(m.e.ID)
	_, err := r.svc.RegradeSweep(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*)::int from exams where id=$1 and regrading`, m.e.ID), "bản nộp còn QUEUED")
	m.judged(sub, "AC")
	require.NoError(t, r.svc.OnSubmissionDone(t.Context(), r.course, m.v.Attempt.ID))
	_, err = r.svc.RegradeSweep(t.Context())
	require.NoError(t, err)
	require.Zero(t, r.count(`select count(*)::int from exams where id=$1 and regrading`, m.e.ID))
}

// TestResultPageReadsBreakdownDuringRegrade — AC3/AC6: trang kết quả đọc từ `breakdown` — không trống khi bản nộp đang bị chấm lại.
func TestResultPageReadsBreakdownDuringRegrade(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	r.regradeEnv(m)
	res, err := r.svc.AttemptResult(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.NoError(t, err)
	require.Equal(t, "8.33", *res.Score)
	require.Equal(t, &exam.ResultHiddenView{Passed: 1, Total: 1}, res.Items[1].Hidden)
	require.Len(t, res.Items[1].Samples, 1)
}

// ---- AC4 / AC5: công bố ------------------------------------------------------------------------------------------------

// TestAutoPublishWaitsForRegrade — AC4: `regrading` → chưa công bố; hết chấm lại → công bố.
func TestAutoPublishWaitsForRegrade(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set regrading = true where id=$1`, m.e.ID)
	n, err := r.svc.PublishDue(t.Context())
	require.NoError(t, err)
	require.Zero(t, n)
	r.exec(`update exams set regrading = false where id=$1`, m.e.ID)
	n, err = r.svc.PublishDue(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, n)
}

// TestPublishHoldBlocksAuto — AC5: hoãn → tick không công bố (nhiều vòng).
func TestPublishHoldBlocksAuto(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	for range 3 {
		n, err := r.svc.PublishDue(t.Context())
		require.NoError(t, err)
		require.Zero(t, n)
	}
	require.Equal(t, "CLOSED", r.examStatus(m.e.ID))
}

// TestReleaseHoldPublishes — AC5: bỏ hoãn khi đã chấm xong → công bố ngay, một sự kiện.
func TestReleaseHoldPublishes(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	d, err := r.svc.GetExam(t.Context(), r.course, m.e.ID)
	require.NoError(t, err)
	out, err := r.svc.SetHold(t.Context(), r.teacher, r.course, m.e.ID, false, d.Version)
	require.NoError(t, err)
	require.Equal(t, "PUBLISHED", out.Status)
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.published' and payload->>'exam_id'=$1`, m.e.ID.String()))
}

// TestHoldAfterPublished409 — AC5: sau PUBLISHED không hoãn được nữa (409 EXAM_LOCKED).
func TestHoldAfterPublished409(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	r.published(m)
	d, err := r.svc.GetExam(t.Context(), r.course, m.e.ID)
	require.NoError(t, err)
	_, err = r.svc.SetHold(t.Context(), r.teacher, r.course, m.e.ID, true, d.Version)
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "EXAM_LOCKED", code)
}

// TestHoldTeacherOnly — AC5: #26 chỉ Giảng viên (TA / sinh viên / người ngoài / Admin → 403).
func TestHoldTeacherOnly(t *testing.T) { t.Parallel(); onlyTeacher(t, 26) }

// ---- AC6 / AC7: kết quả sinh viên --------------------------------------------------------------------------------------

func (r *rig) publishedMix(reveal bool) *mixEnv {
	r.t.Helper()
	m := r.mixExam()
	sub := m.finish(true, true)
	m.judged(sub, "WA")
	require.NoError(r.t, r.svc.OnSubmissionDone(r.t.Context(), r.course, m.v.Attempt.ID))
	r.exec(`update exams set reveal_answers=$2 where id=$1`, m.e.ID, reveal)
	r.published(m)
	return m
}

func keysOfJSON(t *testing.T, v any) []string {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &m))
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// TestResultPayloadWhitelist — AC6: khoá JSON của kết quả đúng danh sách cho phép; không có trọng số, `override`, lời giải mẫu, tên / input / expected test ẩn, nhật ký liêm chính.
func TestResultPayloadWhitelist(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.publishedMix(true)
	res, err := r.svc.AttemptResult(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"appeal", "exam", "items", "score", "score_adjusted"}, keysOfJSON(t, res))
	code := keysOfJSON(t, res.Items[1])
	require.Equal(t, []string{"answer", "compile_log", "correct", "earned", "explanation", "final_submission", "hidden", "item_id", "max", "mine", "options", "overridden", "position", "samples", "stem", "type"}, code)
	raw, _ := json.Marshal(res)
	for _, bad := range []string{"weight", "answer_key", "override\"", "reference", "tests_version", "integrity", "similarity", "tab_hidden", canary} {
		require.NotContains(t, string(raw), bad)
	}
}

// TestResultRevealAnswersToggle — AC6: `reveal_answers=false` ẩn `answer` + `explanation`; `correct` / `earned` luôn có.
func TestResultRevealAnswersToggle(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	for _, reveal := range []bool{true, false} {
		m := r.publishedMix(reveal)
		res, err := r.svc.AttemptResult(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID)
		require.NoError(t, err)
		it := res.Items[0]
		require.NotNil(t, it.Correct)
		require.NotEmpty(t, it.Earned)
		require.Equal(t, reveal, it.Answer != nil, "answer chỉ khi reveal")
	}
}

// TestResultHiddenOnlyCounts — AC6: test ẩn chỉ là SỐ đạt / tổng.
func TestResultHiddenOnlyCounts(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.publishedMix(true)
	res, err := r.svc.AttemptResult(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"passed", "total"}, keysOfJSON(t, res.Items[1].Hidden))
	for _, sm := range res.Items[1].Samples {
		require.NotContains(t, sm.Name, "CANARY")
	}
}

// TestResultOfficialScore — AC6: điểm hiển thị = adjusted nếu có, nếu không auto.
func TestResultOfficialScore(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.publishedMix(true)
	res, _ := r.svc.AttemptResult(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.Equal(t, "8.33", *res.Score)
	require.False(t, res.ScoreAdjusted)
	var ver int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select version from exam_attempts where id=$1`, m.v.Attempt.ID).Scan(&ver))
	_, err := r.svc.AdjustScore(t.Context(), r.teacher, r.course, m.e.ID, m.v.Attempt.ID, exam.AdjustIn{Score: new("6.5"), Reason: "chấm tay", Version: ver})
	require.NoError(t, err)
	res, _ = r.svc.AttemptResult(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.Equal(t, "6.50", *res.Score)
	require.True(t, res.ScoreAdjusted)
}

// ---- AC8: vắng --------------------------------------------------------------------------------------------------------

// TestAbsentStudentRow — AC8: sinh viên ACTIVE không có lượt → hàng ABSENT, không điểm.
func TestAbsentStudentRow(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.publishedMix(true)
	absent := r.student("ACTIVE")
	_, rows, err := r.svc.ResultsList(t.Context(), r.course, m.e.ID, exam.ResultsFilter{}, nil, 10)
	require.NoError(t, err)
	var found bool
	for _, x := range rows {
		if x.Student.ID == absent {
			found = true
			require.Equal(t, "ABSENT", x.Status)
			require.Nil(t, x.Score)
			require.Nil(t, x.AttemptID)
		}
	}
	require.True(t, found)
}

// TestAbsentCannotAppeal — AC8: sinh viên vắng không phúc khảo được (không có lượt nào để gửi; 404, không lộ tồn tại).
func TestAbsentCannotAppeal(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.publishedMix(true)
	absent := r.student("ACTIVE")
	_, err := r.svc.CreateAppeal(t.Context(), absent, r.course, m.e.ID, m.v.Attempt.ID, "của người khác")
	require.Equal(t, 404, must(apiStatus(t, err)))
	_, err = r.svc.CreateAppeal(t.Context(), absent, r.course, m.e.ID, uuid.New(), "không có lượt")
	require.Equal(t, 404, must(apiStatus(t, err)))
	require.Zero(t, r.count(`select count(*) from exam_appeals where student_id=$1`, absent))
}

// ---- AC9: bảng điểm Staff ---------------------------------------------------------------------------------------------

// TestResultsStaffPayload — AC9: khoá JSON của một hàng bảng điểm.
func TestResultsStaffPayload(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.publishedMix(true)
	_, rows, err := r.svc.ResultsList(t.Context(), r.course, m.e.ID, exam.ResultsFilter{Teacher: true}, nil, 10)
	require.NoError(t, err)
	require.Equal(t, []string{"adjusted", "attempt_id", "auto_score", "flags", "score", "status", "student", "submit_reason", "submitted_at"}, keysOfJSON(t, rows[0]))
	require.Equal(t, []string{"full_name", "id", "student_code"}, keysOfJSON(t, rows[0].Student))
}

// TestResultsProgressWhileOpen — AC9: bài đang mở: `progress` có "đang làm / đã nộp / chưa bắt đầu" và hàng KHÔNG có điểm.
func TestResultsProgressWhileOpen(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	r.student("ACTIVE") // chưa bắt đầu
	page, _, err := r.svc.ResultsList(t.Context(), r.course, m.e.ID, exam.ResultsFilter{}, nil, 10)
	require.NoError(t, err)
	require.Equal(t, exam.ResultsProgress{NotStarted: 1, InProgress: 1}, page.Progress)
	m.finish(true, false)
	page, rows, err := r.svc.ResultsList(t.Context(), r.course, m.e.ID, exam.ResultsFilter{}, nil, 10)
	require.NoError(t, err)
	require.Equal(t, exam.ResultsProgress{NotStarted: 1, Graded: 1}, page.Progress)
	for _, x := range rows {
		require.Nil(t, x.Score)
		require.Nil(t, x.AutoScore)
	}
}

// TestResultsFlagsTeacherOnly — AC9: `flags` chỉ có với Giảng viên (khoá không tồn tại với TA).
func TestResultsFlagsTeacherOnly(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.publishedMix(true)
	_, gv, _ := r.svc.ResultsList(t.Context(), r.course, m.e.ID, exam.ResultsFilter{Teacher: true}, nil, 10)
	_, ta, _ := r.svc.ResultsList(t.Context(), r.course, m.e.ID, exam.ResultsFilter{Teacher: false}, nil, 10)
	require.Contains(t, keysOfJSON(t, gv[0]), "flags")
	require.NotContains(t, keysOfJSON(t, ta[0]), "flags")
}

// TestResultsCursorStable — AC9: duyệt hết bằng con trỏ → mỗi sinh viên đúng một lần, thứ tự giống một lần lấy cả danh sách (theo điểm và theo tên).
func TestResultsCursorStable(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	for range 6 {
		sv, tab := r.student("ACTIVE"), uuid.New()
		v, _ := r.start(m.e, sv, tab)
		_, err := r.svc.SubmitAttempt(t.Context(), sv, r.course, m.e.ID, v.Attempt.ID, tab)
		require.NoError(t, err)
	}
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	for _, sortBy := range []string{"score", "name"} {
		_, all, err := r.svc.ResultsList(t.Context(), r.course, m.e.ID, exam.ResultsFilter{Sort: sortBy}, nil, 100)
		require.NoError(t, err)
		var got []uuid.UUID
		var after *uuid.UUID
		for {
			_, page, err := r.svc.ResultsList(t.Context(), r.course, m.e.ID, exam.ResultsFilter{Sort: sortBy}, after, 2)
			require.NoError(t, err)
			if len(page) == 0 {
				break
			}
			for _, x := range page {
				got = append(got, x.Student.ID)
			}
			after = &page[len(page)-1].Student.ID
		}
		want := make([]uuid.UUID, len(all))
		for i, x := range all {
			want[i] = x.Student.ID
		}
		require.Equal(t, want, got, sortBy)
	}
}

// ---- AC11: thống kê ---------------------------------------------------------------------------------------------------

func (r *rig) statsEnv() (*mixEnv, exam.Stats) {
	r.t.Helper()
	m := r.mixExam()
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	st, err := r.svc.ExamStats(r.t.Context(), r.course, m.e.ID)
	require.NoError(r.t, err)
	return m, st
}

// TestStatsDistribution — AC11: 10 khoảng 1 điểm, `[a,b)`, tổng count = số lượt GRADED.
func TestStatsDistribution(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, st := r.statsEnv()
	require.Len(t, st.Distribution, 10)
	require.Equal(t, exam.StatsBucket{From: "5.00", To: "6.00", Count: 1}, st.Distribution[5])
}

// TestStatsHardest — AC11: câu trắc nghiệm đúng ít nhất, có tỉ lệ đúng.
func TestStatsHardest(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, st := r.statsEnv()
	require.Len(t, st.Hardest, 1)
	require.Equal(t, "1.00", st.Hardest[0].CorrectRate)
}

// TestStatsMatchesResults — AC11: số liệu khớp bảng điểm (tổng count = số lượt GRADED; mean = trung bình điểm trong bảng).
func TestStatsMatchesResults(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m, st := r.statsEnv()
	_, rows, err := r.svc.ResultsList(t.Context(), r.course, m.e.ID, exam.ResultsFilter{}, nil, 100)
	require.NoError(t, err)
	graded, total := 0, 0
	for _, x := range rows {
		if x.Status == "GRADED" {
			graded++
		}
	}
	for _, b := range st.Distribution {
		total += b.Count
	}
	require.Equal(t, graded, total)
	require.Equal(t, "5.00", st.Mean)
}

// ---- AC12: CSV --------------------------------------------------------------------------------------------------------

func (r *rig) csvOut(m *mixEnv) string {
	r.t.Helper()
	var b strings.Builder
	require.NoError(r.t, r.svc.ResultsCSV(r.t.Context(), r.course, m.e.ID, &b, func() {}))
	return b.String()
}

func (r *rig) csvEnv() *mixEnv {
	r.t.Helper()
	m := r.mixExam()
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	return m
}

// TestCSVFormat — AC12: BOM, `;`, dấu phẩy thập phân, đúng cột.
func TestCSVFormat(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	out := r.csvOut(r.csvEnv())
	require.True(t, strings.HasPrefix(out, "\ufeffmssv;ho_ten;trang_thai;diem_tu_dong;diem_chinh_thuc;nop_luc;ly_do_nop;cau_1;cau_2\n"))
	require.Contains(t, out, ";GRADED;5,00;5,00;")
}

// TestCSVFormulaInjection — AC12: ô bắt đầu bằng `= + - @` được thêm `'`.
func TestCSVFormulaInjection(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.csvEnv()
	for i, v := range []string{"=1+1", "+84", "-2", "@cmd"} {
		sv := r.student("ACTIVE")
		r.exec(`update users set full_name=$2, student_code=$3 where id=$1`, sv, v, fmt.Sprintf("%s%d", v, i))
	}
	out := r.csvOut(m)
	for _, v := range []string{"'=1+1", "'+84", "'-2", "'@cmd"} {
		require.Contains(t, out, v)
	}
	require.NotContains(t, out, ";=1+1;")
}

// TestCSVAbsent — AC12: sinh viên vắng có trạng thái ABSENT và điểm trống.
func TestCSVAbsent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.csvEnv()
	r.student("ACTIVE")
	require.Contains(t, r.csvOut(m), ";ABSENT;;;;;")
}

type countingWriter struct{ writes, bytes int }

func (c *countingWriter) Write(p []byte) (int, error) {
	c.writes++
	c.bytes += len(p)
	return len(p), nil
}

// TestCSVStreams — AC12: ghi theo luồng (nhiều lần `Write` cho lớp lớn, không dựng cả tệp một lần).
func TestCSVStreams(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.csvEnv()
	r.exec(`with u as (insert into users (email, full_name, role) select 'bulk.' || g || '.' || $1, 'Sinh Viên ' || g, 'STUDENT' from generate_series(1, 600) g returning id)
		insert into enrollments (course_id, user_id, role_in_course, status, joined_via) select $2, id, 'STUDENT', 'ACTIVE', 'ADMIN' from u`, uuid.NewString()[:8], r.course)
	var w countingWriter
	require.NoError(t, r.svc.ResultsCSV(t.Context(), r.course, m.e.ID, &w, func() {}))
	require.Greater(t, w.writes, 2, "nhiều lần ghi = theo luồng")
	require.Greater(t, w.bytes, 15_000)
}

// ---- AC13: sửa điểm ---------------------------------------------------------------------------------------------------

func (r *rig) adjustEnv() (*mixEnv, int) {
	r.t.Helper()
	m := r.mixExam()
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	var ver int
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select version from exam_attempts where id=$1`, m.v.Attempt.ID).Scan(&ver))
	return m, ver
}

// TestAdjustKeepsAutoScore — AC13: điều chỉnh không ghi đè điểm máy.
func TestAdjustKeepsAutoScore(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m, ver := r.adjustEnv()
	_, err := r.svc.AdjustScore(t.Context(), r.teacher, r.course, m.e.ID, m.v.Attempt.ID, exam.AdjustIn{Score: new("9"), Reason: "x", Version: ver})
	require.NoError(t, err)
	var auto, adj string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select auto_score::text, adjusted_score::text from exam_attempts where id=$1`, m.v.Attempt.ID).Scan(&auto, &adj))
	require.Equal(t, "5.00", auto)
	require.Equal(t, "9.00", adj)
}

// TestAdjustAudit — AC13: `audit_log exam.score_adjust` trước / sau, không chứa lý do / nội dung bài.
func TestAdjustAudit(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m, ver := r.adjustEnv()
	_, err := r.svc.AdjustScore(t.Context(), r.teacher, r.course, m.e.ID, m.v.Attempt.ID, exam.AdjustIn{Score: new("9"), Reason: "lý do riêng tư XYZ", Version: ver})
	require.NoError(t, err)
	var after string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select "after"::text from audit_log where action='exam.score_adjust' and entity_id=$1`, m.v.Attempt.ID.String()).Scan(&after))
	require.Contains(t, after, "9.00")
	require.NotContains(t, after, "XYZ")
}

// TestAdjustAfterPublishNotifies — AC13: sau công bố → `EXAM_REGRADED` cho chủ lượt.
func TestAdjustAfterPublishNotifies(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.publishedMix(true)
	var ver int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select version from exam_attempts where id=$1`, m.v.Attempt.ID).Scan(&ver))
	_, err := r.svc.AdjustScore(t.Context(), r.teacher, r.course, m.e.ID, m.v.Attempt.ID, exam.AdjustIn{Score: new("7"), Reason: "x", Version: ver})
	require.NoError(t, err)
	require.NoError(t, (&exam.ResultNotifier{Pool: r.pool}).HandleRegraded(t.Context(), r.message("exam.regraded")))
	require.Equal(t, 1, r.count(`select count(*) from notifications where type='EXAM_REGRADED' and user_id=$1`, m.sv))
}

// TestAdjustTeacherOnly — AC13: #46 chỉ Giảng viên.
func TestAdjustTeacherOnly(t *testing.T) { t.Parallel(); onlyTeacher(t, 46) }

// ---- AC14: override ---------------------------------------------------------------------------------------------------

// TestOverrideAnswerKeyRecomputes — AC14: đổi đáp án đúng → lượt chọn đáp án cũ thành sai.
func TestOverrideAnswerKeyRecomputes(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m, _ := r.adjustEnv()
	var opt uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `select o.id from question_options o join exam_items i on i.question_id=o.question_id where i.id=$1 and o.body='A'`, m.mcqItem).Scan(&opt))
	out, err := r.svc.OverrideItem(t.Context(), r.teacher, r.course, m.e.ID, m.mcqItem, exam.OverrideIn{AnswerKey: json.RawMessage(`{"option_ids":["` + opt.String() + `"]}`), Reason: "đổi"})
	require.NoError(t, err)
	require.Equal(t, 1, out.Recomputed)
	_, _, sc, _ := r.attemptRow(m.v.Attempt.ID)
	require.Equal(t, "0.00", *sc)
}

// TestOverrideVoidGivesFullPoints — AC14: `void` = trọn điểm câu đó với mọi lượt (kể cả bỏ trống), giữ nguyên mẫu số.
func TestOverrideVoidGivesFullPoints(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(false, false) // bỏ trống cả hai câu → 0
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	_, _, sc, _ := r.attemptRow(m.v.Attempt.ID)
	require.Equal(t, "0.00", *sc)
	_, err := r.svc.OverrideItem(t.Context(), r.teacher, r.course, m.e.ID, m.mcqItem, exam.OverrideIn{Void: new(true), Reason: "đề sai"})
	require.NoError(t, err)
	_, _, sc, _ = r.attemptRow(m.v.Attempt.ID)
	require.Equal(t, "5.00", *sc, "1/2 điểm (mẫu số vẫn là 2)")
}

// TestOverrideNotifiesAfterPublish — AC14: sau công bố lượt bị đổi điểm nhận EXAM_REGRADED; lượt không đổi thì không.
func TestOverrideNotifiesAfterPublish(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.publishedMix(true)
	var opt uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `select o.id from question_options o join exam_items i on i.question_id=o.question_id where i.id=$1 and o.body='A'`, m.mcqItem).Scan(&opt))
	_, err := r.svc.OverrideItem(t.Context(), r.teacher, r.course, m.e.ID, m.mcqItem, exam.OverrideIn{AnswerKey: json.RawMessage(`{"option_ids":["` + opt.String() + `"]}`), Reason: "đổi"})
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.regraded' and payload->>'attempt_id'=$1`, m.v.Attempt.ID.String()))
	require.NoError(t, (&exam.ResultNotifier{Pool: r.pool}).HandleRegraded(t.Context(), r.message("exam.regraded")))
	require.Equal(t, 1, r.count(`select count(*) from notifications where type='EXAM_REGRADED' and user_id=$1`, m.sv))
}

// ---- AC15: chấm lại ---------------------------------------------------------------------------------------------------

// TestRegradeAllNewTestsVersion — AC15: test đổi (`tests_version` mới) → `all` đặt lại bản nộp cuối về QUEUED.
func TestRegradeAllNewTestsVersion(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := r.regradeEnv(m)
	var st string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select status::text from code_submissions where id=$1`, sub).Scan(&st))
	require.Equal(t, "QUEUED", st)
}

// TestRegradeErrorsOnly — AC15: `errors` chỉ chạm bản lỗi.
func TestRegradeErrorsOnly(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := m.finish(true, true)
	m.judged(sub, "AC")
	require.NoError(t, r.svc.OnSubmissionDone(t.Context(), r.course, m.v.Attempt.ID))
	m.closeExam()
	r.exec(`update code_problems set tests_version = tests_version + 1 where question_id = (select question_id from exam_items where id=$1)`, m.codeI)
	jid, err := r.svc.Regrade(t.Context(), r.teacher, r.course, m.e.ID, exam.RegradeIn{Scope: "errors", Reason: "lỗi"})
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", runJob(t, r, newRunner(t, r, &exam.Worker{}), jid).Status)
	var st string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select status::text from code_submissions where id=$1`, sub).Scan(&st))
	require.Equal(t, "DONE", st, "bản DONE không phải lỗi: không đụng")
}

// TestRegradeAfterPublishNotifies — AC15: sau công bố, chấm lại làm điểm đổi → EXAM_REGRADED đúng một lần.
func TestRegradeAfterPublishNotifies(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := r.regradeEnv(m)
	m.judged(sub, "AC")
	require.NoError(t, r.svc.OnSubmissionDone(t.Context(), r.course, m.v.Attempt.ID))
	rn := &exam.ResultNotifier{Pool: r.pool}
	msg := r.message("exam.regraded")
	require.NoError(t, rn.HandleRegraded(t.Context(), msg))
	require.NoError(t, rn.HandleRegraded(t.Context(), msg))
	require.Equal(t, 1, r.count(`select count(*) from notifications where type='EXAM_REGRADED' and user_id=$1`, m.sv))
}

// TestRegradeWhileOpenRejected — AC15: bài còn mở → 409 EXAM_LOCKED.
func TestRegradeWhileOpenRejected(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	_, err := r.svc.Regrade(t.Context(), r.teacher, r.course, m.e.ID, exam.RegradeIn{Scope: "all", Reason: "sớm"})
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "EXAM_LOCKED", code)
}

// ---- AC16 / AC17: phúc khảo -------------------------------------------------------------------------------------------

// TestAppealCreate — AC16: 201, `OPEN`, outbox `exam.appeal_created`.
func TestAppealCreate(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.publishedMix(true)
	ap, err := r.svc.CreateAppeal(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID, "Câu 2 chấm sai")
	require.NoError(t, err)
	require.Equal(t, "OPEN", ap.Status)
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.appeal_created'`))
}

// TestAppealOnlyOnce — AC16: một yêu cầu mỗi lượt (409 APPEAL_EXISTS), kể cả gửi song song.
func TestAppealOnlyOnce(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.publishedMix(true)
	res := make(chan error, 6)
	for range 6 {
		go func() {
			_, err := r.svc.CreateAppeal(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID, "song song")
			res <- err
		}()
	}
	ok := 0
	for range 6 {
		if err := <-res; err == nil {
			ok++
		} else {
			_, code := apiStatus(t, err)
			require.Equal(t, "APPEAL_EXISTS", code)
		}
	}
	require.Equal(t, 1, ok)
	require.Equal(t, 1, r.count(`select count(*) from exam_appeals where attempt_id=$1`, m.v.Attempt.ID))
}

// TestAppealNotPublished — AC16: chưa công bố → 409.
func TestAppealNotPublished(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	m.closeExam()
	_, err := r.svc.CreateAppeal(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID, "sớm")
	require.Equal(t, 409, must(apiStatus(t, err)))
}

// TestAppealNoLLM — AC16: không file nào của phúc khảo (service / handler) import `internal/llm`.
func TestAppealNoLLM(t *testing.T) {
	t.Parallel()
	for _, f := range []string{"appeals.go", "adjust.go", "../httpapi/examhttp/results.go"} {
		src, err := os.ReadFile(f)
		require.NoError(t, err)
		af, err := parser.ParseFile(token.NewFileSet(), f, src, parser.ImportsOnly)
		require.NoError(t, err)
		for _, im := range af.Imports {
			require.NotContains(t, im.Path.Value, "/llm", f)
		}
	}
}

func (r *rig) appealEnv() (*mixEnv, exam.AppealView) {
	r.t.Helper()
	m := r.publishedMix(true)
	ap, err := r.svc.CreateAppeal(r.t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID, "xin xem lại")
	require.NoError(r.t, err)
	return m, ap
}

// TestAppealAnswerUpheld — AC17: giữ điểm; điểm không đổi; sinh viên thấy phản hồi.
func TestAppealAnswerUpheld(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m, ap := r.appealEnv()
	out, err := r.svc.AnswerAppeal(t.Context(), r.teacher, r.course, m.e.ID, ap.ID, exam.AppealAnswerIn{Decision: "UPHELD", Response: "giữ nguyên", Version: ap.Version})
	require.NoError(t, err)
	require.Equal(t, "UPHELD", out.Status)
	res, _ := r.svc.AttemptResult(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.Equal(t, "8.33", *res.Score)
	require.Equal(t, "giữ nguyên", *res.Appeal.Response)
}

// TestAppealAnswerAdjustedAppliesScore — AC17: ADJUSTED áp điểm cùng giao dịch, `score_before` / `score_after`, audit `appeal.answer`.
func TestAppealAnswerAdjustedAppliesScore(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m, ap := r.appealEnv()
	out, err := r.svc.AnswerAppeal(t.Context(), r.teacher, r.course, m.e.ID, ap.ID, exam.AppealAnswerIn{Decision: "ADJUSTED", Response: "nâng", Score: new("9"), Version: ap.Version})
	require.NoError(t, err)
	require.Equal(t, "8.33", *out.ScoreBefore)
	require.Equal(t, "9.00", *out.ScoreAfter)
	res, _ := r.svc.AttemptResult(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.Equal(t, "9.00", *res.Score)
	require.Equal(t, 1, r.count(`select count(*) from audit_log where action='appeal.answer' and entity_id=$1`, ap.ID.String()))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where action='exam.score_adjust' and entity_id=$1`, m.v.Attempt.ID.String()))
}

// TestAppealAnswerOnce — AC17: trả lời một lần (409).
func TestAppealAnswerOnce(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m, ap := r.appealEnv()
	out, err := r.svc.AnswerAppeal(t.Context(), r.teacher, r.course, m.e.ID, ap.ID, exam.AppealAnswerIn{Decision: "UPHELD", Response: "ok", Version: ap.Version})
	require.NoError(t, err)
	_, err = r.svc.AnswerAppeal(t.Context(), r.teacher, r.course, m.e.ID, ap.ID, exam.AppealAnswerIn{Decision: "UPHELD", Response: "lại", Version: out.Version})
	require.Equal(t, 409, must(apiStatus(t, err)))
}

// TestAppealTeacherOnlyAnswer — AC17: #56 chỉ Giảng viên; #55 (đọc) cho cả TA.
func TestAppealTeacherOnlyAnswer(t *testing.T) {
	t.Parallel()
	onlyTeacher(t, 56)
	code, _ := serve(t, routeNo(55), auth.RoleTA, auth.Membership{Found: true, Role: auth.RoleTA, Status: "ACTIVE"})
	require.Equal(t, http.StatusNoContent, code)
}

// TestAppealTodayProviders — AC16/AC17: EXAM_APPEAL cho Giảng viên khi còn OPEN (TA không); EXAM_APPEAL_REPLY cho sinh viên sau khi được trả lời.
func TestAppealTodayProviders(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m, ap := r.appealEnv()
	now := time.Now().UTC()
	staff := func(role string, rl today.Role) []today.Item {
		items, err := today.StaffProvider{Pool: r.pool}.Items(t.Context(), today.Viewer{UserID: r.teacher, Role: rl, Now: now, Courses: []today.CourseRef{{ID: r.course, ClassCode: "T", RoleInCourse: role}}}, today.Scope{})
		require.NoError(t, err)
		return items
	}
	kinds := func(items []today.Item) []today.Kind {
		var out []today.Kind
		for _, it := range items {
			out = append(out, it.Kind)
		}
		return out
	}
	require.Contains(t, kinds(staff("TEACHER", today.RoleTeacher)), today.KindExamAppeal)
	require.NotContains(t, kinds(staff("TA", today.RoleTA)), today.KindExamAppeal)
	_, err := r.svc.AnswerAppeal(t.Context(), r.teacher, r.course, m.e.ID, ap.ID, exam.AppealAnswerIn{Decision: "UPHELD", Response: "ok", Version: ap.Version})
	require.NoError(t, err)
	require.NotContains(t, kinds(staff("TEACHER", today.RoleTeacher)), today.KindExamAppeal)
	items, err := today.StudentProvider{Pool: r.pool}.Items(t.Context(), today.Viewer{UserID: m.sv, Role: today.RoleStudent, EmailVerified: true, Now: now, Courses: []today.CourseRef{{ID: r.course, ClassCode: "T", RoleInCourse: "STUDENT"}}}, today.Scope{})
	require.NoError(t, err)
	require.Contains(t, kinds(items), today.KindExamAppealReply)
}

// ---- AC19 / PE-07 -----------------------------------------------------------------------------------------------------

// TestEventsTeacherOnlyMatrix — US-PE-07 AC4: #51, #52, #54 chỉ Giảng viên (vai × 3 route).
func TestEventsTeacherOnlyMatrix(t *testing.T) {
	t.Parallel()
	for _, no := range []int{51, 52, 54} {
		onlyTeacher(t, no)
	}
}

// TestResultsPermissionMatrix — AC19: mọi route của US-PE-08 × vai × tình trạng ghi danh qua CourseAccessGuard thật: ADMIN → 403 mọi route; sinh viên chỉ route sinh viên; TA đọc nhưng không sửa; Giảng viên mọi route;
// người ngoài lớp → 403. (Lượt của người khác → 404: `TestStudentResult`, và ma trận HTTP đầy đủ ở `contract.TestResultsPermissionMatrix`.)
func TestResultsPermissionMatrix(t *testing.T) {
	t.Parallel()
	nos := []int{26, 42, 44, 45, 46, 47, 48, 49, 50, 55, 56, 41, 51, 52, 54}
	n := 0
	for _, no := range nos {
		rt := routeNo(no)
		for _, jwt := range []auth.Role{auth.RoleAdmin, auth.RoleTeacher, auth.RoleTA, auth.RoleStudent} {
			for _, e := range enrolments() {
				code, _ := serve(t, rt, jwt, e.m)
				active := e.m.Found && e.m.Status == "ACTIVE" && jwt != auth.RoleAdmin
				want := active && allowedRoles(rt.Mode)[e.m.Role]
				if want {
					require.Equal(t, http.StatusNoContent, code, fmt.Sprintf("#%d %s %s", no, jwt, e.name))
				} else {
					require.Equal(t, http.StatusForbidden, code, fmt.Sprintf("#%d %s %s", no, jwt, e.name))
				}
				n++
			}
		}
	}
	require.GreaterOrEqual(t, len(nos), 15)
	require.Greater(t, n, 400)
}
