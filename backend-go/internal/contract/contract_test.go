package contract

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
)

func loadBoth(t *testing.T) (prod, test *Spec) {
	t.Helper()
	ctx := context.Background()
	prod, err := Load(ctx, ProdSpecPath())
	if err != nil {
		t.Fatalf("%v", err)
	}
	test, err = Load(ctx, TestSpecPath())
	if err != nil {
		t.Fatalf("%v", err)
	}
	return prod, test
}

// AC1: hai tệp nạp được, Validate được, cùng một phiên bản `openapi:` và khai báo đủ operationId.
func TestSpec_LoadsAndValidates(t *testing.T) {
	prod, test := loadBoth(t)
	if prod.Doc.OpenAPI != test.Doc.OpenAPI {
		t.Fatalf("hai tệp phải cùng phiên bản openapi: %q ≠ %q", prod.Doc.OpenAPI, test.Doc.OpenAPI)
	}
	for _, s := range []*Spec{prod, test} {
		ids := map[string]string{}
		for _, o := range s.Operations() {
			if o.ID == "" {
				t.Errorf("%s: %s thiếu operationId", s.File, o.Key())
			}
			if prev, dup := ids[o.ID]; dup {
				t.Errorf("operationId %q trùng giữa %s và %s", o.ID, prev, o.Key())
			}
			ids[o.ID] = o.Key()
		}
	}
	// Cả hai tệp là 3.1: cấm từ khoá `nullable` của 3.0 (kin-openapi không bắt, null vẫn lọt — dùng `type: [x, "null"]`).
	for _, p := range []string{ProdSpecPath(), TestSpecPath()} {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "nullable:") {
			t.Errorf("%s dùng `nullable:` (cú pháp 3.0) trong tệp 3.1", p)
		}
	}
	// AC2: openapi.yaml không chứa route thử.
	for _, o := range prod.Operations() {
		if strings.Contains(o.Path, "_test") {
			t.Errorf("openapi.yaml không được chứa route thử: %s", o.Key())
		}
	}
	if hasTestRoutes {
		// 5 thao tác của PG + 13 thao tác của API cấu hình LLM (FEAT-llm-gateway SRS 6.2) + 3 thao tác phiên (US-P2-02) + 3 thao tác đăng ký / xác minh (US-P2-03).
		if n := len(prod.Operations()); n != 24 {
			t.Errorf("openapi.yaml: %d thao tác (cần 24)", n)
		}
		// 15 thao tác của PG + 3 route thử của cổng LLM (FEAT-llm-gateway SRS 6.4: chat, stats, fake).
		if n := len(test.Operations()); n != 18 {
			t.Errorf("openapi.test.yaml: %d thao tác (cần 18)", n)
		}
	}
}

// AC4: route trong mã ⇄ spec phải khớp hai chiều; thông báo nêu `METHOD /đường-dẫn` lệch.
func TestRouteSpecParity(t *testing.T) {
	r := getRig(t)
	code, err := Routes(r.handle)
	if err != nil {
		t.Fatalf("liệt kê route: %v", err)
	}
	prod, test := loadBoth(t)

	check := func(name string, inCode, inSpec []string) {
		for _, x := range Diff(inCode, inSpec) {
			t.Errorf("[%s] route có trong mã nhưng THIẾU trong spec: %s", name, x)
		}
		for _, x := range Diff(inSpec, inCode) {
			t.Errorf("[%s] route có trong spec nhưng KHÔNG có trong mã: %s", name, x)
		}
	}
	check("openapi.yaml", CodeRoutes(code, false), SpecRoutes(prod, false))
	if hasTestRoutes {
		check("openapi.test.yaml", CodeRoutes(code, true), SpecRoutes(test, true))
	} else if got := CodeRoutes(code, true); len(got) != 0 {
		t.Errorf("bản dựng mặc định không được có route thử, nhưng có: %v", got)
	}
}

// AC4 (status): status thật không có trong spec → đỏ, nêu thao tác và status.
func TestStatusParity(t *testing.T) {
	for _, o := range observations(t) {
		if o.Op != "" && !o.Declared {
			t.Errorf("%s trả status %d nhưng spec KHÔNG khai báo (gọi %s)", o.Op, o.Status, o.Requested)
		}
	}
}

// AC3: mọi response thật khớp spec (status, trường bắt buộc, kiểu, header bắt buộc, hình dạng lỗi và phân trang).
func TestContract_ResponsesMatchSpec(t *testing.T) {
	obs := observations(t)
	if len(obs) < 10 {
		t.Fatalf("chỉ có %d quan sát — kịch bản không chạy đủ", len(obs))
	}
	for _, o := range obs {
		if o.Err != nil {
			t.Errorf("%s (status %d): %v", o.Requested, o.Status, o.Err)
		}
	}
}

// AC5: components.responses có đủ 8 response dùng chung và mọi response lỗi dùng schema Error.
func TestSpec_ErrorResponsesDeclared(t *testing.T) {
	prod, test := loadBoth(t)
	for _, s := range []*Spec{prod, test} {
		for _, name := range []string{"Unauthorized", "Forbidden", "NotFound", "Conflict", "ValidationFailed", "RateLimited", "Unavailable", "DeadlineExceeded"} {
			ref := s.Doc.Components.Responses[name]
			if ref == nil || ref.Value == nil {
				t.Errorf("%s: thiếu components.responses.%s", s.File, name)
				continue
			}
			assertErrorSchema(t, s.File+" components.responses."+name, ref.Value)
		}
		for _, o := range s.Operations() {
			for _, st := range o.Statuses() {
				if st < 400 {
					continue
				}
				rr := o.Op.Responses.Map()[itoa(st)]
				if rr == nil || rr.Value == nil {
					t.Errorf("%s %d: response rỗng", o.Key(), st)
					continue
				}
				assertErrorSchema(t, o.Key()+" "+itoa(st), rr.Value)
			}
		}
	}
}

func assertErrorSchema(t *testing.T, where string, r *openapi3.Response) {
	t.Helper()
	mt := r.Content.Get("application/json")
	if mt == nil || mt.Schema == nil || !strings.HasSuffix(mt.Schema.Ref, "/schemas/Error") {
		t.Errorf("%s: response lỗi phải $ref schemas/Error", where)
	}
}

// AC5: mọi (thao tác, status) đã khai báo đều được gọi thật, trừ danh sách miễn trừ có lý do.
func TestSpec_EveryDocumentedStatusExercised(t *testing.T) {
	prod, test := loadBoth(t)
	seen := map[string]bool{}
	for _, o := range observations(t) {
		seen[o.Op+"|"+itoa(o.Status)] = true
	}
	specs := []*Spec{prod}
	if hasTestRoutes {
		specs = append(specs, test)
	}
	for _, s := range specs {
		for _, o := range s.Operations() {
			for _, st := range o.Statuses() {
				if !seen[o.Key()+"|"+itoa(st)] && !IsExempt(o.Key(), st) {
					t.Errorf("%s khai báo status %d nhưng không test nào sinh ra (thêm kịch bản hoặc miễn trừ có lý do ở exempt.go)", o.Key(), st)
				}
			}
		}
	}
	for _, e := range Exemptions() {
		if strings.TrimSpace(e.Reason) == "" || e.Operation == "" || e.Status == 0 {
			t.Errorf("miễn trừ %+v thiếu operation / status / reason", e)
		}
	}
}

// AC6 (đối chứng âm): validator PHẢI bắt được response sai.
func TestValidator_RejectsBadResponses(t *testing.T) {
	_, test := loadBoth(t)
	ctx := context.Background()
	hdr := func(kv ...string) http.Header {
		h := http.Header{"Content-Type": {"application/json; charset=utf-8"}}
		for i := 0; i+1 < len(kv); i += 2 {
			h.Set(kv[i], kv[i+1])
		}
		return h
	}
	list := func() *http.Request {
		return httptest.NewRequest(http.MethodGet, "http://localhost/api/v1/_test/items", nil)
	}
	good := `{"items":[],"next_cursor":null}`
	goodErr := `{"code":"NOT_FOUND","message":"Không tìm thấy.","trace_id":"0123456789abcdef0123456789abcdef"}`

	if err := test.ValidateResponse(ctx, list(), 200, hdr("ETag", `W/"abc"`), []byte(good), ValidateOpts{}); err != nil {
		t.Fatalf("đối chứng dương phải qua: %v", err)
	}
	if err := test.ValidateResponse(ctx, list(), 401, hdr(), []byte(strings.Replace(goodErr, "NOT_FOUND", "UNAUTHENTICATED", 1)), ValidateOpts{}); err != nil {
		t.Fatalf("lỗi đúng hình dạng phải qua: %v", err)
	}
	bad := []struct {
		name   string
		status int
		hdr    http.Header
		body   string
	}{
		{"items null", 200, hdr("ETag", `W/"abc"`), `{"items":null,"next_cursor":null}`},
		{"thiếu next_cursor", 200, hdr("ETag", `W/"abc"`), `{"items":[]}`},
		{"khoá lạ ngoài hợp đồng", 200, hdr("ETag", `W/"abc"`), `{"items":[],"next_cursor":null,"total":0}`},
		{"thiếu header ETag bắt buộc", 200, hdr(), good},
		{"code ngoài danh sách", 401, hdr(), strings.Replace(goodErr, "NOT_FOUND", "LOI_LA", 1)},
		{"lỗi thiếu trace_id", 401, hdr(), `{"code":"UNAUTHENTICATED","message":"x"}`},
		{"trace_id sai định dạng", 401, hdr(), strings.Replace(goodErr, "0123456789abcdef0123456789abcdef", "khong-phai-hex", 1)},
		{"status không khai báo", 418, hdr(), goodErr},
	}
	for _, c := range bad {
		if err := test.ValidateResponse(ctx, list(), c.status, c.hdr, []byte(c.body), ValidateOpts{}); err == nil {
			t.Errorf("validator KHÔNG bắt được response sai: %s", c.name)
		}
	}
}

// Góp ý #3(a): `format: uuid` được kiểm thật (RFC 9562, nhận cả UUIDv7); id sai định dạng bị bắt.
func TestValidator_UUIDFormat(t *testing.T) {
	prod, _ := loadBoth(t)
	req := httptest.NewRequest(http.MethodGet, "http://localhost/api/v1/jobs/"+uuid.NewString(), nil)
	h := http.Header{"Content-Type": {"application/json; charset=utf-8"}}
	job := func(id string) []byte {
		return []byte(`{"id":"` + id + `","kind":"k","status":"QUEUED","progress":0,"created_at":"2026-10-02T00:00:00Z","updated_at":"2026-10-02T00:00:00Z"}`)
	}
	v7, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	for _, ok := range []string{v7.String(), uuid.NewString()} {
		if err := prod.ValidateResponse(context.Background(), req, 200, h, job(ok), ValidateOpts{}); err != nil {
			t.Errorf("uuid hợp lệ %s bị từ chối: %v", ok, err)
		}
	}
	for _, bad := range []string{"x", "not-a-uuid", "0192a3b4-c5d6-7e8f-9a0b-1c2d3e4f5a6", ""} {
		if err := prod.ValidateResponse(context.Background(), req, 200, h, job(bad), ValidateOpts{}); err == nil {
			t.Errorf("Job.id %q sai định dạng uuid mà validator không bắt", bad)
		}
	}
}

// AC7: security khai đúng.
func TestSpec_SecurityDeclared(t *testing.T) {
	prod, test := loadBoth(t)
	want := map[string]bool{ // thao tác → cần bearerAuth
		"GET /healthz": false, "GET /api/v1/healthz": false, "GET /api/v1/readyz": false,
		"GET /api/v1/jobs/{id}": true, "GET /api/v1/events": true,
		"GET /api/v1/admin/llm/providers": true, "POST /api/v1/admin/llm/providers": true,
		"POST /api/v1/admin/llm/providers/test": true, "PUT /api/v1/admin/llm/providers/{id}": true,
		"DELETE /api/v1/admin/llm/providers/{id}": true, "POST /api/v1/admin/llm/providers/{id}/test": true,
		"GET /api/v1/admin/llm/routes": true, "PUT /api/v1/admin/llm/routes": true, "GET /api/v1/admin/llm/usage": true,
		"GET /api/v1/admin/llm/budget": true, "PUT /api/v1/admin/llm/budget": true,
		"GET /api/v1/courses/{id}/llm-budget": true, "PUT /api/v1/courses/{id}/llm-budget": true,
		"POST /api/v1/auth/login": false, "POST /api/v1/auth/refresh": false, "POST /api/v1/auth/logout": false,
		"POST /api/v1/auth/register": false, "POST /api/v1/auth/verify-email": false, "POST /api/v1/auth/resend-verification": false,
	}
	for _, o := range prod.Operations() {
		need, ok := want[o.Key()]
		if !ok {
			t.Errorf("thao tác lạ trong openapi.yaml: %s", o.Key())
			continue
		}
		if got := prod.RequiresAuth(o); got != need {
			t.Errorf("%s: bearerAuth = %v (cần %v)", o.Key(), got, need)
		}
		if need && !o.Declared(401) {
			t.Errorf("%s cần bearerAuth nhưng không khai 401", o.Key())
		}
		if !need && o.Op.Security == nil {
			t.Errorf("%s phải khai `security: []` tường minh", o.Key())
		}
	}
	for _, o := range test.Operations() {
		if strings.HasPrefix(o.Path, "/api/v1/_test/rbac/") && !o.Declared(403) {
			t.Errorf("%s: route chỉ vai trò phải khai 403", o.Key())
		}
		if test.RequiresAuth(o) && !o.Declared(401) {
			t.Errorf("%s: cần bearerAuth nhưng không khai 401", o.Key())
		}
	}
}

var pathSamples = map[string]string{"id": "", "courseId": "", "status": "400"}

func samplePath(p string) string {
	out := p
	for k, v := range pathSamples {
		if v == "" {
			v = uuid.NewString()
		}
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}

// AC7 (thật): gọi KHÔNG token vào thao tác có bearerAuth → 401 đúng như spec; thao tác công khai → không 401.
func TestContract_UnauthenticatedMatchesSpec(t *testing.T) {
	r := getRig(t)
	prod, test := loadBoth(t)
	specs := []*Spec{prod}
	if hasTestRoutes {
		specs = append(specs, test)
	}
	for _, s := range specs {
		for _, o := range s.Operations() {
			if o.Path == "/api/v1/events" {
				req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, r.srv.URL+o.Path, nil)
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatalf("%s: %v", o.Key(), err)
				}
				_ = resp.Body.Close()
				if resp.StatusCode != http.StatusUnauthorized {
					t.Errorf("%s không token → %d (cần 401)", o.Key(), resp.StatusCode)
				}
				continue
			}
			path := samplePath(o.Path)
			var body *strings.Reader
			if o.Method == http.MethodPost || o.Method == http.MethodPut {
				body = strings.NewReader(`{}`)
			} else {
				body = strings.NewReader("")
			}
			req, _ := http.NewRequestWithContext(context.Background(), o.Method, r.srv.URL+path, body)
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("%s: %v", o.Key(), err)
			}
			_ = resp.Body.Close()
			needs := s.RequiresAuth(o)
			if o.Key() == "POST /api/v1/auth/refresh" {
				// Công khai (không Bearer) nhưng cần cookie `ep_rt`: thiếu cookie → 401 UNAUTHENTICATED là đúng hợp đồng (US-P2-02 AC5).
				continue
			}
			switch {
			case needs && resp.StatusCode != http.StatusUnauthorized:
				t.Errorf("%s có bearerAuth nhưng không token → %d (cần 401)", o.Key(), resp.StatusCode)
			case !needs && resp.StatusCode == http.StatusUnauthorized:
				t.Errorf("%s khai công khai nhưng trả 401", o.Key())
			}
		}
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
