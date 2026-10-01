// Package sse: hạ tầng Server-Sent Events (SRS 4.5, 6.8). File này do lead viết trước để các story khác (jobs, route thử)
// phát sự kiện qua `Publisher`; agent US-PG-05 bổ sung handler `GET /api/v1/events`, giới hạn kết nối, đọc bù và test.
package sse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/platform/redis"
)

// MaxDataBytes là kích thước tối đa của `data` một sự kiện (64 KiB).
const MaxDataBytes = 64 << 10

var typeRE = regexp.MustCompile(`^[a-z][a-z0-9_.]{0,63}$`)

// ErrInvalidType / ErrDataTooLarge: lỗi do người gọi (map sang 422 ở route thử).
var (
	ErrInvalidType  = errors.New("sse: type sai định dạng")
	ErrDataTooLarge = errors.New("sse: data vượt 64 KiB")
)

// BufKey dựng khoá Redis của một người dùng (SRS 5.6).
func BufKey(userID string) string { return redis.Key("sse", "buf", userID) }

// ChanKey là kênh Pub/Sub "có sự kiện mới" của một người dùng.
func ChanKey(userID string) string { return redis.Key("sse", "ch", userID) }

// ConnKey là ZSET đếm kết nối SSE của một người dùng.
func ConnKey(userID string) string { return redis.Key("sse", "conn", userID) }

// Publisher phát một sự kiện tới stream của một người dùng. Dùng được từ gateway lẫn worker.
type Publisher interface {
	// Publish ghi sự kiện vào bộ đệm Redis (id = id Redis Stream, tăng nghiêm ngặt) rồi báo qua Pub/Sub; trả id.
	Publish(ctx context.Context, userID, eventType string, data any) (string, error)
}

// NewPublisher dựng Publisher; maxLen = SSE_BUFFER_MAXLEN, ttl = SSE_BUFFER_TTL.
func NewPublisher(rdb *redis.Client, maxLen int64, ttl time.Duration) Publisher {
	return &publisher{rdb: rdb, maxLen: maxLen, ttl: ttl}
}

type publisher struct {
	rdb    *redis.Client
	maxLen int64
	ttl    time.Duration
}

func (p *publisher) Publish(ctx context.Context, userID, eventType string, data any) (string, error) {
	if !typeRE.MatchString(eventType) {
		return "", ErrInvalidType
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("sse publish: marshal data: %w", err)
	}
	if len(raw) > MaxDataBytes {
		return "", ErrDataTooLarge
	}
	id, err := p.rdb.XAdd(ctx, &goredis.XAddArgs{Stream: BufKey(userID), MaxLen: p.maxLen, Approx: true, Values: map[string]any{"type": eventType, "data": string(raw)}}).Result()
	if err != nil {
		return "", fmt.Errorf("sse publish: xadd: %w", err)
	}
	if err := p.rdb.PExpire(ctx, BufKey(userID), p.ttl).Err(); err != nil {
		return "", fmt.Errorf("sse publish: pexpire: %w", err)
	}
	if err := p.rdb.Publish(ctx, ChanKey(userID), id).Err(); err != nil {
		return "", fmt.Errorf("sse publish: publish: %w", err)
	}
	return id, nil
}
