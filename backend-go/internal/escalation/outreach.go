package escalation

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Giới hạn của `Nhắn sinh viên` (SRS FEAT-crm-360 4.8.7): thân 1–2000 ký tự, mặc định 3 tin / ngày tới một sinh viên (OUTREACH_PER_DAY).
const (
	OutreachMaxChars  = 2000
	DefaultOutreachPD = 3
)

// OutreachIn là đầu vào của CreateOutreach. PerDay 0 = mặc định 3.
type OutreachIn struct {
	CourseID, StudentID, SenderID uuid.UUID
	Body                          string
	PerDay                        int
	Now                           time.Time // mốc "hôm nay" theo giờ Asia/Ho_Chi_Minh; zero = đồng hồ của Service
}

var ictOutreach = time.FixedZone("Asia/Ho_Chi_Minh", 7*3600) //nolint:gochecknoglobals // múi giờ cố định, bất biến

// CreateOutreach (SRS escalation 4.2.7) chạy TRONG giao dịch của người gọi: kiểm sinh viên ACTIVE (409 STUDENT_NOT_ACTIVE), thân tin (422), định danh sinh viên KHÁC (422 PII_IN_ANSWER),
// trần tin / ngày (429 OUTREACH_LIMIT), rồi ticket TEACHER_OUTREACH đã ANSWERED + tin STAFF_ANSWER trong phiên "Tin nhắn từ giảng viên" + thư `outreach_message` + outbox `escalation.answered`.
// Trần ticket mở (5) KHÔNG áp. Trả ticket và tin đã chèn.
func (s *Service) CreateOutreach(ctx context.Context, tx pgx.Tx, in OutreachIn) (store.EscalationTicket, store.ChatMessage, error) {
	body := strings.TrimSpace(in.Body)
	if n := utf8.RuneCountInString(body); n == 0 || n > OutreachMaxChars {
		return store.EscalationTicket{}, store.ChatMessage{}, apierr.Validation(apierr.FieldError{Field: "body", Code: "length", Message: fmt.Sprintf("Tin nhắn dài từ 1 đến %d ký tự.", OutreachMaxChars)})
	}
	q := store.New(tx)
	active, err := q.OutreachStudentActive(ctx, store.OutreachStudentActiveParams{CourseID: in.CourseID, StudentID: in.StudentID})
	if err != nil {
		return store.EscalationTicket{}, store.ChatMessage{}, unavailable()
	}
	if !active {
		return store.EscalationTicket{}, store.ChatMessage{}, apierr.New(http.StatusConflict, apierr.StudentNotActive)
	}
	if err := s.checkAnswerPII(ctx, in.CourseID, in.StudentID, body); err != nil {
		return store.EscalationTicket{}, store.ChatMessage{}, err
	}
	if err := q.OutreachLock(ctx, store.OutreachLockParams{CourseID: in.CourseID.String(), StudentID: in.StudentID.String()}); err != nil {
		return store.EscalationTicket{}, store.ChatMessage{}, unavailable()
	}
	now := in.Now
	if now.IsZero() {
		now = s.now()
	}
	l := now.In(ictOutreach)
	dayStart := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, ictOutreach)
	perDay := in.PerDay
	if perDay <= 0 {
		perDay = DefaultOutreachPD
	}
	n, err := q.OutreachCountSince(ctx, store.OutreachCountSinceParams{CourseID: in.CourseID, StudentID: in.StudentID, Since: dayStart})
	if err != nil {
		return store.EscalationTicket{}, store.ChatMessage{}, unavailable()
	}
	if int(n) >= perDay {
		return store.EscalationTicket{}, store.ChatMessage{}, apierr.New(http.StatusTooManyRequests, apierr.OutreachLimit).WithRetryAfter(max(1, int(dayStart.AddDate(0, 0, 1).Sub(now).Seconds())))
	}
	sess, err := ensureStaffSession(ctx, q, in.CourseID, in.StudentID)
	if err != nil {
		return store.EscalationTicket{}, store.ChatMessage{}, err
	}
	t, err := q.TicketInsertOutreach(ctx, store.TicketInsertOutreachParams{CourseID: in.CourseID, StudentID: in.StudentID, SessionID: &sess, Sender: &in.SenderID, Body: &body})
	if err != nil {
		return store.EscalationTicket{}, store.ChatMessage{}, unavailable()
	}
	brief, err := q.TicketUserBrief(ctx, store.TicketUserBriefParams{CourseID: in.CourseID, UserID: in.SenderID})
	if err != nil {
		return store.EscalationTicket{}, store.ChatMessage{}, unavailable()
	}
	msg, err := s.postStaffMessage(ctx, q, t, brief, body)
	if err != nil {
		return store.EscalationTicket{}, store.ChatMessage{}, err
	}
	if err := enqueueAnswerMail(ctx, tx, MailOutreach, t, msg.ID); err != nil {
		return store.EscalationTicket{}, store.ChatMessage{}, err
	}
	if err := s.audit(ctx, q, in.CourseID, in.SenderID, t.ID, "escalation_ticket.outreach", "", store.TicketStatusANSWERED); err != nil {
		return store.EscalationTicket{}, store.ChatMessage{}, err
	}
	if _, err := outbox.Write(ctx, tx, TopicAnswered, map[string]any{"ticket_id": t.ID, "course_id": in.CourseID, "answer_message_id": msg.ID, "user_id": t.StudentID}); err != nil {
		return store.EscalationTicket{}, store.ChatMessage{}, unavailable()
	}
	return t, msg, nil
}
