//go:build testroutes

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appdb "github.com/edupilot/backend-go/internal/platform/db"
	applog "github.com/edupilot/backend-go/internal/platform/log"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
	"github.com/google/uuid"
)

// Một ngăn xếp dùng chung cho mọi test route thử: Postgres đã migrate + Redis thật + router đầy đủ.
// Dữ liệu của từng test tách nhau bằng `owner_id` nên dùng chung vẫn chạy song song được.
var (
	tOnce  sync.Once
	tDeps  Deps
	tSrv   *httptest.Server
	tSetup error
)

// tStack trả server thật (router có route thử) + Deps của nó.
func tStack(t *testing.T) (*httptest.Server, Deps) {
	t.Helper()
	testutil.RequireContainers(t)
	tOnce.Do(func() {
		coreOtel(t)
		cfg := coreConfig()
		cfg.JWTSecretKey = rlSecret
		cfg.RateLimitIPPerMin, cfg.RateLimitUserPerMin = 1_000_000, 1_000_000
		cfg.SSEBufferMaxLen, cfg.SSEBufferTTL = 1000, time.Hour
		cfg.DatabaseURL = testutil.MigratedPostgresURL(t)
		cfg.RedisURL = testutil.RedisURL(t)
		log := applog.NewTo(io.Discard, cfg, "gateway")

		ctx := context.Background()
		pool, err := appdb.NewPool(ctx, cfg, log)
		if err != nil {
			tSetup = fmt.Errorf("NewPool: %w", err)
			return
		}
		rdb, err := appredis.New(ctx, cfg.RedisURL)
		if err != nil {
			tSetup = fmt.Errorf("redis.New: %w", err)
			return
		}
		tDeps = withDefaults(Deps{
			Cfg: cfg, Log: log, DB: pool, Redis: rdb, Clock: clock.Real{}, State: NewState(),
		})
		tSrv = httptest.NewServer(NewRouter(tDeps))
	})
	if tSetup != nil {
		t.Fatalf("dựng ngăn xếp test: %v", tSetup)
	}
	return tSrv, tDeps
}

// tUser sinh một chủ sở hữu riêng cho mỗi test + token STUDENT của người đó.
func tUser(t *testing.T) (string, string) {
	t.Helper()
	_, d := tStack(t)
	sub := uuid.NewString()
	tok, err := auth.NewIssuer(rlSecret, time.Hour, d.Clock).Issue(sub, auth.RoleStudent, "")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return sub, tok
}

func tTokenRole(t *testing.T, sub string, role auth.Role) string {
	t.Helper()
	_, d := tStack(t)
	tok, err := auth.NewIssuer(rlSecret, time.Hour, d.Clock).Issue(sub, role, "")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return tok
}

// tDo gọi một route thử. header là các cặp tên/giá trị thêm; body rỗng = không gửi thân.
func tDo(t *testing.T, method, path, token, body string, header ...string) *http.Response {
	t.Helper()
	srv, _ := tStack(t)
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, rd)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// tBody đọc toàn bộ thân (giữ nguyên byte — cần cho băm ETag danh sách).
func tBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("đọc thân: %v", err)
	}
	return b
}

func tJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(tBody(t, resp), &m); err != nil {
		t.Fatalf("thân không phải JSON: %v", err)
	}
	return m
}

// tIdemKey sinh khoá Idempotency-Key hợp lệ (8–128 ký tự [A-Za-z0-9._:-]).
func tIdemKey(t *testing.T) string {
	t.Helper()
	return "test-" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

// tNewItem tạo một bản ghi thử và trả thân JSON của nó; kiểm luôn 201 + Location (#Q-QC-03-3).
func tNewItem(t *testing.T, token, name string) map[string]any {
	t.Helper()
	resp := tDo(t, http.MethodPost, "/api/v1/_test/items", token,
		`{"name":"`+name+`"}`, "Idempotency-Key", tIdemKey(t))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /_test/items → %d (cần 201): %s", resp.StatusCode, tBody(t, resp))
	}
	body := tJSON(t, resp)
	if loc := resp.Header.Get("Location"); loc != "/api/v1/_test/items/"+fmt.Sprint(body["id"]) {
		t.Fatalf("Location = %q, id = %v", loc, body["id"])
	}
	return body
}

// tSeed chèn n dòng cho một chủ sở hữu; mỗi dòng cách nhau 1 giây (created_at giảm dần).
func tSeed(t *testing.T, owner string, n int) {
	t.Helper()
	_, d := tStack(t)
	_, err := d.DB.Exec(t.Context(),
		`insert into _test_items (name, owner_id, created_at)
		 select 'seed-'||g, $1::uuid, now() - (g || ' seconds')::interval from generate_series(1, $2::int) g`,
		owner, n)
	if err != nil {
		t.Fatalf("seed %d dòng: %v", n, err)
	}
}

// TestErrorFormat_AllStatuses: 13 status của SRS 6.1 — đúng `code`, chỉ các khoá cho phép,
// `trace_id` 32 hex, message tiếng Việt không lộ nội bộ, 429/503 có `retry_after` = header Retry-After (03-AC4).
func TestErrorFormat_AllStatuses(t *testing.T) {
	t.Parallel()
	want := map[int]string{
		400: apierr.BadRequest, 401: apierr.Unauthenticated, 403: apierr.Forbidden, 404: apierr.NotFound,
		405: apierr.MethodNotAllowed, 409: apierr.Conflict, 413: apierr.PayloadTooLarge,
		415: apierr.UnsupportedMediaType, 422: apierr.ValidationFailed, 429: apierr.RateLimited,
		500: apierr.Internal, 503: apierr.ServiceUnavailable, 504: apierr.DeadlineExceeded,
	}
	allowed := map[string]bool{"code": true, "message": true, "trace_id": true, "details": true, "retry_after": true}

	for status, code := range want {
		resp := tDo(t, http.MethodGet, "/api/v1/_test/error/"+strconv.Itoa(status), "", "")
		if resp.StatusCode != status {
			t.Errorf("status %d → %d", status, resp.StatusCode)
			continue
		}
		if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("status %d: Content-Type = %q", status, ct)
		}
		body := tJSON(t, resp)
		if body["code"] != code {
			t.Errorf("status %d: code = %v, cần %s", status, body["code"], code)
		}
		for k := range body {
			if !allowed[k] {
				t.Errorf("status %d: khoá lạ %q", status, k)
			}
		}
		if id, _ := body["trace_id"].(string); !coreHex32.MatchString(id) {
			t.Errorf("status %d: trace_id = %v", status, body["trace_id"])
		}
		msg, _ := body["message"].(string)
		if msg == "" {
			t.Errorf("status %d: message rỗng", status)
		}
		for _, leak := range []string{"panic", ".go", "sql", "select ", "/"} {
			if strings.Contains(strings.ToLower(msg), leak) {
				t.Errorf("status %d: message lộ nội bộ: %q", status, msg)
			}
		}
		if status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable {
			if body["retry_after"] != float64(7) {
				t.Errorf("status %d: retry_after = %v", status, body["retry_after"])
			}
			if h := resp.Header.Get("Retry-After"); h != "7" {
				t.Errorf("status %d: Retry-After = %q", status, h)
			}
		}
	}

	// 401 của route thử là cố định: token hợp lệ vẫn 401 (route ẩn danh).
	_, tok := tUser(t)
	if resp := tDo(t, http.MethodGet, "/api/v1/_test/error/401", tok, ""); resp.StatusCode != 401 {
		t.Errorf("error/401 với token hợp lệ → %d", resp.StatusCode)
	}
}

// TestValidation: JSON hỏng → 400; mọi lỗi dữ liệu liệt kê cùng lúc theo thứ tự trường → 422;
// Content-Type sai/thiếu → 415 (03-AC5).
func TestValidation(t *testing.T) {
	t.Parallel()
	_, tok := tUser(t)
	const path = "/api/v1/_test/items"

	resp := tDo(t, http.MethodPost, path, tok, `{"name":`, "Idempotency-Key", tIdemKey(t))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("JSON hỏng → %d, cần 400", resp.StatusCode)
	}
	if code := tJSON(t, resp)["code"]; code != apierr.BadRequest {
		t.Errorf("JSON hỏng: code = %v", code)
	}

	resp = tDo(t, http.MethodPost, path, tok, `{"name":"","extra":1}`, "Idempotency-Key", tIdemKey(t))
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("name rỗng + trường lạ → %d, cần 422", resp.StatusCode)
	}
	body := tJSON(t, resp)
	if body["code"] != apierr.ValidationFailed {
		t.Fatalf("code = %v", body["code"])
	}
	details, _ := body["details"].([]any)
	var fields []string
	for _, d := range details {
		m, _ := d.(map[string]any)
		if m["field"] == nil || m["code"] == nil || m["message"] == nil {
			t.Errorf("phần tử details thiếu field/code/message: %v", m)
		}
		fields = append(fields, fmt.Sprint(m["field"]))
	}
	if strings.Join(fields, ",") != "name,extra" {
		t.Errorf("details.field = %v, cần [name extra]", fields)
	}

	// Thiếu name → 422 field name; biên 200 ký tự được, 201 ký tự → 422.
	resp = tDo(t, http.MethodPost, path, tok, `{}`, "Idempotency-Key", tIdemKey(t))
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("thiếu name → %d", resp.StatusCode)
	}
	if f := tFirstField(t, resp); f != "name" {
		t.Errorf("thiếu name: details[0].field = %q", f)
	}
	if resp := tDo(t, http.MethodPost, path, tok, `{"name":"`+strings.Repeat("x", 200)+`"}`,
		"Idempotency-Key", tIdemKey(t)); resp.StatusCode != http.StatusCreated {
		t.Errorf("name 200 ký tự → %d, cần 201", resp.StatusCode)
	}
	resp = tDo(t, http.MethodPost, path, tok, `{"name":"`+strings.Repeat("x", 201)+`"}`, "Idempotency-Key", tIdemKey(t))
	if resp.StatusCode != http.StatusUnprocessableEntity || tFirstField(t, resp) != "name" {
		t.Errorf("name 201 ký tự → %d", resp.StatusCode)
	}

	// Content-Type: text/plain và thiếu → 415; application/json (± charset) được nhận.
	for _, ct := range []string{"text/plain", ""} {
		resp := tDo(t, http.MethodPost, path, tok, `{"name":"ct"}`, "Idempotency-Key", tIdemKey(t), "Content-Type", ct)
		if resp.StatusCode != http.StatusUnsupportedMediaType {
			t.Errorf("Content-Type %q → %d, cần 415", ct, resp.StatusCode)
		}
		if code := tJSON(t, resp)["code"]; code != apierr.UnsupportedMediaType {
			t.Errorf("Content-Type %q: code = %v", ct, code)
		}
	}
	if resp := tDo(t, http.MethodPost, path, tok, `{"name":"ct-ok"}`,
		"Idempotency-Key", tIdemKey(t), "Content-Type", "application/json; charset=utf-8"); resp.StatusCode != http.StatusCreated {
		t.Errorf("application/json; charset=utf-8 → %d, cần 201", resp.StatusCode)
	}
}

func tFirstField(t *testing.T, resp *http.Response) string {
	t.Helper()
	details, _ := tJSON(t, resp)["details"].([]any)
	if len(details) == 0 {
		return ""
	}
	m, _ := details[0].(map[string]any)
	return fmt.Sprint(m["field"])
}

// TestTestRoutes_AllOperations: 15 thao tác của SRS 6.3 trả đúng mã khi gọi qua router thật
// (quyền ẩn danh / đăng nhập / RBAC / guard, 413 trước 422, ETag, 202 việc dài, phát SSE).
func TestTestRoutes_AllOperations(t *testing.T) {
	t.Parallel()
	sub, student := tUser(t)
	admin := tTokenRole(t, sub, auth.RoleAdmin)
	teacher := tTokenRole(t, sub, auth.RoleTeacher)
	it := tNewItem(t, student, "all-ops")
	id := fmt.Sprint(it["id"])

	cases := []struct {
		name, method, path, token, body string
		header                          []string
		want                            int
	}{
		{name: "1 GET items", method: "GET", path: "/items", token: student, want: 200},
		{name: "1 GET items ẩn danh", method: "GET", path: "/items", want: 401},
		{name: "2 POST items", method: "POST", path: "/items", token: student, body: `{"name":"op2"}`,
			header: []string{"Idempotency-Key", tIdemKey(t)}, want: 201},
		{name: "2 POST items thiếu khoá", method: "POST", path: "/items", token: student, body: `{"name":"op2b"}`, want: 422},
		{name: "3 GET items/{id}", method: "GET", path: "/items/" + id, token: student, want: 200},
		{name: "3 GET items/{id} id lạ", method: "GET", path: "/items/" + uuid.NewString(), token: student, want: 404},
		{name: "4 PUT items/{id}", method: "PUT", path: "/items/" + id, token: student,
			body: `{"name":"op4","version":1}`, want: 200},
		{name: "5 GET slow", method: "GET", path: "/slow?ms=1", want: 200},
		{name: "6 GET db-sleep", method: "GET", path: "/db-sleep?seconds=0.05", want: 200},
		{name: "7 GET redis-block", method: "GET", path: "/redis-block?seconds=0.05", want: 200},
		{name: "8 GET panic", method: "GET", path: "/panic", want: 500},
		{name: "9 GET error/409", method: "GET", path: "/error/409", want: 409},
		{name: "10 GET whoami", method: "GET", path: "/whoami", token: student, want: 200},
		{name: "11 GET rbac/admin · ADMIN", method: "GET", path: "/rbac/admin", token: admin, want: 200},
		{name: "11 GET rbac/admin · STUDENT", method: "GET", path: "/rbac/admin", token: student, want: 403},
		{name: "12 GET rbac/staff · TEACHER", method: "GET", path: "/rbac/staff", token: teacher, want: 200},
		{name: "12 GET rbac/staff · STUDENT", method: "GET", path: "/rbac/staff", token: student, want: 403},
		{name: "13 GET courses/{id}/ping", method: "GET", path: "/courses/" + uuid.NewString() + "/ping",
			token: student, want: 403},
		{name: "13 GET courses/{id}/ping ẩn danh", method: "GET", path: "/courses/" + uuid.NewString() + "/ping", want: 401},
		{name: "14 POST jobs", method: "POST", path: "/jobs", token: student, body: `{"steps":2}`, want: 202},
		{name: "14 POST jobs steps lạ", method: "POST", path: "/jobs", token: student, body: `{"steps":101}`, want: 422},
		{name: "15 POST events", method: "POST", path: "/events", token: student,
			body: `{"type":"test.ping","data":{"n":1}}`, want: 202},
		{name: "15 POST events type sai", method: "POST", path: "/events", token: student,
			body: `{"type":"Test Ping","data":{}}`, want: 422},
		{name: "15 POST events user khác", method: "POST", path: "/events", token: student,
			body: `{"type":"test.ping","data":{},"user_id":"` + uuid.NewString() + `"}`, want: 403},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp := tDo(t, c.method, "/api/v1/_test"+c.path, c.token, c.body, c.header...)
			if resp.StatusCode != c.want {
				t.Fatalf("%s %s → %d, cần %d: %s", c.method, c.path, resp.StatusCode, c.want, tBody(t, resp))
			}
		})
	}

	// whoami lấy danh tính từ claim.
	who := tJSON(t, tDo(t, http.MethodGet, "/api/v1/_test/whoami", student, ""))
	if who["sub"] != sub || who["role"] != string(auth.RoleStudent) {
		t.Errorf("whoami = %v", who)
	}

	// 202 việc dài: Location trỏ /api/v1/jobs/<job_id>.
	job := tDo(t, http.MethodPost, "/api/v1/_test/jobs", student, `{"steps":4}`)
	jb := tJSON(t, job)
	if loc := job.Header.Get("Location"); loc != "/api/v1/jobs/"+fmt.Sprint(jb["job_id"]) {
		t.Errorf("Location = %q, job_id = %v", loc, jb["job_id"])
	}

	// 413 chạy TRƯỚC kiểm Idempotency-Key: thân quá lớn mà thiếu khoá vẫn là 413 (SRS 3.1).
	huge := `{"name":"` + strings.Repeat("a", int(tDeps.Cfg.MaxBodyBytes)+10) + `"}`
	if resp := tDo(t, http.MethodPost, "/api/v1/_test/items", student, huge); resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("thân quá lớn + thiếu Idempotency-Key → %d, cần 413", resp.StatusCode)
	}
}
