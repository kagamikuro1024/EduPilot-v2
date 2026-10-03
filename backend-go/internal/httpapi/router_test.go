package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appdb "github.com/edupilot/backend-go/internal/platform/db"
	applog "github.com/edupilot/backend-go/internal/platform/log"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	coreOtelOnce sync.Once
	coreHex32    = regexp.MustCompile(`^[0-9a-f]{32}$`)
)

// coreOtel cài tracer provider + propagator W3C một lần cho cả gói test (như otel.Setup lúc chạy thật).
func coreOtel(t *testing.T) {
	t.Helper()
	coreOtelOnce.Do(func() {
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		otel.SetTextMapPropagator(propagation.TraceContext{})
	})
}

// coreBuf là bộ đệm log an toàn khi chạy song song với server.
type coreBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *coreBuf) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *coreBuf) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func coreConfig() config.Config {
	return config.Config{
		Role:            config.Gateway,
		AppEnv:          "test",
		HTTPAddr:        "127.0.0.1:0",
		InstanceID:      "gw-test",
		LogLevel:        "debug",
		DBMaxConns:      4,
		DBSlowQueryMS:   200,
		RequestTimeout:  30 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		StartupTimeout:  5 * time.Second,
		MaxBodyBytes:    1048576,
	}
}

// coreDeps dựng Deps không có DB/Redis (test không cần phụ thuộc thật).
func coreDeps(t *testing.T, mutate func(*config.Config)) (Deps, *coreBuf) {
	t.Helper()
	coreOtel(t)
	cfg := coreConfig()
	if mutate != nil {
		mutate(&cfg)
	}
	buf := &coreBuf{}
	return Deps{Cfg: cfg, Log: applog.NewTo(buf, cfg, "gateway"), Clock: clock.Real{}, State: NewState()}, buf
}

// coreDepsReal dựng Deps với Postgres + Redis thật (testcontainers).
func coreDepsReal(t *testing.T, mutate func(*config.Config)) (Deps, *coreBuf) {
	t.Helper()
	d, buf := coreDeps(t, mutate)
	d.Cfg.DatabaseURL = testutil.PostgresURL(t)
	d.Cfg.RedisURL = testutil.RedisURL(t)

	pool, err := appdb.NewPool(t.Context(), d.Cfg, d.Log)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(pool.Close)
	rdb, err := appredis.New(t.Context(), d.Cfg.RedisURL)
	if err != nil {
		t.Fatalf("redis.New: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	d.DB, d.Redis = pool, rdb
	return d, buf
}

func coreJSON(t *testing.T, r io.Reader) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		t.Fatalf("thân không phải JSON: %v", err)
	}
	return m
}

func coreLogLines(t *testing.T, raw string) []map[string]any {
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
		out = append(out, m)
	}
	return out
}

func TestHealthz(t *testing.T) {
	t.Parallel()
	d, _ := coreDeps(t, nil)
	srv := httptest.NewServer(NewRouter(d))
	t.Cleanup(srv.Close)

	for _, path := range []string{"/healthz", "/api/v1/healthz"} {
		resp, err := srv.Client().Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body := coreJSON(t, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || body["status"] != "ok" {
			t.Fatalf("%s → %d %v", path, resp.StatusCode, body)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("%s content-type = %q", path, ct)
		}
		if resp.Header.Get("X-Instance-Id") != "gw-test" {
			t.Errorf("%s thiếu X-Instance-Id", path)
		}
	}

	// Sai method → 405 theo định dạng lỗi chung.
	resp, err := srv.Client().Post(srv.URL+"/healthz", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /healthz: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST /healthz → %d", resp.StatusCode)
	}
	if code := coreJSON(t, resp.Body)["code"]; code != "METHOD_NOT_ALLOWED" {
		t.Fatalf("code = %v", code)
	}
}

// TestTraceID_InErrorBodyAndLogs: traceparent đến → thân lỗi và mọi dòng log của request mang đúng
// trace_id; không có traceparent → trace mới, X-Request-Id = trace_id (US-PG-01 AC5).
func TestTraceID_InErrorBodyAndLogs(t *testing.T) {
	t.Parallel()
	d, buf := coreDeps(t, nil)
	srv := httptest.NewServer(NewRouter(d))
	t.Cleanup(srv.Close)

	const incoming = "4bf92f3577b34da6a3ce929d0e0e4736"
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/api/v1/khong-co", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("traceparent", "00-"+incoming+"-00f067aa0ba902b7-01")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body := coreJSON(t, resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Fatalf("status = %d, body = %v", resp.StatusCode, body)
	}
	if body["trace_id"] != incoming {
		t.Fatalf("trace_id trong thân = %v, muốn %s", body["trace_id"], incoming)
	}
	if resp.Header.Get("X-Request-Id") == "" {
		t.Fatal("thiếu header X-Request-Id")
	}
	if msg, _ := body["message"].(string); msg == "" {
		t.Fatal("thiếu message tiếng Việt")
	}
	for k := range body {
		switch k {
		case "code", "message", "trace_id", "details", "retry_after":
		default:
			t.Errorf("thân lỗi có trường lạ %q (SRS 6.1)", k)
		}
	}

	var seen int
	for _, l := range coreLogLines(t, buf.String()) {
		if l["trace_id"] == incoming {
			seen++
		}
	}
	if seen == 0 {
		t.Fatalf("không dòng log nào mang trace_id của request: %s", buf.String())
	}

	// Không có traceparent → trace mới, X-Request-Id = trace_id.
	resp2, err := srv.Client().Get(srv.URL + "/api/v1/khong-co")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body2 := coreJSON(t, resp2.Body)
	_ = resp2.Body.Close()
	tid, _ := body2["trace_id"].(string)
	if !coreHex32.MatchString(tid) || tid == strings.Repeat("0", 32) || tid == incoming {
		t.Fatalf("trace_id tự sinh = %q", tid)
	}
	if rid := resp2.Header.Get("X-Request-Id"); rid != tid {
		t.Fatalf("X-Request-Id = %q, muốn = trace_id %q", rid, tid)
	}
}

// coreServer dựng server thật (đủ giới hạn của NewServer) với route tạm trong nhóm nghiệp vụ.
func coreServer(t *testing.T, d Deps, mount func(chi.Router)) string {
	t.Helper()
	srv := NewServer(d)
	srv.Handler = newRouterWith(d, mount)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})
	return ln.Addr().String()
}

func coreEcho(r chi.Router) {
	r.Post("/echo", func(w http.ResponseWriter, r *http.Request) {
		n, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"status": "lỗi đọc thân"})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]int64{"bytes": n})
	})
}

// TestServerLimits: thân lớn hơn MAX_BODY_BYTES → 413 PAYLOAD_TOO_LARGE kèm details.max_bytes;
// đúng giới hạn thì qua (US-PG-01 AC8).
func TestServerLimits(t *testing.T) {
	t.Parallel()
	d, _ := coreDeps(t, nil)
	addr := coreServer(t, d, coreEcho)
	client := &http.Client{Timeout: 30 * time.Second}

	post := func(size int) (*http.Response, map[string]any) {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
			"http://"+addr+"/api/v1/echo", bytes.NewReader(bytes.Repeat([]byte("a"), size)))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("POST %d byte: %v", size, err)
		}
		m := coreJSON(t, resp.Body)
		_ = resp.Body.Close()
		return resp, m
	}

	resp, body := post(2_000_000)
	if resp.StatusCode != http.StatusRequestEntityTooLarge || body["code"] != "PAYLOAD_TOO_LARGE" {
		t.Fatalf("thân 2 MB → %d %v", resp.StatusCode, body)
	}
	details, _ := body["details"].(map[string]any)
	if details["max_bytes"] != float64(1048576) {
		t.Fatalf("details.max_bytes = %v", details["max_bytes"])
	}

	if resp, body := post(1048576); resp.StatusCode != http.StatusOK || body["bytes"] != float64(1048576) {
		t.Fatalf("thân đúng 1 MiB → %d %v", resp.StatusCode, body)
	}
	if resp, body := post(1048577); resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("thân 1 MiB + 1 → %d %v", resp.StatusCode, body)
	}
}

// TestServerLimits_SlowHeader: header gửi chậm hơn ReadHeaderTimeout (5 s) → server đóng kết nối.
func TestServerLimits_SlowHeader(t *testing.T) {
	t.Parallel()
	d, _ := coreDeps(t, nil)
	addr := coreServer(t, d, coreEcho)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := io.WriteString(conn, "GET /healthz HTTP/1.1\r\nHost: x\r\n"); err != nil {
		t.Fatalf("ghi header dở: %v", err)
	}

	start := time.Now()
	_ = conn.SetReadDeadline(time.Now().Add(ReadHeaderTimeout + 8*time.Second))
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	elapsed := time.Since(start)

	if err == nil && strings.HasPrefix(string(buf[:n]), "HTTP/1.1 2") {
		t.Fatalf("server trả 2xx cho header chưa kết thúc: %q", buf[:n])
	}
	if elapsed < 3*time.Second {
		t.Fatalf("đóng sau %v, muốn ≈ ReadHeaderTimeout %v", elapsed, ReadHeaderTimeout)
	}
	if elapsed > ReadHeaderTimeout+5*time.Second {
		t.Fatalf("đóng sau %v, quá muộn", elapsed)
	}
}

// TestServerLimits_HugeHeader: header lớn hơn MaxHeaderBytes (64 KiB) → 431.
func TestServerLimits_HugeHeader(t *testing.T) {
	t.Parallel()
	d, _ := coreDeps(t, nil)
	addr := coreServer(t, d, coreEcho)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()
	req := fmt.Sprintf("GET /healthz HTTP/1.1\r\nHost: x\r\nX-Big: %s\r\n\r\n", strings.Repeat("a", 70000))
	if _, err := io.WriteString(conn, req); err != nil {
		t.Fatalf("ghi request: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := make([]byte, 128)
	n, _ := conn.Read(buf)
	if got := string(buf[:n]); !strings.HasPrefix(got, "HTTP/1.1 431") {
		t.Fatalf("dòng trạng thái = %q, muốn 431", got)
	}
}

// TestDeadline_DB: quá REQUEST_TIMEOUT → 504 DEADLINE_EXCEEDED và truy vấn bị HUỶ ở Postgres (AC9).
func TestDeadline_DB(t *testing.T) {
	t.Parallel()
	d, _ := coreDepsReal(t, func(c *config.Config) { c.RequestTimeout = 500 * time.Millisecond })
	srv := httptest.NewServer(newRouterWith(d, func(r chi.Router) {
		r.Get("/db-sleep", func(w http.ResponseWriter, r *http.Request) {
			var n int
			if err := d.DB.QueryRow(r.Context(),
				"-- name: TestDBSleep :one\nselect 1 from pg_sleep($1::float)", 5).Scan(&n); err != nil {
				return // middleware timeout trả 504
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]int{"n": n})
		})
	}))
	t.Cleanup(srv.Close)

	start := time.Now()
	resp, err := srv.Client().Get(srv.URL + "/api/v1/db-sleep")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	elapsed := time.Since(start)
	body := coreJSON(t, resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusGatewayTimeout || body["code"] != "DEADLINE_EXCEEDED" {
		t.Fatalf("status = %d, body = %v", resp.StatusCode, body)
	}
	if elapsed > 1500*time.Millisecond {
		t.Fatalf("trả lời sau %v, muốn ≤ 1,5 s", elapsed)
	}
	coreWaitNoSleepQuery(t, d)
}

// coreWaitNoSleepQuery đợi tới 3 s cho tới khi không còn truy vấn pg_sleep nào chạy trong DB của test.
func coreWaitNoSleepQuery(t *testing.T, d Deps) {
	t.Helper()
	const q = `select count(*) from pg_stat_activity
		where query ilike '%pg_sleep%' and state = 'active'
		  and pid <> pg_backend_pid() and datname = current_database()`
	deadline := time.Now().Add(3 * time.Second)
	var n int
	for time.Now().Before(deadline) {
		if err := d.DB.QueryRow(t.Context(), q).Scan(&n); err == nil && n == 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("còn %d truy vấn pg_sleep chạy sau khi request bị huỷ", n)
}

// TestDeadline_Redis: lệnh Redis chặn (BLPOP) cũng bị huỷ theo deadline của request → 504 (AC9).
func TestDeadline_Redis(t *testing.T) {
	t.Parallel()
	d, _ := coreDepsReal(t, func(c *config.Config) { c.RequestTimeout = 500 * time.Millisecond })
	key := appredis.Key("test", testutil.TestPrefix(t))
	srv := httptest.NewServer(newRouterWith(d, func(r chi.Router) {
		r.Get("/redis-block", func(w http.ResponseWriter, r *http.Request) {
			if err := d.Redis.BLPop(r.Context(), 5*time.Second, key).Err(); err != nil {
				return // middleware timeout trả 504
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
	}))
	t.Cleanup(srv.Close)

	start := time.Now()
	resp, err := srv.Client().Get(srv.URL + "/api/v1/redis-block")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	elapsed := time.Since(start)
	body := coreJSON(t, resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusGatewayTimeout || body["code"] != "DEADLINE_EXCEEDED" {
		t.Fatalf("status = %d, body = %v", resp.StatusCode, body)
	}
	if elapsed > 1500*time.Millisecond {
		t.Fatalf("trả lời sau %v, muốn ≤ 1,5 s", elapsed)
	}

	// Không còn client nào kẹt ở BLPOP.
	deadline := time.Now().Add(3 * time.Second)
	for {
		info, err := d.Redis.Info(t.Context(), "clients").Result()
		if err != nil {
			t.Fatalf("INFO clients: %v", err)
		}
		if !strings.Contains(info, "blocked_clients:0") && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if !strings.Contains(info, "blocked_clients:0") {
			t.Fatalf("còn client Redis bị chặn: %s", info)
		}
		return
	}
}

// TestDeadline_ClientCancel: client ngắt giữa chừng cũng huỷ truy vấn dưới DB (AC9).
func TestDeadline_ClientCancel(t *testing.T) {
	t.Parallel()
	d, _ := coreDepsReal(t, nil) // REQUEST_TIMEOUT mặc định 30 s: chỉ client cancel mới huỷ
	done := make(chan struct{})
	srv := httptest.NewServer(newRouterWith(d, func(r chi.Router) {
		r.Get("/db-sleep", func(w http.ResponseWriter, r *http.Request) {
			defer close(done)
			var n int
			if err := d.DB.QueryRow(r.Context(),
				"-- name: TestDBSleep :one\nselect 1 from pg_sleep($1::float)", 5).Scan(&n); err != nil {
				return
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]int{"n": n})
		})
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/v1/db-sleep", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp, err := srv.Client().Do(req); err == nil {
		_ = resp.Body.Close()
		t.Fatal("muốn client bị huỷ trước khi có phản hồi")
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("handler không kết thúc sau khi client ngắt")
	}
	coreWaitNoSleepQuery(t, d)
}

// TestReadyz_DependencyDown: readyz 200 khi DB + Redis sống; phụ thuộc chết → 503 NOT_READY với
// details tương ứng; đang tắt → details.draining = true (AC14, AC7).
func TestReadyz_DependencyDown(t *testing.T) {
	t.Parallel()
	d, _ := coreDepsReal(t, nil)
	srv := httptest.NewServer(NewRouter(d))
	t.Cleanup(srv.Close)

	get := func() (int, map[string]any) {
		t.Helper()
		resp, err := srv.Client().Get(srv.URL + "/api/v1/readyz")
		if err != nil {
			t.Fatalf("GET readyz: %v", err)
		}
		m := coreJSON(t, resp.Body)
		_ = resp.Body.Close()
		return resp.StatusCode, m
	}

	status, body := get()
	if status != http.StatusOK || body["status"] != "ready" || body["db"] != "up" || body["redis"] != "up" {
		t.Fatalf("readyz khi mọi thứ sống = %d %v", status, body)
	}

	// healthz không phụ thuộc DB/Redis.
	resp, err := srv.Client().Get(srv.URL + "/api/v1/healthz")
	if err != nil {
		t.Fatalf("GET healthz: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz = %d, muốn 200", resp.StatusCode)
	}

	// Đang tắt: 503 ngay dù phụ thuộc còn sống.
	d.State.Drain()
	status, body = get()
	details, _ := body["details"].(map[string]any)
	if status != http.StatusServiceUnavailable || body["code"] != "NOT_READY" || details["draining"] != true {
		t.Fatalf("readyz khi đang tắt = %d %v", status, body)
	}
	if details["db"] != "up" || details["redis"] != "up" {
		t.Fatalf("details khi đang tắt = %v", details)
	}

	// Redis chết (client đóng) → details.redis = down, tiến trình vẫn phục vụ.
	down, _ := coreDepsReal(t, nil)
	downSrv := httptest.NewServer(NewRouter(down))
	t.Cleanup(downSrv.Close)
	_ = down.Redis.Close()

	// healthz vẫn 200 khi phụ thuộc chết (AC14).
	hz, err := downSrv.Client().Get(downSrv.URL + "/api/v1/healthz")
	if err != nil {
		t.Fatalf("GET healthz: %v", err)
	}
	_ = hz.Body.Close()
	if hz.StatusCode != http.StatusOK {
		t.Fatalf("healthz khi Redis chết = %d, muốn 200", hz.StatusCode)
	}

	resp, err = downSrv.Client().Get(downSrv.URL + "/api/v1/readyz")
	if err != nil {
		t.Fatalf("GET readyz: %v", err)
	}
	body = coreJSON(t, resp.Body)
	_ = resp.Body.Close()
	details, _ = body["details"].(map[string]any)
	if resp.StatusCode != http.StatusServiceUnavailable || details["redis"] != "down" || details["db"] != "up" {
		t.Fatalf("readyz khi Redis chết = %d %v", resp.StatusCode, body)
	}

	// Postgres chết (pool đóng) → details.db = down.
	down.DB.Close()
	resp, err = downSrv.Client().Get(downSrv.URL + "/api/v1/readyz")
	if err != nil {
		t.Fatalf("GET readyz: %v", err)
	}
	body = coreJSON(t, resp.Body)
	_ = resp.Body.Close()
	details, _ = body["details"].(map[string]any)
	if resp.StatusCode != http.StatusServiceUnavailable || details["db"] != "down" {
		t.Fatalf("readyz khi Postgres chết = %d %v", resp.StatusCode, body)
	}
}

// TestRecover_PanicGives500: panic trong handler → log + 500 INTERNAL, server vẫn chạy.
func TestRecover_PanicGives500(t *testing.T) {
	t.Parallel()
	d, buf := coreDeps(t, nil)
	srv := httptest.NewServer(newRouterWith(d, func(r chi.Router) {
		r.Get("/panic", func(http.ResponseWriter, *http.Request) { panic("bùm") })
	}))
	t.Cleanup(srv.Close)

	resp, err := srv.Client().Get(srv.URL + "/api/v1/panic")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body := coreJSON(t, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError || body["code"] != "INTERNAL" {
		t.Fatalf("status = %d, body = %v", resp.StatusCode, body)
	}
	if !strings.Contains(buf.String(), "panic trong handler") {
		t.Fatalf("thiếu dòng log panic: %s", buf.String())
	}

	resp2, err := srv.Client().Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("server chết sau panic: %v", err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("healthz sau panic = %d", resp2.StatusCode)
	}
}
