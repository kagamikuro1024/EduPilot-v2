package escalation_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// capSender ghi lại thư; fail(n) (n = số lần gọi, từ 1) trả lỗi giả lập.
type capSender struct {
	mu   sync.Mutex
	sent []mail.Mail
	n    int
	fail func(n int) error
}

func (c *capSender) Send(_ context.Context, m mail.Mail) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
	if c.fail != nil {
		if err := c.fail(c.n); err != nil {
			return err
		}
	}
	c.sent = append(c.sent, m)
	return nil
}

func (r *rig) verify(ids ...uuid.UUID) {
	r.t.Helper()
	for _, id := range ids {
		r.exec(`update users set email_verified_at = now(), status = 'ACTIVE', password_hash = 'x' where id = $1`, id)
	}
}

func (r *rig) mailHandler(s mail.Sender, log *slog.Logger) *mail.Handler {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	h := &mail.Handler{Pool: r.pool, Clock: clock.Real{}, Sender: s, Log: log, Resolvers: map[string]mail.Resolver{},
		Cfg: config.Config{AppPublicURL: "https://app.test", MailSendTimeout: 3 * time.Second, OutboxRetryBackoff: []time.Duration{time.Second, time.Second, time.Second}}}
	escalation.MailResolver{PublicURL: "https://app.test"}.Register(h.Resolvers)
	return h
}

type mailRow struct {
	ID        uuid.UUID
	To        string
	Template  string
	Status    string
	Attempts  int
	LastError string
	Payload   string
}

func (r *rig) mailRows(where string, args ...any) []mailRow {
	r.t.Helper()
	rows, err := r.pool.Query(r.t.Context(), `select id, to_addr, template, status::text, attempts, coalesce(last_error, ''), payload::text from mail_outbox where `+where+` order by created_at, id`, args...)
	require.NoError(r.t, err)
	defer rows.Close()
	var out []mailRow
	for rows.Next() {
		var m mailRow
		require.NoError(r.t, rows.Scan(&m.ID, &m.To, &m.Template, &m.Status, &m.Attempts, &m.LastError, &m.Payload))
		out = append(out, m)
	}
	require.NoError(r.t, rows.Err())
	return out
}

func deliver(t *testing.T, h *mail.Handler, id uuid.UUID, attempts int) error {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"mail_id": id.String()})
	return h.Handle(t.Context(), outbox.Message{ID: uuid.New(), Topic: mail.Topic, Payload: b, Attempts: attempts})
}

func (r *rig) email(id uuid.UUID) string {
	r.t.Helper()
	var e string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select email from users where id = $1`, id).Scan(&e))
	return e
}

func TestMailRecipientIsTicketOwner(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.a.UserID, r.b.UserID)
	id := r.ticket("Hỏi")
	_, msg, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Em xem bài 3.", nil)
	require.NoError(t, err)
	rows := r.mailRows(`template = 'escalation_answered'`)
	require.Len(t, rows, 1)
	require.Equal(t, strings.ToLower(r.email(r.a.UserID)), rows[0].To, "thư tới email đã xác minh của CHÍNH chủ ticket")
	var p map[string]any
	require.NoError(t, json.Unmarshal([]byte(rows[0].Payload), &p))
	require.Equal(t, map[string]any{"ticket_id": id.String(), "answer_message_id": msg.ID.String()}, p, "payload chỉ có định danh")
	require.Equal(t, 1, r.count(`select count(*) from mail_outbox where dedupe_key = $1`, "escalation_answered:"+msg.ID.String()))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'mail.send'`), "thư và ticket cùng giao dịch")
}

func TestMailRespectsSettings(t *testing.T) {
	r := newRig(t, escalation.Config{OpenCap: 10})
	id := r.ticket("Hỏi")
	// email chưa xác minh → không thư
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Trả lời một", nil)
	require.NoError(t, err)
	require.Empty(t, r.mailRows(`template = 'escalation_answered'`), "email chưa xác minh")
	// đã xác minh nhưng tắt cờ → không thư, chuông vẫn có (tin chat vẫn tạo)
	r.verify(r.a.UserID)
	r.exec(`insert into user_settings (user_id, notify_answer_by_mail) values ($1, false)`, r.a.UserID)
	_, _, err = r.svc.Answer(t.Context(), r.gv(), r.course, id, "Trả lời hai", nil)
	require.NoError(t, err)
	require.Empty(t, r.mailRows(`template = 'escalation_answered'`), "notify_answer_by_mail=false")
	require.Equal(t, 2, r.count(`select count(*) from chat_messages where intent = 'STAFF_ANSWER'`))
	// bật lại → có thư
	r.exec(`update user_settings set notify_answer_by_mail = true where user_id = $1`, r.a.UserID)
	_, _, err = r.svc.Answer(t.Context(), r.gv(), r.course, id, "Trả lời ba", nil)
	require.NoError(t, err)
	require.Len(t, r.mailRows(`template = 'escalation_answered'`), 1)

	// ticket_new: mặc định tắt → không thư cho tới khi bật
	r.verify(r.teacher, r.ta)
	n := &escalation.Notifier{Pool: r.pool, Notify: notify.New(r.pool, &capture{}, nil)}
	pl, _ := json.Marshal(map[string]any{"course_id": r.course})
	require.NoError(t, n.HandleCreated(t.Context(), outbox.Message{ID: uuid.New(), Topic: escalation.TopicCreated, Payload: pl}))
	require.Empty(t, r.mailRows(`template = 'ticket_new'`), "mặc định tắt")
	r.exec(`insert into user_settings (user_id, notify_ticket_by_mail) values ($1, true)`, r.teacher)
	require.NoError(t, n.HandleCreated(t.Context(), outbox.Message{ID: uuid.New(), Topic: escalation.TopicCreated, Payload: pl}))
	rows := r.mailRows(`template = 'ticket_new'`)
	require.Len(t, rows, 1, "chỉ giảng viên đã bật; TA chưa bật")
	require.Equal(t, strings.ToLower(r.email(r.teacher)), rows[0].To)
}

// TestMailNoCrossStudentLeak + TestMailPayloadHasNoContent — AC2: nội dung đọc LÚC GỬI theo đúng ticket; payload chỉ có định danh.
func TestMailNoCrossStudentLeak(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.a.UserID, r.b.UserID)
	ta := r.ticket("CANARY-A1 câu hỏi của A")
	sessB := r.session(r.course, r.b.UserID)
	uq := r.msg(r.course, sessB, r.b.UserID, "USER", "CANARY-B2 câu hỏi của B", "", nil)
	am := r.msg(r.course, sessB, r.b.UserID, "ASSISTANT", "AI trả lời B", "COURSE_QA", &uq)
	cb, err := r.svc.Escalate(t.Context(), r.b, am)
	require.NoError(t, err)
	_, _, err = r.svc.Answer(t.Context(), r.gv(), r.course, ta, "Trả lời CANARY-A1-ANS", nil)
	require.NoError(t, err)
	_, _, err = r.svc.Answer(t.Context(), r.gv(), r.course, cb.Ticket.ID, "Trả lời CANARY-B2-ANS", nil)
	require.NoError(t, err)

	cs := &capSender{}
	h := r.mailHandler(cs, nil)
	for _, row := range r.mailRows(`template = 'escalation_answered'`) {
		require.NoError(t, deliver(t, h, row.ID, 0))
		require.NotContains(t, row.Payload, "CANARY", "payload không có nội dung")
		require.NotContains(t, row.Payload, "SV A")
	}
	require.Len(t, cs.sent, 2)
	for _, m := range cs.sent {
		body := m.Subject + m.Text + m.HTML
		switch m.To {
		case strings.ToLower(r.email(r.a.UserID)):
			require.Contains(t, body, "CANARY-A1")
			require.NotContains(t, body, "CANARY-B2")
			require.Contains(t, m.Text, "Chào SV A,")
		case strings.ToLower(r.email(r.b.UserID)):
			require.Contains(t, body, "CANARY-B2")
			require.NotContains(t, body, "CANARY-A1")
		default:
			t.Fatalf("thư tới người lạ: %s", m.To)
		}
		require.NotContains(t, m.Subject, "SV", "tiêu đề không chứa tên người")
		require.Contains(t, m.Text, "https://app.test/chat?session=")
		require.Contains(t, m.Text, "Giảng viên GV trả lời: ")
	}
}

func TestMailPayloadHasNoContent(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.a.UserID, r.teacher, r.ta)
	r.exec(`insert into user_settings (user_id, notify_ticket_by_mail) values ($1, true)`, r.teacher)
	id := r.ticket("CANARY-Q câu hỏi riêng")
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "CANARY-ANS trả lời riêng", nil)
	require.NoError(t, err)
	n := &escalation.Notifier{Pool: r.pool, Notify: notify.New(r.pool, &capture{}, nil)}
	pl, _ := json.Marshal(map[string]any{"course_id": r.course})
	require.NoError(t, n.HandleCreated(t.Context(), outbox.Message{ID: uuid.New(), Topic: escalation.TopicCreated, Payload: pl}))
	for _, row := range r.mailRows(`true`) {
		require.NotContains(t, row.Payload, "CANARY", row.Template)
		require.NotContains(t, strings.ToLower(row.Payload), "sv a", row.Template)
	}
	require.Zero(t, r.count(`select count(*) from outbox where topic = 'mail.send' and payload::text like '%CANARY%'`))
	// thư nhắc / thư ticket mới: chỉ số lượng + mã lớp + liên kết /inbox
	cs := &capSender{}
	h := r.mailHandler(cs, nil)
	for _, row := range r.mailRows(`template = 'ticket_new'`) {
		require.NoError(t, deliver(t, h, row.ID, 0))
	}
	require.Len(t, cs.sent, 1)
	require.Contains(t, cs.sent[0].Text, "/inbox?course="+r.course.String())
	require.NotContains(t, cs.sent[0].Text+cs.sent[0].HTML, "CANARY")
	require.NotContains(t, cs.sent[0].Text+cs.sent[0].HTML, "SV A")
}

func TestEscalationMailMissingTicket(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.a.UserID)
	id := r.ticket("Hỏi")
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "x", nil)
	require.NoError(t, err)
	row := r.mailRows(`template = 'escalation_answered'`)[0]
	r.exec(`delete from escalation_tickets where id = $1`, id)
	cs := &capSender{}
	require.NoError(t, deliver(t, r.mailHandler(cs, nil), row.ID, 0), "lỗi vĩnh viễn: không trả lỗi để outbox thử lại")
	got := r.mailRows(`id = $1`, row.ID)[0]
	require.Equal(t, [2]string{"DEAD", "ticket_missing"}, [2]string{got.Status, got.LastError})
	require.Empty(t, cs.sent, "không gửi thư rỗng")
}

func TestEscalationMailBadPayload(t *testing.T) {
	r := newRig(t, escalation.Config{})
	var id uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into mail_outbox (to_addr, template, payload) values ('a@example.test', 'escalation_answered', '{"answer_message_id":"x"}') returning id`).Scan(&id))
	cs := &capSender{}
	require.NoError(t, deliver(t, r.mailHandler(cs, nil), id, 0))
	got := r.mailRows(`id = $1`, id)[0]
	require.Equal(t, [2]string{"DEAD", "payload_invalid"}, [2]string{got.Status, got.LastError})
	require.Empty(t, cs.sent)
}

func TestMailRetryThenDead(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.a.UserID)
	id := r.ticket("Hỏi")
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "x", nil)
	require.NoError(t, err)
	row := r.mailRows(`template = 'escalation_answered'`)[0]
	cs := &capSender{fail: func(int) error { return &mail.SendFailure{Class: "smtp_temporary"} }}
	h := r.mailHandler(cs, nil)
	for attempt := range 4 { // 1 lần đầu + 3 lần thử lại
		err := deliver(t, h, row.ID, attempt)
		require.Error(t, err, "lần %d: trả lỗi để outbox thử lại / dead-letter", attempt+1)
	}
	got := r.mailRows(`id = $1`, row.ID)[0]
	require.Equal(t, "DEAD", got.Status)
	require.Equal(t, "smtp_temporary", got.LastError, "chỉ nhãn lớp lỗi")
	require.Equal(t, 4, got.Attempts)
	require.Equal(t, 4, cs.n, "đúng 4 lần gọi SMTP")
	require.NoError(t, deliver(t, h, row.ID, 4), "đã DEAD: giao lại là no-op")
	require.Equal(t, 4, cs.n)
}

func TestMailPermanentFailureNoRetry(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.a.UserID)
	id := r.ticket("Hỏi")
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "x", nil)
	require.NoError(t, err)
	row := r.mailRows(`template = 'escalation_answered'`)[0]
	cs := &capSender{fail: func(int) error { return &mail.SendFailure{Permanent: true, Class: "mailbox_unavailable"} }}
	require.NoError(t, deliver(t, r.mailHandler(cs, nil), row.ID, 0))
	got := r.mailRows(`id = $1`, row.ID)[0]
	require.Equal(t, [3]any{"DEAD", "mailbox_unavailable", 1}, [3]any{got.Status, got.LastError, cs.n})
}

// Thư lỗi không ảnh hưởng ứng dụng: ticket vẫn ANSWERED, tin vẫn trong chat, chuông vẫn tới.
func TestMailFailureDoesNotAffectAnswer(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.a.UserID)
	id := r.ticket("Hỏi")
	_, msg, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Trả lời", nil)
	require.NoError(t, err)
	row := r.mailRows(`template = 'escalation_answered'`)[0]
	cs := &capSender{fail: func(int) error { return errors.New("boom") }}
	h := r.mailHandler(cs, nil)
	for attempt := range 4 {
		_ = deliver(t, h, row.ID, attempt)
	}
	require.Equal(t, "DEAD", r.mailRows(`id = $1`, row.ID)[0].Status)
	require.Equal(t, 1, r.count(`select count(*) from escalation_tickets where id = $1 and status = 'ANSWERED'`, id))
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where id = $1`, msg.ID))
	n := &escalation.Notifier{Pool: r.pool, Notify: notify.New(r.pool, &capture{}, nil)}
	pl, _ := json.Marshal(map[string]any{"course_id": r.course, "answer_message_id": msg.ID})
	require.NoError(t, n.HandleAnswered(t.Context(), outbox.Message{ID: uuid.New(), Topic: escalation.TopicAnswered, Payload: pl}))
	require.Equal(t, 1, r.count(`select count(*) from notifications where type = 'TICKET_ANSWERED' and user_id = $1`, r.a.UserID), "chuông vẫn tới")
}

func TestMailLogsHaveNoAddress(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.a.UserID)
	id := r.ticket("Hỏi")
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "BÍ-MẬT-TRẢ-LỜI", nil)
	require.NoError(t, err)
	row := r.mailRows(`template = 'escalation_answered'`)[0]
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, nil))
	cs := &capSender{fail: func(int) error { return &mail.SendFailure{Class: "smtp_temporary"} }}
	h := r.mailHandler(cs, log)
	for attempt := range 4 {
		_ = deliver(t, h, row.ID, attempt)
	}
	require.NotEmpty(t, buf.String())
	require.NotContains(t, buf.String(), "@", "log không có địa chỉ hộp thư")
	require.NotContains(t, buf.String(), "BÍ-MẬT")
	require.Contains(t, buf.String(), row.ID.String())
}

func TestReminderMailRecipients(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.teacher, r.ta, r.admin)
	other := r.user("GV lớp khác", "TEACHER")
	r.verify(other)
	third := r.newCourse("ACTIVE")
	r.enroll(third, other, "TEACHER", "ACTIVE")
	r.exec(`update enrollments set status = 'REMOVED', removed_at = now() where course_id = $1 and user_id = $2`, r.course2, r.ta) // TA đã rời lớp 2
	now := time.Date(2026, 10, 13, 3, 30, 0, 0, time.UTC)
	tx, err := r.pool.Begin(t.Context())
	require.NoError(t, err)
	n, err := escalation.EnqueueReminder(t.Context(), tx, r.course, now)
	require.NoError(t, err)
	n2, err := escalation.EnqueueReminder(t.Context(), tx, r.course, now.Add(20*time.Minute))
	require.NoError(t, err)
	require.NoError(t, tx.Commit(t.Context()))
	require.Equal(t, [2]int{2, 0}, [2]int{n, n2}, "TEACHER + TA; cùng giờ không gửi lại")
	got := map[string]bool{}
	for _, m := range r.mailRows(`template = 'ticket_reminder'`) {
		got[m.To] = true
	}
	require.Equal(t, map[string]bool{strings.ToLower(r.email(r.teacher)): true, strings.ToLower(r.email(r.ta)): true}, got, "không ADMIN, không giảng viên lớp khác")

	tx, err = r.pool.Begin(t.Context())
	require.NoError(t, err)
	n3, err := escalation.EnqueueReminder(t.Context(), tx, r.course, now.Add(time.Hour))
	require.NoError(t, err)
	require.NoError(t, tx.Commit(t.Context()))
	require.Equal(t, 2, n3, "giờ kế tiếp được gửi lại")
}

func TestTicketNewMailCoalesced(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.teacher)
	r.exec(`insert into user_settings (user_id, notify_ticket_by_mail) values ($1, true)`, r.teacher)
	rc, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })
	clk := clock.NewFake(time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC))
	n := &escalation.Notifier{Pool: r.pool, Notify: notify.New(r.pool, &capture{}, nil), Gate: escalation.MailGate{Redis: rc, Clock: clk, Window: 15 * time.Minute}}
	pl, _ := json.Marshal(map[string]any{"course_id": r.course})
	same := uuid.New()
	for _, d := range []time.Duration{0, time.Minute, 4 * time.Minute} { // 3 ticket trong 5 phút → 1 thư
		clk.Set(time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC).Add(d))
		require.NoError(t, n.HandleCreated(t.Context(), outbox.Message{ID: uuid.New(), Topic: escalation.TopicCreated, Payload: pl}))
	}
	require.Len(t, r.mailRows(`template = 'ticket_new'`), 1)
	// giao lại đúng tin outbox: không thêm thư (dedupe) và không tiêu mốc
	require.NoError(t, n.HandleCreated(t.Context(), outbox.Message{ID: same, Topic: escalation.TopicCreated, Payload: pl}))
	require.Len(t, r.mailRows(`template = 'ticket_new'`), 1)
}

func TestTicketNewMailSlidingWindow(t *testing.T) {
	rc, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })
	base := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
	allow := func(at time.Time, gate escalation.MailGate, u, c uuid.UUID) bool {
		gate.Clock = clock.NewFake(at)
		return gate.Allow(t.Context(), u, c)
	}
	g := escalation.MailGate{Redis: rc, Window: 15 * time.Minute}
	// 10:14:59 và 10:15:01 cách nhau 2 giây: KHÔNG phải hai ô cố định 15 phút → 1 thư
	u, c := uuid.New(), uuid.New()
	require.True(t, allow(base.Add(14*time.Minute+59*time.Second), g, u, c))
	require.False(t, allow(base.Add(15*time.Minute+1*time.Second), g, u, c))
	// 10:00 và 10:16 cách 16 phút → 2 thư
	u2 := uuid.New()
	require.True(t, allow(base, g, u2, c))
	require.True(t, allow(base.Add(16*time.Minute), g, u2, c))
	// khác lớp / khác người: cửa sổ riêng
	require.True(t, allow(base, g, u, uuid.New()))
	// Redis lỗi → vẫn gửi
	bad, err := appredis.New(t.Context(), "redis://127.0.0.1:1/0")
	if err == nil {
		g.Redis = bad
		g.Clock = clock.NewFake(base)
		require.True(t, g.Allow(t.Context(), u, c))
	}
}

// ---- thư thật tới Mailpit: giao lại cùng Message-ID chỉ còn một thư; chuỗi hoàn chỉnh ticket → trả lời → thư ----

type mpMsg struct {
	ID        string `json:"ID"`
	MessageID string `json:"MessageID"`
	Subject   string
	Text      string
}

func mpSearch(t *testing.T, api, addr string) []mpMsg {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, api+"/api/v1/search?query="+url.QueryEscape("to:"+addr), http.NoBody)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	var list struct {
		Messages []struct{ ID string } `json:"messages"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
	var out []mpMsg
	for _, m := range list.Messages {
		rq, err := http.NewRequestWithContext(t.Context(), http.MethodGet, api+"/api/v1/message/"+m.ID, http.NoBody)
		require.NoError(t, err)
		rs, err := http.DefaultClient.Do(rq)
		require.NoError(t, err)
		var full mpMsg
		require.NoError(t, json.NewDecoder(rs.Body).Decode(&full))
		_ = rs.Body.Close()
		out = append(out, full)
	}
	return out
}

func smtpCfg(t *testing.T, smtpAddr string) config.Config {
	t.Helper()
	host, port, ok := strings.Cut(smtpAddr, ":")
	require.True(t, ok)
	var p int
	for _, c := range port {
		p = p*10 + int(c-'0')
	}
	return config.Config{SMTPHost: host, SMTPPort: p, SMTPTLS: "none", MailFrom: "EduPilot <no-reply@edupilot.local>", MailSendTimeout: 3 * time.Second,
		AppPublicURL: "https://app.test", OutboxRetryBackoff: []time.Duration{time.Second, time.Second, time.Second}}
}

// killAfterSend gửi thật rồi "chết" (panic) trước khi handler kịp ghi SENT: mô phỏng SIGKILL giữa DATA và COMMIT.
type killAfterSend struct {
	inner mail.Sender
	kill  bool
}

func (k *killAfterSend) Send(ctx context.Context, m mail.Mail) error {
	if err := k.inner.Send(ctx, m); err != nil {
		return err
	}
	if k.kill {
		panic("SIGKILL")
	}
	return nil
}

func TestOutboxSurvivesWorkerKill(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.a.UserID)
	smtpAddr, api := testutil.Mailpit(t)
	cfg := smtpCfg(t, smtpAddr)
	id := r.ticket("Hỏi")
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Trả lời sống sót", nil)
	require.NoError(t, err)
	row := r.mailRows(`template = 'escalation_answered'`)[0]
	to := row.To
	ks := &killAfterSend{inner: mail.SMTP{Cfg: cfg}, kill: true}
	h := r.mailHandler(ks, nil)
	h.Cfg = cfg
	func() {
		defer func() { _ = recover() }()
		_ = deliver(t, h, row.ID, 0) // chết sau khi Mailpit đã nhận DATA, trước khi ghi SENT
	}()
	require.Equal(t, "QUEUED", r.mailRows(`id = $1`, row.ID)[0].Status, "chưa ghi SENT")
	ks.kill = false
	require.NoError(t, deliver(t, h, row.ID, 1)) // outbox giao lại
	require.Equal(t, "SENT", r.mailRows(`id = $1`, row.ID)[0].Status)
	want := "<" + mail.MessageID(row.ID, cfg.MailFrom) + ">"
	require.Eventually(t, func() bool { return len(mpSearch(t, api, to)) >= 1 }, 10*time.Second, 50*time.Millisecond)
	time.Sleep(300 * time.Millisecond)
	n := 0
	for _, m := range mpSearch(t, api, to) {
		if m.MessageID == want || "<"+m.MessageID+">" == want {
			n++
		}
	}
	require.Equal(t, 1, n, "SMTP ít nhất một lần, nhưng Mailpit chỉ có đúng một thư mang Message-ID của dòng")
}

func TestMailSendIdempotentRedelivery(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.a.UserID)
	id := r.ticket("Hỏi")
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "x", nil)
	require.NoError(t, err)
	row := r.mailRows(`template = 'escalation_answered'`)[0]
	cs := &capSender{}
	h := r.mailHandler(cs, nil)
	require.NoError(t, deliver(t, h, row.ID, 0))
	require.NoError(t, deliver(t, h, row.ID, 1), "giao lại sau SENT")
	require.Len(t, cs.sent, 1)
	require.Equal(t, row.ID, cs.sent[0].ID, "thư mang id dòng outbox (→ Message-ID tất định)")
}

func TestEscalationToMail(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.verify(r.a.UserID)
	smtpAddr, api := testutil.Mailpit(t)
	cfg := smtpCfg(t, smtpAddr)
	_, m := r.pair("CANARY-E2E câu hỏi", "AI chưa chắc")
	first, err := r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)
	again, err := r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)
	require.Equal(t, first.Ticket.ID, again.Ticket.ID, "gửi lại cùng message_id không tạo ticket thứ hai")

	start := time.Now()
	_, _, err = r.svc.Answer(t.Context(), r.gv(), r.course, first.Ticket.ID, "Em xem bài 3 nhé.", nil)
	require.NoError(t, err)
	// relay + consumer thật: outbox → mail.send → Mailpit
	rc, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })
	pfx := testutil.TestPrefix(t)
	cfg.InstanceID, cfg.OutboxPollInterval, cfg.OutboxBatch = pfx, 20*time.Millisecond, 100
	cfg.OutboxClaimIdle, cfg.OutboxStaleAfter = 5*time.Second, time.Minute
	streams := outbox.Streams{Dispatch: pfx + ".dispatch", Dead: pfx + ".dispatch.dead", Consumer: pfx + "-c1"}
	t.Cleanup(func() { _ = rc.Del(context.Background(), streams.Dispatch, streams.Dead).Err() })
	h := r.mailHandler(mail.SMTP{Cfg: cfg}, nil)
	h.Cfg = cfg
	reg := outbox.NewRegistry()
	reg.Register(mail.Topic, h.Handle)
	d := outbox.Deps{Pool: r.pool, Redis: rc, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Clock: clock.Real{}, Cfg: cfg, Streams: streams}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, task := range []interface{ Run(context.Context) error }{outbox.NewRelay(d), outbox.NewConsumer(d, reg)} {
		wg.Add(1)
		go func() { defer wg.Done(); _ = task.Run(ctx) }()
	}
	t.Cleanup(func() { cancel(); wg.Wait() })

	to := strings.ToLower(r.email(r.a.UserID))
	var got []mpMsg
	require.Eventually(t, func() bool { got = mpSearch(t, api, to); return len(got) >= 1 }, 60*time.Second, 50*time.Millisecond)
	require.Less(t, time.Since(start), 60*time.Second)
	require.Len(t, got, 1)
	require.Equal(t, "Giảng viên đã trả lời câu hỏi của bạn", got[0].Subject)
	require.Contains(t, got[0].Text, "CANARY-E2E câu hỏi")
	require.Contains(t, got[0].Text, "Em xem bài 3 nhé.")
	require.Contains(t, got[0].Text, "https://app.test/chat?session="+r.sessA.String()+"&m=")
}
