package chat_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/chat"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/llm"
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
