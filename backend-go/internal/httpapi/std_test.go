package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/go-chi/chi/v5"
)

// stdRoutes là vài route ad-hoc đủ để kiểm middleware chung mà không cần DB/Redis.
func stdRoutes(r chi.Router) {
	r.Get("/ok", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/boom", func(http.ResponseWriter, *http.Request) { panic("nổ trong handler") })
	r.Get("/abort", func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) })
	r.Get("/unauth", func(w http.ResponseWriter, r *http.Request) {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
	})
	r.Get("/limited", func(w http.ResponseWriter, r *http.Request) {
		apierr.Write(w, r, apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(7))
	})
	r.Get("/etag", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSONETag(w, r, map[string]int{"version": 1}, httpx.ETagVersion(1))
	})
}

// stdServer dựng server thật với router đầy đủ + route ad-hoc ở trên.
func stdServer(t *testing.T, mutate func(*config.Config)) *httptest.Server {
	t.Helper()
	d, _ := coreDeps(t, mutate)
	d = withDefaults(d)
	srv := httptest.NewServer(newRouterWith(d, stdRoutes))
	t.Cleanup(srv.Close)
	return srv
}

func stdGet(t *testing.T, srv *httptest.Server, path string, header ...string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// TestRequestID: X-Request-Id hợp lệ được giữ, giá trị sai bị thay bằng trace_id (32 hex),
// và header có mặt trên mọi loại phản hồi kể cả 304 / lỗi (03-AC1, SRS 6.5).
func TestRequestID(t *testing.T) {
	t.Parallel()
	srv := stdServer(t, nil)

	keep := []string{"abc-123", strings.Repeat("a", 64), "A.b_c-1"}
	for _, id := range keep {
		if got := stdGet(t, srv, "/api/v1/ok", "X-Request-Id", id).Header.Get("X-Request-Id"); got != id {
			t.Errorf("X-Request-Id %q bị đổi thành %q", id, got)
		}
	}

	replaced := []string{strings.Repeat("a", 65), "bad id!", "", "xin chào"}
	for _, id := range replaced {
		got := stdGet(t, srv, "/api/v1/ok", "X-Request-Id", id).Header.Get("X-Request-Id")
		if got == id || !coreHex32.MatchString(got) {
			t.Errorf("X-Request-Id %q → %q (cần 32 hex sinh mới)", id, got)
		}
	}

	etag := stdGet(t, srv, "/api/v1/etag").Header.Get("ETag")
	cases := []struct {
		name, path, want string
		header           []string
	}{
		{name: "200", path: "/api/v1/ok"},
		{name: "404", path: "/api/v1/khong-co"},
		{name: "401", path: "/api/v1/unauth"},
		{name: "429", path: "/api/v1/limited"},
		{name: "304", path: "/api/v1/etag", header: []string{"If-None-Match", etag}},
	}
	for _, c := range cases {
		resp := stdGet(t, srv, c.path, c.header...)
		if resp.Header.Get("X-Request-Id") == "" {
			t.Errorf("%s: thiếu X-Request-Id", c.name)
		}
		if resp.Header.Get("X-Instance-Id") != "gw-test" {
			t.Errorf("%s: thiếu X-Instance-Id", c.name)
		}
	}
	if resp := stdGet(t, srv, "/api/v1/etag", "If-None-Match", etag); resp.StatusCode != http.StatusNotModified {
		t.Fatalf("If-None-Match khớp → %d, cần 304", resp.StatusCode)
	}
}

// TestRecover: panic → 500 INTERNAL chung (không lộ nội bộ), tiến trình sống, request sau vẫn 200;
// http.ErrAbortHandler KHÔNG bị nuốt (03-AC2, FR-26).
func TestRecover(t *testing.T) {
	t.Parallel()
	d, buf := coreDeps(t, nil)
	d = withDefaults(d)
	srv := httptest.NewServer(newRouterWith(d, stdRoutes))
	t.Cleanup(srv.Close)

	resp := stdGet(t, srv, "/api/v1/boom")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("panic → %d, cần 500", resp.StatusCode)
	}
	body := coreJSON(t, resp.Body)
	if body["code"] != apierr.Internal {
		t.Fatalf("code = %v", body["code"])
	}
	msg, _ := body["message"].(string)
	if msg == "" {
		t.Fatal("message rỗng")
	}
	for _, leak := range []string{"panic", ".go", "sql", "/"} {
		if strings.Contains(strings.ToLower(msg), leak) {
			t.Errorf("message lộ nội bộ (%q): %q", leak, msg)
		}
	}
	if !coreHex32.MatchString(body["trace_id"].(string)) {
		t.Errorf("trace_id = %v", body["trace_id"])
	}

	// Log mức error có stack và trace_id của chính request đó.
	var logged bool
	for _, line := range coreLogLines(t, buf.String()) {
		if line["level"] == "ERROR" && line["trace_id"] == body["trace_id"] &&
			strings.Contains(line["stack"].(string), ".go:") {
			logged = true
		}
	}
	if !logged {
		t.Error("thiếu dòng log error có stack + trace_id của request panic")
	}

	// Tiến trình vẫn sống.
	if resp := stdGet(t, srv, "/api/v1/ok"); resp.StatusCode != http.StatusOK {
		t.Fatalf("request kế tiếp → %d", resp.StatusCode)
	}

	// ErrAbortHandler: net/http cắt kết nối, client thấy lỗi chứ không thấy 500 JSON.
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/api/v1/abort", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if resp, err := srv.Client().Do(req); err == nil {
		_ = resp.Body.Close()
		t.Fatalf("ErrAbortHandler bị nuốt: nhận %d", resp.StatusCode)
	}
}

// TestCORS: chỉ origin trong CORS_ORIGINS mới có Allow-Origin + Allow-Credentials; preflight 204
// kèm Methods/Headers/Max-Age/Expose; origin lạ không có gì; không bao giờ `*` (03-AC3, SRS 6.7).
func TestCORS(t *testing.T) {
	t.Parallel()
	const good = "http://localhost:3000"
	srv := stdServer(t, func(c *config.Config) { c.CORSOrigins = []string{good, "https://localhost"} })

	preflight := func(origin string) *http.Response {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodOptions, srv.URL+"/api/v1/ok", nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		req.Header.Set("Access-Control-Request-Headers", "Authorization, Idempotency-Key")
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatalf("OPTIONS: %v", err)
		}
		t.Cleanup(func() { _ = resp.Body.Close() })
		return resp
	}

	resp := preflight(good)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("preflight → %d, cần 204", resp.StatusCode)
	}
	h := resp.Header
	if h.Get("Access-Control-Allow-Origin") != good {
		t.Errorf("Allow-Origin = %q", h.Get("Access-Control-Allow-Origin"))
	}
	if h.Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("thiếu Allow-Credentials")
	}
	if h.Get("Access-Control-Max-Age") != "600" {
		t.Errorf("Max-Age = %q", h.Get("Access-Control-Max-Age"))
	}
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
		if !strings.Contains(h.Get("Access-Control-Allow-Methods"), m) {
			t.Errorf("Allow-Methods thiếu %s", m)
		}
	}
	for _, k := range []string{"authorization", "content-type", "idempotency-key", "if-match", "if-none-match", "last-event-id", "x-request-id"} {
		if !strings.Contains(strings.ToLower(h.Get("Access-Control-Allow-Headers")), k) {
			t.Errorf("Allow-Headers thiếu %s", k)
		}
	}
	for _, k := range []string{"etag", "x-request-id", "retry-after", "idempotent-replayed"} {
		if !strings.Contains(strings.ToLower(h.Get("Access-Control-Expose-Headers")), k) {
			t.Errorf("Expose-Headers thiếu %s", k)
		}
	}
	if !strings.Contains(strings.Join(h.Values("Vary"), ","), "Origin") {
		t.Errorf("Vary thiếu Origin: %v", h.Values("Vary"))
	}

	for _, origin := range []string{"https://evil.example", "null", ""} {
		got := preflight(origin)
		if v := got.Header.Get("Access-Control-Allow-Origin"); v != "" {
			t.Errorf("origin %q: Allow-Origin = %q (phải rỗng)", origin, v)
		}
		if v := got.Header.Get("Access-Control-Allow-Credentials"); v != "" {
			t.Errorf("origin %q: Allow-Credentials = %q (phải rỗng)", origin, v)
		}
	}

	// Request thường (không preflight) cũng được gắn header, kể cả trên health.
	ok := stdGet(t, srv, "/api/v1/healthz", "Origin", good)
	if ok.Header.Get("Access-Control-Allow-Origin") != good {
		t.Errorf("GET: Allow-Origin = %q", ok.Header.Get("Access-Control-Allow-Origin"))
	}
	bad := stdGet(t, srv, "/api/v1/healthz", "Origin", "https://evil.example")
	if v := bad.Header.Get("Access-Control-Allow-Origin"); v != "" {
		t.Errorf("GET origin lạ: Allow-Origin = %q", v)
	}
}

// TestNotFoundAndMethodNotAllowed: route lạ → 404 JSON (không phải văn bản của net/http),
// sai method → 405 JSON kèm Allow (03-AC4).
func TestNotFoundAndMethodNotAllowed(t *testing.T) {
	t.Parallel()
	srv := stdServer(t, nil)

	resp := stdGet(t, srv, "/api/v1/khong-co")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("route lạ → %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	body := coreJSON(t, resp.Body)
	if body["code"] != apierr.NotFound || body["message"] == "" {
		t.Fatalf("thân 404 = %v", body)
	}
	if !coreHex32.MatchString(body["trace_id"].(string)) {
		t.Errorf("trace_id = %v", body["trace_id"])
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, srv.URL+"/api/v1/healthz", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	del, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer func() { _ = del.Body.Close() }()
	if del.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE /api/v1/healthz → %d", del.StatusCode)
	}
	if !strings.Contains(del.Header.Get("Allow"), http.MethodGet) {
		t.Errorf("Allow = %q", del.Header.Get("Allow"))
	}
	if code := coreJSON(t, del.Body)["code"]; code != apierr.MethodNotAllowed {
		t.Errorf("code = %v", code)
	}
}
