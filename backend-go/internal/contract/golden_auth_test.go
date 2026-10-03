package contract

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/edupilot/backend-go/internal/auth"
)

// Golden của 3 thao tác phiên (US-P2-02; SRS FEAT-account-security 9): thân phản hồi, trường động được che.
// Ghi lại: UPDATE_GOLDEN=1 go test ./internal/contract -run TestGolden_Auth — chỉ khi hợp đồng đổi có chủ ý (đổi spec + PM duyệt).
func TestGolden_Auth(t *testing.T) {
	r := getRig(t)
	ctx := context.Background()
	const password = "Edupilot#2026-demo"
	hash, err := auth.HashPassword(password, 4)
	if err != nil {
		t.Fatal(err)
	}
	email := "golden.auth@example.test"
	if _, err := r.deps.DB.Exec(ctx, `insert into users (email, full_name, role, status, password_hash) values ($1, 'Nguyễn Minh Trung', 'STUDENT', 'ACTIVE', $2)`, email, hash); err != nil {
		t.Fatal(err)
	}
	do := func(path, body string, hdr map[string]string, name string, want int) http.Header {
		t.Helper()
		var rd io.Reader
		if body != "" {
			rd = strings.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.srv.URL+"/api/v1/auth"+path, rd)
		if err != nil {
			t.Fatal(err)
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("POST %s → %d (cần %d): %s", path, resp.StatusCode, want, b)
		}
		if name != "" && len(bytes.TrimSpace(b)) > 0 {
			checkGoldenIn(t, "auth", name, b)
		}
		return resp.Header
	}
	origin := map[string]string{"Origin": "https://localhost"}
	do("/login", `{"email":"`+email+`","password":"sai-mat-khau-1"}`, origin, "login.401", 401)
	h := do("/login", `{"email":"`+email+`","password":"`+password+`"}`, origin, "login", 200)
	rt := strings.TrimPrefix(strings.SplitN(h.Get("Set-Cookie"), ";", 2)[0], "ep_rt=")
	do("/refresh", "", origin, "refresh.401", 401)
	do("/refresh", "", map[string]string{"Origin": "https://localhost", "Cookie": "ep_rt=" + rt}, "refresh", 200)
	do("/refresh", "", map[string]string{"Origin": "https://evil.example"}, "refresh.403", 403)
}
