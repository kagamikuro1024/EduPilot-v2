package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/edupilot/backend-go/internal/platform/clock"
)

// whoami trả [sub, role] của Principal trong context — dùng để chứng minh danh tính không lẫn.
func whoami(w http.ResponseWriter, r *http.Request) {
	p := MustFromContext(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"sub": p.Sub, "role": string(p.Role)})
}

// testRouter dựng router thử: whoami, rbac/admin, rbac/staff, courses/{courseId}/ping.
func testRouter(v *Verifier, resolver CourseResolver) http.Handler {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(Middleware(v))
		r.Get("/whoami", whoami)
		r.With(RequireRole(RoleAdmin)).Get("/rbac/admin", whoami)
		r.With(RequireRole(RoleTeacher, RoleTA)).Get("/rbac/staff", whoami)
		r.With(CourseAccessGuard(resolver)).Get("/courses/{courseId}/ping", func(w http.ResponseWriter, r *http.Request) {
			a, ok := CourseFromContext(r.Context())
			if !ok {
				http.Error(w, "thiếu CourseAccess", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"course_id": a.CourseID, "course_role": string(a.CourseRole)})
		})
	})
	return r
}

func do(t *testing.T, h http.Handler, path, authz string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("thân không phải JSON (%q): %v", rec.Body.String(), err)
	}
	return m
}

func TestRBAC_Matrix(t *testing.T) {
	clk := fixedClock()
	iss := NewIssuer(testSecret, 0, clk)
	h := testRouter(NewVerifier(testSecret, clk), DenyAll{})

	tests := []struct {
		role Role
		path string
		code int
	}{
		{RoleAdmin, "/rbac/admin", http.StatusOK},
		{RoleAdmin, "/rbac/staff", http.StatusForbidden},
		{RoleTeacher, "/rbac/admin", http.StatusForbidden},
		{RoleTeacher, "/rbac/staff", http.StatusOK},
		{RoleTA, "/rbac/admin", http.StatusForbidden},
		{RoleTA, "/rbac/staff", http.StatusOK},
		{RoleStudent, "/rbac/admin", http.StatusForbidden},
		{RoleStudent, "/rbac/staff", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(string(tt.role)+" "+tt.path, func(t *testing.T) {
			tok, err := iss.Issue(testSub, tt.role, testEmail)
			if err != nil {
				t.Fatalf("Issue: %v", err)
			}
			rec := do(t, h, tt.path, "Bearer "+tok)
			if rec.Code != tt.code {
				t.Fatalf("status=%d, muốn %d (thân %s)", rec.Code, tt.code, rec.Body.String())
			}
			body := decodeBody(t, rec)
			if tt.code == http.StatusOK {
				if body["role"] != string(tt.role) || body["sub"] != testSub {
					t.Fatalf("danh tính=%v", body)
				}
				return
			}
			if body["code"] != "FORBIDDEN" {
				t.Fatalf("code=%v, muốn FORBIDDEN", body["code"])
			}
			d, _ := body["details"].(map[string]any)
			if d["reason"] != "role" {
				t.Fatalf("details=%v, muốn reason=role", body["details"])
			}
			if msg, _ := body["message"].(string); strings.Contains(msg, string(tt.role)) {
				t.Fatalf("message nêu vai trò: %q", msg)
			}
		})
	}

	// Ẩn danh ở route RBAC → 401 UNAUTHENTICATED (không phải 403/404)
	for _, p := range []string{"/rbac/admin", "/rbac/staff"} {
		t.Run("ẩn danh "+p, func(t *testing.T) {
			rec := do(t, h, p, "")
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d, muốn 401", rec.Code)
			}
			if c := decodeBody(t, rec)["code"]; c != "UNAUTHENTICATED" {
				t.Fatalf("code=%v, muốn UNAUTHENTICATED", c)
			}
		})
	}
}

func TestMiddleware_Unauthorized(t *testing.T) {
	clk := fixedClock()
	iss := NewIssuer(testSecret, 0, clk)
	h := testRouter(NewVerifier(testSecret, clk), DenyAll{})
	valid, err := iss.Issue(testSub, RoleStudent, testEmail)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	expired, err := NewIssuer(testSecret, -time.Minute, clk).Issue(testSub, RoleStudent, testEmail)
	if err != nil {
		t.Fatalf("Issue hết hạn: %v", err)
	}

	tests := []struct {
		name, authz, code string
	}{
		{"không có header", "", "UNAUTHENTICATED"},
		{"Basic", "Basic dTpw", "UNAUTHENTICATED"},
		{"Bearer rỗng", "Bearer ", "UNAUTHENTICATED"},
		{"chỉ chữ Bearer", "Bearer", "UNAUTHENTICATED"},
		{"token rác", "Bearer garbage", "TOKEN_INVALID"},
		{"token hết hạn", "Bearer " + expired, "TOKEN_EXPIRED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, h, "/whoami", tt.authz)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d, muốn 401", rec.Code)
			}
			body := decodeBody(t, rec)
			if body["code"] != tt.code {
				t.Fatalf("code=%v, muốn %s", body["code"], tt.code)
			}
			if len(body) != 3 || body["message"] == nil || body["trace_id"] == nil {
				t.Fatalf("thân 401 phải có đúng code/message/trace_id, có %v", body)
			}
			www := rec.Header().Get("WWW-Authenticate")
			if !strings.Contains(www, `Bearer realm="edupilot"`) {
				t.Fatalf("WWW-Authenticate=%q", www)
			}
			hasErr := strings.Contains(www, `error="invalid_token"`)
			if want := tt.code != "UNAUTHENTICATED"; hasErr != want {
				t.Fatalf("WWW-Authenticate=%q, error=invalid_token=%v muốn %v", www, hasErr, want)
			}
		})
	}

	// scheme không phân biệt hoa thường (RFC 7235)
	for _, s := range []string{"bearer", "BEARER", "Bearer", "BeArEr"} {
		t.Run("scheme "+s, func(t *testing.T) {
			if rec := do(t, h, "/whoami", s+" "+valid); rec.Code != http.StatusOK {
				t.Fatalf("scheme %q → status=%d, muốn 200", s, rec.Code)
			}
		})
	}
}

// ===== CourseAccessGuard =====

const testCourseA = "00000000-0000-7000-8000-0000000000aa"
const testCourseB = "00000000-0000-7000-8000-0000000000bb"

// fakeResolver cho phép một (sub, course) duy nhất và đếm số lần được gọi.
type fakeResolver struct {
	sub, course string
	err         error
	calls       atomic.Int64
}

func (f *fakeResolver) CanAccess(_ context.Context, p Principal, courseID string) (bool, error) {
	f.calls.Add(1)
	if f.err != nil {
		return false, f.err
	}
	return p.Sub == f.sub && courseID == f.course, nil
}

func guardRequest(t *testing.T, resolver CourseResolver, role Role, sub, courseID string) *httptest.ResponseRecorder {
	t.Helper()
	clk := fixedClock()
	tok, err := NewIssuer(testSecret, 0, clk).Issue(sub, role, testEmail)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	h := testRouter(NewVerifier(testSecret, clk), resolver)
	return do(t, h, "/courses/"+courseID+"/ping", "Bearer "+tok)
}

func TestCourseAccessGuard_DefaultDenyAll(t *testing.T) {
	for _, role := range []Role{RoleAdmin, RoleTeacher, RoleTA, RoleStudent} {
		t.Run(string(role), func(t *testing.T) {
			rec := guardRequest(t, DenyAll{}, role, testSub, testCourseA)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status=%d, muốn 403 (mặc định từ chối kể cả ADMIN)", rec.Code)
			}
			body := decodeBody(t, rec)
			if body["code"] != "FORBIDDEN" {
				t.Fatalf("code=%v", body["code"])
			}
			d, _ := body["details"].(map[string]any)
			if d["reason"] != "course" {
				t.Fatalf("details=%v, muốn reason=course", body["details"])
			}
		})
	}
	// resolver nil cũng phải là deny-all, không panic
	t.Run("resolver nil", func(t *testing.T) {
		if rec := guardRequest(t, nil, RoleAdmin, testSub, testCourseA); rec.Code != http.StatusForbidden {
			t.Fatalf("status=%d, muốn 403", rec.Code)
		}
	})
	// ẩn danh vào route guard → 401 (xác thực chạy trước guard)
	t.Run("ẩn danh", func(t *testing.T) {
		h := testRouter(NewVerifier(testSecret, fixedClock()), DenyAll{})
		rec := do(t, h, "/courses/"+testCourseA+"/ping", "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d, muốn 401", rec.Code)
		}
		if c := decodeBody(t, rec)["code"]; c != "UNAUTHENTICATED" {
			t.Fatalf("code=%v, muốn UNAUTHENTICATED", c)
		}
	})
}

func TestCourseAccessGuard_Membership(t *testing.T) {
	r := &fakeResolver{sub: testSub, course: testCourseA}

	rec := guardRequest(t, r, RoleStudent, testSub, testCourseA)
	if rec.Code != http.StatusOK {
		t.Fatalf("lớp A: status=%d, muốn 200 (thân %s)", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if body["course_id"] != testCourseA || body["course_role"] != string(RoleStudent) {
		t.Fatalf("context CourseAccess=%v", body)
	}

	if rec := guardRequest(t, r, RoleStudent, testSub, testCourseB); rec.Code != http.StatusForbidden {
		t.Fatalf("lớp B: status=%d, muốn 403", rec.Code)
	}
	if rec := guardRequest(t, r, RoleStudent, testSub2, testCourseA); rec.Code != http.StatusForbidden {
		t.Fatalf("người khác, lớp A: status=%d, muốn 403", rec.Code)
	}
	if n := r.calls.Load(); n != 3 {
		t.Fatalf("resolver được gọi %d lần, muốn đúng 1 lần mỗi request (3)", n)
	}
}

func TestCourseAccessGuard_BadID(t *testing.T) {
	r := &fakeResolver{sub: testSub, course: testCourseA}
	for _, id := range []string{
		"abc", "123", "not-a-uuid-at-all",
		"00000000-0000-7000-8000-0000000000a",   // thiếu 1 ký tự
		"00000000-0000-7000-8000-0000000000aaa", // thừa 1 ký tự
	} {
		t.Run(id, func(t *testing.T) {
			rec := guardRequest(t, r, RoleAdmin, testSub, id)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status=%d, muốn 404", rec.Code)
			}
			if c := decodeBody(t, rec)["code"]; c != "NOT_FOUND" {
				t.Fatalf("code=%v, muốn NOT_FOUND", c)
			}
		})
	}
	if n := r.calls.Load(); n != 0 {
		t.Fatalf("resolver được gọi %d lần với id sai, muốn 0", n)
	}
}

func TestCourseAccessGuard_ResolverError(t *testing.T) {
	r := &fakeResolver{sub: testSub, course: testCourseA, err: fmt.Errorf("enrollments tạm hỏng")}
	rec := guardRequest(t, r, RoleStudent, testSub, testCourseA)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, muốn 503 (không cho qua)", rec.Code)
	}
	body := decodeBody(t, rec)
	if body["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("code=%v", body["code"])
	}
	if msg, _ := body["message"].(string); strings.Contains(msg, "enrollments") {
		t.Fatalf("message lộ nội bộ: %q", msg)
	}
}

func TestCourseAccessGuard_NoCache(t *testing.T) {
	r := &fakeResolver{sub: testSub, course: testCourseA}
	clk := fixedClock()
	tok, err := NewIssuer(testSecret, 0, clk).Issue(testSub, RoleStudent, testEmail)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	h := testRouter(NewVerifier(testSecret, clk), r)
	path := "/courses/" + testCourseA + "/ping"

	const n = 5
	for i := range n {
		if rec := do(t, h, path, "Bearer "+tok); rec.Code != http.StatusOK {
			t.Fatalf("lần %d: status=%d, muốn 200", i, rec.Code)
		}
	}
	if got := r.calls.Load(); got != n {
		t.Fatalf("resolver được gọi %d lần trong %d request, muốn %d (không cache)", got, n, n)
	}

	// Quyết định đổi giữa chừng phải có hiệu lực ngay ở request kế tiếp → không cache kết quả cũ.
	r.course = testCourseB
	if rec := do(t, h, path, "Bearer "+tok); rec.Code != http.StatusForbidden {
		t.Fatalf("sau khi resolver đổi: status=%d, muốn 403", rec.Code)
	}
	if got := r.calls.Load(); got != n+1 {
		t.Fatalf("resolver được gọi %d lần, muốn %d", got, n+1)
	}
}

// ===== Danh tính chỉ đi qua context, chạy song song =====

func TestPrincipal_ContextOnly(t *testing.T) {
	if _, ok := FromContext(context.Background()); ok {
		t.Fatal("context trắng mà vẫn có Principal")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("MustFromContext không panic với context trắng")
			}
		}()
		MustFromContext(context.Background())
	}()

	// Hai nhánh context từ cùng một gốc không nhìn thấy danh tính của nhau; gốc không bị đổi.
	root := context.Background()
	a := WithPrincipal(root, Principal{Sub: testSub, Role: RoleAdmin})
	b := WithPrincipal(root, Principal{Sub: testSub2, Role: RoleStudent})
	if pa, _ := FromContext(a); pa.Sub != testSub || pa.Role != RoleAdmin {
		t.Fatalf("nhánh a=%+v", pa)
	}
	if pb, _ := FromContext(b); pb.Sub != testSub2 || pb.Role != RoleStudent {
		t.Fatalf("nhánh b=%+v", pb)
	}
	if _, ok := FromContext(root); ok {
		t.Fatal("context gốc bị thay đổi")
	}

	// Song song: mỗi goroutine chỉ thấy danh tính của chính nó (chạy kèm -race).
	const n = 64
	var wg sync.WaitGroup
	errs := make(chan string, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			want := Principal{Sub: fmt.Sprintf("sub-%d", i), Role: RoleTA, JTI: fmt.Sprintf("jti-%d", i)}
			got, ok := FromContext(WithPrincipal(context.Background(), want))
			if !ok || got != want {
				errs <- fmt.Sprintf("goroutine %d: got=%+v want=%+v", i, got, want)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

func TestAuth_ParallelRequests(t *testing.T) {
	clk := clock.Real{}
	iss := NewIssuer(testSecret, time.Hour, clk)
	h := testRouter(NewVerifier(testSecret, clk), DenyAll{})

	type cred struct {
		sub   string
		role  Role
		token string
	}
	creds := []cred{
		{testSub, RoleAdmin, ""},
		{testSub, RoleTeacher, ""},
		{testSub2, RoleTA, ""},
		{testSub2, RoleStudent, ""},
	}
	for i := range creds {
		tok, err := iss.Issue(creds[i].sub, creds[i].role, testEmail)
		if err != nil {
			t.Fatalf("Issue: %v", err)
		}
		creds[i].token = tok
	}

	const perCred = 25
	var wg sync.WaitGroup
	bad := make(chan string, len(creds)*perCred)
	for _, c := range creds {
		for range perCred {
			wg.Add(1)
			go func() {
				defer wg.Done()
				rec := do(t, h, "/whoami", "Bearer "+c.token)
				if rec.Code != http.StatusOK {
					bad <- fmt.Sprintf("%s/%s: status=%d", c.sub, c.role, rec.Code)
					return
				}
				var got map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					bad <- err.Error()
					return
				}
				if got["sub"] != c.sub || got["role"] != string(c.role) {
					bad <- fmt.Sprintf("lẫn danh tính: got=%v want=%s/%s", got, c.sub, c.role)
				}
			}()
		}
	}
	wg.Wait()
	close(bad)
	for e := range bad {
		t.Error(e)
	}
}

// ===== Không truy DB mỗi request =====

// spanCounter đếm mọi span được mở; otelpgx mở một span cho mỗi câu lệnh SQL, nên đếm = 0
// chứng minh đường xác thực không chạm DB (FR-39).
type spanCounter struct{ n atomic.Int64 }

func (c *spanCounter) OnStart(context.Context, sdktrace.ReadWriteSpan) { c.n.Add(1) }
func (c *spanCounter) OnEnd(sdktrace.ReadOnlySpan)                     {}
func (c *spanCounter) Shutdown(context.Context) error                  { return nil }
func (c *spanCounter) ForceFlush(context.Context) error                { return nil }

func TestAuth_NoDBQueryPerRequest(t *testing.T) {
	counter := &spanCounter{}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(counter))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(prev)
		_ = tp.Shutdown(context.Background())
	})

	clk := fixedClock()
	iss := NewIssuer(testSecret, 0, clk)
	h := testRouter(NewVerifier(testSecret, clk), DenyAll{})

	for _, role := range []Role{RoleAdmin, RoleTeacher, RoleTA, RoleStudent} {
		tok, err := iss.Issue(testSub, role, testEmail)
		if err != nil {
			t.Fatalf("Issue: %v", err)
		}
		for range 5 {
			if rec := do(t, h, "/whoami", "Bearer "+tok); rec.Code != http.StatusOK {
				t.Fatalf("vai %s: status=%d", role, rec.Code)
			}
		}
	}
	if n := counter.n.Load(); n != 0 {
		t.Fatalf("có %d span trong 20 request đã xác thực, muốn 0 (không truy DB)", n)
	}

	// Bộ đếm phải thật sự đếm được — nếu không, phép đo ở trên vô nghĩa.
	_, span := tp.Tracer("test").Start(context.Background(), "mẫu")
	span.End()
	if n := counter.n.Load(); n != 1 {
		t.Fatalf("bộ đếm span hỏng: %d, muốn 1", n)
	}
}

// ===== Không log bí mật =====

func TestAuth_NoSecretsInLogs(t *testing.T) {
	const password = "mat-khau-thu-nghiem-9"
	clk := fixedClock()
	iss := NewIssuer(testSecret, 0, clk)
	tok, err := iss.Issue(testSub, RoleTeacher, testEmail)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	hash, err := HashPassword(password, 4)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Handler ghi log danh tính theo cách khuyến nghị: slog.Any với Principal (LogValue tự che).
	h := chi.NewRouter()
	h.Use(Middleware(NewVerifier(testSecret, clk)))
	h.Get("/whoami", func(w http.ResponseWriter, r *http.Request) {
		p := MustFromContext(r.Context())
		logger.Info("request đã xác thực", "principal", p, "path", r.URL.Path)
		whoami(w, r)
	})

	for range 3 {
		if rec := do(t, h, "/whoami", "Bearer "+tok); rec.Code != http.StatusOK {
			t.Fatalf("status=%d", rec.Code)
		}
	}
	// Các nhánh lỗi cũng không được ghi gì chứa token.
	for _, a := range []string{"Bearer garbage", "Bearer " + tok + "x", ""} {
		do(t, h, "/whoami", a)
	}
	logger.Info("đã băm mật khẩu", "cost", 4, "ok", hash != "")

	out := buf.String()
	jti, _ := jwtPart(t, tok, 1)["jti"].(string)
	forbidden := map[string]string{
		"token đầy đủ":   tok,
		"chữ ký token":   tok[strings.LastIndex(tok, ".")+1:],
		"JWT_SECRET_KEY": testSecret,
		"mật khẩu rõ":    password,
		"hash bcrypt":    hash,
		"jti đầy đủ":     jti,
		"email (PII)":    testEmail,
		"chữ Bearer":     "Bearer ",
	}
	for name, secret := range forbidden {
		if strings.Contains(out, secret) {
			t.Errorf("log chứa %s", name)
		}
	}
	// Vẫn phải đủ thông tin để lần vết: sub, role và jti rút gọn 8 ký tự.
	for _, want := range []string{testSub, string(RoleTeacher), jti[:8]} {
		if !strings.Contains(out, want) {
			t.Errorf("log thiếu %q (không lần vết được)", want)
		}
	}
}
