package notify

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/store"
)

// Agg là một thông báo GỘP (SRS 4.4.3): một dòng chưa đọc cho mỗi (người nhận, lớp, loại); tiêu đề là số ĐẾM HIỆN TẠI, không đếm sự kiện.
type Agg struct {
	CourseID   uuid.UUID
	Type       string
	Recipients []uuid.UUID
	Link       string
	Body       *string // mã lớp (không nội dung riêng tư)
	// Title dựng tiêu đề từ số hiện tại; PHẢI bắt đầu bằng số ("3 câu hỏi …") — số cũ được đọc lại từ tiêu đề dòng hiện có.
	Title func(n int) string
	// Count đếm số hiện tại; chạy TRONG giao dịch dưới khoá advisory (hai handler song song cho số cuối đúng).
	Count func(ctx context.Context, q *store.Queries) (int, error)
	// DedupeKey mặc định `agg:<TYPE>:<course_id>`; đặt khi cần khoá khác (nhóm / cá nhân).
	DedupeKey func(user uuid.UUID) string
}

func leadingInt(title string) int {
	f, _, _ := strings.Cut(title, " ")
	n, err := strconv.Atoi(f)
	if err != nil {
		return -1
	}
	return n
}

// Aggregate cập nhật dòng gộp cho từng người nhận trong MỘT giao dịch:
// chưa có dòng và n > 0 → tạo; n tăng → nổi lên + chưa đọc + phát SSE; n bằng → không đổi, không phát; n giảm → đổi chữ, GIỮ read_at, không phát; n = 0 → đánh dấu đã đọc.
func (s *Service) Aggregate(ctx context.Context, a Agg) error {
	if len(a.Recipients) == 0 {
		return nil
	}
	dk := a.DedupeKey
	if dk == nil {
		base := "agg:" + a.Type + ":" + a.CourseID.String()
		dk = func(uuid.UUID) string { return base }
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("notify: mở giao dịch gộp: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if err := q.NotifyAggLock(ctx, store.NotifyAggLockParams{CourseID: a.CourseID.String(), Type: a.Type}); err != nil {
		return fmt.Errorf("notify: khoá gộp: %w", err)
	}
	n, err := a.Count(ctx, q)
	if err != nil {
		return fmt.Errorf("notify: đếm gộp: %w", err)
	}
	course := a.CourseID
	link := a.Link
	var announce []Row
	for _, u := range a.Recipients {
		k := dk(u)
		cur, err := q.NotifyAggGet(ctx, store.NotifyAggGetParams{UserID: u, DedupeKey: &k})
		missing := errors.Is(err, pgx.ErrNoRows)
		if err != nil && !missing {
			return fmt.Errorf("notify: đọc dòng gộp: %w", err)
		}
		sp := Spec{UserID: u, CourseID: &course, Type: a.Type, Body: a.Body, Link: &link, DedupeKey: k}
		switch {
		case n == 0:
			if !missing && cur.ReadAt == nil {
				if err := q.NotifyAggMarkRead(ctx, cur.ID); err != nil {
					return fmt.Errorf("notify: đánh dấu đã đọc: %w", err)
				}
			}
		case missing:
			sp.Title = a.Title(n)
			r, err := q.NotifyAggInsert(ctx, store.NotifyAggInsertParams{UserID: u, CourseID: &course, Type: a.Type, Title: sp.Title, Body: a.Body, Link: &link, DedupeKey: &k})
			if err != nil {
				return fmt.Errorf("notify: tạo dòng gộp: %w", err)
			}
			announce = append(announce, Row{ID: r.ID, UserID: u, Spec: sp, CreatedAt: r.CreatedAt})
		case n > leadingInt(cur.Title):
			sp.Title = a.Title(n)
			r, err := q.NotifyAggRaise(ctx, store.NotifyAggRaiseParams{ID: cur.ID, Title: sp.Title})
			if err != nil {
				return fmt.Errorf("notify: nâng dòng gộp: %w", err)
			}
			announce = append(announce, Row{ID: r.ID, UserID: u, Spec: sp, CreatedAt: r.CreatedAt})
		case n < leadingInt(cur.Title):
			if err := q.NotifyAggRetitle(ctx, store.NotifyAggRetitleParams{ID: cur.ID, Title: a.Title(n)}); err != nil {
				return fmt.Errorf("notify: đổi chữ dòng gộp: %w", err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("notify: commit gộp: %w", err)
	}
	s.Announce(ctx, announce...)
	return nil
}
