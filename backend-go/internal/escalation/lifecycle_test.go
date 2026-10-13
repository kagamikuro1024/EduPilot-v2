package escalation_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/workhours"
)

// withClock đặt đồng hồ giả cho service (mốc SLA, hỏi tiếp, tự đóng đều theo đồng hồ này, không theo now() của DB).
func (r *rig) withClock(t time.Time) *clock.Fake {
	c := clock.NewFake(t)
	r.svc.Clock = c
	return c
}

func (r *rig) setTicket(id uuid.UUID, set string, args ...any) {
	r.t.Helper()
	r.exec(`update escalation_tickets set `+set+` where id = $1`, append([]any{id}, args...)...)
}

func (r *rig) ticketRow(id uuid.UUID) (status string, stage int, reopened int, waiting time.Time) {
	r.t.Helper()
	var st string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select status::text, sla_stage, reopened_count, waiting_since from escalation_tickets where id = $1`, id).Scan(&st, &stage, &reopened, &waiting))
	return st, stage, reopened, waiting
}

func (r *rig) answered(q string) uuid.UUID {
	r.t.Helper()
	id := r.ticket(q)
	_, _, err := r.svc.Answer(r.t.Context(), r.gv(), r.course, id, "Trả lời", nil)
	require.NoError(r.t, err)
	return id
}

func TestFollowupReopens(t *testing.T) {
	r := newRig(t, escalation.Config{})
	clk := r.withClock(time.Date(2026, 10, 14, 10, 0, 0, 0, workhoursICT()))
	id := r.answered("Hỏi")
	r.setTicket(id, `claimed_by = $2, sla_stage = 1`, r.teacher)
	res, err := r.svc.Followup(t.Context(), r.a, id, "Em vẫn chưa hiểu bước 2", uuid.New())
	require.NoError(t, err)
	require.Equal(t, "OPEN", res.Ticket.Status)
	st, stage, reopened, waiting := r.ticketRow(id)
	require.Equal(t, [3]any{"OPEN", 0, 1}, [3]any{st, stage, reopened})
	require.True(t, waiting.Equal(clk.Now()), "waiting_since = bây giờ")
	require.Equal(t, 1, r.count(`select count(*) from escalation_tickets where id = $1 and claimed_by is null`, id))
	var role, intent string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select role::text, intent from chat_messages where id = $1`, *res.MessageID).Scan(&role, &intent))
	require.Equal(t, [2]string{"USER", "ESCALATION_FOLLOWUP"}, [2]string{role, intent})
	d, err := r.svc.GetDetail(t.Context(), r.course, id)
	require.NoError(t, err)
	var seen bool
	for _, m := range d.Transcript {
		if m.Content == "Em vẫn chưa hiểu bước 2" && m.Role == "USER" {
			seen = true
		}
	}
	require.True(t, seen, "giảng viên thấy tin hỏi tiếp trong transcript")
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'escalation.followup'`))
}

func TestFollowupIdempotentClientMsgID(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.withClock(time.Date(2026, 10, 14, 10, 0, 0, 0, workhoursICT()))
	id := r.answered("Hỏi")
	cid := uuid.New()
	first, err := r.svc.Followup(t.Context(), r.a, id, "Hỏi tiếp", cid)
	require.NoError(t, err)
	again, err := r.svc.Followup(t.Context(), r.a, id, "Hỏi tiếp", cid)
	require.NoError(t, err)
	require.Equal(t, *first.MessageID, *again.MessageID)
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where intent = 'ESCALATION_FOLLOWUP'`))
	_, _, reopened, _ := r.ticketRow(id)
	require.Equal(t, 1, reopened)
}

func TestFollowupExpired(t *testing.T) {
	r := newRig(t, escalation.Config{})
	clk := r.withClock(time.Date(2026, 10, 14, 10, 0, 0, 0, workhoursICT()))
	id := r.answered("Hỏi")
	r.setTicket(id, `status = 'CLOSED', closed_reason = 'AUTO_7D', closed_at = $2`, clk.Now().Add(-31*24*time.Hour))
	_, err := r.svc.Followup(t.Context(), r.a, id, "Hỏi tiếp", uuid.New())
	require.Equal(t, "TICKET_REOPEN_EXPIRED", code(err))
	r.setTicket(id, `closed_at = $2`, clk.Now().Add(-29*24*time.Hour))
	res, err := r.svc.Followup(t.Context(), r.a, id, "Hỏi tiếp", uuid.New())
	require.NoError(t, err, "chưa quá 30 ngày thì hỏi tiếp được")
	require.Equal(t, "OPEN", res.Ticket.Status)
}

func TestFollowupStateGuard(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.withClock(time.Date(2026, 10, 14, 10, 0, 0, 0, workhoursICT()))
	id := r.ticket("Hỏi")
	_, err := r.svc.Followup(t.Context(), r.a, id, "Hỏi tiếp", uuid.New())
	require.Equal(t, apiTicketStateInvalid, code(err), "OPEN: đang chờ rồi")
	_, err = r.svc.Claim(t.Context(), r.gv(), r.course, id, nil)
	require.NoError(t, err)
	_, err = r.svc.Followup(t.Context(), r.a, id, "Hỏi tiếp", uuid.New())
	require.Equal(t, apiTicketStateInvalid, code(err), "CLAIMED")
	_, _, err = r.svc.Answer(t.Context(), r.gv(), r.course, id, "x", nil)
	require.NoError(t, err)
	for name, body := range map[string]string{"rỗng": "  ", "quá dài": string(make([]byte, escalation.FollowupMaxChars+1))} {
		_, err = r.svc.Followup(t.Context(), r.a, id, body, uuid.New())
		require.Equal(t, 422, status(err), name)
	}
	_, err = r.svc.Followup(t.Context(), r.b, id, "Của người khác", uuid.New())
	require.Equal(t, 404, status(err), "sinh viên khác")
	_, err = r.svc.Followup(t.Context(), escalation.Actor{UserID: r.teacher, Role: auth.RoleTeacher}, id, "x", uuid.New())
	require.Equal(t, 403, status(err), "vai khác STUDENT")
}

func TestFollowupResetsWaitingSince(t *testing.T) {
	r := newRig(t, escalation.Config{})
	clk := r.withClock(time.Date(2026, 10, 14, 10, 0, 0, 0, workhoursICT()))
	id := r.answered("Hỏi")
	r.setTicket(id, `waiting_since = $2`, clk.Now().Add(-10*24*time.Hour))
	_, err := r.svc.Followup(t.Context(), r.a, id, "Hỏi tiếp", uuid.New())
	require.NoError(t, err)
	_, _, _, w := r.ticketRow(id)
	require.True(t, w.Equal(clk.Now()))
}

func TestReopenedTicketNotInstantlyOverdue(t *testing.T) {
	r := newRig(t, escalation.Config{})
	clk := r.withClock(time.Date(2026, 10, 14, 10, 0, 0, 0, workhoursICT()))
	id := r.ticket("Hỏi")
	r.setTicket(id, `created_at = $2, waiting_since = $2`, clk.Now().Add(-10*24*time.Hour))
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Trả lời", nil)
	require.NoError(t, err)
	_, err = r.svc.Followup(t.Context(), r.a, id, "Hỏi tiếp", uuid.New())
	require.NoError(t, err)
	clk.Advance(5 * time.Minute)
	require.NoError(t, r.svc.SLATick(t.Context()))
	_, stage, _, _ := r.ticketRow(id)
	require.Equal(t, 0, stage, "đồng hồ tính từ waiting_since, không từ created_at")
}

func TestConfirmClosesTicket(t *testing.T) {
	r := newRig(t, escalation.Config{})
	clk := r.withClock(time.Date(2026, 10, 14, 10, 0, 0, 0, workhoursICT()))
	id := r.answered("Hỏi")
	res, err := r.svc.Confirm(t.Context(), r.a, id)
	require.NoError(t, err)
	require.Equal(t, "CLOSED", res.Ticket.Status)
	var reason string
	var at time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select closed_reason, closed_at from escalation_tickets where id = $1`, id).Scan(&reason, &at))
	require.Equal(t, "STUDENT_CONFIRMED", reason)
	require.True(t, at.Equal(clk.Now()))
	_, err = r.svc.Confirm(t.Context(), r.a, id)
	require.Equal(t, apiTicketStateInvalid, code(err), "đã đóng")
}

func TestConfirmUndoWindow(t *testing.T) {
	r := newRig(t, escalation.Config{})
	clk := r.withClock(time.Date(2026, 10, 14, 10, 0, 0, 0, workhoursICT()))
	id := r.answered("Hỏi")
	_, err := r.svc.Confirm(t.Context(), r.a, id)
	require.NoError(t, err)
	clk.Advance(4 * time.Second)
	res, err := r.svc.ConfirmUndo(t.Context(), r.a, id)
	require.NoError(t, err, "trong 5 giây")
	require.Equal(t, "ANSWERED", res.Ticket.Status)
	_, err = r.svc.Confirm(t.Context(), r.a, id)
	require.NoError(t, err)
	clk.Advance(6 * time.Second)
	_, err = r.svc.ConfirmUndo(t.Context(), r.a, id)
	require.Equal(t, apiTicketStateInvalid, code(err), "quá 5 giây")
	// hành động khác đã đổi trạng thái → không hoàn tác được
	id2 := r.answered("Hỏi hai")
	_, err = r.svc.Confirm(t.Context(), r.a, id2)
	require.NoError(t, err)
	_, err = r.svc.Followup(t.Context(), r.a, id2, "Hỏi tiếp", uuid.New())
	require.NoError(t, err)
	_, err = r.svc.ConfirmUndo(t.Context(), r.a, id2)
	require.Equal(t, apiTicketStateInvalid, code(err))
}

func TestConfirmOwnerOnly(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.withClock(time.Date(2026, 10, 14, 10, 0, 0, 0, workhoursICT()))
	id := r.answered("Hỏi")
	_, err := r.svc.Confirm(t.Context(), r.b, id)
	require.Equal(t, 404, status(err))
	_, err = r.svc.Confirm(t.Context(), escalation.Actor{UserID: r.ta, Role: auth.RoleTA}, id)
	require.Equal(t, 403, status(err))
	_, err = r.svc.ConfirmUndo(t.Context(), r.b, id)
	require.Equal(t, 404, status(err))
}

func TestAutoCloseAfter7Days(t *testing.T) {
	r := newRig(t, escalation.Config{OpenCap: 10})
	clk := r.withClock(time.Date(2026, 10, 14, 10, 0, 0, 0, workhoursICT()))
	old, fresh := r.answered("Cũ"), r.answered("Mới")
	r.setTicket(old, `answered_at = $2`, clk.Now().Add(-7*24*time.Hour-time.Minute))
	r.setTicket(fresh, `answered_at = $2`, clk.Now().Add(-6*24*time.Hour))
	n, err := r.svc.AutoClose(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	st, _, _, _ := r.ticketRow(old)
	require.Equal(t, "CLOSED", st)
	require.Equal(t, 1, r.count(`select count(*) from escalation_tickets where id = $1 and closed_reason = 'AUTO_7D'`, old))
	st, _, _, _ = r.ticketRow(fresh)
	require.Equal(t, "ANSWERED", st)
	require.Equal(t, 1, r.count(`select count(*) from audit_log where action = 'escalation_ticket.autoclosed' and entity_id = $1`, old.String()))
	// thêm câu trả lời mới đặt lại mốc
	_, _, err = r.svc.Answer(t.Context(), r.gv(), r.course, fresh, "Bổ sung", nil)
	require.NoError(t, err)
	clk.Advance(2 * 24 * time.Hour)
	n, err = r.svc.AutoClose(t.Context())
	require.NoError(t, err)
	require.Zero(t, n)
}

func TestAutoCloseIdempotent(t *testing.T) {
	r := newRig(t, escalation.Config{})
	clk := r.withClock(time.Date(2026, 10, 14, 10, 0, 0, 0, workhoursICT()))
	id := r.answered("Hỏi")
	r.setTicket(id, `answered_at = $2`, clk.Now().Add(-8*24*time.Hour))
	var wg sync.WaitGroup
	got := make([]int, 2)
	for i := range got {
		wg.Add(1)
		go func() { defer wg.Done(); got[i], _ = r.svc.AutoClose(t.Context()) }()
	}
	wg.Wait()
	require.Equal(t, 1, got[0]+got[1], "chạy song song / hai lần = một lần")
	require.Equal(t, 1, r.count(`select count(*) from audit_log where action = 'escalation_ticket.autoclosed'`))
	n, err := r.svc.AutoClose(t.Context())
	require.NoError(t, err)
	require.Zero(t, n)
	res, err := r.svc.Followup(t.Context(), r.a, id, "Vẫn hỏi tiếp được", uuid.New())
	require.NoError(t, err, "đóng tự động vẫn hỏi tiếp được trong 30 ngày")
	require.Equal(t, "OPEN", res.Ticket.Status)
}

func TestAutoCloseSkipsOpen(t *testing.T) {
	r := newRig(t, escalation.Config{OpenCap: 10})
	clk := r.withClock(time.Date(2026, 10, 14, 10, 0, 0, 0, workhoursICT()))
	open, claimed := r.ticket("Mở"), r.ticket("Nhận")
	_, err := r.svc.Claim(t.Context(), r.gv(), r.course, claimed, nil)
	require.NoError(t, err)
	r.setTicket(open, `answered_at = $2`, clk.Now().Add(-30*24*time.Hour))
	r.setTicket(claimed, `answered_at = $2`, clk.Now().Add(-30*24*time.Hour))
	n, err := r.svc.AutoClose(t.Context())
	require.NoError(t, err)
	require.Zero(t, n)
	st, _, _, _ := r.ticketRow(open)
	require.Equal(t, "OPEN", st)
	st, _, _, _ = r.ticketRow(claimed)
	require.Equal(t, "CLAIMED", st)
}

// ---- SLA ----

// drain chạy handler outbox của job đúng một lần cho mỗi dòng chưa xử lý của topic (thay cho consumer).
type drainer struct {
	r    *rig
	done map[uuid.UUID]bool
	n    *escalation.Notifier
}

func (r *rig) drainer(clk clock.Clock) *drainer {
	return &drainer{r: r, done: map[uuid.UUID]bool{}, n: &escalation.Notifier{Pool: r.pool, Notify: notify.New(r.pool, &capture{}, nil), Clock: clk}}
}

func (d *drainer) run(topic string, h func(context.Context, outbox.Message) error) int {
	d.r.t.Helper()
	rows, err := d.r.pool.Query(d.r.t.Context(), `select id, payload from outbox where topic = $1 order by created_at, id`, topic)
	require.NoError(d.r.t, err)
	var msgs []outbox.Message
	for rows.Next() {
		var m outbox.Message
		require.NoError(d.r.t, rows.Scan(&m.ID, &m.Payload))
		m.Topic = topic
		msgs = append(msgs, m)
	}
	rows.Close()
	n := 0
	for _, m := range msgs {
		if d.done[m.ID] {
			continue
		}
		d.done[m.ID] = true
		require.NoError(d.r.t, h(d.r.t.Context(), m))
		n++
	}
	return n
}

func (r *rig) openTicketAt(waiting time.Time) uuid.UUID {
	r.t.Helper()
	id := r.ticket("Hỏi lúc " + waiting.Format(time.RFC3339))
	r.setTicket(id, `waiting_since = $2, created_at = $2`, waiting)
	return id
}

func (r *rig) stale(user uuid.UUID) int {
	return r.count(`select count(*) from notifications where type = 'TICKET_STALE' and user_id = $1 and read_at is null`, user)
}

// Mở T6 16/10 12:00 → nhắc T2 19/10 12:00 (không nhắc T7 / CN).
func TestSLAReminderOnce(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.teacher, r.ta)
	t6 := time.Date(2026, 10, 16, 12, 0, 0, 0, workhoursICT())
	clk := r.withClock(t6.Add(24 * time.Hour)) // T7 12:00
	id := r.openTicketAt(t6)
	d := r.drainer(clk)
	for _, at := range []time.Time{t6.Add(24 * time.Hour), t6.Add(48 * time.Hour), t6.Add(72*time.Hour - time.Minute)} { // T7, CN, T2 11:59
		clk.Set(at)
		require.NoError(t, r.svc.SLATick(t.Context()))
		_, stage, _, _ := r.ticketRow(id)
		require.Equal(t, 0, stage, at.String())
	}
	clk.Set(t6.Add(72*time.Hour + time.Minute)) // T2 12:01
	require.NoError(t, r.svc.SLATick(t.Context()))
	_, stage, _, _ := r.ticketRow(id)
	require.Equal(t, 1, stage)
	require.Equal(t, 1, d.run(escalation.TopicSLAReminder, d.n.HandleSLAReminder))
	require.Equal(t, [2]int{1, 1}, [2]int{r.stale(r.teacher), r.stale(r.ta)}, "một thông báo gộp mỗi người")
	require.Equal(t, 2, r.count(`select count(*) from mail_outbox where template = 'ticket_reminder'`), "một thư mỗi TA / TEACHER ACTIVE")
	// chạy lại: không gửi thêm
	clk.Advance(15 * time.Minute)
	require.NoError(t, r.svc.SLATick(t.Context()))
	require.Zero(t, d.run(escalation.TopicSLAReminder, d.n.HandleSLAReminder))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'escalation.sla_reminder'`))
	require.Equal(t, 2, r.count(`select count(*) from mail_outbox where template = 'ticket_reminder'`))
}

func TestSLATwoWorkersOnce(t *testing.T) {
	r := newRig(t, escalation.Config{OpenCap: 10})
	t2 := time.Date(2026, 10, 12, 9, 0, 0, 0, workhoursICT())
	clk := r.withClock(t2.Add(25 * time.Hour))
	ids := []uuid.UUID{r.openTicketAt(t2), r.openTicketAt(t2.Add(time.Minute)), r.openTicketAt(t2.Add(2 * time.Minute))}
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); require.NoError(t, r.svc.SLATick(t.Context())) }()
	}
	wg.Wait()
	for _, id := range ids {
		_, stage, _, _ := r.ticketRow(id)
		require.Equal(t, 1, stage)
	}
	require.LessOrEqual(t, r.count(`select count(*) from outbox where topic = 'escalation.sla_reminder'`), 2, "mỗi lớp mỗi lượt chạy ≤ 1 sự kiện; hai bản ≤ 2 (cập nhật có điều kiện: mỗi ticket một lần)")
	require.Equal(t, 3, r.count(`select count(*) from escalation_tickets where sla_stage = 1 and sla_stage_at = $1`, clk.Now())+r.count(`select count(*) from escalation_tickets where sla_stage = 1 and sla_stage_at <> $1`, clk.Now()))
}

func TestSLAClaimedRemindsOwnerOnly(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.teacher, r.ta)
	t2 := time.Date(2026, 10, 12, 9, 0, 0, 0, workhoursICT())
	clk := r.withClock(t2)
	r.svc.Notify = notify.New(r.pool, &capture{}, nil)
	id := r.ticket("Hỏi")
	_, err := r.svc.Claim(t.Context(), r.gv(), r.course, id, nil)
	require.NoError(t, err)
	r.setTicket(id, `claimed_at = $2, waiting_since = $2`, t2)
	clk.Set(t2.Add(25 * time.Hour))
	require.NoError(t, r.svc.SLATick(t.Context()))
	require.NoError(t, r.svc.SLATick(t.Context()))
	require.Equal(t, [2]int{1, 0}, [2]int{r.stale(r.teacher), r.stale(r.ta)}, "chỉ người đã nhận, đúng một lần")
	require.Zero(t, r.count(`select count(*) from mail_outbox where template = 'ticket_reminder'`), "ticket CLAIMED: không thư")
	require.Zero(t, r.count(`select count(*) from outbox where topic = 'escalation.sla_reminder'`), "không nhắc cả nhóm")
}

func TestSLAClaimedRemindsAgainAfterReclaim(t *testing.T) {
	r := newRig(t, escalation.Config{})
	t2 := time.Date(2026, 10, 12, 9, 0, 0, 0, workhoursICT())
	clk := r.withClock(t2)
	r.svc.Notify = notify.New(r.pool, &capture{}, nil)
	id := r.ticket("Hỏi")
	_, err := r.svc.Claim(t.Context(), r.gv(), r.course, id, nil)
	require.NoError(t, err)
	r.setTicket(id, `claimed_at = $2`, t2)
	clk.Set(t2.Add(25 * time.Hour))
	require.NoError(t, r.svc.SLATick(t.Context()))
	require.Equal(t, 1, r.stale(r.teacher))
	_, err = r.svc.Release(t.Context(), r.gv(), r.course, id, nil)
	require.NoError(t, err)
	_, err = r.svc.Claim(t.Context(), r.gv(), r.course, id, nil)
	require.NoError(t, err)
	r.setTicket(id, `claimed_at = $2`, t2.Add(26*time.Hour))
	clk.Set(t2.Add(51 * time.Hour))
	require.NoError(t, r.svc.SLATick(t.Context()))
	require.Equal(t, 2, r.stale(r.teacher), "nhận lại sau khi bỏ nhận được nhắc lại")
}

func TestSLAFilterUsesWaitingSince(t *testing.T) {
	r := newRig(t, escalation.Config{OpenCap: 10})
	t2 := time.Date(2026, 10, 12, 9, 0, 0, 0, workhoursICT())
	clk := r.withClock(t2.Add(23 * time.Hour))
	old := r.openTicketAt(t2)
	fresh := r.ticket("Mới tạo")
	r.setTicket(fresh, `created_at = $2, waiting_since = $3`, t2.Add(-30*24*time.Hour), t2.Add(22*time.Hour)) // tạo từ lâu nhưng vừa hỏi tiếp
	require.NoError(t, r.svc.SLATick(t.Context()))
	_, s1, _, _ := r.ticketRow(old)
	_, s2, _, _ := r.ticketRow(fresh)
	require.Equal(t, [2]int{0, 0}, [2]int{s1, s2}, "chưa đủ 24 giờ làm việc")
	clk.Advance(2 * time.Hour)
	require.NoError(t, r.svc.SLATick(t.Context()))
	_, s1, _, _ = r.ticketRow(old)
	_, s2, _, _ = r.ticketRow(fresh)
	require.Equal(t, [2]int{1, 0}, [2]int{s1, s2}, "chỉ ticket có waiting_since đủ lâu, không theo created_at")
}

func TestSLA72Escalates(t *testing.T) {
	r := newRig(t, escalation.Config{})
	t2 := time.Date(2026, 10, 12, 8, 0, 0, 0, workhoursICT())
	clk := r.withClock(t2.Add(25 * time.Hour))
	id := r.openTicketAt(t2)
	require.NoError(t, r.svc.SLATick(t.Context()))
	_, stage, _, _ := r.ticketRow(id)
	require.Equal(t, 1, stage)
	clk.Set(t2.Add(72*time.Hour - time.Minute)) // T5 07:59
	require.NoError(t, r.svc.SLATick(t.Context()))
	_, stage, _, _ = r.ticketRow(id)
	require.Equal(t, 1, stage)
	clk.Set(t2.Add(72*time.Hour + time.Minute)) // T5 08:01
	require.NoError(t, r.svc.SLATick(t.Context()))
	_, stage, _, _ = r.ticketRow(id)
	require.Equal(t, 2, stage)
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'escalation.sla_escalated'`))
	require.NoError(t, r.svc.SLATick(t.Context()))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'escalation.sla_escalated'`), "chạy lại: không phát thêm")
}

func (r *rig) staleSetup() (clk *clock.Fake, id uuid.UUID, d *drainer) {
	r.t.Helper()
	r.verify(r.admin)
	r.exec(`update users set status = 'ACTIVE', password_hash = 'x' where id = $1`, r.admin)
	t2 := time.Date(2026, 10, 12, 8, 0, 0, 0, workhoursICT())
	clk = r.withClock(t2.Add(80 * time.Hour)) // T5 16:00: ticket đã quá 72 giờ làm việc
	id = r.openTicketAt(t2)
	r.setTicket(id, `sla_stage = 2`)
	return clk, id, r.drainer(clk)
}

func TestAdminNotifiedOnlyWhenInactive(t *testing.T) {
	r := newRig(t, escalation.Config{})
	clk, _, d := r.staleSetup()
	// có TEACHER dùng phiên gần đây → không báo Admin
	r.exec(`insert into auth_sessions (user_id, refresh_hash, last_used_at, expires_at, absolute_expires_at) values ($1, repeat('a', 64), $2, $3, $3)`, r.teacher, clk.Now().Add(-time.Hour), clk.Now().Add(24*time.Hour))
	require.NoError(t, r.svc.SLATick(t.Context()))
	require.Zero(t, r.count(`select count(*) from outbox where topic = 'escalation.course_stalled'`), "có người hoạt động")
	// phiên cũ hơn 72 giờ → không ai hoạt động
	r.exec(`update auth_sessions set last_used_at = $1`, clk.Now().Add(-80*time.Hour))
	require.NoError(t, r.svc.SLATick(t.Context()))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'escalation.course_stalled'`))
	require.Equal(t, 1, d.run(escalation.TopicCourseStall, d.n.HandleCourseStalled))
	require.Equal(t, 1, r.count(`select count(*) from notifications where type = 'COURSE_TICKETS_STALLED' and user_id = $1`, r.admin))
	// thao tác ticket (audit) gần đây cũng tính là hoạt động
	r.exec(`delete from notifications where type = 'COURSE_TICKETS_STALLED'`)
	r.exec(`insert into audit_log (course_id, actor_id, entity, entity_id, action, before, after) values ($1, $2, 'escalation_ticket', $3, 'escalation_ticket.claimed', '{}', '{}')`, r.course, r.ta, uuid.NewString())
	r.exec(`delete from outbox where topic = 'escalation.course_stalled'`)
	require.NoError(t, r.svc.SLATick(t.Context()))
	require.Zero(t, r.count(`select count(*) from outbox where topic = 'escalation.course_stalled'`))
}

func TestAdminNotifiedDailyNotOnce(t *testing.T) {
	r := newRig(t, escalation.Config{})
	clk, _, d := r.staleSetup()
	for day := range 3 {
		for range 3 { // nhiều tick trong ngày: chỉ MỘT thông báo
			require.NoError(t, r.svc.SLATick(t.Context()))
			d.run(escalation.TopicCourseStall, d.n.HandleCourseStalled)
			clk.Advance(15 * time.Minute)
		}
		require.Equal(t, day+1, r.count(`select count(*) from notifications where type = 'COURSE_TICKETS_STALLED' and user_id = $1`, r.admin), "ngày %d", day+1)
		clk.Advance(24*time.Hour - 45*time.Minute)
	}
	require.Equal(t, 3, r.count(`select count(*) from notifications where type = 'COURSE_TICKETS_STALLED' and user_id = $1`, r.admin), "đúng 3 thông báo, mỗi ngày 1")
}

func TestAdminNeverSeesTicketContent(t *testing.T) {
	r := newRig(t, escalation.Config{})
	clk, id, d := r.staleSetup()
	r.exec(`update chat_messages set content = 'CANARY-NOI-DUNG' where session_id = $1`, r.sessA)
	require.NoError(t, r.svc.SLATick(t.Context()))
	d.run(escalation.TopicCourseStall, d.n.HandleCourseStalled)
	var title, body string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select title, coalesce(body, '') from notifications where type = 'COURSE_TICKETS_STALLED'`).Scan(&title, &body))
	require.Contains(t, title, "1 câu hỏi chờ quá 72 giờ làm việc")
	require.NotContains(t, title+body, "CANARY")
	require.NotContains(t, title+body, "SV A")
	var payload string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select payload::text from outbox where topic = 'escalation.course_stalled'`).Scan(&payload))
	require.NotContains(t, payload, "CANARY")
	require.NotContains(t, payload, id.String(), "không có id ticket")
	_ = clk
}

// SLA chết giữa chừng: mỗi bước ghi giai đoạn + outbox trong MỘT giao dịch — không có "đã gửi mà chưa ghi" và chạy lại hoàn tất đúng một lần.
func TestSLAJobCrashSafe(t *testing.T) {
	r := newRig(t, escalation.Config{})
	t2 := time.Date(2026, 10, 12, 9, 0, 0, 0, workhoursICT())
	r.withClock(t2.Add(25 * time.Hour))
	id := r.openTicketAt(t2)
	// "chết" ngay trước khi outbox được ghi: ràng buộc làm INSERT outbox lỗi ⇒ giao dịch huỷ cả cập nhật giai đoạn
	r.exec(`alter table outbox add constraint no_sla_reminder check (topic <> 'escalation.sla_reminder')`)
	require.Error(t, r.svc.SLATick(t.Context()))
	_, stage, _, _ := r.ticketRow(id)
	require.Equal(t, 0, stage, "giai đoạn không đổi khi sự kiện không ghi được")
	require.Zero(t, r.count(`select count(*) from outbox where topic = 'escalation.sla_reminder'`))
	r.exec(`alter table outbox drop constraint no_sla_reminder`)
	require.NoError(t, r.svc.SLATick(t.Context()))
	require.NoError(t, r.svc.SLATick(t.Context()))
	_, stage, _, _ = r.ticketRow(id)
	require.Equal(t, 1, stage)
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'escalation.sla_reminder'`), "chạy lại: đúng một lần")
}

const apiTicketStateInvalid = "TICKET_STATE_INVALID"

func workhoursICT() *time.Location { return workhours.ICT }
