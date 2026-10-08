// Package redis: client Redis dùng chung và hàm dựng khoá duy nhất (SRS 5.6).
// Mọi khoá do mã sinh ra đi qua Key() nên luôn có tiền tố "ep:"; mọi khoá ngoài Stream đều có TTL.
package redis

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Prefix là tiền tố bắt buộc của mọi khoá EduPilot.
const Prefix = "ep:"

// TTL mặc định theo SRS 5.6. Khoá nào cũng phải đặt TTL — không khoá nào để -1.
const (
	TTLIdempotency     = 24 * time.Hour    // ep:idem:{uid}:{ep}:{key}
	TTLIdempotencyLock = 30 * time.Second  // ep:idem:…:lock
	TTLRateLimit       = 120 * time.Second // ep:rl:ip|user:…
	TTLSSEConn         = 150 * time.Second // ep:sse:conn:{uid}
	TTLSSEBuffer       = time.Hour         // ep:sse:buf:{uid}
	TTLCache           = 5 * time.Minute   // ep:cache:…
)

// Stream dùng tên riêng, KHÔNG mang tiền tố "ep:" (SYSTEM_DESIGN §3.3, SRS 5.6).
const (
	StreamOutboxDispatch = "outbox.dispatch"
	StreamOutboxDead     = "outbox.dispatch.dead"
)

// Client bọc go-redis để mọi gói dùng chung một kiểu.
type Client struct{ *goredis.Client }

// New dựng client từ URL `redis://host:port/db`. Không nối mạng ở đây: việc chờ Redis sẵn sàng
// do bước chờ phụ thuộc lúc khởi động lo (SRS 3.4).
func New(_ context.Context, url string) (*Client, error) {
	opt, err := goredis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("REDIS_URL không hợp lệ: %w", err)
	}
	// Mọi lệnh (kể cả lệnh chặn như BLPOP) phải huỷ theo ctx của request — SRS 4.1 FR-9, AC9.
	opt.ContextTimeoutEnabled = true
	return &Client{Client: goredis.NewClient(opt)}, nil
}

// Key dựng khoá có tiền tố "ep:" từ các phần, ví dụ Key("idem", uid, hash) → "ep:idem:<uid>:<hash>".
func Key(parts ...string) string { return Prefix + strings.Join(parts, ":") }

// SetLogger bắt go-redis ghi log qua slog JSON thay vì stderr dạng text — mọi dòng log của tiến
// trình phải là JSON có trace_id (US-PG-01 AC4). Gọi một lần lúc khởi động.
func SetLogger(l *slog.Logger) { goredis.SetLogger(slogLogger{l}) }

type slogLogger struct{ l *slog.Logger }

func (s slogLogger) Printf(ctx context.Context, format string, v ...any) {
	s.l.WarnContext(ctx, "redis client: "+fmt.Sprintf(format, v...))
}

// Leader giữ khoá `key` cho `name` (SET NX PX ttl) hoặc gia hạn nếu khoá đang là của chính `name`; true = người gọi là leader trong ttl tới.
// Các bộ lập lịch cùng một tiến trình dùng chung một `name` nên cùng nhận ra mình (SRS FEAT-weekly-exam 4.2.6).
func (c *Client) Leader(ctx context.Context, key, name string, ttl time.Duration) bool {
	ok, err := c.SetNX(ctx, key, name, ttl).Result()
	if err != nil {
		return false
	}
	if ok {
		return true
	}
	if v, err := c.Get(ctx, key).Result(); err == nil && v == name {
		_ = c.PExpire(ctx, key, ttl).Err()
		return true
	}
	return false
}
