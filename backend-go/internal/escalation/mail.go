package escalation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Tên mẫu thư của vòng đời ticket (SRS 4.5.2).
const (
	MailAnswered = "escalation_answered"
	MailOutreach = "outreach_message"
	MailReminder = "ticket_reminder"
	MailNew      = "ticket_new"
)

const (
	questionClip = 300
	answerClip   = 2000
)

// MailResolver nạp biến hiển thị của bốn mẫu thư LÚC GỬI (SRS 4.5.1): payload của `mail_outbox` chỉ có định danh, nội dung riêng tư
// (câu hỏi, trả lời) đọc ở đây theo ĐÚNG ticket nên không lẫn giữa sinh viên.
type MailResolver struct{ PublicURL string }

// Register gắn resolver cho cả bốn mẫu vào bảng của mail.Handler.
func (m MailResolver) Register(into map[string]mail.Resolver) {
	for _, t := range []string{MailAnswered, MailOutreach, MailReminder, MailNew} {
		into[t] = m
	}
}

type mailPayload struct {
	TicketID        uuid.UUID `json:"ticket_id"`
	AnswerMessageID uuid.UUID `json:"answer_message_id"`
	CourseID        uuid.UUID `json:"course_id"`
}

// Resolve implements mail.Resolver.
func (m MailResolver) Resolve(ctx context.Context, tx pgx.Tx, row store.MailOutbox) (map[string]string, error) {
	var p mailPayload
	if err := json.Unmarshal(row.Payload, &p); err != nil {
		return nil, mail.Permanent("payload_invalid")
	}
	q := store.New(tx)
	switch row.Template {
	case MailReminder, MailNew:
		if p.CourseID == uuid.Nil {
			return nil, mail.Permanent("payload_invalid")
		}
		c, err := q.GetCourseBasic(ctx, p.CourseID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, mail.Permanent("course_missing")
		}
		if err != nil {
			return nil, fmt.Errorf("escalation: đọc lớp: %w", err)
		}
		n, err := q.TicketCountOpenInCourse(ctx, p.CourseID) // đếm LÚC GỬI
		if err != nil {
			return nil, fmt.Errorf("escalation: đếm ticket: %w", err)
		}
		return map[string]string{"ClassCode": c.ClassCode, "Count": fmt.Sprint(n), "InboxURL": m.PublicURL + "/inbox?course=" + c.ID.String()}, nil
	case MailAnswered, MailOutreach:
		return m.answered(ctx, q, row.Template, p)
	}
	return nil, mail.Permanent("template_unknown")
}

func (m MailResolver) answered(ctx context.Context, q *store.Queries, tmpl string, p mailPayload) (map[string]string, error) {
	if p.TicketID == uuid.Nil || p.AnswerMessageID == uuid.Nil {
		return nil, mail.Permanent("payload_invalid")
	}
	t, err := q.TicketByID(ctx, p.TicketID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, mail.Permanent("ticket_missing")
	}
	if err != nil {
		return nil, fmt.Errorf("escalation: đọc ticket: %w", err)
	}
	msg, err := q.GetChatMessageByID(ctx, p.AnswerMessageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, mail.Permanent("ticket_missing")
	}
	if err != nil {
		return nil, fmt.Errorf("escalation: đọc tin trả lời: %w", err)
	}
	if msg.UserID != t.StudentID || msg.CourseID != t.CourseID { // tin phải thuộc đúng sinh viên / lớp của ticket
		return nil, mail.Permanent("payload_invalid")
	}
	stu, err := q.GetUser(ctx, t.StudentID)
	if err != nil {
		return nil, fmt.Errorf("escalation: đọc sinh viên: %w", err)
	}
	c, err := q.GetCourseBasic(ctx, t.CourseID)
	if err != nil {
		return nil, fmt.Errorf("escalation: đọc lớp: %w", err)
	}
	var by struct{ Name, Role string }
	var bs []ticketBlock
	if json.Unmarshal(msg.Blocks, &bs) == nil {
		for _, b := range bs {
			if b.Kind == "ticket" && b.Data.AnsweredBy != nil {
				by.Name, by.Role = b.Data.AnsweredBy.FullName, b.Data.AnsweredBy.Role
			}
		}
	}
	role := "Giảng viên"
	if by.Role == "TA" {
		role = "Trợ giảng"
	}
	out := map[string]string{
		"FullName": stu.FullName, "ClassCode": c.ClassCode, "AnswererRole": role, "AnswererName": by.Name,
		"AnswerText": mail.Clip(msg.Content, answerClip, "… Xem đầy đủ trong EduPilot"),
		"ChatURL":    m.PublicURL + "/chat?session=" + msg.SessionID.String() + "&m=" + msg.ID.String(),
	}
	if tmpl == MailAnswered {
		if t.MessageID == nil {
			return nil, mail.Permanent("ticket_missing")
		}
		ex, err := q.GetEscalatedExchange(ctx, store.GetEscalatedExchangeParams{CourseID: t.CourseID, MessageID: *t.MessageID})
		if err != nil {
			return nil, fmt.Errorf("escalation: đọc câu hỏi: %w", err)
		}
		for _, e := range ex {
			if e.Role == store.ChatRoleUSER {
				out["QuestionText"] = mail.Clip(e.Content, questionClip, "…")
			}
		}
	}
	return out, nil
}

// enqueueAnswerMail xếp thư `escalation_answered` / `outreach_message` TRONG giao dịch của người gọi (ticket + tin + thư cùng commit,
// nên không có câu trả lời nào thiếu thư). Chỉ tới email ĐÃ XÁC MINH của chính chủ ticket và khi `notify_answer_by_mail` bật (mặc định bật).
func enqueueAnswerMail(ctx context.Context, tx pgx.Tx, tmpl string, t store.EscalationTicket, answerMsg uuid.UUID) error {
	tgt, err := store.New(tx).AnswerMailTarget(ctx, t.StudentID)
	if err != nil {
		return fmt.Errorf("escalation: người nhận thư: %w", err)
	}
	if !tgt.Verified || !tgt.WantsMail || tgt.Email == "" {
		return nil
	}
	_, _, err = mail.Enqueue(ctx, tx, mail.Message{
		To: tgt.Email, Template: tmpl, Payload: map[string]any{"ticket_id": t.ID, "answer_message_id": answerMsg},
		DedupeKey: tmpl + ":" + answerMsg.String(),
	})
	if err != nil {
		return fmt.Errorf("escalation: xếp thư: %w", err)
	}
	return nil
}

// slideScript đặt mốc `now` nếu chưa có mốc nào trong `window`; trả 1 = được gửi. Atomic (Lua) và dùng đồng hồ truyền vào (test dùng clock.Fake).
const slideScript = `
local v = redis.call('GET', KEYS[1])
if v and (tonumber(ARGV[1]) - tonumber(v)) < tonumber(ARGV[2]) then return 0 end
redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[3])
return 1`

// MailGate gộp thư `ticket_new` bằng cửa sổ TRƯỢT: tối đa một thư mỗi (người, lớp) trong mọi khoảng `Window` (không theo ô cố định). Redis lỗi → vẫn gửi.
type MailGate struct {
	Redis  *appredis.Client // nil = không gộp
	Clock  clock.Clock
	Window time.Duration
}

// Allow cho biết có được gửi thư `ticket_new` cho (user, course) lúc này không và ghi mốc nếu được.
func (g MailGate) Allow(ctx context.Context, user, course uuid.UUID) bool {
	if g.Redis == nil || g.Window <= 0 {
		return true
	}
	now := g.Clock.Now().UnixMilli()
	res, err := g.Redis.Eval(ctx, slideScript, []string{appredis.Key("mail", "ticket_new", user.String(), course.String())}, now, g.Window.Milliseconds(), (g.Window + time.Minute).Milliseconds()).Int()
	if err != nil {
		return true // thà thừa một thư còn hơn mất
	}
	return res == 1
}

// EnqueueReminder xếp thư `ticket_reminder` cho TA + TEACHER ACTIVE (email đã xác minh) của ĐÚNG lớp — không ADMIN, không giảng viên lớp khác, không phụ thuộc cờ cá nhân.
// dedupe: `ticket_reminder:<user>:<course>:<yyyymmddHH>` (giờ Việt Nam) ⇒ tối đa một thư mỗi giờ mỗi lớp. Trả số thư mới xếp. Chạy trong giao dịch của job SLA.
func EnqueueReminder(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, now time.Time) (int, error) {
	targets, err := store.New(tx).TicketStaffRemindTargets(ctx, courseID)
	if err != nil {
		return 0, fmt.Errorf("escalation: người nhận thư nhắc: %w", err)
	}
	hour := now.In(time.FixedZone("ICT", 7*3600)).Format("2006010215")
	n := 0
	for _, t := range targets {
		_, queued, err := mail.Enqueue(ctx, tx, mail.Message{To: t.Email, Template: MailReminder, Payload: map[string]any{"course_id": courseID},
			DedupeKey: "ticket_reminder:" + t.ID.String() + ":" + courseID.String() + ":" + hour})
		if err != nil {
			return n, fmt.Errorf("escalation: xếp thư nhắc: %w", err)
		}
		if queued {
			n++
		}
	}
	return n, nil
}
