package chat_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/chat"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/rag"
)

// withTickets nối escalation.Service vào chat (như chatService ở gateway).
func (r *rig) withTickets() {
	r.svc.Tickets = &escalation.Service{Pool: r.pool, Members: course.Resolver{Pool: r.pool}, Lock: r.lock, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

// answer: tác tử trả lời `text` với intent + cờ NoContext (→ low_confidence) / NoFormula / Canned.
func (r *rig) answer(intent agent.Intent, noCtx, noFormula, canned bool, text string) {
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		o := agent.Outcome{Plan: intent, NoContext: noCtx, NoFormula: noFormula}
		if canned {
			o.Canned = text
		} else {
			o.Stream = r.ag.stream(ctx, nil, time.Millisecond, llm.Response{}, text)
		}
		return o, nil
	}
}

func TestAutoTicketSameTx(t *testing.T) {
	r := newRig(t)
	r.withTickets()
	r.answer(agent.IntentCourseQA, true, false, false, "Mình chưa chắc về câu này.")
	c, mid, err := r.send(r.student, r.sess, "Câu hỏi ngoài tài liệu")
	require.NoError(t, err)
	done := c.last()
	require.Equal(t, chat.EvDone, done.Ev)
	tk, ok := done.D["ticket"].(map[string]any)
	require.True(t, ok, "khung done có ticket: %v", done.D)
	require.Equal(t, "OPEN", tk["status"])
	require.NotContains(t, tk, "confidence")
	require.NotContains(t, tk, "origin")
	require.Equal(t, 1, r.count(`select count(*) from escalation_tickets where message_id = $1 and origin = 'AUTO_LOW_CONFIDENCE' and confidence is not null`, mid))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'escalation.created'`))

	page, err := r.svc.Messages(r.t.Context(), r.student, r.sess, pageOf(10))
	require.NoError(t, err)
	var seen bool
	for _, m := range page.Items {
		if m.ID == mid {
			require.NotNil(t, m.Ticket)
			require.Equal(t, tk["id"], m.Ticket.ID.String())
			seen = true
		} else {
			require.Nil(t, m.Ticket)
		}
	}
	require.True(t, seen)
}

func TestAutoTicketRollsBackWithAnswer(t *testing.T) {
	r := newRig(t)
	r.withTickets()
	r.answer(agent.IntentCourseQA, true, false, false, "Mình chưa chắc.")
	_, err := r.pool.Exec(r.t.Context(), `alter table outbox add constraint no_escalation_created check (topic <> 'escalation.created')`)
	require.NoError(t, err)
	c, mid, err := r.send(r.student, r.sess, "Câu hỏi")
	require.NoError(t, err)
	require.NotEqual(t, chat.EvDone, c.last().Ev, "lỗi ghi outbox ⇒ câu trả lời không DONE")
	st, _, _, _ := r.row(mid)
	require.Equal(t, "FAILED", st)
	require.Equal(t, 0, r.count(`select count(*) from escalation_tickets`), "không ticket mồ côi")
}

func TestHighConfidenceNoTicket(t *testing.T) {
	r := newRig(t)
	r.withTickets()
	c, mid, err := r.send(r.student, r.sess, "Hỏi bình thường") // fakeAgent mặc định: không low_confidence
	require.NoError(t, err)
	require.Nil(t, c.last().D["ticket"])
	require.Equal(t, 0, r.count(`select count(*) from escalation_tickets where message_id = $1`, mid))
}

func TestCrisisNeverTicketed(t *testing.T) {
	r := newRig(t)
	r.withTickets()
	r.answer(agent.IntentCrisis, true, true, true, "Bạn không đơn độc.") // cả NoContext lẫn NoFormula: vẫn không ticket
	c, _, err := r.send(r.student, r.sess, "tin khủng hoảng giả")
	require.NoError(t, err)
	require.Equal(t, chat.EvDone, c.last().Ev)
	require.Equal(t, 0, r.count(`select count(*) from escalation_tickets`))
	require.Equal(t, 0, r.count(`select count(*) from outbox where topic = 'escalation.created'`))
}

func TestNoFormulaAutoTicket(t *testing.T) {
	r := newRig(t)
	r.withTickets()
	r.answer(agent.IntentGradeFormula, false, true, true, agent.ReplyNoFormula)
	c, mid, err := r.send(r.student, r.sess, "Cách tính điểm môn này?")
	require.NoError(t, err)
	require.NotNil(t, c.last().D["ticket"])
	require.Equal(t, 1, r.count(`select count(*) from escalation_tickets where message_id = $1 and origin = 'AUTO_NO_FORMULA'`, mid))
}

func TestAutoTicketSuppressedAtCap(t *testing.T) {
	r := newRig(t)
	r.withTickets()
	r.answer(agent.IntentCourseQA, true, false, false, "Chưa chắc.")
	for range escalation.DefaultOpenCap {
		_, _, err := r.send(r.student, r.sess, "Câu hỏi")
		require.NoError(t, err)
	}
	require.Equal(t, 5, r.count(`select count(*) from escalation_tickets`))
	c, _, err := r.send(r.student, r.sess, "Câu thứ sáu")
	require.NoError(t, err)
	require.Equal(t, chat.EvDone, c.last().Ev, "câu trả lời vẫn hiện")
	require.Nil(t, c.last().D["ticket"])
	require.Equal(t, true, c.last().D["low_confidence"])
	require.Equal(t, 5, r.count(`select count(*) from escalation_tickets`))
	require.EqualValues(t, 1, r.svc.Tickets.Suppressed.Load())
}

// staffMsg chèn tin của giảng viên (STAFF_ANSWER) vào phiên của sinh viên.
func (r *rig) staffMsg(content string) uuid.UUID {
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into chat_messages (course_id, session_id, user_id, role, content, stream_status, intent)
		values ($1, $2, $3, 'ASSISTANT', $4, 'DONE', 'STAFF_ANSWER') returning id`, r.course, r.sess, r.student.UserID, content).Scan(&id))
	return id
}

func TestChatHistoryExcludesStaffMessages(t *testing.T) {
	r := newRig(t)
	_, _, err := r.send(r.student, r.sess, "câu một")
	require.NoError(t, err)
	r.svc.Wait()
	r.staffMsg("Em xem bài 3 nhé — lời của giảng viên")
	var got []llm.Message
	r.ag.fn = func(_ context.Context, _ agent.TrustedContext, in agent.Input) (agent.Outcome, error) {
		got = in.History
		return agent.Outcome{Plan: agent.IntentSmalltalk, Canned: "ok"}, nil
	}
	_, _, err = r.send(r.student, r.sess, "câu hai")
	require.NoError(t, err)
	require.Equal(t, []llm.Message{{Role: "user", Content: "câu một"}, {Role: "assistant", Content: "Xin chào bạn."}}, got, "lời giảng viên không vào ngữ cảnh LLM")
}

func TestFeedbackOnStaffAnswerRejected(t *testing.T) {
	r := newRig(t)
	mid := r.staffMsg("Trả lời của giảng viên")
	v := "HELPFUL"
	require.Equal(t, 422, httpStatus(r.svc.Feedback(t.Context(), r.student, mid, &v)))
	require.Equal(t, 0, r.count(`select count(*) from chat_messages where id = $1 and feedback is not null`, mid))
}

func TestRegenerateStaffAnswerRejected(t *testing.T) {
	r := newRig(t)
	mid := r.staffMsg("Trả lời của giảng viên")
	_, err := r.svc.Retry(t.Context(), r.student, mid)
	require.Equal(t, 422, httpStatus(err))
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where session_id = $1 and role = 'ASSISTANT'`, r.sess), "không sinh thêm tin nào")
}

// TestThresholdTakesEffectNextMessage — US-P4-05 AC12: ngưỡng đọc MỖI LẦN khi trả lời; đổi ngưỡng có hiệu lực ở tin kế tiếp của sinh viên.
// Cùng một câu trả lời (cosine cao, không nhận định nào đủ căn cứ ⇒ độ tin cậy 0,600): ngưỡng 0,55 → không chuyển; nâng lên 0,65 → tin kế tiếp tự chuyển.
func TestThresholdTakesEffectNextMessage(t *testing.T) {
	r := newRig(t)
	r.withTickets()
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Hits: []rag.Hit{{Cosine: 0.75, Text: "Quy chế thi cuối kỳ"}}, Stream: r.ag.stream(ctx, nil, time.Millisecond, llm.Response{}, "Có.")}, nil
	}
	set := func(v string) {
		_, err := r.pool.Exec(t.Context(), `update courses set escalation_threshold = $1 where id = $2`, v, r.course)
		require.NoError(t, err)
	}
	set("0.55")
	_, m1, err := r.send(r.student, r.sess, "Câu một")
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where id = $1 and confidence = 0.600 and not low_confidence`, m1))
	require.Zero(t, r.count(`select count(*) from escalation_tickets where message_id = $1`, m1), "ngưỡng 0,55 ≤ độ tin cậy 0,60: không chuyển")
	set("0.65")
	_, m2, err := r.send(r.student, r.sess, "Câu hai")
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from escalation_tickets where message_id = $1 and origin = 'AUTO_LOW_CONFIDENCE'`, m2), "ngưỡng mới 0,65 > 0,60: chuyển ngay ở tin kế tiếp")
}
