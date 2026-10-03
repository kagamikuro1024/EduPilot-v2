package httpapi_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/clock"
)

// TestAuth_NoSecretsInLogs (04-AC9, bản ở tầng HTTP): log của một chuỗi middleware có xác thực
// không được chứa token, chữ ký, secret hay header Authorization — chỉ `sub`/`role`/`jti` rút gọn.
func TestAuth_NoSecretsInLogs(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef"
	const sub = "00000000-0000-7000-8000-000000000001"
	clk := clock.NewFake(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))

	tok, err := auth.NewIssuer(secret, 0, clk).Issue(sub, auth.RoleStudent, "qc@example.test")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Middleware log kiểu "access log": ghi phương thức, đường dẫn, trạng thái và danh tính.
	accessLog := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attrs := []any{"method", r.Method, "path", r.URL.Path}
			if p, ok := auth.FromContext(r.Context()); ok {
				attrs = append(attrs, "principal", p)
			}
			logger.Info("http request", attrs...)
			next.ServeHTTP(w, r)
		})
	}
	h := auth.Middleware(auth.NewVerifier(secret, clk))(accessLog(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })))

	for _, authz := range []string{"Bearer " + tok, "Bearer garbage", "Basic dTpw", ""} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/_test/whoami", nil)
		if authz != "" {
			req.Header.Set("Authorization", authz)
		}
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	out := buf.String()
	sig := tok[strings.LastIndex(tok, ".")+1:]
	for name, s := range map[string]string{
		"token đầy đủ":   tok,
		"chữ ký token":   sig,
		"JWT_SECRET_KEY": secret,
		"chữ Bearer":     "Bearer ",
		"email (PII)":    "qc@example.test",
	} {
		if strings.Contains(out, s) {
			t.Errorf("log chứa %s", name)
		}
	}
	if !strings.Contains(out, sub) {
		t.Error("log thiếu sub (không lần vết được)")
	}
}
