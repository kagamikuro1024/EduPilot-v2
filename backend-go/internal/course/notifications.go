package course

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/store"
)

// Notification là một thông báo của chính người gọi. `Link` luôn là đường dẫn nội bộ (CHECK ở DB).
type Notification struct {
	ID        uuid.UUID
	Type      string
	Title     string
	Body      *string
	Link      *string
	CourseID  *uuid.UUID
	ReadAt    *time.Time
	CreatedAt time.Time
}

// NotificationPage là trang kèm tổng số chưa đọc của người gọi.
type NotificationPage struct {
	Rows        []Notification // tối đa limit+1 dòng
	UnreadCount int
}

// Notifications liệt kê thông báo CỦA userID (created_at DESC, id DESC) và đếm chưa đọc.
func (s Service) Notifications(ctx context.Context, userID uuid.UUID, unreadOnly bool, cur *Cursor, limit int) (NotificationPage, error) {
	q := store.New(s.Pool)
	p := store.ListNotificationsParams{UserID: userID, UnreadOnly: unreadOnly, Lim: int32(limit + 1)}
	if cur != nil {
		p.CurAt, p.CurID = &cur.At, &cur.ID
	}
	rows, err := q.ListNotifications(ctx, p)
	if err != nil {
		return NotificationPage{}, fmt.Errorf("course: liệt kê thông báo: %w", err)
	}
	unread, err := q.CountUnreadNotifications(ctx, userID)
	if err != nil {
		return NotificationPage{}, fmt.Errorf("course: đếm chưa đọc: %w", err)
	}
	out := NotificationPage{UnreadCount: int(unread), Rows: make([]Notification, len(rows))}
	for i, r := range rows {
		out.Rows[i] = Notification{ID: r.ID, Type: r.Type, Title: r.Title, Body: r.Body, Link: r.Link, CourseID: r.CourseID, ReadAt: r.ReadAt, CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// MarkRead đánh dấu đã đọc thông báo của chính mình; idempotent. Của người khác hoặc không có ⇒ ErrNotFound (404, không lộ tồn tại).
func (s Service) MarkRead(ctx context.Context, userID, id uuid.UUID) error {
	n, err := store.New(s.Pool).MarkNotificationRead(ctx, store.MarkNotificationReadParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("course: đánh dấu đã đọc: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
