// Package notify là MỘT CỬA tạo thông báo (SRS FEAT-escalation-notify 4.4): ghi `notifications` rồi đẩy khung SSE `notification` tới chuông qua `GET /api/v1/events`
// (Redis Pub/Sub nên chạy đúng với nhiều bản gateway). Ngoài gói này không ai `insert into notifications` (TestNotificationsOnlyViaNotify).
package notify

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/store"
)

// EventType là loại sự kiện SSE của chuông.
const EventType = "notification"

// Loại thông báo gộp (SRS 4.4.3) và cá nhân của P4.
const (
	TypeTicketNew            = "TICKET_NEW"
	TypeAIPending            = "AI_PENDING"
	TypePostReported         = "POST_REPORTED"
	TypeTicketStale          = "TICKET_STALE"
	TypeTicketAnswered       = "TICKET_ANSWERED"
	TypeCourseTicketsStalled = "COURSE_TICKETS_STALLED"
)

// Spec là một thông báo cho MỘT người. DedupeKey rỗng = không khử trùng.
type Spec struct {
	UserID   uuid.UUID
	CourseID *uuid.UUID
	Type     string
	Title    string
	Body     *string
	Link     *string
	// DedupeKey: với Send là khoá đầy đủ; với SendMany là TIỀN TỐ (khoá từng người = `<tiền tố>:<user_id>`).
	DedupeKey string
}

// Frame là khung SSE `notification` (≤ 64 KiB).
type Frame struct {
	ID          uuid.UUID  `json:"id"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Body        *string    `json:"body"`
	Link        *string    `json:"link"`
	CourseID    *uuid.UUID `json:"course_id"`
	Read        bool       `json:"read"`
	CreatedAt   time.Time  `json:"created_at"`
	UnreadCount int32      `json:"unread_count"`
}

// Row là một dòng vừa ghi, chờ `Announce` sau khi giao dịch của người gọi commit.
type Row struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Spec      Spec
	CreatedAt time.Time
}

// Service ghi và phát thông báo. Pub nil → chỉ ghi (chuông rơi về làm mới 30 s).
type Service struct {
	Pool *pgxpool.Pool
	Pub  sse.Publisher
	Log  *slog.Logger

	// PublishErrors đếm lỗi phát SSE (`notify_publish_errors_total`).
	PublishErrors atomic.Int64
}

// New dựng Service.
func New(pool *pgxpool.Pool, pub sse.Publisher, log *slog.Logger) *Service {
	return &Service{Pool: pool, Pub: pub, Log: log}
}

func (s *Service) fail(ctx context.Context, err error) {
	s.PublishErrors.Add(1)
	if s.Log != nil {
		s.Log.WarnContext(ctx, "notify_publish_errors_total", "error", err.Error())
	}
}

func key(k string) *string {
	if k == "" {
		return nil
	}
	return &k
}

// SendTx ghi MỘT dòng trong giao dịch của người gọi (chưa phát SSE: gọi Announce sau commit). created=false khi trùng `DedupeKey`.
func SendTx(ctx context.Context, q *store.Queries, sp Spec) (Row, bool, error) {
	r, err := q.NotifyInsert(ctx, store.NotifyInsertParams{UserID: sp.UserID, CourseID: sp.CourseID, Type: sp.Type, Title: sp.Title, Body: sp.Body, Link: sp.Link, DedupeKey: key(sp.DedupeKey)})
	if errors.Is(err, pgx.ErrNoRows) {
		return Row{}, false, nil
	}
	if err != nil {
		return Row{}, false, fmt.Errorf("notify: ghi thông báo: %w", err)
	}
	return Row{ID: r.ID, UserID: sp.UserID, Spec: sp, CreatedAt: r.CreatedAt}, true, nil
}

// Send ghi (bằng pool) rồi phát SSE khi dòng MỚI. Lỗi phát chỉ log + đếm, trả nil.
func (s *Service) Send(ctx context.Context, sp Spec) (created bool, err error) {
	row, created, err := SendTx(ctx, store.New(s.Pool), sp)
	if err != nil || !created {
		return false, err
	}
	s.Announce(ctx, row)
	return true, nil
}

// SendMany ghi cho cả danh sách bằng MỘT câu INSERT … SELECT unnest rồi phát SSE theo từng dòng mới. `sp.UserID` bị bỏ qua; `sp.DedupeKey` là tiền tố.
func (s *Service) SendMany(ctx context.Context, userIDs []uuid.UUID, sp Spec) (int, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}
	ids := make([]string, len(userIDs))
	for i, u := range userIDs {
		ids[i] = u.String()
	}
	rows, err := store.New(s.Pool).NotifyInsertMany(ctx, store.NotifyInsertManyParams{CourseID: sp.CourseID, Type: sp.Type, Title: sp.Title, Body: sp.Body, Link: sp.Link, DedupePrefix: sp.DedupeKey, UserIds: ids})
	if err != nil {
		return 0, fmt.Errorf("notify: ghi hàng loạt: %w", err)
	}
	out := make([]Row, len(rows))
	for i, r := range rows {
		one := sp
		one.UserID = r.UserID
		out[i] = Row{ID: r.ID, UserID: r.UserID, Spec: one, CreatedAt: r.CreatedAt}
	}
	s.Announce(ctx, out...)
	return len(rows), nil
}

// Announce phát khung SSE cho các dòng ĐÃ commit (kèm số chưa đọc). Lỗi chỉ log + đếm.
func (s *Service) Announce(ctx context.Context, rows ...Row) {
	if s == nil || s.Pub == nil || len(rows) == 0 {
		return
	}
	uniq := map[uuid.UUID]bool{}
	var ids []string
	for _, r := range rows {
		if !uniq[r.UserID] {
			uniq[r.UserID] = true
			ids = append(ids, r.UserID.String())
		}
	}
	counts := map[uuid.UUID]int32{}
	cs, err := store.New(s.Pool).NotifyUnreadCounts(ctx, ids)
	if err != nil {
		s.fail(ctx, err) // vẫn phát, số chưa đọc = 0 → khách hàng làm mới
	}
	for _, c := range cs {
		counts[c.UserID] = c.N
	}
	for _, r := range rows {
		f := Frame{ID: r.ID, Type: r.Spec.Type, Title: r.Spec.Title, Body: r.Spec.Body, Link: r.Spec.Link, CourseID: r.Spec.CourseID, CreatedAt: r.CreatedAt, UnreadCount: counts[r.UserID]}
		if _, err := s.Pub.Publish(ctx, r.UserID.String(), EventType, f); err != nil {
			s.fail(ctx, err)
		}
	}
}

// MarkReadByLink đánh dấu đã đọc mọi thông báo chưa đọc của CHÍNH `user` có `link` trùng; trả số dòng đổi.
func (s *Service) MarkReadByLink(ctx context.Context, user uuid.UUID, link string) (int64, error) {
	n, err := store.New(s.Pool).NotifyMarkReadByLink(ctx, store.NotifyMarkReadByLinkParams{UserID: user, Link: &link})
	if err != nil {
		return 0, fmt.Errorf("notify: đánh dấu đã đọc: %w", err)
	}
	return n, nil
}
