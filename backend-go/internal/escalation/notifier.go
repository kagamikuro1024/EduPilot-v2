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
