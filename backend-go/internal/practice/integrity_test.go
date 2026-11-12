package practice_test

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/practice"
)

// TestPracticeStudentDTOAllowlist — US-P9-01 AC5 / FR-22: thẻ JSON của mọi DTO sinh viên khớp `testdata/student_dto_allowlist.json`.
// Thêm trường vào DTO mà quên xét "lộ đáp án?" → test này đỏ (đổi allowlist = quyết định có chủ ý).
func TestPracticeStudentDTOAllowlist(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("testdata/student_dto_allowlist.json")
	require.NoError(t, err)
	var want map[string][]string
	require.NoError(t, json.Unmarshal(b, &want))
	types := map[string]any{
		"OptionView": practice.OptionView{}, "QuestionView": practice.QuestionView{}, "AnswerView": practice.AnswerView{}, "FeedbackView": practice.FeedbackView{},
		"ItemView": practice.ItemView{}, "TopicResult": practice.TopicResult{}, "ResultView": practice.ResultView{}, "AttemptView": practice.AttemptView{},
		"SaveView": practice.SaveView{}, "TopicInfo": practice.TopicInfo{}, "Recommended": practice.Recommended{}, "InProgressView": practice.InProgressView{},
		"OverviewView": practice.OverviewView{}, "HistoryRow": practice.HistoryRow{}, "WeakTopic": practice.WeakTopic{}, "DocSuggested": practice.DocSuggested{},
		"WeaknessView": practice.WeaknessView{}, "MetaView": practice.MetaView{},
	}
	require.Len(t, want, len(types), "allowlist và danh sách DTO phải khớp tên")
	for name, v := range types {
		var tags []string
		rt := reflect.TypeOf(v)
		for i := range rt.NumField() {
			tag, _, _ := strings.Cut(rt.Field(i).Tag.Get("json"), ",")
			require.NotEmpty(t, tag, "%s.%s thiếu thẻ json", name, rt.Field(i).Name)
			require.NotEqual(t, "-", tag)
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		require.Equal(t, want[name], tags, name)
		for _, forbidden := range []string{"answer_key", "correct_option", "reference", "model_answer"} {
			require.NotContains(t, tags, forbidden, name)
		}
	}
}

// walk kiểm cây JSON của một AttemptView: khoá lộ đáp án (`feedback`) chỉ được khác null khi `allowed(item)`; câu hỏi chỉ có đúng 4 khoá.
func assertNoLeak(t *testing.T, v practice.AttemptView, allowed func(i int) bool, msg string) {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	require.NotContains(t, m, "answer_key", msg)
	items, _ := m["items"].([]any)
	require.Len(t, items, len(v.Items))
	for i, raw := range items {
		it := raw.(map[string]any)
		q := it["question"].(map[string]any)
		keys := make([]string, 0, len(q))
		for k := range q {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		require.Equal(t, []string{"options", "stem", "topic", "type"}, keys, "%s: câu %d", msg, i)
		if !allowed(i) {
			require.Nil(t, it["feedback"], "%s: câu %d lộ phản hồi", msg, i)
		}
	}
	if v.Status == "IN_PROGRESS" {
		require.Nil(t, m["result"], msg)
	}
}

// TestPracticeNoAnswerLeak — US-P9-10 AC2: đáp án / giải thích / trích dẫn chỉ lộ đúng lúc ở mọi trạng thái của lượt (kể cả khi tải lại).
func TestPracticeNoAnswerLeak(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student()
	r.approved("Mạng", 12)
	ctx := t.Context()
	never := func(int) bool { return false }
	always := func(int) bool { return true }

	// TOPIC: chưa trả lời → không; trả lời câu 0 → chỉ câu 0; nộp → tất cả
	tv, _, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Mạng", Count: 5}, uuid.Nil)
	require.NoError(t, err)
	assertNoLeak(t, tv, never, "TOPIC chưa trả lời")
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, tv.ID, tv.Items[0].ID, uuid.Nil, opt(r.correctOption(tv.Items[0].ID)))
	require.NoError(t, err)
	got, err := r.svc.Get(ctx, sv, r.course, tv.ID, uuid.Nil)
	require.NoError(t, err)
	assertNoLeak(t, got, func(i int) bool { return i == 0 }, "TOPIC đã trả lời câu 0")
	done, err := r.svc.Submit(ctx, sv, r.course, tv.ID)
	require.NoError(t, err)
	assertNoLeak(t, done, always, "TOPIC đã nộp")

	// MOCK: đang làm (kể cả sau khi lưu đáp án và tải lại) → không câu nào; đã nộp → tất cả
	tab := uuid.New()
	mv, _, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "MOCK", Count: 5}, tab)
	require.NoError(t, err)
	assertNoLeak(t, mv, never, "MOCK vừa bắt đầu")
	save, err := r.svc.SaveAnswer(ctx, sv, r.course, mv.ID, mv.Items[0].ID, tab, opt(r.correctOption(mv.Items[0].ID)))
	require.NoError(t, err)
	require.Nil(t, save.Feedback, "MOCK: phản hồi lưu không chứa đáp án")
	got, err = r.svc.Get(ctx, sv, r.course, mv.ID, tab)
	require.NoError(t, err)
	assertNoLeak(t, got, never, "MOCK đang làm, tải lại")
	// lỗi cũng không lộ: ghi sai phiên bản
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, mv.ID, mv.Items[0].ID, tab, practice.AnswerIn{Answer: opt(r.wrongOption(mv.Items[0].ID)).Answer, Version: 9})
	require.Error(t, err)
	require.NotContains(t, err.Error(), r.correctOption(mv.Items[0].ID))
	fin, err := r.svc.Submit(ctx, sv, r.course, mv.ID)
	require.NoError(t, err)
	assertNoLeak(t, fin, always, "MOCK đã nộp")
}

// TestPracticeNoExamLeak — US-P9-10 AC1 / US-P9-05 AC3: câu thuộc bài thi chưa công bố KHÔNG vào đề; chỉ xuất hiện khi PUBLISHED.
func TestPracticeNoExamLeak(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	ctx := t.Context()
	fillers := r.approved("Thi", 5)
	require.Len(t, fillers, 5)
	x := r.approved("Thi", 1)[0]
	const stemX = "Câu X đặc biệt của bài thi?"
	r.exec(`update question_bank set stem = $2 where id = $1`, x, stemX)

	appears := func() bool {
		sv := r.student()
		v, _, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Thi", Count: 6}, uuid.Nil)
		if err != nil { // còn 5 câu: vẫn dựng được 5 (không phải lỗi)
			require.NoError(t, err)
		}
		return slices.Contains(qs(v), stemX)
	}
	require.True(t, appears(), "chưa nằm trong bài thi nào: có thể luyện")
	for _, st := range []string{"DRAFT", "SCHEDULED", "OPEN", "CLOSED"} {
		eid := r.examWith(x, st)
		require.False(t, appears(), "bài thi %s: câu bị loại", st)
		r.exec(`delete from exams where id = $1`, eid) // gỡ bài để thử trạng thái kế tiếp
	}
	eid := r.examWith(x, "PUBLISHED")
	require.True(t, appears(), "bài thi đã công bố: được phép (Q-P9-3)")
	// câu còn nằm ở một bài thi chưa công bố khác thì vẫn bị loại dù có bài PUBLISHED
	r.examWith(x, "DRAFT")
	require.False(t, appears())
	_ = eid
}

func (r *rig) examWith(qid uuid.UUID, st string) uuid.UUID {
	r.t.Helper()
	var eid uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into exams (course_id, title, status, opens_at, closes_at, duration_minutes, published_at, created_by)
		select $1, 'Bài thi ' || $2, $2::exam_status, case when $2 <> 'DRAFT' then now() - interval '2 hours' end, case when $2 <> 'DRAFT' then now() + interval '2 hours' end,
		       case when $2 <> 'DRAFT' then 30 end, case when $2 = 'PUBLISHED' then now() end, $3 returning id`, r.course, st, r.teacher).Scan(&eid))
	r.exec(`insert into exam_items (course_id, exam_id, question_id, position) values ($1, $2, $3, 1)`, r.course, eid, qid)
	return eid
}

// TestWrongAnswerSignalsNoIdentity — US-P9-09 AC4: tổng hợp theo chủ đề, k ≥ 3 sinh viên, không định danh.
func TestWrongAnswerSignalsNoIdentity(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	ctx := t.Context()
	r.approved("A", 5)
	r.approved("B", 5)
	var students []uuid.UUID
	wrong := func(sv uuid.UUID, topic string) {
		v, _, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "TOPIC", Topic: topic, Count: 5}, uuid.Nil)
		require.NoError(t, err)
		_, err = r.svc.SaveAnswer(ctx, sv, r.course, v.ID, v.Items[0].ID, uuid.Nil, opt(r.wrongOption(v.Items[0].ID)))
		require.NoError(t, err)
	}
	for range 4 {
		sv := r.student()
		students = append(students, sv)
		wrong(sv, "A")
	}
	wrong(students[0], "B") // chỉ 1 sinh viên sai chủ đề B
	wrong(r.student(), "B") // 2 sinh viên sai chủ đề B: vẫn < 3

	sig, err := r.svc.WrongAnswerSignals(ctx, r.course, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, sig, 1, "chỉ chủ đề có ≥ 3 sinh viên sai")
	require.Equal(t, "A", sig[0].Topic)
	require.Equal(t, 4, sig[0].StudentsAffected)
	require.Equal(t, 4, sig[0].WrongAnswers)
	require.NotEmpty(t, sig[0].SampleQuestionIDs)
	b, _ := json.Marshal(sig)
	for _, sv := range students {
		require.NotContains(t, string(b), sv.String())
	}
	require.NotContains(t, string(b), "@example.test")
	require.NotContains(t, string(b), "student_id")
	// khoảng thời gian không chứa dữ liệu → rỗng
	none, err := r.svc.WrongAnswerSignals(ctx, r.course, time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour))
	require.NoError(t, err)
	require.Empty(t, none)
}

// TestPracticeIsolation — US-P9-06 AC8, US-P9-10 AC8: lượt của người khác / lớp khác → 404; mọi hàng đọc chỉ của chính mình.
func TestPracticeIsolation(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	ctx := t.Context()
	r.approved("Mạng", 6)
	a, b := r.student(), r.student()
	course2 := r.newCourse("PX-" + strings.ToUpper(r.sfx))
	c2 := r.student(course2)
	r.approved("Mạng", 6, course2)

	va, _, err := r.svc.Start(ctx, a, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Mạng", Count: 5}, uuid.Nil)
	require.NoError(t, err)
	_, err = r.svc.Get(ctx, b, r.course, va.ID, uuid.Nil)
	require.Equal(t, 404, status(t, err))
	_, err = r.svc.SaveAnswer(ctx, b, r.course, va.ID, va.Items[0].ID, uuid.Nil, opt(r.correctOption(va.Items[0].ID)))
	require.Equal(t, 404, status(t, err))
	_, err = r.svc.Submit(ctx, b, r.course, va.ID)
	require.Equal(t, 404, status(t, err))
	_, err = r.svc.SetIndex(ctx, b, r.course, va.ID, uuid.Nil, 1)
	require.Equal(t, 404, status(t, err))
	_, err = r.svc.Get(ctx, c2, course2, va.ID, uuid.Nil) // lớp khác: id của lớp 1 không tồn tại ở lớp 2
	require.Equal(t, 404, status(t, err))
	// mục của lượt A không ghi được qua lượt của chính B
	vb, _, err := r.svc.Start(ctx, b, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Mạng", Count: 5}, uuid.Nil)
	require.NoError(t, err)
	_, err = r.svc.SaveAnswer(ctx, b, r.course, vb.ID, va.Items[0].ID, uuid.Nil, opt(r.correctOption(va.Items[0].ID)))
	require.Equal(t, 404, status(t, err))

	// lịch sử chỉ của mình, theo lớp
	h, err := r.svc.History(ctx, a, r.course, nil, 21)
	require.NoError(t, err)
	require.Len(t, h, 1)
	h2, err := r.svc.History(ctx, a, course2, nil, 21)
	require.NoError(t, err)
	require.Empty(t, h2)
	hb, err := r.svc.History(ctx, b, r.course, nil, 21)
	require.NoError(t, err)
	require.Len(t, hb, 1)
	require.Equal(t, vb.ID, hb[0].ID)
}
