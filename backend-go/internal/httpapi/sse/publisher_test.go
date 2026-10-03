package sse

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/platform/config"
)

// 05-AC2 — Publisher kiểm `type` (^[a-z][a-z0-9_.]{0,63}$) và `data` ≤ 64 KiB; sai thì KHÔNG có gì vào Stream.
func TestPublish_Validation(t *testing.T) {
	r := newRig(t, nil)
	uid := newUID()
	ctx := context.Background()

	xlen := func() int64 {
		n, err := r.rdb.XLen(ctx, BufKey(uid)).Result()
		if err != nil {
			t.Fatalf("XLEN: %v", err)
		}
		return n
	}

	bad := []struct {
		name string
		typ  string
	}{
		{"có khoảng trắng", "Bad Type"},
		{"viết hoa", "Type"},
		{"bắt đầu bằng số", "1abc"},
		{"rỗng", ""},
		{"65 ký tự", strings.Repeat("a", 65)},
		{"ký tự lạ", "test..x#"},
	}
	for _, c := range bad {
		before := xlen()
		if _, err := r.pub.Publish(ctx, uid, c.typ, map[string]int{"n": 1}); !errors.Is(err, ErrInvalidType) {
			t.Errorf("type %s: err = %v, muốn ErrInvalidType", c.name, err)
		}
		if after := xlen(); after != before {
			t.Errorf("type %s: XLEN %d → %d, muốn không đổi", c.name, before, after)
		}
	}

	for _, typ := range []string{"a", strings.Repeat("a", 64), "test.ok"} {
		id, err := r.pub.Publish(ctx, uid, typ, map[string]int{"n": 1})
		if err != nil {
			t.Fatalf("type %q: err = %v", typ, err)
		}
		if !idLineRE.MatchString(id) {
			t.Fatalf("id = %q, muốn <ms>-<seq>", id)
		}
	}

	// Biên 64 KiB tính theo JSON gọn: {"s":"<65528 a>"} = 65.536 byte.
	fit := map[string]string{"s": strings.Repeat("a", MaxDataBytes-8)}
	if _, err := r.pub.Publish(ctx, uid, "test.edge", fit); err != nil {
		t.Fatalf("data đúng 64 KiB phải được chấp nhận: %v", err)
	}
	before := xlen()
	tooBig := map[string]string{"s": strings.Repeat("a", MaxDataBytes-7)}
	if _, err := r.pub.Publish(ctx, uid, "test.edge", tooBig); !errors.Is(err, ErrDataTooLarge) {
		t.Fatalf("err = %v, muốn ErrDataTooLarge", err)
	}
	if after := xlen(); after != before {
		t.Fatalf("XLEN %d → %d, muốn không đổi khi data quá lớn", before, after)
	}

	// SRS 5.6: bộ đệm luôn có TTL (không khoá vĩnh viễn) và không quá 1 giờ.
	ttl, err := r.rdb.PTTL(ctx, BufKey(uid)).Result()
	if err != nil || ttl <= 0 || ttl > time.Hour {
		t.Fatalf("PTTL ep:sse:buf = %s (err %v), muốn trong (0, 1h]", ttl, err)
	}
}

// 05-AC3 / 05-AC6 / SRS 8.1 — mặc định của các biến SSE_*.
func TestConfig_SSEDefaults(t *testing.T) {
	t.Parallel()
	env := map[string]string{
		"DATABASE_URL":       "postgres://u:p@127.0.0.1:5432/db",
		"REDIS_URL":          "redis://127.0.0.1:6379/0",
		"JWT_SECRET_KEY":     testSecret,
		"BLOB_ENDPOINT":      "minio:9000",
		"BLOB_BUCKET":        "edupilot",
		"BLOB_ACCESS_KEY":    "ak-dev",
		"BLOB_SECRET_KEY":    "sk-dev",
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=",
	}
	cfg, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"SSE_HEARTBEAT", cfg.SSEHeartbeat, 25 * time.Second},
		{"SSE_MAX_DURATION", cfg.SSEMaxDuration, 120 * time.Second},
		{"SSE_MAX_PER_USER", cfg.SSEMaxPerUser, 2},
		{"SSE_CONN_TTL", cfg.SSEConnTTL, 150 * time.Second},
		{"SSE_BUFFER_MAXLEN", cfg.SSEBufferMaxLen, int64(1000)},
		{"SSE_BUFFER_TTL", cfg.SSEBufferTTL, time.Hour},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, muốn %v", c.name, c.got, c.want)
		}
	}
}
