package exam_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/platform/clock"
)

// mixEnv: bài ĐANG MỞ có một câu MCQ (vị trí 1) và một câu code (vị trí 2), một sinh viên đã bắt đầu.
type mixEnv struct {
	r              *rig
	e              exam.ExamDetail
	sv, tab        uuid.UUID
	v              exam.AttemptStartView
	mcqItem, codeI uuid.UUID
	sample, hidden uuid.UUID
}

func (r *rig) mixExam() *mixEnv {
	r.t.Helper()
	qid, s, h := r.codeQuestion("{cpp17}")
	m := &mixEnv{r: r, sv: r.student("ACTIVE"), tab: uuid.New(), sample: s, hidden: h}
	m.e = r.openExam("hỗn hợp", false, r.mcqSet(1)[0], qid)
	m.v, _ = r.start(m.e, m.sv, m.tab)
	for _, it := range m.v.Items {
		if it.Type == "CODE" {
			m.codeI = it.ItemID
		} else {
			m.mcqItem = it.ItemID
		}
	}
	return m
}

// answerAll: trả lời đúng câu MCQ (nếu `right`) rồi nộp mã (nếu `code`), nộp bài. Trả id bản nộp (hoặc Nil).
func (m *mixEnv) finish(right, code bool) uuid.UUID {
	m.r.t.Helper()
	ctx := m.r.t.Context()
	if right {
		for _, it := range m.v.Items {
			if it.Type != "CODE" {
				_, err := m.r.svc.SaveAnswers(ctx, m.sv, m.r.course, m.e.ID, m.v.Attempt.ID, m.tab, exam.AnswersIn{Items: []exam.AnswerIn{answerCorrect(it)}})
				require.NoError(m.r.t, err)
			}
		}
	}
	var sub uuid.UUID
	if code {
		q, err := m.r.svc.SubmitCode(ctx, m.sv, m.r.course, m.e.ID, m.v.Attempt.ID, m.codeI, m.tab, exam.CodeRunIn{Language: "cpp17", Source: "int main(){}"})
		require.NoError(m.r.t, err)
		sub = q.SubmissionID
	}
	_, err := m.r.svc.SubmitAttempt(ctx, m.sv, m.r.course, m.e.ID, m.v.Attempt.ID, m.tab)
	require.NoError(m.r.t, err)
	return sub
}

// judged giả lập máy chấm: test mẫu `sampleV`, test ẩn AC; trọng số mẫu 1 / ẩn 2 → passed_weight 2 khi mẫu sai.
func (m *mixEnv) judged(sub uuid.UUID, sampleV string) {
	m.r.t.Helper()
	pw := 2
	if sampleV == "AC" {
		pw = 3
	}
	res, _ := json.Marshal([]map[string]any{
		{"test_id": m.sample, "position": 1, "is_sample": true, "verdict": sampleV, "time_ms": 12, "memory_kb": 900},
		{"test_id": m.hidden, "position": 2, "is_sample": false, "verdict": "AC", "time_ms": 7, "memory_kb": 800},
	})
	m.r.exec(`update code_submissions set status='DONE', verdict='WA', compile_ok=true, results=$2::jsonb, passed_weight=$3, total_weight=3, tests_version=(select cp.tests_version from code_problems cp where cp.question_id = code_submissions.problem_id), judged_at=now(), lease_until=null where id=$1`, sub, string(res), pw)
}

func (m *mixEnv) closeExam() {
	m.r.exec(`update exams set status='CLOSED', opens_at = now() - interval '4 hours', closes_at = now() - interval '1 minute' where id=$1`, m.e.ID)
}

func (r *rig) attemptStatus(attempt uuid.UUID) string {
	st, _, _, _ := r.attemptRow(attempt)
	return st
}

func (r *rig) examStatus(id uuid.UUID) string {
	r.t.Helper()
	var s string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select status::text from exams where id=$1`, id).Scan(&s))
	return s
}

// TestGradeCompletesWhenSubmissionDone — US-PE-08 AC2/AC3: MCQ chấm ngay nhưng lượt có bản nộp QUEUED ở lại GRADING; khi bản nộp DONE → `OnSubmissionDone` chuyển GRADED
// với điểm tính bằng decimal: code earned = points × Σweight(đạt) ÷ Σweight, lưu đủ chữ số; điểm cuối làm tròn MỘT lần từ tổng chưa làm tròn.
func TestGradeCompletesWhenSubmissionDone(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := m.finish(true, true)
	require.Equal(t, "GRADING", r.attemptStatus(m.v.Attempt.ID))
	require.NoError(t, r.svc.OnSubmissionDone(t.Context(), r.course, m.v.Attempt.ID)) // bản nộp còn QUEUED: chưa làm gì
	require.Equal(t, "GRADING", r.attemptStatus(m.v.Attempt.ID))
	m.judged(sub, "WA")
	require.NoError(t, r.svc.OnSubmissionDone(t.Context(), r.course, m.v.Attempt.ID))
	st, _, sc, _ := r.attemptRow(m.v.Attempt.ID)
	require.Equal(t, "GRADED", st)
	// MCQ 1/1 + code 2/3: raw = 1 + 0,6666… = 1,6666…; điểm = round(1,6666… × 10 ÷ 2, bước 0,01) = 8,33
	require.Equal(t, "8.33", *sc)
	var raw []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select breakdown from exam_attempts where id=$1`, m.v.Attempt.ID).Scan(&raw))
	var bd []struct {
		ItemID string `json:"item_id"`
		Earned string `json:"earned"`
		Max    string `json:"max"`
		Code   *struct {
			CompileOK bool `json:"compile_ok"`
			Hidden    struct{ Passed, Total int }
			Samples   []struct{ Verdict string }
		} `json:"code"`
	}
	require.NoError(t, json.Unmarshal(raw, &bd))
	require.Len(t, bd, 2)
	require.Equal(t, "0.6666666666666667", bd[1].Earned, "earned lưu đủ chữ số, không làm tròn trung gian")
	require.NotNil(t, bd[1].Code)
	require.Equal(t, 1, bd[1].Code.Hidden.Total)
	require.Equal(t, "WA", bd[1].Code.Samples[0].Verdict)
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.attempt_graded' and payload->>'attempt_id'=$1`, m.v.Attempt.ID.String()))
	// chạy lại: idempotent (không ghi đôi, không đổi điểm)
	require.NoError(t, r.svc.OnSubmissionDone(t.Context(), r.course, m.v.Attempt.ID))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.attempt_graded' and payload->>'attempt_id'=$1`, m.v.Attempt.ID.String()))
}

// TestIEKeepsGrading — AC2: bản nộp ERROR (sau dead-letter) → lượt Ở LẠI GRADING, không điểm 0; chấm lại xong thì hoàn tất.
func TestIEKeepsGrading(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := m.finish(true, true)
	r.exec(`update code_submissions set status='ERROR', verdict='IE', judged_at=now() where id=$1`, sub)
	n, err := r.svc.GradeDue(t.Context())
	require.NoError(t, err)
	require.Zero(t, n)
	st, _, sc, _ := r.attemptRow(m.v.Attempt.ID)
	require.Equal(t, "GRADING", st)
	require.Nil(t, sc)
	m.closeExam()
	n, err = r.svc.PublishDue(t.Context())
	require.NoError(t, err)
	require.Zero(t, n, "còn lượt chưa GRADED thì không công bố")
}

// TestAutoPublishOnce — AC4: công bố chỉ khi mọi lượt GRADED, không hoãn, không đang chấm lại; chạy song song / chạy lại → đúng MỘT lần, một sự kiện.
func TestAutoPublishOnce(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	n, err := r.svc.PublishDue(t.Context())
	require.NoError(t, err)
	require.Zero(t, n, "đang hoãn")
	r.exec(`update exams set publish_hold = false, regrading = true where id=$1`, m.e.ID)
	n, _ = r.svc.PublishDue(t.Context())
	require.Zero(t, n, "đang chấm lại")
	r.exec(`update exams set regrading = false where id=$1`, m.e.ID)
	var wg sync.WaitGroup
	var mu sync.Mutex
	total := 0
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			k, err := r.svc.PublishDue(t.Context())
			require.NoError(t, err)
			mu.Lock()
			total += k
			mu.Unlock()
		}()
	}
	wg.Wait()
	require.Equal(t, 1, total)
	require.Equal(t, "PUBLISHED", r.examStatus(m.e.ID))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.published' and payload->>'exam_id'=$1`, m.e.ID.String()))
	n, _ = r.svc.PublishDue(t.Context())
	require.Zero(t, n)
}

// TestPublishNotificationsOnce — AC4: EXAM_PUBLISHED chỉ cho sinh viên CÓ lượt, khử trùng khi xử lý lại.
func TestPublishNotificationsOnce(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	absent := r.student("ACTIVE")
	m.finish(true, false)
	m.closeExam()
	_, err := r.svc.PublishDue(t.Context())
	require.NoError(t, err)
	n := &exam.ResultNotifier{Pool: r.pool}
	msg := r.message("exam.published")
	require.NoError(t, n.HandlePublished(t.Context(), msg))
	require.NoError(t, n.HandlePublished(t.Context(), msg))
	require.Equal(t, 1, r.count(`select count(*) from notifications where type='EXAM_PUBLISHED' and user_id=$1`, m.sv))
	require.Zero(t, r.count(`select count(*) from notifications where type='EXAM_PUBLISHED' and user_id=$1`, absent))
}

// TestSetHold — #26: hoãn / bỏ hoãn; bỏ hoãn khi đã chấm xong → công bố NGAY; sau PUBLISHED → 409.
func TestSetHold(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	d, err := r.svc.GetExam(t.Context(), r.course, m.e.ID)
	require.NoError(t, err)
	_, err = r.svc.SetHold(t.Context(), r.teacher, r.course, m.e.ID, false, d.Version-1)
	requireVC(t, err)
	out, err := r.svc.SetHold(t.Context(), r.teacher, r.course, m.e.ID, false, d.Version)
	require.NoError(t, err)
	require.Equal(t, "PUBLISHED", out.Status, "bỏ hoãn khi đã chấm xong → công bố ngay")
	_, err = r.svc.SetHold(t.Context(), r.teacher, r.course, m.e.ID, true, out.Version)
	st, _ := apiStatus(t, err)
	require.Equal(t, 409, st)
}

func (r *rig) published(m *mixEnv) {
	r.t.Helper()
	m.closeExam()
	n, err := r.svc.PublishDue(r.t.Context())
	require.NoError(r.t, err)
	require.Equal(r.t, 1, n)
}

// TestStudentResult — AC6: kết quả đủ từng câu; test ẩn chỉ SỐ đạt / tổng, KHÔNG tên / input / expected; `reveal_answers=false` ẩn đáp án + giải thích; chưa công bố → 409 không dữ liệu.
func TestStudentResult(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := m.finish(true, true)
	m.judged(sub, "WA")
	require.NoError(t, r.svc.OnSubmissionDone(t.Context(), r.course, m.v.Attempt.ID))
	m.closeExam()
	_, err := r.svc.AttemptResult(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	st, _ := apiStatus(t, err)
	require.Equal(t, 409, st)
	r.exec(`update exams set status='CLOSED' where id=$1`, m.e.ID)
	_, err = r.svc.PublishDue(t.Context())
	require.NoError(t, err)

	res, err := r.svc.AttemptResult(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.NoError(t, err)
	require.Equal(t, "8.33", *res.Score)
	require.False(t, res.ScoreAdjusted)
	require.Len(t, res.Items, 2)
	mc, cd := res.Items[0], res.Items[1]
	require.True(t, *mc.Correct)
	require.NotNil(t, mc.Answer, "reveal_answers mặc định bật")
	require.Equal(t, "0.67", cd.Earned)
	require.Equal(t, &exam.ResultHiddenView{Passed: 1, Total: 1}, cd.Hidden)
	require.Len(t, cd.Samples, 1)
	require.Equal(t, "WA", cd.Samples[0].Verdict)
	require.Equal(t, "1 2", cd.Samples[0].Input)
	require.Equal(t, "int main(){}", cd.FinalSubmission.Source)
	raw, _ := json.Marshal(res)
	require.NotContains(t, string(raw), canary, "tên / input / expected của test ẩn và lời giải mẫu không được lộ")

	r.exec(`update exams set reveal_answers=false where id=$1`, m.e.ID)
	res, err = r.svc.AttemptResult(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.NoError(t, err)
	require.Nil(t, res.Items[0].Answer)
	require.Nil(t, res.Items[0].Explanation)
	require.NotNil(t, res.Items[0].Correct, "correct / earned luôn có sau công bố")

	other := r.student("ACTIVE")
	_, err = r.svc.AttemptResult(t.Context(), other, r.course, m.e.ID, m.v.Attempt.ID)
	st, _ = apiStatus(t, err)
	require.Equal(t, 404, st, "lượt của người khác")
}

// TestAdjustScore — AC13: sửa tay có lý do, đúng bước / khoảng, version; auto_score giữ nguyên; `score:null` gỡ; sau công bố → EXAM_REGRADED khử trùng theo version.
func TestAdjustScore(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	ctx := t.Context()
	var ver int
	require.NoError(t, r.pool.QueryRow(ctx, `select version from exam_attempts where id=$1`, m.v.Attempt.ID).Scan(&ver))
	adj := func(score *string, reason string, v int) (exam.AdjustOut, error) {
		return r.svc.AdjustScore(ctx, r.teacher, r.course, m.e.ID, m.v.Attempt.ID, exam.AdjustIn{Score: score, Reason: reason, Version: v})
	}
	_, err := adj(new("7"), "", ver)
	require.Equal(t, 422, must(apiStatus(t, err)))
	_, err = adj(new("11"), "vượt", ver)
	require.Equal(t, 422, must(apiStatus(t, err)))
	_, err = adj(new("7.005"), "lẻ", ver)
	require.Equal(t, 422, must(apiStatus(t, err)))
	_, err = adj(new("7"), "sai version", ver+9)
	requireVC(t, err)
	out, err := adj(new("7.5"), "chấm tay", ver)
	require.NoError(t, err)
	require.True(t, out.Adjusted)
	require.Equal(t, "7.50", *out.Score)
	require.Equal(t, "5.00", *out.AutoScore, "auto_score giữ nguyên")
	require.Equal(t, 1, r.count(`select count(*) from audit_log where action='exam.score_adjust' and entity_id=$1`, m.v.Attempt.ID.String()))
	out, err = adj(nil, "gỡ", out.Version)
	require.NoError(t, err)
	require.False(t, out.Adjusted)
	require.Equal(t, "5.00", *out.Score)
	require.Zero(t, r.count(`select count(*) from outbox where topic='exam.regraded'`), "trước công bố không báo sinh viên")

	// sau công bố
	r.exec(`update exams set publish_hold=false where id=$1`, m.e.ID)
	_, err = r.svc.PublishDue(ctx)
	require.NoError(t, err)
	out, err = adj(new("9"), "phúc khảo", out.Version)
	require.NoError(t, err)
	n := &exam.ResultNotifier{Pool: r.pool}
	msg := r.message("exam.regraded")
	require.NoError(t, n.HandleRegraded(ctx, msg))
	require.NoError(t, n.HandleRegraded(ctx, msg))
	require.Equal(t, 1, r.count(`select count(*) from notifications where type='EXAM_REGRADED' and user_id=$1`, m.sv))
	res, err := r.svc.AttemptResult(ctx, m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.NoError(t, err)
	require.Equal(t, "9.00", *res.Score)
	require.True(t, res.ScoreAdjusted)
}

// ageExam lùi `updated_at` 1 phút (bỏ qua trigger `set_updated_at` bằng replica role trong MỘT giao dịch).
func (r *rig) ageExam(id uuid.UUID) {
	r.t.Helper()
	tx, err := r.pool.Begin(r.t.Context())
	require.NoError(r.t, err)
	defer func() { _ = tx.Rollback(r.t.Context()) }()
	_, err = tx.Exec(r.t.Context(), `set local session_replication_role = replica`)
	require.NoError(r.t, err)
	_, err = tx.Exec(r.t.Context(), `update exams set updated_at = now() - interval '1 minute' where id=$1`, id)
	require.NoError(r.t, err)
	require.NoError(r.t, tx.Commit(r.t.Context()))
}

func requireVC(t *testing.T, err error) {
	t.Helper()
	var vc *exam.VersionConflict
	require.ErrorAs(t, err, &vc)
}

func must[T any](v T, _ any) T { return v }

// TestOverrideRecomputes — AC14: đổi đáp án đúng / huỷ câu → mọi lượt GRADED tính lại; auto_score đổi, điểm cũ không "nhảy" ở lượt khác; chỉ câu trắc nghiệm.
func TestOverrideRecomputes(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	ctx := t.Context()
	_, _, sc, _ := r.attemptRow(m.v.Attempt.ID)
	require.Equal(t, "5.00", *sc)
	// huỷ câu: trọn điểm câu đó cho mọi lượt
	out, err := r.svc.OverrideItem(ctx, r.teacher, r.course, m.e.ID, m.mcqItem, exam.OverrideIn{Void: new(true), Reason: "đề sai"})
	require.NoError(t, err)
	require.Equal(t, 1, out.Recomputed)
	_, _, sc, _ = r.attemptRow(m.v.Attempt.ID)
	require.Equal(t, "5.00", *sc, "đã đúng sẵn: không đổi")
	// đáp án đúng mới = đáp án khác → lượt này (chọn B) thành sai
	var opt uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `select o.id from question_options o join exam_items i on i.question_id=o.question_id where i.id=$1 and o.body='A'`, m.mcqItem).Scan(&opt))
	_, err = r.svc.OverrideItem(ctx, r.teacher, r.course, m.e.ID, m.mcqItem, exam.OverrideIn{AnswerKey: json.RawMessage(`{"option_ids":["` + opt.String() + `"]}`), Reason: "đổi đáp án"})
	require.NoError(t, err)
	_, _, sc, _ = r.attemptRow(m.v.Attempt.ID)
	require.Equal(t, "0.00", *sc)
	_, err = r.svc.OverrideItem(ctx, r.teacher, r.course, m.e.ID, m.mcqItem, exam.OverrideIn{AnswerKey: json.RawMessage(`{"option_ids":["` + uuid.NewString() + `"]}`), Reason: "lạ"})
	require.Equal(t, 422, must(apiStatus(t, err)))
	_, err = r.svc.OverrideItem(ctx, r.teacher, r.course, m.e.ID, m.codeI, exam.OverrideIn{Void: new(true), Reason: "x"})
	require.Equal(t, 422, must(apiStatus(t, err)), "câu code không override")
	_, err = r.svc.OverrideItem(ctx, r.teacher, r.course, m.e.ID, m.mcqItem, exam.OverrideIn{Reason: "thiếu"})
	require.Equal(t, 422, must(apiStatus(t, err)))
}

// TestRegradeFlow — AC15: `regrade` giữ lượt GRADED với điểm cũ, đặt `regrading`, đưa bản nộp về QUEUED; khi máy chấm xong → điểm mới, `regrading=false`, KHÔNG công bố sớm,
// EXAM_REGRADED một lần sau công bố.
func TestRegradeFlow(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := m.finish(true, true)
	m.judged(sub, "WA")
	require.NoError(t, r.svc.OnSubmissionDone(t.Context(), r.course, m.v.Attempt.ID))
	r.published(m)
	ctx := t.Context()

	r.exec(`update code_problems set tests_version = tests_version + 1 where question_id = (select question_id from exam_items where id=$1)`, m.codeI) // test đổi → bản nộp đã chấm bằng bộ cũ được chấm lại
	jid, err := r.svc.Regrade(ctx, r.teacher, r.course, m.e.ID, exam.RegradeIn{Scope: "all", Reason: "test đổi"})
	require.NoError(t, err)
	run := newRunner(t, r, &exam.Worker{})
	require.Equal(t, "SUCCEEDED", runJob(t, r, run, jid).Status)
	require.Equal(t, "QUEUED", func() string {
		var s string
		require.NoError(t, r.pool.QueryRow(ctx, `select status::text from code_submissions where id=$1`, sub).Scan(&s))
		return s
	}())
	require.Equal(t, "GRADED", r.attemptStatus(m.v.Attempt.ID), "lượt không đổi trạng thái khi chấm lại")
	res, err := r.svc.AttemptResult(ctx, m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.NoError(t, err)
	require.Equal(t, "8.33", *res.Score, "giữ điểm cũ tới khi tính lại xong")
	require.Equal(t, &exam.ResultHiddenView{Passed: 1, Total: 1}, res.Items[1].Hidden, "kết quả đọc từ ảnh chụp, không trống khi đang chấm lại")
	n, err := r.svc.RegradeSweep(ctx)
	require.NoError(t, err)
	require.Zero(t, n)
	require.Equal(t, 1, r.count(`select count(*)::int from exams where id=$1 and regrading`, m.e.ID))

	m.judged(sub, "AC") // máy chấm xong: mẫu đạt
	require.NoError(t, r.svc.OnSubmissionDone(ctx, r.course, m.v.Attempt.ID))
	res, err = r.svc.AttemptResult(ctx, m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.NoError(t, err)
	require.Equal(t, "10.00", *res.Score)
	require.Equal(t, "GRADED", r.attemptStatus(m.v.Attempt.ID))
	n2 := &exam.ResultNotifier{Pool: r.pool}
	msg := r.message("exam.regraded")
	require.NoError(t, n2.HandleRegraded(ctx, msg))
	require.NoError(t, n2.HandleRegraded(ctx, msg))
	require.Equal(t, 1, r.count(`select count(*) from notifications where type='EXAM_REGRADED' and user_id=$1`, m.sv))
	r.ageExam(m.e.ID) // qua thời gian chờ của tick
	_, err = r.svc.RegradeSweep(ctx)
	require.NoError(t, err)
	require.Zero(t, r.count(`select count(*)::int from exams where id=$1 and regrading`, m.e.ID), "mọi bản nộp xong và mọi lượt đã tính lại → gỡ cờ")
}

// TestAppealFlow — AC16/AC17: trong hạn, một lần; sau hạn → 409; ADJUSTED áp điểm cùng transaction; trả lời một lần; sinh viên khác / vắng không gửi được.
func TestAppealFlow(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	r.published(m)
	ctx := t.Context()
	_, err := r.svc.CreateAppeal(ctx, m.sv, r.course, m.e.ID, m.v.Attempt.ID, "")
	require.Equal(t, 422, must(apiStatus(t, err)))
	other := r.student("ACTIVE")
	_, err = r.svc.CreateAppeal(ctx, other, r.course, m.e.ID, m.v.Attempt.ID, "của người khác")
	require.Equal(t, 404, must(apiStatus(t, err)))
	ap, err := r.svc.CreateAppeal(ctx, m.sv, r.course, m.e.ID, m.v.Attempt.ID, "Câu 2 chấm sai")
	require.NoError(t, err)
	require.Equal(t, "OPEN", ap.Status)
	_, err = r.svc.CreateAppeal(ctx, m.sv, r.course, m.e.ID, m.v.Attempt.ID, "lần hai")
	require.Equal(t, 409, must(apiStatus(t, err)))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.appeal_created'`))

	rows, err := r.svc.ListAppeals(ctx, r.course, m.e.ID, "OPEN", nil, 11)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "Sinh Viên", rows[0].Student.FullName)

	_, err = r.svc.AnswerAppeal(ctx, r.teacher, r.course, m.e.ID, ap.ID, exam.AppealAnswerIn{Decision: "ADJUSTED", Response: "ok", Version: ap.Version})
	require.Equal(t, 422, must(apiStatus(t, err)), "ADJUSTED cần điểm")
	_, err = r.svc.AnswerAppeal(ctx, r.teacher, r.course, m.e.ID, ap.ID, exam.AppealAnswerIn{Decision: "ADJUSTED", Response: "ok", Score: new("6"), Version: ap.Version + 3})
	requireVC(t, err)
	out, err := r.svc.AnswerAppeal(ctx, r.teacher, r.course, m.e.ID, ap.ID, exam.AppealAnswerIn{Decision: "ADJUSTED", Response: "đã xem lại, nâng điểm", Score: new("6"), Version: ap.Version})
	require.NoError(t, err)
	require.Equal(t, "ADJUSTED", out.Status)
	require.Equal(t, "5.00", *out.ScoreBefore)
	require.Equal(t, "6.00", *out.ScoreAfter)
	res, err := r.svc.AttemptResult(ctx, m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.NoError(t, err)
	require.Equal(t, "6.00", *res.Score)
	require.True(t, res.ScoreAdjusted)
	require.Equal(t, "ADJUSTED", *res.Appeal.Status)
	_, err = r.svc.AnswerAppeal(ctx, r.teacher, r.course, m.e.ID, ap.ID, exam.AppealAnswerIn{Decision: "UPHELD", Response: "lại", Version: out.Version})
	require.Equal(t, 409, must(apiStatus(t, err)), "trả lời một lần")
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.appeal_answered'`))
	require.Zero(t, r.count(`select count(*) from outbox where topic='exam.regraded'`), "phúc khảo chỉ báo EXAM_APPEAL_REPLIED, không báo đôi")
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TEACHER', 'ACTIVE', 'ADMIN'), ($1, $3, 'TA', 'ACTIVE', 'ADMIN')`, r.course, r.teacher, r.ta)
	n := &exam.ResultNotifier{Pool: r.pool}
	require.NoError(t, n.HandleAppealCreated(ctx, r.message("exam.appeal_created")))
	require.NoError(t, n.HandleAppealAnswered(ctx, r.message("exam.appeal_answered")))
	require.Equal(t, 2, r.count(`select count(*) from notifications where type='EXAM_APPEAL_NEW'`), "Giảng viên + TA")
	require.Equal(t, 1, r.count(`select count(*) from notifications where type='EXAM_APPEAL_REPLIED' and user_id=$1`, m.sv))
}

// TestAppealWindow — AC16: hết `appeal_days` hoặc `appeal_days = 0` → 409 APPEAL_WINDOW_CLOSED; bài chưa công bố → 409.
func TestAppealWindow(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	ctx := t.Context()
	_, err := r.svc.CreateAppeal(ctx, m.sv, r.course, m.e.ID, m.v.Attempt.ID, "sớm")
	require.Equal(t, 409, must(apiStatus(t, err)))
	r.published(m)
	r.exec(`update exams set published_at = now() - interval '8 days' where id=$1`, m.e.ID)
	_, err = r.svc.CreateAppeal(ctx, m.sv, r.course, m.e.ID, m.v.Attempt.ID, "muộn")
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "APPEAL_WINDOW_CLOSED", code)
	r.exec(`update exams set published_at = now(), appeal_days = 0 where id=$1`, m.e.ID)
	_, err = r.svc.CreateAppeal(ctx, m.sv, r.course, m.e.ID, m.v.Attempt.ID, "tắt")
	_, code = apiStatus(t, err)
	require.Equal(t, "APPEAL_WINDOW_CLOSED", code)
}

// TestResultsListStaff — AC9: tiến độ, vắng, sắp xếp ổn định theo điểm / tên với con trỏ; `flags` chỉ với Giảng viên; trước khi đóng không có điểm.
func TestResultsListStaff(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	absent := r.student("ACTIVE")
	r.exec(`update users set full_name='Vắng Mặt' where id=$1`, absent)
	m.finish(true, false)
	ctx := t.Context()
	page, rows, err := r.svc.ResultsList(ctx, r.course, m.e.ID, exam.ResultsFilter{Teacher: true}, nil, 10)
	require.NoError(t, err)
	require.Equal(t, exam.ResultsProgress{NotStarted: 1, Graded: 1}, page.Progress)
	for _, x := range rows {
		require.Nil(t, x.Score, "bài còn mở: không điểm")
	}
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	page, rows, err = r.svc.ResultsList(ctx, r.course, m.e.ID, exam.ResultsFilter{Teacher: true}, nil, 10)
	require.NoError(t, err)
	require.Equal(t, exam.ResultsProgress{Absent: 1, Graded: 1}, page.Progress)
	require.Len(t, rows, 2)
	require.Equal(t, "GRADED", rows[0].Status)
	require.Equal(t, "5.00", *rows[0].Score)
	require.NotNil(t, rows[0].Flags)
	require.Equal(t, "ABSENT", rows[1].Status)
	require.Nil(t, rows[1].Score)
	pq, q1, err := r.svc.ResultsList(ctx, r.course, m.e.ID, exam.ResultsFilter{Q: "vắng"}, nil, 10)
	require.NoError(t, err)
	require.Len(t, q1, 1, "ô tìm lọc theo tên")
	require.Equal(t, exam.ResultsProgress{Absent: 1, Graded: 1}, pq.Progress, "tiến độ vẫn tính trên cả lớp")
	_, ta, err := r.svc.ResultsList(ctx, r.course, m.e.ID, exam.ResultsFilter{Teacher: false}, nil, 10)
	require.NoError(t, err)
	require.Nil(t, ta[0].Flags, "TA không có `flags`")
	// con trỏ: trang 1 một dòng, trang 2 phần còn lại, không lặp
	_, p1, _ := r.svc.ResultsList(ctx, r.course, m.e.ID, exam.ResultsFilter{Sort: "name"}, nil, 1)
	_, p2, err := r.svc.ResultsList(ctx, r.course, m.e.ID, exam.ResultsFilter{Sort: "name"}, &p1[0].Student.ID, 5)
	require.NoError(t, err)
	require.Len(t, p2, 1)
	require.NotEqual(t, p1[0].Student.ID, p2[0].Student.ID)
	_, _, err = r.svc.ResultsList(ctx, r.course, m.e.ID, exam.ResultsFilter{}, new(uuid.New()), 5)
	require.Equal(t, 422, must(apiStatus(t, err)))
}

// TestResultsCSV — AC12: BOM, `;`, dấu phẩy thập phân, ô bắt đầu bằng `= + - @` thêm `'`, ABSENT điểm trống, không cột liêm chính.
func TestResultsCSV(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	absent := r.student("ACTIVE")
	r.exec(`update users set full_name='=HYPERLINK("x")', student_code='+84' where id=$1`, absent)
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	var b strings.Builder
	require.NoError(t, r.svc.ResultsCSV(t.Context(), r.course, m.e.ID, &b, func() {}))
	out := b.String()
	require.True(t, strings.HasPrefix(out, "\ufeffmssv;ho_ten;trang_thai;diem_tu_dong;diem_chinh_thuc;nop_luc;ly_do_nop;cau_1;cau_2\n"), out)
	require.Contains(t, out, ";GRADED;5,00;5,00;")
	require.Contains(t, out, "'+84;\"'=HYPERLINK(\"\"x\"\")\";ABSENT;;;;")
	require.NotContains(t, strings.ToLower(out), "tab_hidden")
}

// TestExamStats — AC11: phân bố 10 khoảng nửa mở (khoảng cuối đóng), tổng count = số lượt GRADED, trung bình / trung vị, câu sai nhiều loại câu void.
func TestExamStats(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	st, err := r.svc.ExamStats(t.Context(), r.course, m.e.ID)
	require.NoError(t, err)
	require.Len(t, st.Distribution, 10)
	require.Equal(t, 1, st.Distribution[5].Count, "5,00 rơi vào [5;6)")
	total := 0
	for _, b := range st.Distribution {
		total += b.Count
	}
	require.Equal(t, 1, total)
	require.Equal(t, "5.00", st.Mean)
	require.Equal(t, "5.00", st.Median)
	require.Len(t, st.Hardest, 1)
	require.Equal(t, "1.00", st.Hardest[0].CorrectRate)
	_, err = r.svc.OverrideItem(t.Context(), r.teacher, r.course, m.e.ID, m.mcqItem, exam.OverrideIn{Void: new(true), Reason: "x"})
	require.NoError(t, err)
	st, _ = r.svc.ExamStats(t.Context(), r.course, m.e.ID)
	require.Empty(t, st.Hardest, "câu void bị loại")
}

// TestGradeDueSafetyNet — AC2: tin `exam.submission_done` mất → tick `GradeDue` vẫn hoàn tất lượt.
func TestGradeDueSafetyNet(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := m.finish(false, true)
	m.judged(sub, "AC")
	n, err := r.svc.GradeDue(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	_, _, sc, _ := r.attemptRow(m.v.Attempt.ID)
	require.Equal(t, "5.00", *sc)
}

// TestOverrideLargeUsesJob — AC14: nhiều hơn ngưỡng đồng bộ → 202 + việc nền `exam.regrade` (scope recompute) giữ `regrading`; chạy việc xong thì mọi lượt được tính lại.
func TestOverrideLargeUsesJob(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	second := r.student("ACTIVE")
	v2, _ := r.start(m.e, second, uuid.New())
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	small := *r.svc
	small.Results = exam.ResultsConfig{RecomputeSync: 1}
	r.exec(`update exam_attempts set status='GRADED', auto_score=0, graded_at=now(), submitted_at=now(), submit_reason='MANUAL' where id=$1`, v2.Attempt.ID)
	out, err := small.OverrideItem(t.Context(), r.teacher, r.course, m.e.ID, m.mcqItem, exam.OverrideIn{Void: new(true), Reason: "đề sai"})
	require.NoError(t, err)
	require.NotNil(t, out.JobID)
	require.Equal(t, 1, r.count(`select count(*)::int from exams where id=$1 and regrading`, m.e.ID))
	run := newRunner(t, r, &exam.Worker{})
	require.Equal(t, "SUCCEEDED", runJob(t, r, run, *out.JobID).Status)
}

// TestCSVOver5000Rejected — SRS 4.8.6: vượt trần dòng → 422 EXPORT_TOO_LARGE, KHÔNG cắt im lặng, chưa ghi byte nào.
func TestCSVOver5000Rejected(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	r.student("ACTIVE")
	m.finish(true, false)
	m.closeExam()
	small := *r.svc
	small.Results = exam.ResultsConfig{CSVMaxRows: 1}
	var b strings.Builder
	err := small.ResultsCSV(t.Context(), r.course, m.e.ID, &b, func() { t.Fatal("không được ghi header") })
	require.Equal(t, 422, must(apiStatus(t, err)))
	require.Empty(t, b.String())
}

// closingExam: bài đóng sau 10 phút (hạn của lượt = giờ đóng), n sinh viên đã bắt đầu, đồng hồ giả bắt đầu từ bây giờ.
func (r *rig) closingExam(n int) (exam.ExamDetail, []exam.AttemptStartView, []uuid.UUID, *clock.Fake) {
	r.t.Helper()
	e := r.openExam("sắp đóng", false, r.mcqSet(2)...)
	r.exec(`update exams set opens_at = now() - interval '35 minutes', closes_at = now() + interval '10 minutes' where id=$1`, e.ID)
	clk := clock.NewFake(time.Now().UTC())
	var vs []exam.AttemptStartView
	var svs []uuid.UUID
	for range n {
		sv := r.student("ACTIVE")
		v, _, err := r.at(clk).StartAttempt(r.t.Context(), sv, r.course, e.ID, uuid.New())
		require.NoError(r.t, err)
		vs, svs = append(vs, v), append(svs, sv)
	}
	return e, vs, svs, clk
}

// TestCloseForceSubmitsAll — AC1: tới giờ đóng, mọi lượt IN_PROGRESS được tự nộp với `CLOSED` (hạn = giờ đóng) và chấm xong; `exam.closed` đúng một lần dù tick chạy lại; sau đó không bắt đầu / lưu mới.
func TestCloseForceSubmitsAll(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e, vs, svs, clk := r.closingExam(3)
	r.exec(`update exams set publish_hold = true where id=$1`, e.ID)
	clk.Advance(10*time.Minute + 5*time.Second) // trong grace: chưa nộp
	r.tick(clk)
	require.Equal(t, "IN_PROGRESS", r.attemptStatus(vs[0].Attempt.ID))
	clk.Advance(30 * time.Second)
	r.tick(clk)
	r.tick(clk)
	for _, v := range vs {
		st, reason, _, _ := r.attemptRow(v.Attempt.ID)
		require.Equal(t, "GRADED", st)
		require.Equal(t, "CLOSED", *reason)
	}
	require.Equal(t, "CLOSED", r.examStatus(e.ID), "đang hoãn: chưa công bố")
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.closed' and payload->>'exam_id'=$1`, e.ID.String()))
	page, _, err := r.svc.ResultsList(t.Context(), r.course, e.ID, exam.ResultsFilter{}, nil, 10)
	require.NoError(t, err)
	require.Equal(t, exam.ResultsProgress{Graded: 3, Absent: 0}, page.Progress, "Giảng viên thấy Đang chấm {graded}/{graded + grading}")
	// CLOSED: không bắt đầu / lưu / nộp mới
	late := r.student("ACTIVE")
	_, _, err = r.at(clk).StartAttempt(t.Context(), late, r.course, e.ID, uuid.New())
	_, code := apiStatus(t, err)
	require.Equal(t, "EXAM_NOT_OPEN", code)
	_, err = r.at(clk).SaveAnswers(t.Context(), svs[0], r.course, e.ID, vs[0].Attempt.ID, uuid.New(), exam.AnswersIn{Items: []exam.AnswerIn{pick(vs[0].Items[0], 0)}})
	st, _ := apiStatus(t, err)
	require.Equal(t, 409, st)
}

// TestPublishFlowConcurrency — AC20: tick công bố + chấm + chấm lại + sửa điểm + hoãn / bỏ hoãn chạy song song → đúng MỘT trạng thái cuối hợp lệ, `published_at` đặt một lần,
// `exam.published` một sự kiện, thông báo không trùng, không điểm "nhảy" (điểm chính thức cuối = giá trị ghi sau cùng hợp lệ).
func TestPublishFlowConcurrency(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e, vs, _, clk := r.closingExam(5)
	clk.Advance(11 * time.Minute)
	r.exec(`update exams set status='CLOSED', opens_at = now() - interval '4 hours', closes_at = now() - interval '1 minute', publish_hold = true where id=$1`, e.ID)
	for _, v := range vs { // lượt nộp tay trước khi đóng (đã chấm)
		r.exec(`update exam_attempts set status='GRADED', auto_score=5, graded_at=now(), submitted_at=now(), submit_reason='MANUAL' where id=$1`, v.Attempt.ID)
	}
	ctx := t.Context()
	var wg sync.WaitGroup
	for g := range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 12 {
				switch (g + i) % 6 {
				case 0:
					_, _ = r.svc.PublishDue(ctx)
				case 1:
					_, _ = r.svc.GradeDue(ctx)
				case 2:
					_, _ = r.svc.RegradeSweep(ctx)
				case 3:
					d, err := r.svc.GetExam(ctx, r.course, e.ID)
					if err == nil {
						_, _ = r.svc.SetHold(ctx, r.teacher, r.course, e.ID, i%2 == 0, d.Version)
					}
				case 4:
					var ver int
					if r.pool.QueryRow(ctx, `select version from exam_attempts where id=$1`, vs[g%len(vs)].Attempt.ID).Scan(&ver) == nil {
						_, _ = r.svc.AdjustScore(ctx, r.teacher, r.course, e.ID, vs[g%len(vs)].Attempt.ID, exam.AdjustIn{Score: new(fmt.Sprintf("%d", 1+i%9)), Reason: "đồng thời", Version: ver})
					}
				case 5:
					_ = r.ticker(clk).Tick(ctx)
				}
			}
		}()
	}
	wg.Wait()
	// kết thúc: bỏ hoãn và để công bố chạy nốt
	d, err := r.svc.GetExam(ctx, r.course, e.ID)
	require.NoError(t, err)
	if d.PublishHold {
		_, err = r.svc.SetHold(ctx, r.teacher, r.course, e.ID, false, d.Version)
		require.NoError(t, err)
	}
	_, err = r.svc.PublishDue(ctx)
	require.NoError(t, err)
	require.Equal(t, "PUBLISHED", r.examStatus(e.ID))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.published' and payload->>'exam_id'=$1`, e.ID.String()), "một lần công bố, một sự kiện")
	require.Equal(t, 1, r.count(`select count(*) from exams where id=$1 and published_at is not null and not regrading`, e.ID))
	n := &exam.ResultNotifier{Pool: r.pool}
	msg := r.message("exam.published")
	require.NoError(t, n.HandlePublished(ctx, msg))
	require.NoError(t, n.HandlePublished(ctx, msg))
	require.Equal(t, 5, r.count(`select count(*) from notifications where type='EXAM_PUBLISHED' and course_id=$1`, r.course))
	// điểm chính thức của mỗi lượt khớp đúng một lần ghi (adjusted nếu có, nếu không 5,00): không giá trị lạ
	for _, v := range vs {
		var auto, adj *string
		require.NoError(t, r.pool.QueryRow(ctx, `select auto_score::text, adjusted_score::text from exam_attempts where id=$1`, v.Attempt.ID).Scan(&auto, &adj))
		require.Equal(t, "5.00", *auto, "auto_score không bị ghi đè")
	}
}

// TestPublishFlowWorkerKilled — AC20: worker chết SAU khi công bố (giao dịch đã commit, outbox chưa xử lý) hoặc giữa chừng xử lý → xử lý lại cho kết quả như một lần: thông báo không trùng,
// lượt kẹt GRADING được tick nhặt khi bản nộp đã DONE.
func TestPublishFlowWorkerKilled(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := m.finish(true, true)
	ctx := t.Context()
	// worker chết sau khi bản nộp DONE nhưng trước khi xử lý `exam.submission_done`: không có sự kiện nào chạy → tick nhặt
	m.judged(sub, "AC")
	require.Equal(t, "GRADING", r.attemptStatus(m.v.Attempt.ID))
	n, err := r.svc.GradeDue(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	m.closeExam()
	n, err = r.svc.PublishDue(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	// worker chết giữa lúc xử lý `exam.published` rồi được giao lại hai lần
	rn := &exam.ResultNotifier{Pool: r.pool}
	msg := r.message("exam.published")
	require.NoError(t, rn.HandlePublished(ctx, msg))
	require.NoError(t, rn.HandlePublished(ctx, msg))
	require.NoError(t, rn.HandlePublished(ctx, msg))
	require.Equal(t, 1, r.count(`select count(*) from notifications where type='EXAM_PUBLISHED' and user_id=$1`, m.sv))
	require.Equal(t, "PUBLISHED", r.examStatus(m.e.ID))
	n, _ = r.svc.PublishDue(ctx)
	require.Zero(t, n, "chạy lại tick: không công bố lần hai")
}

// TestResultsHideIntegrityFromTA — US-PE-07 AC4 (chuyển sang PE-08): chi tiết một lượt chỉ kèm `integrity` (sự kiện liêm chính) và bảng điểm chỉ kèm `flags` cho Giảng viên.
func TestResultsHideIntegrityFromTA(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	ta, err := r.svc.ResultDetail(t.Context(), r.course, m.e.ID, m.v.Attempt.ID, false)
	require.NoError(t, err)
	require.Nil(t, ta.Integrity)
	gv, err := r.svc.ResultDetail(t.Context(), r.course, m.e.ID, m.v.Attempt.ID, true)
	require.NoError(t, err)
	require.NotNil(t, gv.Integrity)
	raw, _ := json.Marshal(ta)
	require.NotContains(t, string(raw), "integrity")
}

// TestRegradeIdempotent — AC15: chấm lại idempotent theo (bản nộp, tests_version): bản đã chấm bằng bộ test hiện hành không bị chấm đôi; bản lỗi (ERROR / IE) thì luôn được chấm lại;
// `scope:"errors"` chỉ chạm bản lỗi.
func TestRegradeIdempotent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	sub := m.finish(true, true)
	m.judged(sub, "AC")
	require.NoError(t, r.svc.OnSubmissionDone(t.Context(), r.course, m.v.Attempt.ID))
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	run := newRunner(t, r, &exam.Worker{})
	requeued := func(scope string) float64 {
		jid, err := r.svc.Regrade(t.Context(), r.teacher, r.course, m.e.ID, exam.RegradeIn{Scope: scope, Reason: "thử"})
		require.NoError(t, err)
		j := runJob(t, r, run, jid)
		require.Equal(t, "SUCCEEDED", j.Status)
		var res map[string]float64
		require.NoError(t, json.Unmarshal(j.Result, &res))
		return res["requeued"]
	}
	require.Zero(t, requeued("all"), "cùng tests_version: không chấm đôi")
	require.Zero(t, requeued("errors"))
	r.exec(`update code_submissions set status='ERROR', verdict='IE' where id=$1`, sub)
	require.EqualValues(t, 1, requeued("errors"), "bản lỗi được chấm lại")
	require.Equal(t, "QUEUED", func() string {
		var s string
		require.NoError(t, r.pool.QueryRow(t.Context(), `select status::text from code_submissions where id=$1`, sub).Scan(&s))
		return s
	}())
}
