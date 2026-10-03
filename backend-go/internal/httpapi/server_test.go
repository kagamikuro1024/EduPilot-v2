package httpapi

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/go-chi/chi/v5"
)

// coreSlowServer dựng server thật (router + giới hạn như lúc chạy) với một route chậm ad-hoc
// và chạy nó qua Serve với ctx huỷ được.
func coreSlowServer(t *testing.T, d Deps, delay time.Duration) (addr string, serveErr chan error, cancel context.CancelFunc) {
	t.Helper()
	srv := NewServer(d)
	srv.Handler = newRouterWith(d, func(r chi.Router) {
		r.Get("/slow", func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	serveErr = make(chan error, 1)
	go func() { serveErr <- Serve(ctx, srv, ln, d) }()
	return ln.Addr().String(), serveErr, cancel
}

// coreWaitInFlight đợi tới khi có request thật sự đang chạy trong handler (tránh đua với client).
func coreWaitInFlight(t *testing.T, d Deps) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if d.State.InFlight() > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("không có request nào vào được handler")
}

// TestGracefulShutdown: SIGTERM (ctx huỷ) → request đang chạy hoàn tất 200, kết nối mới bị từ chối,
// Serve trả nil (lệnh thoát mã 0) trước SHUTDOWN_TIMEOUT (US-PG-01 AC7).
func TestGracefulShutdown(t *testing.T) {
	t.Parallel()
	d, _ := coreDeps(t, func(c *config.Config) { c.ShutdownTimeout = 10 * time.Second })
	addr, serveErr, cancel := coreSlowServer(t, d, 2*time.Second)

	type result struct {
		code int
		err  error
	}
	slow := make(chan result, 1)
	go func() {
		client := &http.Client{Timeout: 20 * time.Second}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+addr+"/api/v1/slow", nil)
		if err != nil {
			slow <- result{err: err}
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			slow <- result{err: err}
			return
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		slow <- result{code: resp.StatusCode}
	}()

	coreWaitInFlight(t, d) // request chậm đã vào handler
	start := time.Now()
	cancel()                           // ~ SIGTERM
	time.Sleep(100 * time.Millisecond) // cờ draining đã bật

	if !d.State.Draining() {
		t.Error("State phải chuyển sang draining ngay khi bắt đầu tắt")
	}
	// Trong cửa sổ drain, cổng còn mở nhưng request MỚI nhận 503 NOT_READY (AC7 (1) và (4)).
	resp, err := http.Get("http://" + addr + "/api/v1/healthz") //nolint:noctx // test cục bộ
	if err == nil {
		body := coreJSON(t, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "NOT_READY" {
			t.Errorf("request mới khi đang tắt = %d %v, muốn 503 NOT_READY", resp.StatusCode, body)
		}
		if resp.Header.Get(DrainingHeader) != "1" {
			t.Errorf("503 do đang tắt phải mang %s: 1 để Caddy thử lại sang bản khác", DrainingHeader)
		}
	}
	rdz, err := http.Get("http://" + addr + "/api/v1/readyz") //nolint:noctx // test cục bộ
	if err == nil {
		body := coreJSON(t, rdz.Body)
		_ = rdz.Body.Close()
		details, _ := body["details"].(map[string]any)
		if rdz.StatusCode != http.StatusServiceUnavailable || details["draining"] != true {
			t.Errorf("readyz khi đang tắt = %d %v", rdz.StatusCode, body)
		}
		if rdz.Header.Get(DrainingHeader) != "" {
			t.Errorf("readyz không được mang %s", DrainingHeader)
		}
	}

	got := <-slow
	if got.err != nil || got.code != http.StatusOK {
		t.Fatalf("request đang chạy = %d (%v), muốn 200 không bị cắt", got.code, got.err)
	}

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("Serve = %v, muốn nil (thoát mã 0)", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Serve không trả về sau khi tắt êm")
	}
	if elapsed := time.Since(start); elapsed > d.Cfg.ShutdownTimeout {
		t.Fatalf("tắt mất %v, muốn < SHUTDOWN_TIMEOUT %v", elapsed, d.Cfg.ShutdownTimeout)
	}
}

// TestShutdown_ForcedAfterTimeout: request vượt SHUTDOWN_TIMEOUT bị cắt → log warn `forced shutdown`
// kèm `cancelled_requests` ≥ 1 và Serve trả lỗi (lệnh thoát mã 1) (US-PG-01 AC7, #Q-QC-01-5).
func TestShutdown_ForcedAfterTimeout(t *testing.T) {
	t.Parallel()
	d, buf := coreDeps(t, func(c *config.Config) { c.ShutdownTimeout = 300 * time.Millisecond })
	addr, serveErr, cancel := coreSlowServer(t, d, 10*time.Second)

	go func() {
		client := &http.Client{Timeout: 20 * time.Second}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+addr+"/api/v1/slow", nil)
		if err != nil {
			return
		}
		if resp, err := client.Do(req); err == nil {
			_ = resp.Body.Close()
		}
	}()

	coreWaitInFlight(t, d)
	cancel()

	select {
	case err := <-serveErr:
		if err == nil {
			t.Fatal("Serve = nil, muốn lỗi khi phải cắt request quá hạn (thoát mã 1)")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Serve không trả về")
	}

	var forced map[string]any
	for _, l := range coreLogLines(t, buf.String()) {
		if l["msg"] == "forced shutdown" {
			forced = l
		}
	}
	if forced == nil {
		t.Fatalf("thiếu dòng log `forced shutdown`: %s", buf.String())
	}
	if forced["level"] != "WARN" {
		t.Errorf("level = %v, muốn WARN", forced["level"])
	}
	n, ok := forced["cancelled_requests"].(float64)
	if !ok || n < 1 || n != float64(int64(n)) {
		t.Errorf("cancelled_requests = %v, muốn số nguyên ≥ 1", forced["cancelled_requests"])
	}
}

// TestStartup_WaitsForDeps: chờ DB/Redis mỗi giây tới STARTUP_TIMEOUT; mỗi lần lỗi ghi warn
// `dependency not ready` kèm `dependency`; quá hạn ghi error cùng msg rồi trả lỗi (AC14, SRS 3.4).
func TestStartup_WaitsForDeps(t *testing.T) {
	t.Parallel()

	t.Run("phụ thuộc lên muộn thì vẫn khởi động được", func(t *testing.T) {
		t.Parallel()
		d, buf := coreDeps(t, nil)
		var calls atomic.Int32
		dep := Dep{Name: "db", Ping: func(context.Context) error {
			if calls.Add(1) < 3 {
				return errors.New("chưa lên")
			}
			return nil
		}}

		start := time.Now()
		if err := WaitForDeps(t.Context(), d.Log, 10*time.Second, dep); err != nil {
			t.Fatalf("WaitForDeps = %v, muốn nil", err)
		}
		if elapsed := time.Since(start); elapsed < time.Second {
			t.Fatalf("thử lại sau %v, muốn mỗi giây một lần", elapsed)
		}

		var warns int
		for _, l := range coreLogLines(t, buf.String()) {
			if l["msg"] != "dependency not ready" {
				continue
			}
			warns++
			if l["level"] != "WARN" || l["dependency"] != "db" {
				t.Errorf("dòng chờ = %v", l)
			}
		}
		if warns != 2 {
			t.Fatalf("số dòng warn = %d, muốn 2 (mỗi giây một dòng)", warns)
		}
	})

	t.Run("quá hạn thì báo tên phụ thuộc và thất bại", func(t *testing.T) {
		t.Parallel()
		d, buf := coreDeps(t, nil)
		deps := []Dep{
			{Name: "db", Ping: func(context.Context) error { return errors.New("chưa lên") }},
			{Name: "redis", Ping: func(context.Context) error { return nil }},
		}

		start := time.Now()
		err := WaitForDeps(t.Context(), d.Log, 2*time.Second, deps...)
		elapsed := time.Since(start)
		if err == nil {
			t.Fatal("WaitForDeps = nil, muốn lỗi khi quá STARTUP_TIMEOUT")
		}
		if !strings.Contains(err.Error(), "db") || strings.Contains(err.Error(), "redis") {
			t.Fatalf("lỗi = %v, muốn chỉ nêu db", err)
		}
		if elapsed < 2*time.Second || elapsed > 6*time.Second {
			t.Fatalf("chờ %v, muốn ≈ STARTUP_TIMEOUT 2 s", elapsed)
		}

		var warns, errs int
		for _, l := range coreLogLines(t, buf.String()) {
			if l["msg"] != "dependency not ready" {
				continue
			}
			if l["dependency"] != "db" {
				t.Errorf("dependency = %v, muốn db", l["dependency"])
			}
			switch l["level"] {
			case "WARN":
				warns++
			case "ERROR":
				errs++
			}
		}
		if warns < 1 || warns > 4 {
			t.Errorf("số dòng warn = %d, muốn 1–4", warns)
		}
		if errs != 1 {
			t.Errorf("số dòng error = %d, muốn 1", errs)
		}
	})

	// BUG-PG-4: phụ thuộc treo (go-redis tự thử dial lại nhiều lần) không được làm vòng chờ giãn ra — vẫn một warn mỗi giây.
	t.Run("ping treo vẫn mỗi giây một dòng warn", func(t *testing.T) {
		t.Parallel()
		d, buf := coreDeps(t, nil)
		hang := Dep{Name: "redis", Ping: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }}

		if err := WaitForDeps(t.Context(), d.Log, 3500*time.Millisecond, hang); err == nil {
			t.Fatal("WaitForDeps = nil, muốn lỗi khi quá STARTUP_TIMEOUT")
		}
		var warns int
		for _, l := range coreLogLines(t, buf.String()) {
			if l["msg"] == "dependency not ready" && l["level"] == "WARN" {
				warns++
			}
		}
		if warns < 3 {
			t.Errorf("số dòng warn = %d trong 3,5 s, muốn ≥ 3", warns)
		}
	})
}

// TestServe_PortInUse: lỗi mở cổng được trả về ngay (lệnh thoát mã 1), không treo.
func TestServe_PortInUse(t *testing.T) {
	t.Parallel()
	d, _ := coreDeps(t, nil)
	srv := NewServer(d)
	srv.Handler = http.NewServeMux()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	_ = ln.Close() // listener đã đóng → Serve trả lỗi ngay

	done := make(chan error, 1)
	go func() { done <- Serve(t.Context(), srv, ln, d) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Serve = nil, muốn lỗi khi listener hỏng")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve treo khi listener hỏng")
	}
}

// BUG-P103-1: WriteTimeout phải DÀI HƠN REQUEST_TIMEOUT, nếu không 504 DEADLINE_EXCEEDED ghi đúng lúc kết nối bị đóng.
func TestWriteTimeoutOutlivesRequestTimeout(t *testing.T) {
	t.Parallel()
	for _, rt := range []time.Duration{time.Second, 30 * time.Second, 90 * time.Second} {
		d := Deps{Cfg: config.Config{RequestTimeout: rt}}
		srv := NewServer(d)
		if srv.WriteTimeout < rt+WriteMargin {
			t.Errorf("REQUEST_TIMEOUT %v: WriteTimeout %v < %v", rt, srv.WriteTimeout, rt+WriteMargin)
		}
	}
}
