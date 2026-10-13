package escalation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Notifier là các handler outbox của vòng đời ticket ở worker (SRS 4.11). Mỗi mắt xích idempotent; thông báo cho nhân viên chỉ có SỐ ĐẾM và mã lớp.
type Notifier struct {
	Pool   *pgxpool.Pool
	Notify *notify.Service
	Log    *slog.Logger
}

type courseEvent struct {
	CourseID uuid.UUID `json:"course_id"`
}

// HandleCreated (`escalation.created`): đếm lại ticket OPEN của lớp và cập nhật dòng gộp TICKET_NEW của TA + TEACHER.
func (n *Notifier) HandleCreated(ctx context.Context, m outbox.Message) error {
	var p courseEvent
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.CourseID == uuid.Nil {
		return errors.New("payload escalation.created không hợp lệ")
	}
	return n.RecountTicketNew(ctx, p.CourseID)
}

// RecountTicketNew cập nhật dòng gộp TICKET_NEW theo số ticket OPEN HIỆN TẠI của lớp (không đếm theo sự kiện).
func (n *Notifier) RecountTicketNew(ctx context.Context, courseID uuid.UUID) error {
	q := store.New(n.Pool)
	c, err := q.GetCourseBasic(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("escalation: đọc lớp: %w", err)
	}
	staff, err := q.TicketStaffIDs(ctx, courseID)
	if err != nil {
		return fmt.Errorf("escalation: người nhận: %w", err)
	}
	body := c.ClassCode
	return n.Notify.Aggregate(ctx, notify.Agg{
		CourseID: courseID, Type: notify.TypeTicketNew, Recipients: staff, Link: "/inbox?course=" + courseID.String(), Body: &body,
		Title: func(k int) string { return fmt.Sprintf("%d câu hỏi đang chờ giảng viên", k) },
		Count: func(ctx context.Context, q *store.Queries) (int, error) {
			k, err := q.TicketCountOpenInCourse(ctx, courseID)
			return int(k), err
		},
	})
}

// HandleRecount (`escalation.claimed` / `released` / `closed` / `followup`): đếm lại ticket OPEN của lớp và cập nhật dòng gộp TICKET_NEW.
func (n *Notifier) HandleRecount(ctx context.Context, m outbox.Message) error {
	var p courseEvent
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.CourseID == uuid.Nil {
		return fmt.Errorf("payload %s không hợp lệ", m.Topic)
	}
	return n.RecountTicketNew(ctx, p.CourseID)
}

// HandleAnswered (`escalation.answered`): báo sinh viên chủ ticket (thông báo cá nhân, KHÔNG gộp; tiêu đề / thân không chứa nội dung) rồi đếm lại TICKET_NEW cho nhân viên.
// Tin trả lời đọc theo id trong payload để biết phiên chat đã nhận (phiên của ticket hoặc "Tin nhắn từ giảng viên").
func (n *Notifier) HandleAnswered(ctx context.Context, m outbox.Message) error {
	var p struct {
		CourseID        uuid.UUID `json:"course_id"`
		AnswerMessageID uuid.UUID `json:"answer_message_id"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.CourseID == uuid.Nil || p.AnswerMessageID == uuid.Nil {
		return errors.New("payload escalation.answered không hợp lệ")
	}
	q := store.New(n.Pool)
	msg, err := q.GetChatMessageByID(ctx, p.AnswerMessageID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("escalation: đọc tin trả lời: %w", err)
	}
	if err == nil {
		c, err := q.GetCourseBasic(ctx, p.CourseID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("escalation: đọc lớp: %w", err)
		}
		body := c.ClassCode
		link := "/chat?session=" + msg.SessionID.String()
		if _, err := n.Notify.Send(ctx, notify.Spec{UserID: msg.UserID, CourseID: &p.CourseID, Type: notify.TypeTicketAnswered, Title: "Giảng viên đã trả lời câu hỏi của bạn", Body: &body, Link: &link,
			DedupeKey: "escalation.answered:" + p.AnswerMessageID.String()}); err != nil {
			return fmt.Errorf("escalation: thông báo trả lời: %w", err)
		}
	}
	return n.RecountTicketNew(ctx, p.CourseID)
}
