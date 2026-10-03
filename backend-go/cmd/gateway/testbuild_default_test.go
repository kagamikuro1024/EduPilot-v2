//go:build !testroutes

package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
)

// testPaths là 13 đường dẫn thử của SRS 6.3 (đúng danh sách QC dùng ở 03-AC18).
func testPaths(user string) []string {
	return []string{
		"/api/v1/_test/items",
		"/api/v1/_test/items/" + user,
		"/api/v1/_test/slow",
		"/api/v1/_test/db-sleep",
		"/api/v1/_test/redis-block",
		"/api/v1/_test/panic",
		"/api/v1/_test/error/500",
		"/api/v1/_test/whoami",
		"/api/v1/_test/rbac/admin",
		"/api/v1/_test/rbac/staff",
		"/api/v1/_test/courses/" + user + "/ping",
		"/api/v1/_test/jobs",
		"/api/v1/_test/events",
	}
}

// TestDefaultBinary_NoTestRoutes: binary KHÔNG dựng bằng build tag `testroutes` không chứa gói
// `internal/testroutes` (không symbol, không chuỗi `/api/v1/_test/`) và trả 404 cho mọi đường dẫn
// thử — kể cả với token ADMIN hợp lệ (03-AC18).
func TestDefaultBinary_NoTestRoutes(t *testing.T) {
	t.Parallel()

	bin := filepath.Join(t.TempDir(), "gateway-default")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build bản mặc định: %v\n%s", err, out)
	}

	nm, err := exec.CommandContext(t.Context(), "go", "tool", "nm", bin).Output()
	if err != nil {
		t.Fatalf("go tool nm: %v", err)
	}
	if n := bytes.Count(nm, []byte("internal/testroutes")); n != 0 {
		t.Errorf("binary mặc định có %d symbol của internal/testroutes", n)
	}
	raw, err := os.ReadFile(bin) //nolint:gosec // đường dẫn do chính test sinh ra
	if err != nil {
		t.Fatalf("đọc binary: %v", err)
	}
	if n := bytes.Count(raw, []byte("/api/v1/_test/")); n != 0 {
		t.Errorf("binary mặc định chứa chuỗi /api/v1/_test/ %d lần", n)
	}

	// Router của chính bản dựng này: mọi đường dẫn thử là route không tồn tại → 404 NOT_FOUND.
	const secret = "0123456789abcdef0123456789abcdef"
	const user = "00000000-0000-7000-8000-000000000001"
	cfg := config.Config{
		Role: config.Gateway, AppEnv: "test", InstanceID: "gw-test", LogLevel: "error",
		JWTSecretKey: secret, RequestTimeout: 5 * time.Second, MaxBodyBytes: 1 << 20,
		RateLimitIPPerMin: 1000, RateLimitUserPerMin: 1000,
	}
	srv := httptest.NewServer(httpapi.NewRouter(httpapi.Deps{Cfg: cfg, Clock: clock.Real{}, State: httpapi.NewState()}))
	t.Cleanup(srv.Close)

	admin, err := auth.NewIssuer(secret, time.Hour, clock.Real{}).Issue(user, auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	for _, path := range testPaths(user) {
		for _, token := range []string{"", admin} {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+path, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatalf("GET %s: %v", path, err)
			}
			body := new(bytes.Buffer)
			_, _ = body.ReadFrom(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("GET %s (token=%t) → %d, cần 404", path, token != "", resp.StatusCode)
			}
			if !strings.Contains(body.String(), `"NOT_FOUND"`) {
				t.Errorf("GET %s: thân = %s", path, body.String())
			}
		}
	}
}
