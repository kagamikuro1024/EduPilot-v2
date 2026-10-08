package exam_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// ---- helper -----------------------------------------------------------------------------------------------------------------

// openExam dựng một bài ĐANG MỞ (đã qua `schedule`) gồm các câu cho trước rồi dời giờ trong DB: mở 1 phút trước, đóng sau 3 giờ, làm 45 phút.
func (r *rig) openExam(title string, shuffle bool, qs ...uuid.UUID) exam.ExamDetail {
	r.t.Helper()
	in := goodIn(title)
	in.ShuffleQuestions, in.ShuffleOptions = &shuffle, &shuffle
	e := r.mustItems(r.newExam(in), qs...)
	_, err := r.schedule(e)
	require.NoError(r.t, err)
	r.exec(`update exams set status='OPEN', opens_at = now() - interval '1 minute', closes_at = now() + interval '3 hours' where id=$1`, e.ID)
	d, err := r.svc.GetExam(r.t.Context(), r.course, e.ID)
	require.NoError(r.t, err)
	return d
}

// mcqSet tạo n câu MCQ_SINGLE đã duyệt, mỗi câu có 4 đáp án (đáp án thứ 4 ghim cuối), đúng là đáp án thứ 2.
func (r *rig) mcqSet(n int) []uuid.UUID {
	r.t.Helper()
	var out []uuid.UUID
	for i := range n {
		q, err := r.svc.Create(r.t.Context(), r.teacher, r.course, exam.QuestionIn{Type: "MCQ_SINGLE", Title: fmt.Sprintf("q%d", i), Topic: "t", Difficulty: "EASY", Stem: fmt.Sprintf("Đề %d", i),
			Options: []exam.OptionIn{{Body: "A"}, {Body: "B"}, {Body: "C"}, {Body: "Tất cả đáp án trên", PinnedLast: true}}, Correct: []int{1}})
		require.NoError(r.t, err)
		_, err = r.svc.Review(r.t.Context(), r.teacher, r.course, q.ID, "APPROVE", q.Version)
		require.NoError(r.t, err)
		out = append(out, q.ID)
	}
	return out
}

func (r *rig) start(e exam.ExamDetail, sv uuid.UUID, tab uuid.UUID) (exam.AttemptStartView, bool) {
	r.t.Helper()
	v, created, err := r.svc.StartAttempt(r.t.Context(), sv, r.course, e.ID, tab)
	require.NoError(r.t, err)
	return v, created
}

func (r *rig) save(v exam.AttemptStartView, sv, tab uuid.UUID, items ...exam.AnswerIn) (exam.SaveView, error) {
	r.t.Helper()
	return r.svc.SaveAnswers(r.t.Context(), sv, r.course, v.Attempt.ExamID, v.Attempt.ID, tab, exam.AnswersIn{Items: items})
}

func pick(item exam.ItemView, idx int) exam.AnswerIn {
	b, _ := json.Marshal(map[string]any{"option_ids": []string{item.Options[idx].ID.String()}})
	return exam.AnswerIn{ItemID: item.ItemID, Answer: b}
}

// correctOption trả id đáp án ĐÚNG của câu (đáp án có nội dung "B") trong thứ tự hiển thị của item.
func correctOption(item exam.ItemView) uuid.UUID {
	for _, o := range item.Options {
		if o.Body == "B" {
			return o.ID
		}
	}
	return uuid.Nil
}

func answerCorrect(item exam.ItemView) exam.AnswerIn {
	b, _ := json.Marshal(map[string]any{"option_ids": []string{correctOption(item).String()}})
	return exam.AnswerIn{ItemID: item.ItemID, Answer: b}
}

func (r *rig) attemptRow(id uuid.UUID) (status string, reason *string, score *string, submitted *time.Time) {
	r.t.Helper()
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select status::text, submit_reason::text, auto_score::text, submitted_at from exam_attempts where id=$1`, id).Scan(&status, &reason, &score, &submitted))
	return
}

// ---- AC1: bắt đầu -----------------------------------------------------------------------------------------------------------

// TestStartAttemptOK — AC1: tạo MỘT lượt IN_PROGRESS với started_at = now (đồng hồ máy chủ), deadline = started + thời lượng, trả cấu trúc lượt kèm server_time; outbox `exam.attempt_started`.
func TestStartAttemptOK(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	e := r.openExam("làm bài", true, r.mcqSet(3)...)
	tab := uuid.New()
	v, created := r.start(e, sv, tab)
	require.True(t, created)
	require.Equal(t, "IN_PROGRESS", v.Attempt.Status)
	require.Equal(t, e.ID, v.Attempt.ExamID)
	require.True(t, v.Attempt.Writer.IsYou)
	require.WithinDuration(t, time.Now(), v.Attempt.ServerTime, 3*time.Second)
	require.True(t, v.Attempt.DeadlineAt.Equal(v.Attempt.StartedAt.Add(45*time.Minute)))
	require.Len(t, v.Items, 3)
	require.Equal(t, 45, v.Exam.DurationMinutes)
	require.Equal(t, "PARTIAL", v.Exam.MultiScoring)
	require.Equal(t, 1, r.count(`select count(*) from exam_attempts where exam_id=$1 and student_id=$2`, e.ID, sv))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.attempt_started' and payload->>'attempt_id'=$1`, v.Attempt.ID.String()))
	// người không phải sinh viên của lớp: guard chặn ở HTTP; ở service, bài của lớp khác không có
	other := r.otherCourse()
	_, _, err := r.svc.StartAttempt(t.Context(), sv, other, e.ID, tab)
	st, _ := apiStatus(t, err)
	require.Equal(t, 404, st)
}

// TestStartAttemptNotOpenReasons — AC1: bài chưa lên lịch / chưa mở / đã đóng / còn dưới 60 s → 409 EXAM_NOT_OPEN với `details.reason` đúng; lớp lưu trữ → 409 COURSE_ARCHIVED.
func TestStartAttemptNotOpenReasons(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	q := r.mcqSet(1)
	try := func(eid uuid.UUID) (string, map[string]any) {
		t.Helper()
		_, _, err := r.svc.StartAttempt(t.Context(), sv, r.course, eid, uuid.New())
		st, code := apiStatus(t, err)
		require.Equal(t, 409, st)
		require.Equal(t, "EXAM_NOT_OPEN", code)
		b, _ := json.Marshal(detailsOf(t, err))
		var d map[string]any
		_ = json.Unmarshal(b, &d)
		return d["reason"].(string), d
	}
	draft := r.mustItems(r.newExam(goodIn("nháp")), q...)
	reason, _ := try(draft.ID)
	require.Equal(t, "not_scheduled", reason)
	sched, err := r.schedule(draft)
	require.NoError(t, err)
	reason, d := try(sched.ID)
	require.Equal(t, "not_yet", reason)
	require.NotEmpty(t, d["opens_at"])
	for _, st := range []string{"CLOSED", "PUBLISHED"} {
		reason, _ = try(r.useIn(q[0], st))
		require.Equal(t, "closed", reason, st)
	}
	// còn 30 s tới giờ đóng: closing; còn 90 s: được
	e := r.openExam("sắp đóng", false, q...)
	r.exec(`update exams set closes_at = now() + interval '30 seconds', duration_minutes = 1 where id=$1`, e.ID)
	reason, _ = try(e.ID)
	require.Equal(t, "closing", reason)
	r.exec(`update exams set closes_at = now() + interval '90 seconds' where id=$1`, e.ID)
	_, created := r.start(e, sv, uuid.New())
	require.True(t, created)
	// lớp lưu trữ
	r.exec(`update courses set status='ARCHIVED', archived_at=now(), join_enabled=false where id=$1`, r.course)
	_, _, err = r.svc.StartAttempt(t.Context(), sv, r.course, e.ID, uuid.New())
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "COURSE_ARCHIVED", code)
}

// TestStartAttemptIdempotent — AC1: gọi lại (cùng hay khác khoá ở tầng HTTP) → CÙNG lượt, không lượt thứ hai, không thêm sự kiện.
func TestStartAttemptIdempotent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	e := r.openExam("lặp", true, r.mcqSet(2)...)
	a, c1 := r.start(e, sv, uuid.New())
	b, c2 := r.start(e, sv, uuid.New())
	require.True(t, c1)
	require.False(t, c2)
	require.Equal(t, a.Attempt.ID, b.Attempt.ID)
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.attempt_started'`))
}

// TestStartAttemptDeadlineMin — AC1: deadline = min(started + duration, closes_at): bắt đầu khi còn 10 phút tới giờ đóng với thời lượng 45 → deadline = closes_at (đồng hồ giả).
func TestStartAttemptDeadlineMin(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	e := r.openExam("hạn chót", false, r.mcqSet(1)...)
	clk := clock.NewFake(time.Now().UTC())
	r.exec(`update exams set opens_at = $2, closes_at = $3 where id=$1`, e.ID, clk.Now().Add(-50*time.Minute), clk.Now().Add(10*time.Minute)) // khung 60 phút ≥ thời lượng 45
	v, _, err := r.at(clk).StartAttempt(t.Context(), sv, r.course, e.ID, uuid.New())
	require.NoError(t, err)
	require.True(t, v.Attempt.DeadlineAt.Equal(clk.Now().Add(10*time.Minute)), "deadline = closes_at, có %s", v.Attempt.DeadlineAt)
	// khi còn dư giờ: deadline = started + 45
	sv2 := r.student("ACTIVE")
	r.exec(`update exams set closes_at = $2 where id=$1`, e.ID, clk.Now().Add(3*time.Hour))
	v2, _, err := r.at(clk).StartAttempt(t.Context(), sv2, r.course, e.ID, uuid.New())
	require.NoError(t, err)
	require.True(t, v2.Attempt.DeadlineAt.Equal(clk.Now().Add(45*time.Minute)))
}

// ---- AC2: làm tiếp ---------------------------------------------------------------------------------------------------------

// TestResumeSameAttempt — AC2: `attempts/mine` khi đang làm trả cùng lượt, kèm câu trả lời đã lưu; POST lại cũng cùng lượt.
func TestResumeSameAttempt(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	e := r.openExam("tiếp", true, r.mcqSet(4)...)
	tab := uuid.New()
	v, _ := r.start(e, sv, tab)
	_, err := r.save(v, sv, tab, pick(v.Items[0], 2), pick(v.Items[3], 0))
	require.NoError(t, err)
	got, err := r.svc.MyAttempt(t.Context(), sv, r.course, e.ID, tab)
	require.NoError(t, err)
	m, ok := got.(exam.AttemptStartView)
	require.True(t, ok, "đang làm → khối có đề")
	require.Equal(t, v.Attempt.ID, m.Attempt.ID)
	require.True(t, m.Attempt.StartedAt.Equal(v.Attempt.StartedAt))
	require.True(t, m.Attempt.DeadlineAt.Equal(v.Attempt.DeadlineAt))
	for i, it := range m.Items {
		require.Equal(t, v.Items[i].ItemID, it.ItemID, "cùng thứ tự")
	}
	var a0 map[string][]string
	require.NoError(t, json.Unmarshal(m.Items[0].Answer, &a0))
	require.Equal(t, []string{v.Items[0].Options[2].ID.String()}, a0["option_ids"])
	require.NotNil(t, m.Items[3].Answer)
	require.Nil(t, m.Items[1].Answer)
	// chưa có lượt: dạng giới thiệu
	sv2 := r.student("ACTIVE")
	none, err := r.svc.MyAttempt(t.Context(), sv2, r.course, e.ID, tab)
	require.NoError(t, err)
	_, isNone := none.(exam.NoAttemptView)
	require.True(t, isNone)
}

// TestClockNotResetOnReload — AC2: đồng hồ tính từ DB: bắt đầu → 20 phút sau (đồng hồ giả, như tải lại / đổi thiết bị) → started_at / deadline_at không đổi.
func TestClockNotResetOnReload(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	e := r.openExam("đồng hồ", true, r.mcqSet(2)...)
	clk := clock.NewFake(time.Now().UTC())
	svc := r.at(clk)
	v, _, err := svc.StartAttempt(t.Context(), sv, r.course, e.ID, uuid.New())
	require.NoError(t, err)
	clk.Advance(20 * time.Minute)
	for range 3 {
		got, err := svc.MyAttempt(t.Context(), sv, r.course, e.ID, uuid.New())
		require.NoError(t, err)
		m := got.(exam.AttemptStartView)
		require.True(t, m.Attempt.StartedAt.Equal(v.Attempt.StartedAt))
		require.True(t, m.Attempt.DeadlineAt.Equal(v.Attempt.DeadlineAt))
		require.True(t, m.Attempt.ServerTime.Equal(clk.Now()))
	}
	again, created, err := svc.StartAttempt(t.Context(), sv, r.course, e.ID, uuid.New())
	require.NoError(t, err)
	require.False(t, created)
	require.True(t, again.Attempt.DeadlineAt.Equal(v.Attempt.DeadlineAt))
}

// TestOneAttemptRaceStart — AC2: 20 yêu cầu bắt đầu song song của cùng một sinh viên → đúng MỘT dòng, đúng một "tạo mới", mọi phản hồi cùng một lượt.
func TestOneAttemptRaceStart(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	e := r.openExam("đua", true, r.mcqSet(2)...)
	var wg sync.WaitGroup
	ids := make([]uuid.UUID, 20)
	made := make([]bool, 20)
	for i := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, created, err := r.svc.StartAttempt(t.Context(), sv, r.course, e.ID, uuid.New())
			require.NoError(t, err)
			ids[i], made[i] = v.Attempt.ID, created
		}()
	}
	wg.Wait()
	n := 0
	for i := range ids {
		require.Equal(t, ids[0], ids[i])
		if made[i] {
			n++
		}
	}
	require.Equal(t, 1, n, "đúng một yêu cầu tạo lượt")
	require.Equal(t, 1, r.count(`select count(*) from exam_attempts where exam_id=$1`, e.ID))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.attempt_started'`))
}

// TestStartAfterSubmit409 — AC2: đã nộp (GRADING / GRADED) → bắt đầu lại 409 ATTEMPT_ALREADY_SUBMITTED.
func TestStartAfterSubmit409(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	e := r.openExam("nộp rồi", true, r.mcqSet(2)...)
	tab := uuid.New()
	v, _ := r.start(e, sv, tab)
	_, err := r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
	require.NoError(t, err)
	_, _, err = r.svc.StartAttempt(t.Context(), sv, r.course, e.ID, tab)
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "ATTEMPT_ALREADY_SUBMITTED", code)
	require.Equal(t, 1, r.count(`select count(*) from exam_attempts where exam_id=$1`, e.ID))
}

// ---- AC4: cấu trúc lượt, chống rò -------------------------------------------------------------------------------------------

func canaryExam(t *testing.T, r *rig) (exam.ExamDetail, string) {
	t.Helper()
	const canary = "CANARY7Q-KHONG-DUOC-LO"
	var qs []uuid.UUID
	for i := range 4 {
		q, err := r.svc.Create(t.Context(), r.teacher, r.course, exam.QuestionIn{Type: "MCQ_SINGLE", Title: fmt.Sprintf("q%d", i), Topic: "t", Difficulty: "EASY", Stem: fmt.Sprintf("Đề %d", i), Explanation: new(canary),
			Options: []exam.OptionIn{{Body: "A"}, {Body: "B"}, {Body: "C"}, {Body: "Tất cả", PinnedLast: true}}, Correct: []int{1}})
		require.NoError(t, err)
		_, err = r.svc.Review(t.Context(), r.teacher, r.course, q.ID, "APPROVE", q.Version)
		require.NoError(t, err)
		qs = append(qs, q.ID)
	}
	cq, _, _ := r.codeWithTests("code")
	r.exec(`update code_testcases set name = 'CANARYhidden', input = $2, expected = $2 where problem_id = $1 and not is_sample`, cq.ID, canary)
	r.exec(`update code_problems set reference_source = $2 where question_id = $1`, cq.ID, "// "+canary)
	r.exec(`update question_bank set answer_key = jsonb_set(answer_key, '{canary}', to_jsonb($2::text)) where id = any($1::uuid[])`, qs, canary)
	r.verify(cq.ID)
	_, err := r.svc.Review(t.Context(), r.teacher, r.course, cq.ID, "APPROVE", r.version(cq.ID))
	require.NoError(t, err)
	e := r.openExam("canary", true, append(qs, cq.ID)...)
	r.exec(`update exam_items set override = jsonb_build_object('reason', $2::text) where exam_id = $1`, e.ID, canary)
	return e, canary
}

// TestAttemptPayloadWhitelist — AC4: khoá JSON của mọi mục = thẻ json của ItemView; khối attempt / exam đúng danh sách cho phép (`testdata/student_dto_allowlist.json`).
func TestAttemptPayloadWhitelist(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	e := r.openExam("whitelist", true, r.mcqSet(3)...)
	v, _ := r.start(e, sv, uuid.New())
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var top map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &top))
	require.ElementsMatch(t, []string{"attempt", "exam", "items"}, keysOf(top))
	var blocks struct {
		Attempt map[string]json.RawMessage   `json:"attempt"`
		Exam    map[string]json.RawMessage   `json:"exam"`
		Items   []map[string]json.RawMessage `json:"items"`
	}
	require.NoError(t, json.Unmarshal(raw, &blocks))
	require.ElementsMatch(t, jsonKeys(exam.AttemptView{}), keysOf(blocks.Attempt))
	require.ElementsMatch(t, jsonKeys(exam.ExamBlockView{}), keysOf(blocks.Exam))
	for _, it := range blocks.Items {
		require.ElementsMatch(t, jsonKeys(exam.ItemView{}), keysOf(it))
	}
	// điểm là chuỗi thập phân
	require.Equal(t, "1.00", v.Items[0].Points)
}

// TestAttemptPayloadNoAnswerCanary — AC4: canary trong answer_key, explanation, test ẩn, lời giải mẫu, override: duyệt mọi byte phản hồi (bắt đầu, làm tiếp, lưu, nộp, tóm tắt) → 0 lần xuất hiện.
func TestAttemptPayloadNoAnswerCanary(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	e, canary := canaryExam(t, r)
	tab := uuid.New()
	scan := func(what string, v any) {
		t.Helper()
		raw, err := json.Marshal(v)
		require.NoError(t, err)
		s := string(raw)
		require.NotContains(t, s, canary, what)
		for _, bad := range []string{"answer_key", "correct", "explanation", "pinned_last", "override", "hidden", "reference", "weight", "tests_version", "similarity", "events", "is_correct"} {
			require.NotContains(t, s, bad, what)
		}
	}
	v, _ := r.start(e, sv, tab)
	scan("bắt đầu", v)
	got, err := r.svc.MyAttempt(t.Context(), sv, r.course, e.ID, tab)
	require.NoError(t, err)
	scan("làm tiếp", got)
	save, err := r.save(v, sv, tab, answerCorrect(v.Items[0]))
	require.NoError(t, err)
	scan("lưu", save)
	sub, err := r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
	require.NoError(t, err)
	scan("nộp", sub)
	got, err = r.svc.MyAttempt(t.Context(), sv, r.course, e.ID, tab)
	require.NoError(t, err)
	scan("sau nộp", got)
	// đáp án của sinh viên khác không lộ: sinh viên 2 không thấy lựa chọn của sinh viên 1
	sv2 := r.student("ACTIVE")
	e2 := r.openExam("khác", true, r.mcqSet(2)...)
	v2, _ := r.start(e2, sv2, uuid.New())
	for _, it := range v2.Items {
		require.Nil(t, it.Answer)
	}
}

// ---- AC5: xáo trộn -------------------------------------------------------------------------------------------------------------

func orderOf(v exam.AttemptStartView) string {
	ids := make([]string, 0, len(v.Items)*5)
	for _, it := range v.Items {
		ids = append(ids, it.ItemID.String())
		for _, o := range it.Options {
			ids = append(ids, o.ID.String())
		}
	}
	return strings.Join(ids, ",")
}

// TestShuffleDeterministic — AC5: cùng lượt luôn cùng thứ tự (làm tiếp, đổi thiết bị); thứ tự là hàm của attempt_id (không lưu).
func TestShuffleDeterministic(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	e := r.openExam("tất định", true, r.mcqSet(12)...)
	v, _ := r.start(e, sv, uuid.New())
	for range 5 {
		got, err := r.svc.MyAttempt(t.Context(), sv, r.course, e.ID, uuid.New())
		require.NoError(t, err)
		require.Equal(t, orderOf(v), orderOf(got.(exam.AttemptStartView)))
	}
	require.Equal(t, exam.AttemptSeed(v.Attempt.ID), exam.AttemptSeed(v.Attempt.ID))
	require.NotEqual(t, exam.AttemptSeed(v.Attempt.ID), exam.AttemptSeed(uuid.New()))
}

// TestShuffleDiffersAcrossStudents — AC5: ≥ 95 % cặp sinh viên có thứ tự câu khác nhau (bài ≥ 10 câu).
func TestShuffleDiffersAcrossStudents(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e := r.openExam("khác nhau", true, r.mcqSet(12)...)
	var orders []string
	for range 30 {
		v, _ := r.start(e, r.student("ACTIVE"), uuid.New())
		ids := make([]string, len(v.Items))
		for i, it := range v.Items {
			ids[i] = it.ItemID.String()
		}
		orders = append(orders, strings.Join(ids, ","))
	}
	diff, pairs := 0, 0
	for i := range orders {
		for j := i + 1; j < len(orders); j++ {
			pairs++
			if orders[i] != orders[j] {
				diff++
			}
		}
	}
	require.GreaterOrEqual(t, float64(diff)/float64(pairs), 0.95)
}

// TestShufflePermutationComplete — AC5: mỗi câu và mỗi đáp án xuất hiện đúng MỘT lần; vị trí 1…n liên tục.
func TestShufflePermutationComplete(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	qs := r.mcqSet(10)
	e := r.openExam("hoán vị", true, qs...)
	for range 5 {
		v, _ := r.start(e, r.student("ACTIVE"), uuid.New())
		seen := map[uuid.UUID]bool{}
		for i, it := range v.Items {
			require.Equal(t, i+1, it.Position)
			require.False(t, seen[it.ItemID])
			seen[it.ItemID] = true
			require.Len(t, it.Options, 4)
			bodies := map[string]bool{}
			for _, o := range it.Options {
				bodies[o.Body] = true
			}
			require.Len(t, bodies, 4, "đủ A B C và đáp án ghim, không lặp")
		}
		require.Len(t, seen, 10)
	}
}

// TestShufflePinnedLast — AC5: đáp án ghim luôn ở cuối, dù xáo.
func TestShufflePinnedLast(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e := r.openExam("ghim", true, r.mcqSet(8)...)
	for range 10 {
		v, _ := r.start(e, r.student("ACTIVE"), uuid.New())
		for _, it := range v.Items {
			require.Equal(t, "Tất cả đáp án trên", it.Options[3].Body)
		}
	}
}

// TestShuffleOffKeepsOrder — AC5: tắt xáo → giữ thứ tự gốc của giảng viên cho mọi sinh viên.
func TestShuffleOffKeepsOrder(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	qs := r.mcqSet(6)
	e := r.openExam("không xáo", false, qs...)
	for range 4 {
		v, _ := r.start(e, r.student("ACTIVE"), uuid.New())
		for i, it := range v.Items {
			require.Equal(t, e.Items[i].ID, it.ItemID)
			require.Equal(t, []string{"A", "B", "C", "Tất cả đáp án trên"}, []string{it.Options[0].Body, it.Options[1].Body, it.Options[2].Body, it.Options[3].Body})
		}
	}
}

// TestGradeIndependentOfShuffle — AC5: chấm theo id đáp án nên hai sinh viên cùng chọn đáp án đúng đều đủ điểm dù thứ tự khác nhau.
func TestGradeIndependentOfShuffle(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e := r.openExam("chấm", true, r.mcqSet(10)...)
	var scores []string
	var orders []string
	for range 3 {
		sv, tab := r.student("ACTIVE"), uuid.New()
		v, _ := r.start(e, sv, tab)
		var ans []exam.AnswerIn
		for _, it := range v.Items {
			ans = append(ans, answerCorrect(it))
		}
		_, err := r.save(v, sv, tab, ans...)
		require.NoError(t, err)
		_, err = r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
		require.NoError(t, err)
		_, _, sc, _ := r.attemptRow(v.Attempt.ID)
		scores = append(scores, *sc)
		orders = append(orders, orderOf(v))
	}
	require.Equal(t, []string{"10.00", "10.00", "10.00"}, scores)
	require.NotEqual(t, orders[0], orders[1])
}

// ---- AC6: lưu câu trả lời ------------------------------------------------------------------------------------------------------

// TestSaveAnswersDiff — AC6: chỉ ghi các câu gửi lên; mảng rỗng xoá lựa chọn; một lô cả nhiều câu; không trả đúng / sai.
func TestSaveAnswersDiff(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("diff", true, r.mcqSet(5)...)
	v, _ := r.start(e, sv, tab)
	count := func() int { return r.count(`select count(*) from exam_answers where attempt_id=$1`, v.Attempt.ID) }
	require.Zero(t, count())
	res, err := r.save(v, sv, tab, pick(v.Items[0], 0), pick(v.Items[1], 1), pick(v.Items[2], 2))
	require.NoError(t, err)
	require.Equal(t, 3, count(), "một lô ba câu")
	require.True(t, res.DeadlineAt.Equal(v.Attempt.DeadlineAt))
	// đổi MỘT câu: chỉ câu đó đổi `saved_at`
	var before time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select saved_at from exam_answers where attempt_id=$1 and item_id=$2`, v.Attempt.ID, v.Items[1].ItemID).Scan(&before))
	time.Sleep(20 * time.Millisecond)
	_, err = r.save(v, sv, tab, pick(v.Items[0], 3))
	require.NoError(t, err)
	var after time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select saved_at from exam_answers where attempt_id=$1 and item_id=$2`, v.Attempt.ID, v.Items[1].ItemID).Scan(&after))
	require.True(t, before.Equal(after), "câu không gửi lên không bị đụng")
	require.Equal(t, 3, count())
	// mảng rỗng = xoá lựa chọn
	empty, _ := json.Marshal(map[string]any{"option_ids": []string{}})
	_, err = r.save(v, sv, tab, exam.AnswerIn{ItemID: v.Items[0].ItemID, Answer: empty})
	require.NoError(t, err)
	require.Equal(t, 2, count())
	// câu đúng / sai
	raw, _ := json.Marshal(res)
	require.NotContains(t, string(raw), "correct")
}

// TestSaveAnswersValidation — AC6: id đáp án lạ → 422 INVALID_OPTION_ID; MCQ_SINGLE > 1 id; câu lạ; trùng câu; sai dạng; câu trắc nghiệm dạng đúng / sai; mọi lỗi cùng lúc; lỗi thì không ghi gì.
func TestSaveAnswersValidation(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	tfq, err := r.svc.Create(t.Context(), r.teacher, r.course, exam.QuestionIn{Type: "TRUE_FALSE", Title: "tf", Topic: "t", Stem: "1<2", Value: new(true)})
	require.NoError(t, err)
	_, err = r.svc.Review(t.Context(), r.teacher, r.course, tfq.ID, "APPROVE", tfq.Version)
	require.NoError(t, err)
	e := r.openExam("hợp lệ", false, append(r.mcqSet(2), tfq.ID)...)
	v, _ := r.start(e, sv, tab)
	raw := func(m map[string]any) json.RawMessage { b, _ := json.Marshal(m); return b }
	two, _ := json.Marshal(map[string]any{"option_ids": []string{v.Items[0].Options[0].ID.String(), v.Items[0].Options[1].ID.String()}})
	for name, c := range map[string]struct {
		in   exam.AnswerIn
		want string
	}{
		"id đáp án lạ":            {exam.AnswerIn{ItemID: v.Items[0].ItemID, Answer: raw(map[string]any{"option_ids": []string{uuid.NewString()}})}, "INVALID_OPTION_ID"},
		"đáp án của câu khác":     {exam.AnswerIn{ItemID: v.Items[0].ItemID, Answer: raw(map[string]any{"option_ids": []string{v.Items[1].Options[0].ID.String()}})}, "INVALID_OPTION_ID"},
		"câu một đáp án chọn hai": {exam.AnswerIn{ItemID: v.Items[0].ItemID, Answer: two}, "INVALID_OPTION_ID"},
		"câu không thuộc bài":     {exam.AnswerIn{ItemID: uuid.New(), Answer: raw(map[string]any{"option_ids": []string{}})}, "INVALID_ITEM"},
		"sai dạng":                {exam.AnswerIn{ItemID: v.Items[0].ItemID, Answer: raw(map[string]any{"value": true})}, "INVALID_ANSWER"},
		"trường lạ":               {exam.AnswerIn{ItemID: v.Items[0].ItemID, Answer: raw(map[string]any{"option_ids": []string{}, "x": 1})}, "INVALID_ANSWER"},
		"đúng sai thiếu value":    {exam.AnswerIn{ItemID: v.Items[2].ItemID, Answer: raw(map[string]any{"option_ids": []string{}})}, "INVALID_ANSWER"},
	} {
		_, err := r.save(v, sv, tab, c.in)
		require.Equal(t, []string{c.want}, fieldCodes(t, err), name)
	}
	_, err = r.save(v, sv, tab, pick(v.Items[0], 0), pick(v.Items[0], 1))
	require.Equal(t, []string{"DUPLICATE_ITEM"}, fieldCodes(t, err))
	_, err = r.save(v, sv, tab, exam.AnswerIn{ItemID: v.Items[2].ItemID, Answer: raw(map[string]any{"value": false})}, exam.AnswerIn{ItemID: uuid.New(), Answer: raw(map[string]any{"value": true})})
	require.Equal(t, []string{"INVALID_ITEM"}, fieldCodes(t, err))
	require.Zero(t, r.count(`select count(*) from exam_answers where attempt_id=$1`, v.Attempt.ID), "lỗi thì không ghi gì (kể cả câu hợp lệ trong cùng lô)")
	_, err = r.save(v, sv, tab)
	require.Contains(t, fieldCodes(t, err), "VALUE_REQUIRED")
	// đúng / sai hợp lệ
	_, err = r.save(v, sv, tab, exam.AnswerIn{ItemID: v.Items[2].ItemID, Answer: raw(map[string]any{"value": false})})
	require.NoError(t, err)
}

// TestSaveRateLimit — AC6: quá EXAM_SAVE_RATE_PER_MIN lần lưu / phút / lượt → 429 RATE_LIMITED kèm retry_after (Redis thật, cấu hình nhỏ).
func TestSaveRateLimit(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	svc := *r.svc
	svc.Redis, svc.Attempt = rdb, exam.AttemptConfig{SaveRate: 3}
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("giới hạn", true, r.mcqSet(2)...)
	v, _ := r.start(e, sv, tab)
	for range 3 {
		_, err := svc.SaveAnswers(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab, exam.AnswersIn{Items: []exam.AnswerIn{pick(v.Items[0], 0)}})
		require.NoError(t, err)
	}
	_, err = svc.SaveAnswers(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab, exam.AnswersIn{Items: []exam.AnswerIn{pick(v.Items[0], 1)}})
	st, code := apiStatus(t, err)
	require.Equal(t, 429, st)
	require.Equal(t, "RATE_LIMITED", code)
}

// TestSaveAfterDeadlineGrace — AC6 / AC7: lưu tới deadline + grace vẫn nhận (bản lưu cuối ở giây cuối); quá ngưỡng → 409 ATTEMPT_CLOSED.
func TestSaveAfterDeadlineGrace(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("grace", true, r.mcqSet(2)...)
	clk := clock.NewFake(time.Now().UTC())
	svc := r.at(clk)
	v, _, err := svc.StartAttempt(t.Context(), sv, r.course, e.ID, tab)
	require.NoError(t, err)
	clk.Advance(45*time.Minute + 9*time.Second) // quá hạn 9 s < grace 10 s
	_, err = svc.SaveAnswers(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab, exam.AnswersIn{Items: []exam.AnswerIn{pick(v.Items[0], 1)}})
	require.NoError(t, err, "bản lưu cuối trong grace vẫn được nhận")
	clk.Advance(2 * time.Second) // 11 s
	_, err = svc.SaveAnswers(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab, exam.AnswersIn{Items: []exam.AnswerIn{pick(v.Items[0], 2)}})
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "ATTEMPT_CLOSED", code)
}

// TestGraceAcceptsLastSave — AC7: bản lưu tới trong grace được TÍNH khi tự nộp.
func TestGraceAcceptsLastSave(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("lưu cuối", false, r.mcqSet(2)...)
	clk := clock.NewFake(time.Now().UTC())
	svc := r.at(clk)
	v, _, err := svc.StartAttempt(t.Context(), sv, r.course, e.ID, tab)
	require.NoError(t, err)
	clk.Advance(45*time.Minute + 5*time.Second)
	_, err = svc.SaveAnswers(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab, exam.AnswersIn{Items: []exam.AnswerIn{answerCorrect(v.Items[0]), answerCorrect(v.Items[1])}})
	require.NoError(t, err)
	clk.Advance(10 * time.Second)
	n, err := svc.AutoSubmitDue(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	_, _, sc, _ := r.attemptRow(v.Attempt.ID)
	require.Equal(t, "10.00", *sc, "cả hai câu của bản lưu cuối trong grace được tính")
}

// TestSaveOtherUsersAttempt404 — AC6: lưu vào lượt của người khác → 404 (không lộ tồn tại); không đổi gì.
func TestSaveOtherUsersAttempt404(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	a, b := r.student("ACTIVE"), r.student("ACTIVE")
	e := r.openExam("của người khác", true, r.mcqSet(2)...)
	va, _ := r.start(e, a, uuid.New())
	tab := uuid.New()
	_, err := r.svc.SaveAnswers(t.Context(), b, r.course, e.ID, va.Attempt.ID, tab, exam.AnswersIn{Items: []exam.AnswerIn{pick(va.Items[0], 0)}})
	st, _ := apiStatus(t, err)
	require.Equal(t, 404, st)
	_, err = r.svc.SubmitAttempt(t.Context(), b, r.course, e.ID, va.Attempt.ID, tab)
	st, _ = apiStatus(t, err)
	require.Equal(t, 404, st)
	_, err = r.svc.Takeover(t.Context(), b, r.course, e.ID, va.Attempt.ID, tab, false)
	st, _ = apiStatus(t, err)
	require.Equal(t, 404, st)
	require.Zero(t, r.count(`select count(*) from exam_answers where attempt_id=$1`, va.Attempt.ID))
}

// ---- AC7: tự nộp ----------------------------------------------------------------------------------------------------------------

func (r *rig) tick(clk clock.Clock) {
	r.t.Helper()
	require.NoError(r.t, r.ticker(clk).Tick(r.t.Context()))
}

// TestAutoSubmitOnTimeout — AC7: quá deadline + grace (hạn do thời lượng) → tick nộp với TIMEOUT, submitted_at = deadline_at, trắc nghiệm chấm ngay.
func TestAutoSubmitOnTimeout(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("hết giờ", true, r.mcqSet(4)...)
	clk := clock.NewFake(time.Now().UTC())
	v, _, err := r.at(clk).StartAttempt(t.Context(), sv, r.course, e.ID, tab)
	require.NoError(t, err)
	clk.Advance(45*time.Minute + 5*time.Second) // trong grace: chưa nộp
	r.tick(clk)
	st, _, _, _ := r.attemptRow(v.Attempt.ID)
	require.Equal(t, "IN_PROGRESS", st)
	clk.Advance(10 * time.Second)
	r.tick(clk)
	st, reason, score, at := r.attemptRow(v.Attempt.ID)
	require.Equal(t, "GRADED", st)
	require.Equal(t, "TIMEOUT", *reason)
	require.True(t, at.Equal(v.Attempt.DeadlineAt))
	require.Equal(t, "0.00", *score, "chưa trả lời câu nào")
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.attempt_submitted' and payload->>'attempt_id'=$1`, v.Attempt.ID.String()))
	r.tick(clk) // tick lần hai: idempotent
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.attempt_submitted' and payload->>'attempt_id'=$1`, v.Attempt.ID.String()))
	// sinh viên mở lại: tóm tắt, không đề
	got, err := r.at(clk).MyAttempt(t.Context(), sv, r.course, e.ID, tab)
	require.NoError(t, err)
	sum := got.(exam.AttemptSummaryView)
	require.Equal(t, "TIMEOUT", sum.Attempt.SubmitReason)
}

// TestAutoSubmitOnClose — AC7: hạn do GIỜ ĐÓNG của lớp → CLOSED.
func TestAutoSubmitOnClose(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("đóng bài", true, r.mcqSet(2)...)
	clk := clock.NewFake(time.Now().UTC())
	r.exec(`update exams set opens_at = $2, closes_at = $3 where id=$1`, e.ID, clk.Now().Add(-50*time.Minute), clk.Now().Add(10*time.Minute)) // khung 60 phút ≥ thời lượng 45
	v, _, err := r.at(clk).StartAttempt(t.Context(), sv, r.course, e.ID, tab)
	require.NoError(t, err)
	clk.Advance(10*time.Minute + 11*time.Second)
	r.tick(clk)
	st, reason, _, at := r.attemptRow(v.Attempt.ID)
	require.Equal(t, "GRADED", st)
	require.Equal(t, "CLOSED", *reason)
	require.True(t, at.Equal(v.Attempt.DeadlineAt))
}

// TestAutoSubmitGradesMCQ — AC7: tự nộp chấm các câu đã lưu: 3 / 4 câu đúng → 7,50 / 10 (làm tròn 0,01 mặc định); câu để trống 0.
func TestAutoSubmitGradesMCQ(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("chấm khi hết giờ", true, r.mcqSet(4)...)
	clk := clock.NewFake(time.Now().UTC())
	svc := r.at(clk)
	v, _, err := svc.StartAttempt(t.Context(), sv, r.course, e.ID, tab)
	require.NoError(t, err)
	_, err = svc.SaveAnswers(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab, exam.AnswersIn{Items: []exam.AnswerIn{answerCorrect(v.Items[0]), answerCorrect(v.Items[1]), answerCorrect(v.Items[2])}})
	require.NoError(t, err)
	clk.Advance(46 * time.Minute)
	r.tick(clk)
	_, _, sc, _ := r.attemptRow(v.Attempt.ID)
	require.Equal(t, "7.50", *sc)
	var bd []map[string]any
	var raw []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select breakdown from exam_attempts where id=$1`, v.Attempt.ID).Scan(&raw))
	require.NoError(t, json.Unmarshal(raw, &bd))
	require.Len(t, bd, 4)
	// bài có câu code: nộp xong ở GRADING (hoàn tất ở US-PE-06), không có điểm
	cq := r.approvedCode("code")
	mixed := r.openExam("hỗn hợp", false, r.approvedMCQ("m"), cq)
	sv2 := r.student("ACTIVE")
	v2, _ := r.start(mixed, sv2, tab)
	_, err = r.svc.SubmitAttempt(t.Context(), sv2, r.course, mixed.ID, v2.Attempt.ID, tab)
	require.NoError(t, err)
	st, _, sc2, _ := r.attemptRow(v2.Attempt.ID)
	require.Equal(t, "GRADING", st)
	require.Nil(t, sc2)
}

// ---- AC8: nộp tay ----------------------------------------------------------------------------------------------------------------

// TestSubmitManual — AC8: MANUAL, trắc nghiệm chấm ngay, trả CHỈ {status, submitted_at, answered, total}; sau nộp lưu / nộp lại → 409.
func TestSubmitManual(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("nộp tay", true, r.mcqSet(5)...)
	v, _ := r.start(e, sv, tab)
	_, err := r.save(v, sv, tab, answerCorrect(v.Items[0]), answerCorrect(v.Items[1]))
	require.NoError(t, err)
	sub, err := r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
	require.NoError(t, err)
	require.Equal(t, "GRADED", sub.Status)
	require.Equal(t, 2, sub.Answered)
	require.Equal(t, 5, sub.Total)
	st, reason, sc, at := r.attemptRow(v.Attempt.ID)
	require.Equal(t, "GRADED", st)
	require.Equal(t, "MANUAL", *reason)
	require.Equal(t, "4.00", *sc, "2/5 câu đúng, thang 10")
	require.True(t, at.Equal(sub.SubmittedAt))
	_, err = r.save(v, sv, tab, pick(v.Items[2], 0))
	s, code := apiStatus(t, err)
	require.Equal(t, 409, s)
	require.Equal(t, "ATTEMPT_ALREADY_SUBMITTED", code)
}

// TestSubmitIdempotentReplay — AC8: nộp lại một lượt đã nộp KHÔNG chấm lại, không thêm sự kiện (cùng khoá: middleware trả đúng phản hồi cũ; xem kịch bản hợp đồng).
func TestSubmitIdempotentReplay(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("lặp nộp", true, r.mcqSet(3)...)
	v, _ := r.start(e, sv, tab)
	_, err := r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
	require.NoError(t, err)
	_, _, sc1, _ := r.attemptRow(v.Attempt.ID)
	_, err = r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
	require.Error(t, err)
	_, _, sc2, _ := r.attemptRow(v.Attempt.ID)
	require.Equal(t, *sc1, *sc2)
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.attempt_submitted'`))
}

// TestSubmitTwiceDifferentKey409 — AC8: nộp lần hai (khoá khác) → 409 ATTEMPT_ALREADY_SUBMITTED kèm submitted_at; nộp đồng thời → đúng một thành công.
func TestSubmitTwiceDifferentKey409(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("hai lần", true, r.mcqSet(3)...)
	v, _ := r.start(e, sv, tab)
	var wg sync.WaitGroup
	errs := make([]error, 5)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
		}()
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
			continue
		}
		st, code := apiStatus(t, err)
		require.Equal(t, 409, st)
		require.Equal(t, "ATTEMPT_ALREADY_SUBMITTED", code)
	}
	require.Equal(t, 1, ok)
	require.Equal(t, 1, r.count(`select count(*) from exam_attempts where id=$1 and status<>'IN_PROGRESS'`, v.Attempt.ID))
}

// TestSubmitNoScoreInResponse — AC8: phản hồi nộp và mọi đường đọc của sinh viên sau nộp KHÔNG có điểm / đúng sai.
func TestSubmitNoScoreInResponse(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("không điểm", true, r.mcqSet(3)...)
	v, _ := r.start(e, sv, tab)
	_, err := r.save(v, sv, tab, answerCorrect(v.Items[0]))
	require.NoError(t, err)
	sub, err := r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
	require.NoError(t, err)
	raw, _ := json.Marshal(sub)
	var keys map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &keys))
	require.ElementsMatch(t, []string{"status", "submitted_at", "answered", "total"}, keysOf(keys))
	got, err := r.svc.MyAttempt(t.Context(), sv, r.course, e.ID, tab)
	require.NoError(t, err)
	raw, _ = json.Marshal(got)
	for _, bad := range []string{"score", "earned", "correct", "items", "breakdown"} {
		require.NotContains(t, string(raw), bad)
	}
}

// ---- AC9: một nơi được ghi -----------------------------------------------------------------------------------------------------

// TestSingleWriterTab — AC9: tab đã ghi gần đây là người ghi; tab khác → 409 ATTEMPT_OTHER_TAB kèm writer_seen_at; tab cũ bỏ quá EXAM_TAB_STALE (20 s) thì tab khác ghi được.
func TestSingleWriterTab(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, a, b := r.student("ACTIVE"), uuid.New(), uuid.New()
	e := r.openExam("hai tab", true, r.mcqSet(3)...)
	clk := clock.NewFake(time.Now().UTC())
	svc := r.at(clk)
	v, _, err := svc.StartAttempt(t.Context(), sv, r.course, e.ID, a)
	require.NoError(t, err)
	_, err = svc.SaveAnswers(t.Context(), sv, r.course, e.ID, v.Attempt.ID, a, exam.AnswersIn{Items: []exam.AnswerIn{pick(v.Items[0], 0)}})
	require.NoError(t, err)
	_, err = svc.SaveAnswers(t.Context(), sv, r.course, e.ID, v.Attempt.ID, b, exam.AnswersIn{Items: []exam.AnswerIn{pick(v.Items[0], 1)}})
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "ATTEMPT_OTHER_TAB", code)
	require.Contains(t, fmt.Sprint(detailsOf(t, err)), "writer_seen_at")
	clk.Advance(15 * time.Second) // < 20 s: tab A vẫn là người ghi
	_, err = svc.SaveAnswers(t.Context(), sv, r.course, e.ID, v.Attempt.ID, b, exam.AnswersIn{Items: []exam.AnswerIn{pick(v.Items[0], 1)}})
	require.Error(t, err)
	clk.Advance(6 * time.Second) // 21 s im lặng: tab B giành được
	_, err = svc.SaveAnswers(t.Context(), sv, r.course, e.ID, v.Attempt.ID, b, exam.AnswersIn{Items: []exam.AnswerIn{pick(v.Items[0], 1)}})
	require.NoError(t, err)
	// `mine` báo người ghi theo tab gọi, không đổi người ghi
	got, err := svc.MyAttempt(t.Context(), sv, r.course, e.ID, a)
	require.NoError(t, err)
	require.False(t, got.(exam.AttemptStartView).Attempt.Writer.IsYou)
	got, err = svc.MyAttempt(t.Context(), sv, r.course, e.ID, b)
	require.NoError(t, err)
	require.True(t, got.(exam.AttemptStartView).Attempt.Writer.IsYou)
}

// TestTakeoverFlipsWriter — AC9: takeover chuyển quyền ghi ngay (không đợi stale); tab cũ nhận 409 ở lần ghi kế; ghi sự kiện TAB_TAKEOVER.
func TestTakeoverFlipsWriter(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, a, b := r.student("ACTIVE"), uuid.New(), uuid.New()
	e := r.openExam("takeover", true, r.mcqSet(2)...)
	v, _ := r.start(e, sv, a)
	_, err := r.svc.Takeover(t.Context(), sv, r.course, e.ID, v.Attempt.ID, b, false)
	require.NoError(t, err)
	_, err = r.save(v, sv, a, pick(v.Items[0], 0))
	_, code := apiStatus(t, err)
	require.Equal(t, "ATTEMPT_OTHER_TAB", code, "tab cũ bị chặn")
	_, err = r.save(v, sv, b, pick(v.Items[0], 0))
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from exam_events where attempt_id=$1 and type='TAB_TAKEOVER'`, v.Attempt.ID))
}

// TestStaleTabCannotOverwrite — AC9: sau takeover, tab cũ KHÔNG ghi đè câu trả lời / bản nháp của tab mới dù gửi lại nhiều lần.
func TestStaleTabCannotOverwrite(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, a, b := r.student("ACTIVE"), uuid.New(), uuid.New()
	e := r.openExam("không ghi chồng", true, r.mcqSet(2)...)
	v, _ := r.start(e, sv, a)
	_, err := r.svc.Takeover(t.Context(), sv, r.course, e.ID, v.Attempt.ID, b, false)
	require.NoError(t, err)
	_, err = r.save(v, sv, b, pick(v.Items[0], 3))
	require.NoError(t, err)
	for range 5 {
		_, err = r.save(v, sv, a, pick(v.Items[0], 0))
		require.Error(t, err)
	}
	var raw []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select answer from exam_answers where attempt_id=$1 and item_id=$2`, v.Attempt.ID, v.Items[0].ItemID).Scan(&raw))
	require.Contains(t, string(raw), v.Items[0].Options[3].ID.String())
	// nộp từ tab cũ cũng bị chặn
	_, err = r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, a)
	_, code := apiStatus(t, err)
	require.Equal(t, "ATTEMPT_OTHER_TAB", code)
}

// TestWriterSeenThrottle — AC9: `writer_seen_at` cập nhật tối đa một lần / 10 s mỗi lượt (không tốn một lệnh ghi cho mỗi lần lưu).
func TestWriterSeenThrottle(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, a := r.student("ACTIVE"), uuid.New()
	e := r.openExam("tiết chế", true, r.mcqSet(2)...)
	clk := clock.NewFake(time.Now().UTC())
	svc := r.at(clk)
	v, _, err := svc.StartAttempt(t.Context(), sv, r.course, e.ID, a)
	require.NoError(t, err)
	seen := func() time.Time {
		var t0 time.Time
		require.NoError(t, r.pool.QueryRow(t.Context(), `select writer_seen_at from exam_attempts where id=$1`, v.Attempt.ID).Scan(&t0))
		return t0
	}
	save := func() {
		_, err := svc.SaveAnswers(t.Context(), sv, r.course, e.ID, v.Attempt.ID, a, exam.AnswersIn{Items: []exam.AnswerIn{pick(v.Items[0], 0)}})
		require.NoError(t, err)
	}
	t0 := seen()
	clk.Advance(5 * time.Second)
	save()
	require.True(t, seen().Equal(t0), "dưới 10 s: không ghi lại")
	clk.Advance(6 * time.Second) // 11 s kể từ lần ghi
	save()
	require.True(t, seen().After(t0), "từ 10 s: cập nhật")
}

// TestTakeoverReloadMeta — AC9: takeover sau khi tải lại mang `meta.reload = true` (giảng viên thấy như một lần tải lại).
func TestTakeoverReloadMeta(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, a, b := r.student("ACTIVE"), uuid.New(), uuid.New()
	e := r.openExam("tải lại", true, r.mcqSet(2)...)
	v, _ := r.start(e, sv, a)
	_, err := r.svc.Takeover(t.Context(), sv, r.course, e.ID, v.Attempt.ID, b, true)
	require.NoError(t, err)
	var meta string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select meta::text from exam_events where attempt_id=$1 and type='TAB_TAKEOVER'`, v.Attempt.ID).Scan(&meta))
	require.JSONEq(t, `{"reload": true}`, meta)
}

// ---- AC13: sau nộp -------------------------------------------------------------------------------------------------------------

// TestAfterSubmitSummaryOnly — AC13: sau nộp và bài chưa công bố, `mine` chỉ có tóm tắt (không `items`, không điểm).
func TestAfterSubmitSummaryOnly(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("tóm tắt", true, r.mcqSet(3)...)
	v, _ := r.start(e, sv, tab)
	_, err := r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
	require.NoError(t, err)
	got, err := r.svc.MyAttempt(t.Context(), sv, r.course, e.ID, tab)
	require.NoError(t, err)
	sum, ok := got.(exam.AttemptSummaryView)
	require.True(t, ok)
	raw, _ := json.Marshal(sum)
	var top map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &top))
	require.ElementsMatch(t, []string{"attempt", "exam"}, keysOf(top))
	var inner map[string]map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &inner))
	require.ElementsMatch(t, []string{"id", "status", "submitted_at", "submit_reason"}, keysOf(inner["attempt"]))
	require.ElementsMatch(t, []string{"id", "title", "closes_at", "status"}, keysOf(inner["exam"]))
	require.Equal(t, "MANUAL", sum.Attempt.SubmitReason)
}

// TestResultBeforePublish409 — AC13: kết quả trước công bố → 409 RESULT_NOT_PUBLISHED, thân KHÔNG chứa dữ liệu; sau công bố trả điểm.
func TestResultBeforePublish409(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("kết quả", true, r.mcqSet(2)...)
	v, _ := r.start(e, sv, tab)
	_, err := r.save(v, sv, tab, answerCorrect(v.Items[0]), answerCorrect(v.Items[1]))
	require.NoError(t, err)
	_, err = r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
	require.NoError(t, err)
	_, err = r.svc.AttemptResult(t.Context(), sv, r.course, e.ID, v.Attempt.ID)
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "RESULT_NOT_PUBLISHED", code)
	require.Nil(t, detailsOf(t, err), "thân không chứa dữ liệu")
	r.exec(`update exams set status='PUBLISHED', published_at=now() where id=$1`, e.ID)
	res, err := r.svc.AttemptResult(t.Context(), sv, r.course, e.ID, v.Attempt.ID)
	require.NoError(t, err)
	require.Equal(t, "10.00", *res.Score)
	require.True(t, decimal.RequireFromString(res.Exam.MaxScore).Equal(decimal.NewFromInt(10)))
}

// TestOtherStudentAttempt404 — AC13: lượt của người khác → 404 ở mọi đường (kết quả, lưu, nộp).
func TestOtherStudentAttempt404(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	a, b := r.student("ACTIVE"), r.student("ACTIVE")
	e := r.openExam("người khác", true, r.mcqSet(2)...)
	va, _ := r.start(e, a, uuid.New())
	r.exec(`update exams set status='PUBLISHED', published_at=now() where id=$1`, e.ID)
	_, err := r.svc.AttemptResult(t.Context(), b, r.course, e.ID, va.Attempt.ID)
	st, _ := apiStatus(t, err)
	require.Equal(t, 404, st)
}
