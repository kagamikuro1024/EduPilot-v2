package redis_test

import (
	"context"
	"strings"
	"testing"
	"time"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// TestKeys: mọi khoá do mã sinh ra đi qua Key() nên luôn có tiền tố "ep:" (US-PG-01 AC6).
func TestKeys(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"idempotency", appredis.Key("idem", "u1", "ab12", "k1"), "ep:idem:u1:ab12:k1"},
		{"khoá đang chạy", appredis.Key("idem", "u1", "ab12", "k1", "lock"), "ep:idem:u1:ab12:k1:lock"},
		{"rate limit theo ip", appredis.Key("rl", "ip", "10.0.0.1", "29123456"), "ep:rl:ip:10.0.0.1:29123456"},
		{"rate limit theo người dùng", appredis.Key("rl", "user", "u1", "29123456"), "ep:rl:user:u1:29123456"},
		{"kết nối SSE", appredis.Key("sse", "conn", "u1"), "ep:sse:conn:u1"},
		{"bộ đệm SSE", appredis.Key("sse", "buf", "u1"), "ep:sse:buf:u1"},
		{"kênh SSE", appredis.Key("sse", "ch", "u1"), "ep:sse:ch:u1"},
		{"cache", appredis.Key("cache", "course", "c1"), "ep:cache:course:c1"},
		{"một phần", appredis.Key("x"), "ep:x"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: Key = %q, muốn %q", c.name, c.got, c.want)
		}
		if !strings.HasPrefix(c.got, appredis.Prefix) {
			t.Errorf("%s: khoá %q thiếu tiền tố %q", c.name, c.got, appredis.Prefix)
		}
	}
	// Ngoại lệ tên: Stream outbox KHÔNG mang tiền tố ep: (SRS 5.6).
	if strings.HasPrefix(appredis.StreamOutboxDispatch, appredis.Prefix) ||
		strings.HasPrefix(appredis.StreamOutboxDead, appredis.Prefix) {
		t.Errorf("Stream outbox không được mang tiền tố ep:")
	}
}

// TestTTLConstants: hằng TTL đúng SRS 5.6.
func TestTTLConstants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"idempotency 24 h", appredis.TTLIdempotency, 24 * time.Hour},
		{"khoá đang chạy 30 s", appredis.TTLIdempotencyLock, 30 * time.Second},
		{"rate limit 120 s", appredis.TTLRateLimit, 120 * time.Second},
		{"kết nối SSE 150 s", appredis.TTLSSEConn, 150 * time.Second},
		{"bộ đệm SSE 1 h", appredis.TTLSSEBuffer, time.Hour},
		{"cache 5 phút", appredis.TTLCache, 5 * time.Minute},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: %v, muốn %v", c.name, c.got, c.want)
		}
	}
}

// TestNew_SetWithTTL: client thật đặt được khoá kèm TTL và khoá không bao giờ ở trạng thái -1.
func TestNew_SetWithTTL(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c, err := appredis.New(ctx, testutil.RedisURL(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	key := appredis.Key("cache", testutil.TestPrefix(t))
	if err := c.Set(ctx, key, "v", appredis.TTLCache).Err(); err != nil {
		t.Fatalf("Set: %v", err)
	}
	ttl, err := c.TTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if ttl <= 0 || ttl > appredis.TTLCache {
		t.Fatalf("TTL = %v, muốn trong (0, %v]", ttl, appredis.TTLCache)
	}
}

func TestNew_InvalidURL(t *testing.T) {
	t.Parallel()
	if _, err := appredis.New(context.Background(), "không-phải-url"); err == nil {
		t.Fatal("muốn lỗi với REDIS_URL sai")
	}
}
