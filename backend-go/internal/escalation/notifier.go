package escalation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Notifier là các handler outbox của vòng đời ticket ở worker (SRS 4.11). Mỗi mắt xích idempotent; thông báo cho nhân viên chỉ có SỐ ĐẾM và mã lớp.
type Notifier struct {
	Pool   *pgxpool.Pool
	Notify *notify.Service
	Log    *slog.Logger
	Gate   MailGate // thư `ticket_new`: cửa sổ trượt (Redis nil = không gộp)
	Clock  clock.Clock
}

func (n *Notifier) now() time.Time {
	if n.Clock != nil {
		return n.Clock.Now()
	}
	return time.Now().UTC()
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
	if err := n.RecountTicketNew(ctx, p.CourseID); err != nil {
		return err
	}
	return n.mailTicketNew(ctx, m.ID, p.CourseID)
}

// mailTicketNew xếp thư `ticket_new` cho TA / TEACHER ĐÃ bật `notify_ticket_by_mail` (mặc định tắt), gộp theo cửa sổ trượt mỗi (người, lớp).
// Giao lại tin outbox: `dedupe_key = ticket_new:<outbox_message_id>:<user_id>` chặn thư thứ hai; thư không chứa nội dung (chỉ số lượng lúc gửi + liên kết).
func (n *Notifier) mailTicketNew(ctx context.Context, outboxID, courseID uuid.UUID) error {
	targets, err := store.New(n.Pool).TicketStaffMailTargets(ctx, courseID)
	if err != nil {
		return fmt.Errorf("escalation: người nhận thư: %w", err)
	}
	for _, t := range targets {
		key := "ticket_new:" + outboxID.String() + ":" + t.ID.String()
		if done, err := n.mailQueued(ctx, key); err != nil {
			return err
		} else if done {
			continue // giao lại: đã xếp thư cho người này
		}
		if !n.Gate.Allow(ctx, t.ID, courseID) {
			continue
		}
		tx, err := n.Pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("escalation: mở giao dịch: %w", err)
		}
		_, _, err = mail.Enqueue(ctx, tx, mail.Message{To: t.Email, Template: MailNew, Payload: map[string]any{"course_id": courseID}, DedupeKey: key})
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("escalation: xếp thư: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("escalation: commit thư: %w", err)
		}
	}
	return nil
}

// mailQueued: đã có dòng mail_outbox với dedupe_key này chưa (giao lại outbox không được tiêu mốc cửa sổ gộp một lần nữa).
func (n *Notifier) mailQueued(ctx context.Context, key string) (bool, error) {
	done, err := store.New(n.Pool).TicketMailQueued(ctx, &key)
	if err != nil {
		return false, fmt.Errorf("escalation: kiểm thư đã xếp: %w", err)
	}
	return done, nil
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
	if err := n.Notify.Aggregate(ctx, notify.Agg{
		CourseID: courseID, Type: notify.TypeTicketNew, Recipients: staff, Link: "/inbox?course=" + courseID.String(), Body: &body,
		Title: func(k int) string { return fmt.Sprintf("%d câu hỏi đang chờ giảng viên", k) },
		Count: func(ctx context.Context, q *store.Queries) (int, error) {
			k, err := q.TicketCountOpenInCourse(ctx, courseID)
			return int(k), err
		},
	}); err != nil {
		return err
	}
	// lớp hết ticket quá hạn: mục "Hôm nay" của Admin (COURSE_TICKETS_STALLED chưa đọc) biến
	if k, err := q.TicketOpenAtStage(ctx, store.TicketOpenAtStageParams{CourseID: courseID, Stage: 2}); err == nil && k == 0 {
		if err := q.TicketMarkStalledRead(ctx, &courseID); err != nil {
			return fmt.Errorf("escalation: đóng thông báo quá hạn: %w", err)
		}
	}
	return nil
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
