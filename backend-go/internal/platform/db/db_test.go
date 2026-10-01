package db_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/db"
	applog "github.com/edupilot/backend-go/internal/platform/log"
	"github.com/edupilot/backend-go/internal/testutil"
)

// safeBuf cho phép test đọc log trong khi pool còn ghi (go test -race).
type safeBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuf) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuf) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func cfgFor(t *testing.T, maxConns int32, slowMS int) (config.Config, *slog.Logger, *safeBuf) {
	t.Helper()
	buf := &safeBuf{}
	cfg := config.Config{
		Role:          config.Gateway,
		DatabaseURL:   testutil.PostgresURL(t),
		DBMaxConns:    maxConns,
		DBSlowQueryMS: slowMS,
		LogLevel:      "debug",
		InstanceID:    "db-test",
	}
	return cfg, applog.NewTo(buf, cfg, "gateway"), buf
}

// TestPool_MaxConns: pool không bao giờ mở quá DB_MAX_CONNS kết nối, đo bằng pg_stat_activity
// trên kết nối trực tiếp với application_name = edupilot-gateway (US-PG-01 AC10).
func TestPool_MaxConns(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	cfg, log, _ := cfgFor(t, 3, 200)

	pool, err := db.NewPool(ctx, cfg, log)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Kết nối quan sát riêng (ngoài pool) để đếm kết nối của pool.
	watchCfg := cfg
	watchCfg.DBMaxConns = 2
	watchCfg.Role = config.Worker // application_name khác để không tự đếm mình
	watch, err := db.NewPool(ctx, watchCfg, log)
	if err != nil {
		t.Fatalf("NewPool (quan sát): %v", err)
	}
	t.Cleanup(watch.Close)

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var n int
			_ = pool.QueryRow(ctx, "-- name: TestSleep :one\nselect pg_sleep(1), 1").Scan(new(any), &n)
		}()
	}

	var maxSeen int
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		if err := watch.QueryRow(ctx,
			"select count(*) from pg_stat_activity where application_name = 'edupilot-gateway' and datname = current_database()").Scan(&n); err == nil {
			maxSeen = max(maxSeen, n)
		}
		time.Sleep(100 * time.Millisecond)
	}
	wg.Wait()

	if maxSeen == 0 {
		t.Fatal("không quan sát được kết nối nào của pool (phép đo rỗng)")
	}
	if maxSeen > int(cfg.DBMaxConns) {
		t.Fatalf("số kết nối lớn nhất = %d, vượt DB_MAX_CONNS = %d", maxSeen, cfg.DBMaxConns)
	}
}

// TestPool_AcquireHonorsDeadline: request chờ lấy kết nối vẫn tuân deadline — hết hạn thì lỗi ngay,
// không treo (US-PG-01 AC10).
func TestPool_AcquireHonorsDeadline(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	cfg, log, _ := cfgFor(t, 1, 200)

	pool, err := db.NewPool(ctx, cfg, log)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(pool.Close)

	busy, err := pool.Acquire(ctx) // giữ kết nối duy nhất
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer busy.Release()

	start := time.Now()
	short, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	_, err = pool.Exec(short, "select 1")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("muốn lỗi deadline khi pool đã hết kết nối")
	}
	if !errorsIsDeadline(err) {
		t.Fatalf("lỗi = %v, muốn deadline exceeded", err)
	}
	if elapsed > time.Second {
		t.Fatalf("chờ %v, muốn nhả ngay khi hết deadline", elapsed)
	}
}

func errorsIsDeadline(err error) bool {
	return strings.Contains(err.Error(), context.DeadlineExceeded.Error())
}

// TestSlowQueryLog: truy vấn ≥ DB_SLOW_QUERY_MS sinh ĐÚNG một dòng warn `slow query` có query_name,
// duration_ms, trace_id; truy vấn nhanh không sinh dòng nào (US-PG-01 AC11).
func TestSlowQueryLog(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	cfg, log, buf := cfgFor(t, 2, 200)

	pool, err := db.NewPool(ctx, cfg, log)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "-- name: TestFast :exec\nselect pg_sleep(0.05)"); err != nil {
		t.Fatalf("truy vấn nhanh: %v", err)
	}
	if n := countSlow(t, buf.String()); n != 0 {
		t.Fatalf("truy vấn 50 ms sinh %d dòng slow query", n)
	}

	if _, err := pool.Exec(ctx, "-- name: TestSlowSleep :exec\nselect pg_sleep(0.25)"); err != nil {
		t.Fatalf("truy vấn chậm: %v", err)
	}
	lines := slowLines(t, buf.String())
	if len(lines) != 1 {
		t.Fatalf("số dòng slow query = %d, muốn 1: %s", len(lines), buf.String())
	}
	l := lines[0]
	if l["level"] != "WARN" {
		t.Errorf("level = %v, muốn WARN", l["level"])
	}
	if l["query_name"] != "TestSlowSleep" {
		t.Errorf("query_name = %v, muốn TestSlowSleep", l["query_name"])
	}
	ms, ok := l["duration_ms"].(float64)
	if !ok || ms < 200 {
		t.Errorf("duration_ms = %v, muốn số ≥ 200", l["duration_ms"])
	}
	if id, _ := l["trace_id"].(string); len(id) != 32 {
		t.Errorf("trace_id = %v, muốn 32 hex", l["trace_id"])
	}
}

// TestSlowQueryLog_NoArgs: dòng `slow query` không chứa giá trị tham số (US-PG-01 AC11).
func TestSlowQueryLog_NoArgs(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	cfg, log, buf := cfgFor(t, 2, 200)

	pool, err := db.NewPool(ctx, cfg, log)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(pool.Close)

	const secret = "0.37"
	if _, err := pool.Exec(ctx, "-- name: TestSleepArg :exec\nselect pg_sleep($1::float)", secret); err != nil {
		t.Fatalf("truy vấn: %v", err)
	}
	lines := slowLines(t, buf.String())
	if len(lines) != 1 {
		t.Fatalf("số dòng slow query = %d, muốn 1: %s", len(lines), buf.String())
	}
	raw, _ := json.Marshal(lines[0])
	if strings.Contains(string(raw), secret) {
		t.Fatalf("dòng slow query chứa giá trị tham số %q: %s", secret, raw)
	}
	if lines[0]["query_name"] != "TestSleepArg" {
		t.Errorf("query_name = %v", lines[0]["query_name"])
	}
}

// TestQueryName: tên lấy từ chú thích sqlc, không có thì dùng từ khoá đầu — không bao giờ rỗng.
func TestQueryName(t *testing.T) {
	t.Parallel()
	cases := []struct{ sql, want string }{
		{"-- name: GetUser :one\nSELECT 1", "GetUser"},
		{"\n-- name: ListItems :many\nSELECT 1", "ListItems"},
		{"SELECT pg_sleep($1)", "select"},
		{"", "unnamed"},
	}
	for _, c := range cases {
		if got := db.QueryName(c.sql); got != c.want {
			t.Errorf("QueryName(%q) = %q, muốn %q", c.sql, got, c.want)
		}
	}
}

func slowLines(t *testing.T, raw string) []map[string]any {
	t.Helper()
	out := []map[string]any{}
	for _, s := range strings.Split(strings.TrimSpace(raw), "\n") {
		if s == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			t.Fatalf("dòng log không phải JSON: %s", s)
		}
		if m["msg"] == "slow query" {
			out = append(out, m)
		}
	}
	return out
}

func countSlow(t *testing.T, raw string) int {
	t.Helper()
	return len(slowLines(t, raw))
}
