package sse

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

type revokedAll struct{}

func (revokedAll) Revoked(context.Context, auth.Principal) (bool, string) { return true, "logout" }

// P-11-31 F-2: /events dùng cùng tuỳ chọn xác thực như nhóm nghiệp vụ — phiên đã thu hồi → 401 SESSION_REVOKED; token không `sid` bị từ chối khi bắt buộc phiên (production).
func TestSSE_RevokedSessionAndRequireSession(t *testing.T) {
	rdb, err := redis.New(context.Background(), testutil.RedisURL(t))
	if err != nil {
		t.Fatalf("redis.New: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	iss := auth.NewIssuer(testSecret, time.Hour, clock.Real{})
	withSid, err := iss.IssueSession(newUID(), auth.RoleStudent, "sse@test.local", newUID())
	if err != nil {
		t.Fatal(err)
	}
	noSid, err := iss.Issue(newUID(), auth.RoleStudent, "sse@test.local")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		mw   []auth.MiddlewareOption
		tok  string
		code string
	}{
		{"phiên thu hồi", []auth.MiddlewareOption{auth.WithRevocation(revokedAll{})}, withSid, apierr.SessionRevoked},
		{"production: token không sid", []auth.MiddlewareOption{auth.RequireSession()}, noSid, apierr.TokenInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(NewHandler(HandlerDeps{Cfg: testCfg(), Log: slog.Default(), Redis: rdb, Verifier: auth.NewVerifier(testSecret, clock.Real{}), Clock: clock.Real{}, MW: c.mw}))
			defer srv.Close()
			req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/api/v1/events", nil)
			req.Header.Set("Authorization", "Bearer "+c.tok)
			resp, err := testClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("mã HTTP = %d, muốn 401", resp.StatusCode)
			}
			if body := errBody(t, resp); body["code"] != c.code {
				t.Errorf("code = %v, muốn %s", body["code"], c.code)
			}
		})
	}
}
