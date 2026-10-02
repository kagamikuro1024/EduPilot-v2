package sse

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/testutil"
)

// wantLimited khẳng định một phản hồi là 429 SSE_LIMIT_REACHED đúng dạng (SRS 6.1, QC #Q-QC-05-5).
func wantLimited(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("mã HTTP = %d, muốn 429", resp.StatusCode)
	}
	if got := resp.Header.Get("Retry-After"); got != "5" {
		t.Errorf("Retry-After = %q, muốn 5", got)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, muốn application/json", ct)
	}
	if got := resp.Header.Get("X-Accel-Buffering"); got != "" {
		t.Errorf("429 không được mang header của stream (X-Accel-Buffering = %q)", got)
	}
	body := errBody(t, resp)
	if body["code"] != apierr.SSELimitReached {
		t.Errorf("code = %v, muốn %s", body["code"], apierr.SSELimitReached)
	}
	if ra, ok := body["retry_after"].(float64); !ok || ra != 5 {
		t.Errorf("retry_after = %v, muốn số nguyên 5", body["retry_after"])
	}
}

// 05-AC5 — tối đa 2 kết nối / người; stream thứ 3 bị 429 TRƯỚC khi mở stream; người khác không bị ảnh hưởng.
func TestSSE_MaxTwoPerUser(t *testing.T) {
	r := newRig(t, nil)
	u1, u2 := newUID(), newUID()
	tok1 := r.token(u1, time.Hour)

	s1 := r.connect(tok1, nil)
	s2 := r.connect(tok1, nil)
	waitSlots(t, r, u1, 2)

	resp, cancel := r.get(tok1, "", nil)
	wantLimited(t, resp)
	cancel()

	// Người dùng khác không bị ảnh hưởng.
	other := r.connect(r.token(u2, time.Hour), nil)
	other.close()

	// Đóng một stream → chỗ được trả, stream mới mở được.
	s1.close()
	waitSlots(t, r, u1, 1)
	s3 := r.connect(tok1, nil)
	s3.close()
	s2.close()
	waitSlots(t, r, u1, 0)
}

// 05-AC5 — giới hạn đếm CHUNG trên Redis giữa hai bản gateway: A giữ 2, B trả 429.
func TestSSE_LimitSharedAcrossInstances(t *testing.T) {
	a := newRig(t, nil)
	b := rigAt(t, testutil.RedisURL(t), nil)
	uid := newUID()
	tok := a.token(uid, time.Hour)

	sa1 := a.connect(tok, nil)
	sa2 := a.connect(tok, nil)
	waitSlots(t, a, uid, 2)

	resp, cancel := b.get(tok, "", nil)
	wantLimited(t, resp)
	cancel()

	sa1.close()
	waitSlots(t, a, uid, 1)
	sb := b.connect(tok, nil) // chỗ vừa trả dùng được ở bản B
	sb.close()
	sa2.close()
}

// 05-AC5 — chỗ của gateway chết (không kịp trả) tự hết sau SSE_CONN_TTL.
func TestSSE_SlotExpiresAfterCrash(t *testing.T) {
	const ttl = time.Second
	r := newRig(t, func(c *config.Config) { c.SSEConnTTL = ttl })
	uid := newUID()
	tok := r.token(uid, time.Hour)
	ctx := context.Background()

	// Hai chỗ do một bản gateway đã chết để lại: hạn sau 1 s, khoá thì còn lâu mới hết.
	expire := time.Now().Add(ttl).UnixMilli()
	if err := r.rdb.ZAdd(ctx, ConnKey(uid),
		goredis.Z{Score: float64(expire), Member: "crashed-1"},
		goredis.Z{Score: float64(expire), Member: "crashed-2"}).Err(); err != nil {
		t.Fatalf("ZADD: %v", err)
	}
	if err := r.rdb.Expire(ctx, ConnKey(uid), time.Minute).Err(); err != nil {
		t.Fatalf("EXPIRE: %v", err)
	}

	resp, cancel := r.get(tok, "", nil)
	wantLimited(t, resp)
	cancel()

	time.Sleep(ttl + 200*time.Millisecond)
	s := r.connect(tok, nil) // chỗ cũ đã hết hạn → mở được
	defer s.close()

	n, err := r.rdb.ZCard(ctx, ConnKey(uid)).Result()
	if err != nil || n != 1 {
		t.Fatalf("ZCARD = %d (err %v), muốn 1 (hai chỗ cũ bị dọn)", n, err)
	}
}

// 05-AC6 — stream tự kết thúc sau SSE_MAX_DURATION với `event: reconnect` {"reason":"max_duration"}.
func TestSSE_MaxDuration(t *testing.T) {
	r := newRig(t, func(c *config.Config) { c.SSEMaxDuration = 700 * time.Millisecond })
	uid := newUID()
	s := r.connect(r.token(uid, time.Hour), nil)

	start := time.Now()
	f := s.wantType(2*time.Second, evReconnect)
	if got := f.reason(t); got != reasonMaxDuration {
		t.Fatalf("reason = %q, muốn %q", got, reasonMaxDuration)
	}
	if f.id != "" {
		t.Fatalf("reconnect không được mang id, có %q", f.id)
	}
	if took := time.Since(start); took < 400*time.Millisecond {
		t.Fatalf("đóng sau %s, muốn ≈ SSE_MAX_DURATION", took)
	}
	s.eof(2 * time.Second)
	waitSlots(t, r, uid, 0)
}

// 05-AC6 — token hết hạn trước SSE_MAX_DURATION thì stream đóng với reason token_expired.
func TestSSE_TokenExpiryMidStream(t *testing.T) {
	r := newRig(t, nil) // SSE_MAX_DURATION = 30 s
	uid := newUID()
	s := r.connect(r.token(uid, 700*time.Millisecond), nil)

	f := s.wantType(3*time.Second, evReconnect)
	if got := f.reason(t); got != reasonTokenExpired {
		t.Fatalf("reason = %q, muốn %q", got, reasonTokenExpired)
	}
	s.eof(2 * time.Second)
	waitSlots(t, r, uid, 0)
}
