package practice_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/practice"
	"github.com/edupilot/backend-go/internal/today"
)

func raw(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

// TestStartTopic — US-P9-06 AC2, AC11, US-P9-05 AC4/AC5/AC7: tạo lượt, làm tiếp lượt dở, không đủ câu, tham số sai, xáo ổn định.
func TestStartTopic(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student()
	r.approved("Mạng", 12)
	r.approved("Ít", 4)
	ctx := t.Context()

	v, created, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Mạng"}, uuid.Nil)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, 10, v.ItemCount, "mặc định 10 câu")
	require.Len(t, v.Items, 10)
	require.Equal(t, "IN_PROGRESS", v.Status)
	require.Nil(t, v.Result)
	for _, it := range v.Items {
		require.Nil(t, it.Feedback, "chưa trả lời: không lộ đáp án")
		require.Nil(t, it.Answer)
		require.Len(t, it.Question.Options, 3)
	}
	// làm tiếp: cùng chủ đề → 200 cùng lượt; chủ đề khác → lượt mới
	again, created, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Mạng", Count: 20}, uuid.Nil)
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, v.ID, again.ID)
	// tải lại: thứ tự câu và phương án không đổi
	got, err := r.svc.Get(ctx, sv, r.course, v.ID, uuid.Nil)
	require.NoError(t, err)
	require.Equal(t, raw(t, v.Items), raw(t, got.Items))

	// lỗi
	_, _, err = r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Ít"}, uuid.Nil)
	require.Equal(t, "NOT_ENOUGH_QUESTIONS", code(t, err))
	require.Equal(t, 409, status(t, err))
	_, _, err = r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Không có"}, uuid.Nil)
	require.Equal(t, "VALIDATION_FAILED", code(t, err))
	for _, c := range []int{4, 51} {
		_, _, err = r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Mạng", Count: c}, uuid.Nil)
		require.Equal(t, "VALIDATION_FAILED", code(t, err), "count %d", c)
	}
	_, _, err = r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "OTHER"}, uuid.Nil)
	require.Equal(t, "VALIDATION_FAILED", code(t, err))
	// hai sinh viên cùng chủ đề: đề khác nhau (xáo riêng từng lượt)
	sv2 := r.student()
	other, _, err := r.svc.Start(ctx, sv2, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Mạng"}, uuid.Nil)
	require.NoError(t, err)
	require.NotEqual(t, raw(t, qs(v)), raw(t, qs(other)))
}

func qs(v practice.AttemptView) []string {
	var out []string
	for _, it := range v.Items {
		out = append(out, it.Question.Stem)
	}
	return out
}

// TestTopicAnswerFeedback — US-P9-06 AC4: phản hồi ngay trong cùng response, đáp án khoá, gửi lại cùng đáp án idempotent, id lạ → 422.
func TestTopicAnswerFeedback(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student()
	r.approved("Mạng", 6)
	ctx := t.Context()
	v, _, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Mạng", Count: 5}, uuid.Nil)
	require.NoError(t, err)

	it := v.Items[0]
	ok := r.correctOption(it.ID)
	save, err := r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it.ID, uuid.Nil, opt(ok))
	require.NoError(t, err)
	require.True(t, save.Locked)
	require.NotNil(t, save.Feedback)
	require.True(t, *save.Feedback.Correct)
	require.JSONEq(t, `{"option_ids":["`+ok+`"]}`, string(save.Feedback.Key))
	require.Equal(t, "Vì B đúng.", *save.Feedback.Explanation)

	// gửi lại đúng đáp án đó (mất phản hồi) → cùng kết quả; đổi đáp án → ANSWER_LOCKED
	again, err := r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it.ID, uuid.Nil, opt(ok))
	require.NoError(t, err)
	require.Equal(t, save.Version, again.Version)
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it.ID, uuid.Nil, opt(r.wrongOption(it.ID)))
	require.Equal(t, "ANSWER_LOCKED", code(t, err))

	// sai → correct=false và lộ đáp án đúng; id lạ → 422 INVALID_OPTION_ID; không chọn → 422
	it2 := v.Items[1]
	bad, err := r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it2.ID, uuid.Nil, opt(r.wrongOption(it2.ID)))
	require.NoError(t, err)
	require.False(t, *bad.Feedback.Correct)
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, v.ID, v.Items[2].ID, uuid.Nil, opt(uuid.NewString()))
	require.Equal(t, "INVALID_OPTION_ID", code(t, err))
	require.Equal(t, 422, status(t, err))
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, v.ID, v.Items[2].ID, uuid.Nil, practice.AnswerIn{Answer: json.RawMessage(`{"option_ids":[]}`)})
	require.Equal(t, "VALIDATION_FAILED", code(t, err))

	// tải lại: câu đã trả lời có phản hồi, câu chưa trả lời thì không
	got, err := r.svc.Get(ctx, sv, r.course, v.ID, uuid.Nil)
	require.NoError(t, err)
	require.NotNil(t, got.Items[0].Feedback)
	require.NotNil(t, got.Items[0].Answer)
	require.Nil(t, got.Items[2].Feedback)

	// nộp: các câu còn lại tính sai; gọi lại → cùng kết quả
	res, err := r.svc.Submit(ctx, sv, r.course, v.ID)
	require.NoError(t, err)
	require.Equal(t, "SUBMITTED", res.Status)
	require.Equal(t, 1, res.Result.Correct)
	require.Equal(t, 5, res.Result.Total)
	require.Equal(t, "2.0", res.Result.Score)
	require.Equal(t, "Mạng", *res.Result.Weakest)
	res2, err := r.svc.Submit(ctx, sv, r.course, v.ID)
	require.NoError(t, err)
	require.Equal(t, raw(t, res.Result), raw(t, res2.Result))
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, v.ID, v.Items[3].ID, uuid.Nil, opt(r.correctOption(v.Items[3].ID)))
	require.Equal(t, "ATTEMPT_CLOSED", code(t, err))
}

// TestMockFlow — US-P9-07: MOCK không phản hồi tới khi nộp; version, tab, vị trí; nộp tính đúng / tổng; hết giờ → ATTEMPT_CLOSED; worker đóng + chấm.
func TestMockFlow(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student()
	r.approved("Mạng", 8)
	ctx := t.Context()
	tab1, tab2 := uuid.New(), uuid.New()
	v, created, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "MOCK", Count: 6, DurationMinutes: 20}, tab1)
	require.NoError(t, err)
	require.True(t, created)
	require.NotNil(t, v.DeadlineAt)
	require.Len(t, v.Items, 6)

	// đúng 4 câu, sai 1, bỏ trống 1
	for i, it := range v.Items {
		var in practice.AnswerIn
		switch {
		case i < 4:
			in = opt(r.correctOption(it.ID))
		case i == 4:
			in = opt(r.wrongOption(it.ID))
		default:
			continue
		}
		sv1, err := r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it.ID, tab1, in)
		require.NoError(t, err)
		require.False(t, sv1.Locked)
		require.Nil(t, sv1.Feedback, "MOCK: không phản hồi khi đang làm")
		require.Equal(t, 1, sv1.Version)
	}
	// đổi đáp án câu 0 với version đúng; sai version → 409; cùng nội dung → idempotent
	it := v.Items[0]
	chg, err := r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it.ID, tab1, practice.AnswerIn{Answer: opt(r.wrongOption(it.ID)).Answer, Version: 1})
	require.NoError(t, err)
	require.Equal(t, 2, chg.Version)
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it.ID, tab1, practice.AnswerIn{Answer: opt(r.correctOption(it.ID)).Answer, Version: 1})
	require.Equal(t, "VERSION_CONFLICT", code(t, err))
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it.ID, tab1, practice.AnswerIn{Answer: opt(r.correctOption(it.ID)).Answer, Version: 2})
	require.NoError(t, err) // câu 0 đúng lại
	// thiếu tab → 422
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it.ID, uuid.Nil, opt(r.correctOption(it.ID)))
	require.Equal(t, "VALIDATION_FAILED", code(t, err))

	// mở ở tab thứ hai → tab cũ 409 TAB_TAKEN_OVER; vị trí lưu và tải lại đúng câu
	_, err = r.svc.SetIndex(ctx, sv, r.course, v.ID, tab1, 3)
	require.NoError(t, err)
	got, err := r.svc.Get(ctx, sv, r.course, v.ID, tab2)
	require.NoError(t, err)
	require.Equal(t, 3, got.CurrentIndex)
	for _, it := range got.Items {
		require.Nil(t, it.Feedback, "đang làm: không lộ đáp án kể cả khi tải lại")
	}
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, v.ID, v.Items[5].ID, tab1, opt(r.correctOption(v.Items[5].ID)))
	require.Equal(t, "TAB_TAKEN_OVER", code(t, err))
	_, err = r.svc.SetIndex(ctx, sv, r.course, v.ID, tab1, 1)
	require.Equal(t, "TAB_TAKEN_OVER", code(t, err))
	_, err = r.svc.SetIndex(ctx, sv, r.course, v.ID, tab2, 6)
	require.Equal(t, "VALIDATION_FAILED", code(t, err))

	// nộp: 4 đúng / 6, điểm 6.7; mọi câu có đáp án + giải thích
	res, err := r.svc.Submit(ctx, sv, r.course, v.ID)
	require.NoError(t, err)
	require.Equal(t, 4, res.Result.Correct)
	require.Equal(t, 6, res.Result.Total)
	require.Equal(t, "6.7", res.Result.Score)
	for _, it := range res.Items {
		require.NotNil(t, it.Feedback)
		require.NotEmpty(t, it.Feedback.Key)
	}
	again, err := r.svc.Submit(ctx, sv, r.course, v.ID)
	require.NoError(t, err)
	require.Equal(t, raw(t, res.Result), raw(t, again.Result))

	// lượt mới: hết giờ trên máy chủ
	v2, created, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "MOCK", Count: 5}, tab1)
	require.NoError(t, err)
	require.True(t, created, "lượt cũ đã đóng nên tạo lượt mới")
	r.exec(`update practice_attempts set deadline_at = now() - interval '2 minutes' where id = $1`, v2.ID)
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, v2.ID, v2.Items[0].ID, tab1, opt(r.correctOption(v2.Items[0].ID)))
	require.Equal(t, "ATTEMPT_CLOSED", code(t, err))
}

// TestMockOfflineTolerance — FR-17 / US-P9-07 AC4: lựa chọn ghi ngoại tuyến có client_ts ≤ hạn + 60 s được nhận khi gửi muộn trong dung sai; sau dung sai thì không.
func TestMockOfflineTolerance(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student()
	r.approved("Mạng", 6)
	ctx := t.Context()
	tab := uuid.New()
	v, _, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "MOCK", Count: 5}, tab)
	require.NoError(t, err)
	// quá hạn 20 s (trong dung sai 60 s)
	r.exec(`update practice_attempts set deadline_at = now() - interval '20 seconds' where id = $1`, v.ID)
	var dl = *v.DeadlineAt
	_ = dl
	var deadline = r.deadline(v.ID)
	in := opt(r.correctOption(v.Items[0].ID))
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, v.ID, v.Items[0].ID, tab, in)
	require.Equal(t, "ATTEMPT_CLOSED", code(t, err), "không có client_ts: không nhận")
	ts := deadline.Add(-5 * time.Second)
	in.ClientTS = &ts
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, v.ID, v.Items[0].ID, tab, in)
	require.NoError(t, err, "client_ts trước hạn, gửi muộn trong dung sai")
	late := deadline.Add(2 * time.Minute)
	in2 := opt(r.correctOption(v.Items[1].ID))
	in2.ClientTS = &late
	_, err = r.svc.SaveAnswer(ctx, sv, r.course, v.ID, v.Items[1].ID, tab, in2)
	require.Equal(t, "ATTEMPT_CLOSED", code(t, err), "client_ts sau hạn + dung sai")

	// worker: quá hạn + dung sai → đóng EXPIRED, chấm câu đã lưu (1 đúng / 5)
	r.exec(`update practice_attempts set deadline_at = now() - interval '5 minutes' where id = $1`, v.ID)
	n, err := r.svc.CloseDue(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 1)
	got, err := r.svc.Get(ctx, sv, r.course, v.ID, uuid.Nil)
	require.NoError(t, err)
	require.Equal(t, "EXPIRED", got.Status)
	require.True(t, got.Result.Expired)
	require.Equal(t, 1, got.Result.Correct)
	require.Equal(t, 5, got.Result.Total)
	n, err = r.svc.CloseDue(ctx) // idempotent
	require.NoError(t, err)
	require.Zero(t, n)
}

// TestAnswerTypes — SHORT chấm sau chuẩn hoá (giữ dấu), đúng–sai, tự luận không chấm; ESSAY không vào MOCK (Q-P9-6).
func TestAnswerTypes(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student()
	ctx := t.Context()
	mk := func(in exam.QuestionIn) {
		d, err := r.ex.Create(ctx, r.teacher, r.course, in)
		require.NoError(t, err)
		_, err = r.ex.Review(ctx, r.teacher, r.course, d.ID, "APPROVE", d.Version)
		require.NoError(t, err)
	}
	for i := range 2 {
		mk(exam.QuestionIn{Type: "SHORT", Title: "ngắn", Topic: "Hỗn hợp", Stem: fmt.Sprintf("Thủ đô %d?", i), Accepted: []string{"Hà Nội"}})
		mk(exam.QuestionIn{Type: "TRUE_FALSE", Title: "đs", Topic: "Hỗn hợp", Stem: fmt.Sprintf("1 < 2 (%d)?", i), Value: new(true)})
		mk(exam.QuestionIn{Type: "ESSAY", Title: "luận", Topic: "Hỗn hợp", Stem: fmt.Sprintf("Phân tích %d.", i), ModelAnswer: new("Nêu A, B."), Rubric: new("A 5đ; B 5đ")})
	}
	_, _, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "MOCK"}, uuid.New())
	require.Equal(t, "NOT_ENOUGH_QUESTIONS", code(t, err), "ESSAY không vào MOCK: chỉ còn 4 câu đủ điều kiện")

	v, _, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Hỗn hợp", Count: 6}, uuid.Nil)
	require.NoError(t, err)
	var shortOK, tfOK, essays int
	for _, it := range v.Items {
		switch it.Question.Type {
		case "SHORT":
			ans := map[int]string{0: "  hà nội. ", 1: "ha noi"}[shortOK] // câu đầu đúng sau chuẩn hoá, câu sau thiếu dấu → sai
			s, err := r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it.ID, uuid.Nil, practice.AnswerIn{Answer: json.RawMessage(raw(t, map[string]string{"text": ans}))})
			require.NoError(t, err)
			require.Equal(t, shortOK == 0, *s.Feedback.Correct, ans)
			shortOK++
		case "TRUE_FALSE":
			s, err := r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it.ID, uuid.Nil, practice.AnswerIn{Answer: json.RawMessage(`{"value":true}`)})
			require.NoError(t, err)
			require.True(t, *s.Feedback.Correct)
			tfOK++
		case "ESSAY":
			s, err := r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it.ID, uuid.Nil, practice.AnswerIn{Answer: json.RawMessage(`{"text":"Bài làm của em."}`)})
			require.NoError(t, err)
			require.Nil(t, s.Feedback.Correct, "tự luận không chấm")
			require.Contains(t, string(s.Feedback.Key), "Nêu A, B.")
			essays++
		}
	}
	require.Equal(t, [3]int{2, 2, 2}, [3]int{shortOK, tfOK, essays})
	res, err := r.svc.Submit(ctx, sv, r.course, v.ID)
	require.NoError(t, err)
	require.Equal(t, 4, res.Result.Total, "tự luận không tính vào tổng")
	require.Equal(t, 3, res.Result.Correct)
	require.Equal(t, "7.5", res.Result.Score)
	// quá dài: 422 TOO_LONG, không tốn lời gọi nào
	sv2 := r.student()
	v, _, err = r.svc.Start(ctx, sv2, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Hỗn hợp", Count: 6}, uuid.Nil)
	require.NoError(t, err)
	for _, it := range v.Items {
		if it.Question.Type == "ESSAY" {
			_, err := r.svc.SaveAnswer(ctx, sv2, r.course, v.ID, it.ID, uuid.Nil, practice.AnswerIn{Answer: json.RawMessage(raw(t, map[string]string{"text": strings.Repeat("a", practice.MaxEssayChars+1)}))})
			require.Equal(t, "TOO_LONG", code(t, err))
			break
		}
	}
}

// TestOverviewWeakness — US-P9-06 AC1, US-P9-09 AC2: khuyến nghị theo lỗi 30 ngày; "Chủ đề cần ôn" kèm tài liệu từ citations của câu sai; ẩn khi < 10 câu.
func TestOverviewWeakness(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student()
	ctx := t.Context()
	var doc uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `insert into documents (course_id, title, type, status, visible_to_students) values ($1, 'Bài giảng mạng', 'LECTURE', 'READY', true) returning id`, r.course).Scan(&doc))
	r.approved("Mạng", 8)
	r.approved("Bảo mật", 12)
	r.exec(`update question_bank set citations = jsonb_build_array(jsonb_build_object('document_id', $2::text, 'page', 3)) where course_id = $1 and topic = 'Mạng'`, r.course, doc)

	ov, err := r.svc.Overview(ctx, sv, r.course)
	require.NoError(t, err)
	require.Equal(t, "Bảo mật", ov.Recommended.Topic, "chưa có lỗi: chủ đề nhiều câu nhất")
	require.Equal(t, 10, ov.Recommended.Count)
	require.Equal(t, []practice.TopicInfo{{Topic: "Bảo mật", Available: 12}, {Topic: "Mạng", Available: 8}}, ov.Topics)
	w, err := r.svc.Weakness(ctx, sv, r.course)
	require.NoError(t, err)
	require.Empty(t, w.Topics, "chưa đủ dữ liệu")

	do := func(topic string, n int, rightEvery int) {
		v, _, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "TOPIC", Topic: topic, Count: n}, uuid.Nil)
		require.NoError(t, err)
		for i, it := range v.Items {
			o := r.wrongOption(it.ID)
			if rightEvery > 0 && i%rightEvery == 0 {
				o = r.correctOption(it.ID)
			}
			_, err := r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it.ID, uuid.Nil, opt(o))
			require.NoError(t, err)
		}
		_, err = r.svc.Submit(ctx, sv, r.course, v.ID)
		require.NoError(t, err)
	}
	do("Mạng", 8, 0)    // 8 sai
	do("Bảo mật", 6, 2) // 3 đúng, 3 sai
	ov, err = r.svc.Overview(ctx, sv, r.course)
	require.NoError(t, err)
	require.Equal(t, "Mạng", ov.Recommended.Topic, "chủ đề sai nhiều nhất")
	w, err = r.svc.Weakness(ctx, sv, r.course)
	require.NoError(t, err)
	require.Len(t, w.Topics, 2)
	require.Equal(t, "Mạng", w.Topics[0].Topic)
	require.Equal(t, 8, w.Topics[0].Wrong)
	require.Equal(t, []practice.DocSuggested{{DocumentID: doc, Title: "Bài giảng mạng"}}, w.Topics[0].Documents)

	// lịch sử: mới nhất trước, phân trang bằng con trỏ
	h, err := r.svc.History(ctx, sv, r.course, nil, 2)
	require.NoError(t, err)
	require.Len(t, h, 2)
	require.Equal(t, "Bảo mật", *h[0].Topic)
	next, err := r.svc.History(ctx, sv, r.course, &practice.Cursor{At: h[0].StartedAt, ID: h[0].ID}, 5)
	require.NoError(t, err)
	require.Len(t, next, 1)
	require.Equal(t, "Mạng", *next[0].Topic)
}

// US-P9-09 AC3: PRACTICE_WEAK_TOPIC chỉ cho STUDENT, chỉ khi chủ đề yếu chưa luyện trong 3 ngày; luyện / nộp ghi `practice.updated`.
func TestTodayWeakTopic(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student()
	ctx := t.Context()
	r.approved("Mạng", 10)
	v, _, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "TOPIC", Topic: "Mạng", Count: 10}, uuid.Nil)
	require.NoError(t, err)
	for _, it := range v.Items {
		_, err := r.svc.SaveAnswer(ctx, sv, r.course, v.ID, it.ID, uuid.Nil, opt(r.wrongOption(it.ID)))
		require.NoError(t, err)
	}
	_, err = r.svc.Submit(ctx, sv, r.course, v.ID)
	require.NoError(t, err)
	var events int
	require.NoError(t, r.pool.QueryRow(ctx, `select count(*) from outbox where topic = $1 and payload ->> 'user_id' = $2`, practice.TopicUpdated, sv.String()).Scan(&events))
	require.Equal(t, 2, events, "bắt đầu + nộp")

	p := today.PracticeProvider{Pool: r.pool}
	view := today.Viewer{UserID: sv, Role: today.RoleStudent, Courses: []today.CourseRef{{ID: r.course, ClassCode: "X", RoleInCourse: "STUDENT"}}}
	items, err := p.Items(ctx, view, today.Scope{})
	require.NoError(t, err)
	require.Empty(t, items, "vừa luyện xong chủ đề: chưa nhắc")

	r.exec(`update practice_attempts set started_at = now() - interval '4 days' where student_id = $1`, sv)
	items, err = p.Items(ctx, view, today.Scope{})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, today.KindPracticeWeakTopic, items[0].Kind)
	require.Equal(t, 85, items[0].Tier)
	require.Equal(t, "Ôn chủ đề Mạng", items[0].Title)
	require.Contains(t, items[0].Href, "/practice?topic=M")

	view.Role = today.RoleTeacher
	items, err = p.Items(ctx, view, today.Scope{})
	require.NoError(t, err)
	require.Empty(t, items, "chỉ STUDENT")
}
