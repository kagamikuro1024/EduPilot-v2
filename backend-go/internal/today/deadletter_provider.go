package today

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/store"
)

// DeadLetterWindow: dòng outbox chết trong chừng này còn được báo; quá thì mục tự hết (không có thao tác «đã xem»).
const DeadLetterWindow = 7 * 24 * time.Hour

// DeadLetterProvider: DEAD_LETTER (bậc 60, chỉ ADMIN) — MỘT mục khi có ≥ 1 dòng `outbox.dead_at` trong 7 ngày (gồm thư `mail.send` chết).
// Chỉ dùng topic, đếm và dead_at: KHÔNG đọc payload / last_error (không PII). Không href: chưa có trang dead-letter. Một truy vấn gộp; vô hiệu cache bằng TTL 60 giây.
type DeadLetterProvider struct {
	Pool *pgxpool.Pool
}

// Name implements Provider.
func (DeadLetterProvider) Name() string { return "dead_letter" }

// Items implements Provider.
func (p DeadLetterProvider) Items(ctx context.Context, v Viewer, _ Scope) ([]Item, error) {
	if v.Role != RoleAdmin {
		return nil, nil // không tốn truy vấn cho vai khác
	}
	rows, err := store.New(p.Pool).TodayDeadLetters(ctx, v.Now.Add(-DeadLetterWindow))
	if err != nil {
		return nil, fmt.Errorf("today: tin lỗi hẳn: %w", err)
	}
	total, top := 0, -1
	var latest time.Time
	for i, r := range rows { // rows theo topic tăng dần ⇒ hoà thì lấy topic đứng trước theo thứ tự chữ cái
		total += int(r.N)
		if top < 0 || r.N > rows[top].N {
			top = i
		}
		if r.Latest.After(latest) {
			latest = r.Latest
		}
	}
	if total == 0 {
		return nil, nil
	}
	age := v.Now.Sub(latest)
	return []Item{mk(Item{
		ID: "DEAD_LETTER", Kind: KindDeadLetter, Tier: TierDeadLetter, Title: fmt.Sprintf("Hàng đợi có %d tin lỗi hẳn", total),
		Reason: fmt.Sprintf("%s (%d) · mới nhất %s trước.", rows[top].Topic, rows[top].N, Age(age)), AgeMinutes: int(age / time.Minute),
	})}, nil
}
