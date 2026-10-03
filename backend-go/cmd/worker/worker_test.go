package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/platform/config"
	appdb "github.com/edupilot/backend-go/internal/platform/db"
	applog "github.com/edupilot/backend-go/internal/platform/log"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// buf là bộ đệm log an toàn khi worker còn chạy.
type buf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *buf) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *buf) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

// fakeTask là việc nền thử, đại diện cho relay/consumer outbox sẽ cắm vào qua newTasks.
type fakeTask struct {
	started atomic.Bool
	stopped atomic.Bool
}

func (f *fakeTask) Name() string { return "fake" }

func (f *fakeTask) Run(ctx context.Context) error {
	f.started.Store(true)
	<-ctx.Done()
	f.stopped.Store(true)
	return ctx.Err()
}

func workerDeps(t *testing.T) (Deps, *buf) {
	t.Helper()
	out := &buf{}
	cfg := config.Config{
		Role:             config.Worker,
		AppEnv:           "test",
		InstanceID:       "wk-test",
		LogLevel:         "debug",
		DatabaseURL:      testutil.PostgresURL(t),
		RedisURL:         testutil.RedisURL(t),
		DBMaxConns:       2,
		DBSlowQueryMS:    200,
		StartupTimeout:   5 * time.Second,
		WorkerHealthAddr: "127.0.0.1:0",
	}
	log := applog.NewTo(out, cfg, "worker")
	pool, err := appdb.NewPool(t.Context(), cfg, log)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(pool.Close)
	rdb, err := appredis.New(t.Context(), cfg.RedisURL)
	if err != nil {
		t.Fatalf("redis.New: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	return Deps{Cfg: cfg, Log: log, DB: pool, Redis: rdb}, out
}

func startWorker(t *testing.T, d Deps, tasks ...Task) (addr string, cancel context.CancelFunc, done chan int) {
	t.Helper()
	ln, err := net.Listen("tcp", d.Cfg.WorkerHealthAddr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done = make(chan int, 1)
	go func() { done <- runLoop(ctx, d, ln, tasks) }()
	return ln.Addr().String(), cancel, done
}

// TestWorker_Health: `/healthz` nội bộ trả 200 khi vòng lặp còn nhịp và DB + Redis ping được;
// `worker -healthcheck` thoát 0; DB chết → 503 (US-PG-01 AC12).
func TestWorker_Health(t *testing.T) {
	t.Parallel()
	d, _ := workerDeps(t)
	task := &fakeTask{}
	addr, cancel, done := startWorker(t, d, task)
	defer func() { cancel(); <-done }()

	var body map[string]any
	code := 0
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/healthz") //nolint:noctx // test cục bộ
		if err == nil {
			code = resp.StatusCode
			_ = json.NewDecoder(resp.Body).Decode(&body)
			_ = resp.Body.Close()
			if code == http.StatusOK {
				break
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	if code != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("/healthz = %d %v", code, body)
	}
	if !task.started.Load() {
		t.Fatal("việc nền (điểm cắm outbox) không được khởi chạy")
	}

	// `worker -healthcheck` dùng chính endpoint này.
	env := map[string]string{
		"DATABASE_URL":       d.Cfg.DatabaseURL,
		"REDIS_URL":          d.Cfg.RedisURL,
		"WORKER_HEALTH_ADDR": addr,
	}
	getenv := func(k string) string { return env[k] }
	if rc := run([]string{"-healthcheck"}, getenv, &buf{}, &buf{}); rc != 0 {
		t.Fatalf("worker -healthcheck = %d, muốn 0", rc)
	}

	// Phụ thuộc chết → 503 (không phải 200), tiến trình vẫn chạy.
	d.DB.Close()
	resp, err := http.Get("http://" + addr + "/healthz") //nolint:noctx // test cục bộ
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("/healthz khi DB chết = %d, muốn 503", resp.StatusCode)
	}
	if rc := run([]string{"-healthcheck"}, getenv, &buf{}, &buf{}); rc != 1 {
		t.Fatalf("worker -healthcheck khi DB chết = %d, muốn 1", rc)
	}
}

// TestWorker_Shutdown: SIGTERM (ctx huỷ) → việc nền dừng, vòng lặp thoát mã 0 trong ≤ 10 s (AC12).
func TestWorker_Shutdown(t *testing.T) {
	t.Parallel()
	d, out := workerDeps(t)
	task := &fakeTask{}
	addr, cancel, done := startWorker(t, d, task)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) && !task.started.Load() {
		time.Sleep(50 * time.Millisecond)
	}
	if !task.started.Load() {
		t.Fatal("việc nền chưa chạy")
	}

	start := time.Now()
	cancel()
	var rc int
	select {
	case rc = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("worker không thoát trong 10 s")
	}
	elapsed := time.Since(start)

	if rc != 0 {
		t.Fatalf("mã thoát = %d, muốn 0", rc)
	}
	if elapsed > 10*time.Second {
		t.Fatalf("tắt mất %v, muốn ≤ 10 s", elapsed)
	}
	if !task.stopped.Load() {
		t.Fatal("việc nền không dừng theo ctx")
	}
	if _, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err == nil {
		t.Error("cổng health vẫn mở sau khi tắt")
	}

	logs := out.String()
	if !strings.Contains(logs, `"msg":"worker stopped"`) {
		t.Fatalf("thiếu dòng log lúc dừng: %s", logs)
	}
	// Mọi dòng log (kể cả lúc dừng) phải có trace_id 32 hex khác 0 (AC4).
	for _, line := range strings.Split(strings.TrimSpace(logs), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("dòng log không phải JSON: %s", line)
		}
		id, _ := m["trace_id"].(string)
		if len(id) != 32 || id == strings.Repeat("0", 32) || m["service"] != "worker" || m["instance"] == nil {
			t.Fatalf("dòng log thiếu trường: %s", line)
		}
	}
}

// TestWorker_MissingEnv: thiếu biến bắt buộc → đúng một dòng JSON mức error có `missing`, thoát 1 (AC1).
func TestWorker_MissingEnv(t *testing.T) {
	t.Parallel()
	out := &buf{}
	rc := run(nil, func(string) string { return "" }, out, out)
	if rc != 1 {
		t.Fatalf("rc = %d, muốn 1", rc)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("số dòng stdout = %d, muốn 1: %s", len(lines), out.String())
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &m); err != nil {
		t.Fatalf("không phải JSON: %s", lines[0])
	}
	missing, _ := m["missing"].([]any)
	if m["level"] != "ERROR" || len(missing) != 2 {
		t.Fatalf("dòng log = %v, muốn error + missing [DATABASE_URL REDIS_URL]", m)
	}
}
