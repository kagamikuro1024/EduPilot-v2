package contract

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/google/uuid"
)

// Observation là kết quả MỘT lần gọi thật: thao tác trong spec, status thật và (nếu có) lý do response không khớp spec.
type Observation struct {
	Spec      string // "prod" | "test"
	Op        string // "METHOD /mẫu-đường-dẫn" của spec; rỗng nếu spec không khai báo route đó
	Requested string // "METHOD /đường-dẫn-thật"
	Status    int
	Declared  bool
	Err       error
}

// call mô tả một request thử.
type call struct {
	method, path string
	token        string
	headers      map[string]string
	body         string
	ctype        string // mặc định application/json khi có body
}

type runner struct {
	t     *testing.T
	rig   *rig
	prod  *Spec
	test  *Spec
	mu    sync.Mutex
	obs   []Observation
	cache map[string]string // giá trị dùng lại giữa các bước (id, version…)
}

var (
	scenOnce sync.Once
	scenObs  []Observation
	scenErr  error
)

// observations chạy toàn bộ kịch bản đúng một lần cho cả gói test và trả mọi quan sát.
func observations(t *testing.T) []Observation {
	t.Helper()
	scenOnce.Do(func() {
		r := getRig(t)
		ctx := context.Background()
		prod, err := Load(ctx, ProdSpecPath())
		if err != nil {
			scenErr = err
			return
		}
		var test *Spec
		if hasTestRoutes {
			if test, err = Load(ctx, TestSpecPath()); err != nil {
				scenErr = err
				return
			}
		}
		run := &runner{t: t, rig: r, prod: prod, test: test, cache: map[string]string{}}
		run.prodScenarios()
		if hasTestRoutes {
			run.testScenarios()
		}
		scenObs = run.obs
	})
	if scenErr != nil {
		t.Fatalf("kịch bản hợp đồng: %v", scenErr)
	}
	return scenObs
}

func (r *runner) specFor(path string) (*Spec, string) {
	if strings.HasPrefix(path, TestPathPrefix) {
		return r.test, "test"
	}
	return r.prod, "prod"
}

func (r *runner) request(c call, base string) (*http.Request, error) {
	var body io.Reader
	if c.body != "" {
		body = strings.NewReader(c.body)
	}
	req, err := http.NewRequestWithContext(context.Background(), c.method, base+c.path, body)
	if err != nil {
		return nil, err
	}
	if c.body != "" {
		ct := c.ctype
		if ct == "" {
			ct = "application/json"
		}
		req.Header.Set("Content-Type", ct)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

// do gọi qua server HTTP thật, ghi quan sát và trả (status, header, thân).
func (r *runner) do(c call) (int, http.Header, []byte) {
	r.t.Helper()
	req, err := r.request(c, r.rig.srv.URL)
	if err != nil {
		r.t.Fatalf("dựng request %s %s: %v", c.method, c.path, err)
	}
	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose // đóng ngay bên dưới
	if err != nil {
		r.t.Fatalf("%s %s: %v", c.method, c.path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	r.record(c, resp.StatusCode, resp.Header, b, false)
	return resp.StatusCode, resp.Header, b
}

// doHandler gọi thẳng vào một handler (router có State riêng để thử readyz khi đang tắt).
func (r *runner) doHandler(h http.Handler, c call) int {
	r.t.Helper()
	req, err := r.request(c, "http://localhost")
	if err != nil {
		r.t.Fatalf("dựng request: %v", err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	defer func() { _ = res.Body.Close() }()
	b, _ := io.ReadAll(res.Body)
	r.record(c, res.StatusCode, res.Header, b, false)
	return res.StatusCode
}

func (r *runner) record(c call, status int, h http.Header, body []byte, skipBody bool) {
	spec, name := r.specFor(c.path)
	o := Observation{Spec: name, Requested: c.method + " " + c.path, Status: status}
	if spec == nil {
		o.Err = fmt.Errorf("không nạp được spec %q", name)
		r.add(o)
		return
	}
	req, err := r.request(c, "http://localhost")
	if err != nil {
		o.Err = err
		r.add(o)
		return
	}
	op, ok := spec.Find(req)
	if !ok {
		o.Err = fmt.Errorf("spec không khai báo %s", o.Requested)
		r.add(o)
		return
	}
	o.Op = op.Key()
	o.Declared = op.Declared(status)
	o.Err = spec.ValidateResponse(context.Background(), req, status, h, body, ValidateOpts{SkipBody: skipBody})
	r.add(o)
}

func (r *runner) add(o Observation) {
	r.mu.Lock()
	r.obs = append(r.obs, o)
	r.mu.Unlock()
}

func (r *runner) must(c call, want int) (http.Header, []byte) {
	r.t.Helper()
	st, h, b := r.do(c)
	if st != want {
		r.t.Fatalf("%s %s → %d (cần %d): %s", c.method, c.path, st, want, b)
	}
	return h, b
}

// stream mở GET /api/v1/events và đọc tới hết header; trả status, header và hàm đóng.
func (r *runner) stream(token string) (int, http.Header, func()) {
	r.t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, r.rig.srv.URL+"/api/v1/events", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		r.t.Fatalf("mở SSE: %v", err)
	}
	closeFn := func() { cancel(); _ = resp.Body.Close() }
	if resp.StatusCode == http.StatusOK {
		// đọc tới khi thấy `event: ready` để chắc chắn khe kết nối đã được giữ
		buf := make([]byte, 512)
		deadline := time.Now().Add(5 * time.Second)
		var got []byte
		for time.Now().Before(deadline) && !bytes.Contains(got, []byte("event: ready")) {
			n, err := resp.Body.Read(buf)
			got = append(got, buf[:n]...)
			if err != nil {
				break
			}
		}
		r.record(call{method: http.MethodGet, path: "/api/v1/events"}, resp.StatusCode, resp.Header, nil, true)
		return resp.StatusCode, resp.Header, closeFn
	}
	b, _ := io.ReadAll(resp.Body)
	r.record(call{method: http.MethodGet, path: "/api/v1/events"}, resp.StatusCode, resp.Header, b, false)
	return resp.StatusCode, resp.Header, closeFn
}

// prodScenarios gọi thật MỌI (thao tác, status) đã khai báo trong openapi.yaml.
func (r *runner) prodScenarios() {
	t := r.t
	owner := uuid.NewString()
	tok := r.rig.token(t, owner, auth.RoleStudent)

	r.must(call{method: "GET", path: "/healthz"}, 200)
	r.must(call{method: "GET", path: "/api/v1/healthz"}, 200)
	r.must(call{method: "GET", path: "/api/v1/readyz"}, 200)

	// readyz 503: một router riêng đang tắt (State.Drain) — không đụng router dùng chung.
	d2 := r.rig.deps
	d2.State = httpapi.NewState()
	d2.State.Drain()
	if st := r.doHandler(httpapi.NewRouter(d2), call{method: "GET", path: "/api/v1/readyz"}); st != http.StatusServiceUnavailable {
		t.Fatalf("readyz khi tắt → %d (cần 503)", st)
	}

	// jobs: 200 (job vừa tạo bằng service thật), 401, 404.
	j, err := r.rig.deps.Jobs.Enqueue(context.Background(), uuid.MustParse(owner), "test.progress", map[string]int{"steps": 1})
	if err != nil {
		t.Fatalf("tạo job: %v", err)
	}
	r.must(call{method: "GET", path: "/api/v1/jobs/" + j.ID.String(), token: tok}, 200)
	r.must(call{method: "GET", path: "/api/v1/jobs/" + j.ID.String()}, 401)
	r.must(call{method: "GET", path: "/api/v1/jobs/" + uuid.NewString(), token: tok}, 404)

	// events: 401 không token; 200; 429 khi mở khe thứ ba.
	if st, _, c := r.stream(""); st != http.StatusUnauthorized {
		t.Fatalf("events không token → %d", st)
	} else {
		c()
	}
	sseTok := r.rig.token(t, uuid.NewString(), auth.RoleStudent)
	var closers []func()
	for i := range 2 {
		st, _, c := r.stream(sseTok)
		closers = append(closers, c)
		if st != http.StatusOK {
			t.Fatalf("events lần %d → %d (cần 200)", i+1, st)
		}
	}
	st, h, c := r.stream(sseTok)
	c()
	if st != http.StatusTooManyRequests || h.Get("Retry-After") == "" {
		t.Fatalf("events khe thứ ba → %d Retry-After=%q (cần 429 kèm Retry-After)", st, h.Get("Retry-After"))
	}
	for _, c := range closers {
		c()
	}
	r.llmScenarios()
	r.authScenarios()
}

// authScenarios gọi 3 thao tác phiên (FEAT-account-security US-P2-02) với mọi status đã khai báo.
func (r *runner) authScenarios() {
	const password = "Edupilot#2026-demo"
	ctx := context.Background()
	hash, err := auth.HashPassword(password, 4)
	if err != nil {
		r.t.Fatalf("băm mật khẩu: %v", err)
	}
	mk := func(email, status string) {
		if _, err := r.rig.deps.DB.Exec(ctx,
			`insert into users (email, full_name, role, status, password_hash) values ($1, 'Người Thử', 'STUDENT', $2::user_status, $3)`,
			email, status, hash); err != nil {
			r.t.Fatalf("tạo người dùng: %v", err)
		}
	}
	email, off := "ct-"+uuid.NewString()[:8]+"@example.test", "ct-off-"+uuid.NewString()[:8]+"@example.test"
	mk(email, "ACTIVE")
	mk(off, "DISABLED")
	origin := map[string]string{"Origin": "https://localhost"}
	login := func(e, p string, want int) (http.Header, []byte) {
		return r.must(call{method: "POST", path: "/api/v1/auth/login", headers: origin, body: `{"email":"` + e + `","password":"` + p + `"}`}, want)
	}
	login(email, "sai-mat-khau-1", 401)
	login(off, password, 403)
	r.must(call{method: "POST", path: "/api/v1/auth/login", body: `{"email":"` + email + `"}`}, 422)
	h, _ := login(email, password, 200)
	rt := strings.TrimPrefix(strings.SplitN(h.Get("Set-Cookie"), ";", 2)[0], "ep_rt=")
	cookie := func(v string) map[string]string {
		return map[string]string{"Origin": "https://localhost", "Cookie": "ep_rt=" + v}
	}

	r.must(call{method: "POST", path: "/api/v1/auth/refresh", headers: map[string]string{"Origin": "https://evil.example", "Cookie": "ep_rt=" + rt}}, 403)
	r.must(call{method: "POST", path: "/api/v1/auth/refresh", headers: origin}, 401)
	h, _ = r.must(call{method: "POST", path: "/api/v1/auth/refresh", headers: cookie(rt)}, 200)
	rt2 := strings.TrimPrefix(strings.SplitN(h.Get("Set-Cookie"), ";", 2)[0], "ep_rt=")
	r.must(call{method: "POST", path: "/api/v1/auth/logout", headers: map[string]string{"Origin": "https://evil.example"}}, 403)
	r.must(call{method: "POST", path: "/api/v1/auth/logout", headers: cookie(rt2)}, 204)
	r.accountScenarios()
	r.passwordScenarios()
}

// accountScenarios: register / verify-email / resend-verification (US-P2-03) với mọi status đã khai báo.
func (r *runner) accountScenarios() {
	ctx := context.Background()
	email := "ct-reg-" + uuid.NewString()[:8] + "@example.test"
	body := `{"email":"` + email + `","password":"Edupilot#2026-demo","full_name":"Người Thử"}`
	r.must(call{method: "POST", path: "/api/v1/auth/register", body: body}, 202)
	r.must(call{method: "POST", path: "/api/v1/auth/register", body: `{"email":"a@b","password":"x","full_name":""}`}, 422)
	r.must(call{method: "POST", path: "/api/v1/auth/register", body: `{"email":"` + email + `","password":"Edupilot#2026-demo","full_name":"` + strings.Repeat("a", 5000) + `"}`}, 413)

	var uid uuid.UUID
	if err := r.rig.deps.DB.QueryRow(ctx, `select id from users where email = $1`, email).Scan(&uid); err != nil {
		r.t.Fatalf("tra người dùng vừa đăng ký: %v", err)
	}
	tx, err := r.rig.deps.DB.Begin(ctx)
	if err != nil {
		r.t.Fatal(err)
	}
	tok, err := auth.Tokens{Clock: r.rig.deps.Clock}.Issue(ctx, tx, uid, auth.TokenVerifyEmail, time.Hour, nil)
	if err != nil || tx.Commit(ctx) != nil {
		r.t.Fatalf("phát token xác minh: %v", err)
	}
	r.must(call{method: "POST", path: "/api/v1/auth/verify-email", body: `{"token":""}`}, 422)
	r.must(call{method: "POST", path: "/api/v1/auth/verify-email", body: `{"token":"` + tok + `"}`}, 200)
	r.must(call{method: "POST", path: "/api/v1/auth/verify-email", body: `{"token":"` + tok + `"}`}, 410)

	pending := "ct-rs-" + uuid.NewString()[:8] + "@example.test"
	r.must(call{method: "POST", path: "/api/v1/auth/resend-verification", body: `{}`}, 422)
	r.must(call{method: "POST", path: "/api/v1/auth/resend-verification", body: `{"email":"` + pending + `"}`}, 202)
	r.must(call{method: "POST", path: "/api/v1/auth/resend-verification", body: `{"email":"` + pending + `"}`}, 429)
}

// llmScenarios gọi 13 thao tác cấu hình LLM (FEAT-llm-gateway US-P1-04) với mọi status đã khai báo, trên provider `fake`.
// Dọn bảng cấu hình ở cuối để không ảnh hưởng kịch bản khác dùng chung gateway.
func (r *runner) llmScenarios() {
	t := r.t
	admin := r.rig.token(t, uuid.NewString(), auth.RoleAdmin)
	teacher := r.rig.token(t, uuid.NewString(), auth.RoleTeacher)
	ta := r.rig.token(t, uuid.NewString(), auth.RoleTA)
	ctl := r.rig.deps.LLM.Registry.Fake()
	old := ctl.Get()
	ctl.Set(fake.Settings{ValidKey: "good-key"})
	defer func() {
		ctl.Set(old)
		ctx := context.Background()
		if _, err := r.rig.deps.DB.Exec(ctx, `truncate llm_task_routes, llm_models, llm_providers; delete from llm_budgets`); err != nil {
			t.Errorf("dọn cấu hình LLM: %v", err)
		}
		_ = r.rig.deps.LLM.Registry.Reload(ctx)
	}()
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": "ct-" + uuid.NewString()} }
	const (
		base = "/api/v1/admin/llm"
		prov = base + "/providers"
	)
	provBody := func(name, key string) string {
		return `{"type":"fake","name":"` + name + `","api_key":"` + key + `","models":[` +
			`{"model":"fake-chat","kind":"chat","price_in":"0","price_out":"0"},` +
			`{"model":"fake-embed","kind":"embedding","dims":1536,"price_in":"0","price_out":"0"}]}`
	}
	type model struct{ ID, Kind string }
	type provResp struct {
		ID      string  `json:"id"`
		Version int     `json:"version"`
		Models  []model `json:"models"`
	}
	parse := func(b []byte) provResp {
		var p provResp
		if err := json.Unmarshal(b, &p); err != nil {
			t.Fatalf("giải mã nhà cung cấp: %v (%s)", err, b)
		}
		return p
	}

	// POST providers: 201 / 401 / 403 / 409 (trùng tên) / 422 (khoá sai không lưu).
	_, b := r.must(call{method: "POST", path: prov, token: admin, headers: idem(), body: provBody("CT-A", "good-key")}, 201)
	a := parse(b)
	var chatID, embID string
	for _, m := range a.Models {
		if m.Kind == "chat" {
			chatID = m.ID
		} else {
			embID = m.ID
		}
	}
	r.must(call{method: "POST", path: prov, headers: idem(), body: provBody("CT-X", "good-key")}, 401)
	r.must(call{method: "POST", path: prov, token: teacher, headers: idem(), body: provBody("CT-X", "good-key")}, 403)
	r.must(call{method: "POST", path: prov, token: admin, headers: idem(), body: provBody("CT-A", "good-key")}, 409)
	r.must(call{method: "POST", path: prov, token: admin, headers: idem(), body: provBody("CT-B", "bad-key")}, 422)

	// GET providers
	r.must(call{method: "GET", path: prov, token: admin}, 200)
	r.must(call{method: "GET", path: prov, token: teacher}, 200)
	r.must(call{method: "GET", path: prov}, 401)
	r.must(call{method: "GET", path: prov, token: ta}, 403)

	// PUT providers/{id}: 200 / 401 / 403 / 404 / 409 / 422
	one := prov + "/" + a.ID
	r.must(call{method: "PUT", path: one, token: admin, body: `{"type":"fake","name":"CT-A2","version":1}`}, 200)
	r.must(call{method: "PUT", path: one, token: admin, body: `{"type":"fake","name":"CT-A3","version":1}`}, 409)
	r.must(call{method: "PUT", path: one, body: `{"type":"fake","name":"CT-A3","version":2}`}, 401)
	r.must(call{method: "PUT", path: one, token: teacher, body: `{"type":"fake","name":"CT-A3","version":2}`}, 403)
	r.must(call{method: "PUT", path: prov + "/" + uuid.NewString(), token: admin, body: `{"type":"fake","name":"CT-A3","version":1}`}, 404)
	r.must(call{method: "PUT", path: one, token: admin, body: `{"type":"fake","name":"CT-A3","base_url":"file:///etc/passwd","version":2}`}, 422)

	// POST providers/{id}/test và providers/test: 200 / 401 / 403 / 404 / 422 / 429
	r.must(call{method: "POST", path: one + "/test", token: admin}, 200)
	r.must(call{method: "POST", path: one + "/test", token: admin, body: `{"api_key":"wrong"}`}, 200)
	r.must(call{method: "POST", path: one + "/test"}, 401)
	r.must(call{method: "POST", path: one + "/test", token: teacher}, 403)
	r.must(call{method: "POST", path: prov + "/" + uuid.NewString() + "/test", token: admin}, 404)
	r.must(call{method: "POST", path: one + "/test", token: admin, body: `{"base_url":"http://u:p@h/"}`}, 422)
	r.must(call{method: "POST", path: prov + "/test", token: admin, body: `{"type":"fake","api_key":"bad-key","model":"fake-chat"}`}, 200)
	r.must(call{method: "POST", path: prov + "/test", token: admin, body: `{"type":"fake","api_key":"good-key","model":"fake-chat"}`}, 200)
	r.must(call{method: "POST", path: prov + "/test", body: `{}`}, 401)
	r.must(call{method: "POST", path: prov + "/test", token: teacher, body: `{}`}, 403)
	r.must(call{method: "POST", path: prov + "/test", token: admin, body: `{"type":"openai_compatible","base_url":"file:///x","api_key":"x","model":"m"}`}, 422)
	for _, path := range []string{prov + "/test", one + "/test"} { // giới hạn 10 lần / phút / người
		spammer := r.rig.token(t, uuid.NewString(), auth.RoleAdmin)
		body := `{"type":"fake","api_key":"good-key","model":"fake-chat"}`
		if path == one+"/test" {
			body = ""
		}
		for range 10 {
			r.must(call{method: "POST", path: path, token: spammer, body: body}, 200)
		}
		r.must(call{method: "POST", path: path, token: spammer, body: body}, 429)
	}

	// routes: GET 200/401/403; PUT 200/401/403/409/422
	routes := base + "/routes"
	r.must(call{method: "GET", path: routes, token: admin}, 200)
	r.must(call{method: "GET", path: routes}, 401)
	r.must(call{method: "GET", path: routes, token: ta}, 403)
	r.must(call{method: "PUT", path: routes, token: admin, body: `{"task":"CHAT","chain":["` + chatID + `"],"params":{"temperature":0.2},"version":0}`}, 200)
	r.must(call{method: "PUT", path: routes, token: admin, body: `{"task":"EMBEDDING","chain":["` + embID + `"],"version":0}`}, 200)
	r.must(call{method: "PUT", path: routes, token: admin, body: `{"task":"CHAT","chain":["` + chatID + `"],"version":0}`}, 409)
	r.must(call{method: "PUT", path: routes, token: admin, body: `{"task":"CLASSIFY","chain":[],"version":0}`}, 422)
	r.must(call{method: "PUT", path: routes, body: `{}`}, 401)
	r.must(call{method: "PUT", path: routes, token: teacher, body: `{}`}, 403)

	// usage
	usage := base + "/usage"
	r.must(call{method: "GET", path: usage, token: admin}, 200)
	r.must(call{method: "GET", path: usage + "?group=day", token: teacher}, 200)
	r.must(call{method: "GET", path: usage}, 401)
	r.must(call{method: "GET", path: usage, token: ta}, 403)
	r.must(call{method: "GET", path: usage + "?group=week", token: admin}, 422)

	// budget hệ thống và theo lớp
	for _, bp := range []string{base + "/budget", "/api/v1/courses/" + uuid.NewString() + "/llm-budget"} {
		r.must(call{method: "GET", path: bp, token: admin}, 200)
		r.must(call{method: "GET", path: bp}, 401)
		r.must(call{method: "GET", path: bp, token: ta}, 403)
		r.must(call{method: "PUT", path: bp, token: admin, body: `{"daily_limit":"1000","monthly_limit":"30000","version":0}`}, 200)
		r.must(call{method: "PUT", path: bp, token: admin, body: `{"daily_limit":"1000","monthly_limit":"30000","version":0}`}, 409)
		r.must(call{method: "PUT", path: bp, token: admin, body: `{"daily_limit":"-1","version":1}`}, 422)
		r.must(call{method: "PUT", path: bp, body: `{"version":0}`}, 401)
		r.must(call{method: "PUT", path: bp, token: teacher, body: `{"version":0}`}, 403)
	}
	r.must(call{method: "GET", path: base + "/budget", token: teacher}, 200)
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/llm-budget", token: admin}, 422)
	r.must(call{method: "GET", path: "/api/v1/courses/" + uuid.NewString() + "/llm-budget", token: teacher}, 403)

	// DELETE: 409 đang dùng / 401 / 403 / 404 / 204
	r.must(call{method: "DELETE", path: one, token: admin}, 409)
	r.must(call{method: "DELETE", path: one}, 401)
	r.must(call{method: "DELETE", path: one, token: teacher}, 403)
	r.must(call{method: "DELETE", path: prov + "/" + uuid.NewString(), token: admin}, 404)
	_, b = r.must(call{method: "POST", path: prov, token: admin, headers: idem(), body: provBody("CT-C", "good-key")}, 201)
	r.must(call{method: "DELETE", path: prov + "/" + parse(b).ID, token: admin}, 204)
}

// testScenarios gọi thật MỌI (thao tác, status) đã khai báo trong openapi.test.yaml.
func (r *runner) testScenarios() {
	t := r.t
	owner := uuid.NewString()
	tok := r.rig.token(t, owner, auth.RoleStudent)
	admin := r.rig.token(t, uuid.NewString(), auth.RoleAdmin)
	teacher := r.rig.token(t, uuid.NewString(), auth.RoleTeacher)
	const items = "/api/v1/_test/items"
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": "ct-" + uuid.NewString()} }

	// items: tạo, danh sách (+ETag/304), một bản ghi (+ETag/304), sửa (200/409/422/404).
	h, b := r.must(call{method: "POST", path: items, token: tok, headers: idem(), body: `{"name":"một"}`}, 201)
	if h.Get("Location") == "" {
		t.Fatalf("POST items thiếu Location: %s", b)
	}
	id := strings.TrimPrefix(h.Get("Location"), items+"/")
	r.must(call{method: "POST", path: items, token: tok, headers: idem(), body: "{hỏng"}, 400)
	r.must(call{method: "POST", path: items, body: `{"name":"x"}`, headers: idem()}, 401)
	r.must(call{method: "POST", path: items, token: tok, headers: idem(), body: `{"name":"` + strings.Repeat("a", 5000) + `"}`}, 413)
	r.must(call{method: "POST", path: items, token: tok, headers: idem(), body: "x", ctype: "text/plain"}, 415)
	r.must(call{method: "POST", path: items, token: tok, body: `{"name":"x"}`}, 422) // thiếu Idempotency-Key

	lh, _ := r.must(call{method: "GET", path: items + "?limit=1", token: tok}, 200)
	r.must(call{method: "GET", path: items + "?limit=1", token: tok, headers: map[string]string{"If-None-Match": lh.Get("ETag")}}, 304)
	r.must(call{method: "GET", path: items}, 401)
	r.must(call{method: "GET", path: items + "?limit=0", token: tok}, 422)

	gh, _ := r.must(call{method: "GET", path: items + "/" + id, token: tok}, 200)
	r.must(call{method: "GET", path: items + "/" + id, token: tok, headers: map[string]string{"If-None-Match": gh.Get("ETag")}}, 304)
	r.must(call{method: "GET", path: items + "/" + id}, 401)
	r.must(call{method: "GET", path: items + "/" + uuid.NewString(), token: tok}, 404)

	r.must(call{method: "PUT", path: items + "/" + id, token: tok, body: `{"name":"hai","version":1}`}, 200)
	r.must(call{method: "PUT", path: items + "/" + id, token: tok, body: `{"name":"ba","version":1}`}, 409) // đã tăng lên 2
	r.must(call{method: "PUT", path: items + "/" + id, token: tok, body: `{"name":"ba"}`}, 422)             // thiếu version
	r.must(call{method: "PUT", path: items + "/" + id, body: `{"name":"ba","version":2}`}, 401)
	r.must(call{method: "PUT", path: items + "/" + uuid.NewString(), token: tok, body: `{"name":"ba","version":1}`}, 404)

	// chậm / deadline / panic.
	r.must(call{method: "GET", path: "/api/v1/_test/slow?ms=5"}, 200)
	r.must(call{method: "GET", path: "/api/v1/_test/slow?ms=4000"}, 504)
	r.must(call{method: "GET", path: "/api/v1/_test/db-sleep?seconds=0"}, 200)
	r.must(call{method: "GET", path: "/api/v1/_test/db-sleep?seconds=abc"}, 400)
	r.must(call{method: "GET", path: "/api/v1/_test/db-sleep?seconds=4"}, 504)
	if err := r.rig.deps.Redis.LPush(context.Background(), "ep:test:block", "x").Err(); err != nil {
		t.Fatalf("chuẩn bị redis-block: %v", err)
	}
	r.must(call{method: "GET", path: "/api/v1/_test/redis-block?seconds=1"}, 200)
	r.must(call{method: "GET", path: "/api/v1/_test/redis-block?seconds=abc"}, 400)
	r.must(call{method: "GET", path: "/api/v1/_test/redis-block?seconds=4"}, 504)
	r.must(call{method: "GET", path: "/api/v1/_test/panic"}, 500)

	for _, s := range []int{400, 401, 403, 404, 405, 409, 413, 415, 422, 429, 500, 503, 504} {
		r.must(call{method: "GET", path: fmt.Sprintf("/api/v1/_test/error/%d", s)}, s)
	}

	// danh tính và phân quyền.
	r.must(call{method: "GET", path: "/api/v1/_test/whoami", token: tok}, 200)
	r.must(call{method: "GET", path: "/api/v1/_test/whoami"}, 401)
	r.must(call{method: "GET", path: "/api/v1/_test/rbac/admin", token: admin}, 200)
	r.must(call{method: "GET", path: "/api/v1/_test/rbac/admin"}, 401)
	r.must(call{method: "GET", path: "/api/v1/_test/rbac/admin", token: tok}, 403)
	r.must(call{method: "GET", path: "/api/v1/_test/rbac/staff", token: teacher}, 200)
	r.must(call{method: "GET", path: "/api/v1/_test/rbac/staff"}, 401)
	r.must(call{method: "GET", path: "/api/v1/_test/rbac/staff", token: tok}, 403)
	r.must(call{method: "GET", path: "/api/v1/_test/courses/" + uuid.NewString() + "/ping", token: tok}, 403)
	r.must(call{method: "GET", path: "/api/v1/_test/courses/" + uuid.NewString() + "/ping"}, 401)
	r.must(call{method: "GET", path: "/api/v1/_test/courses/khong-phai-uuid/ping", token: tok}, 404)

	// việc dài và sự kiện.
	r.must(call{method: "POST", path: "/api/v1/_test/jobs", token: tok, body: `{"steps":2}`}, 202)
	r.must(call{method: "POST", path: "/api/v1/_test/jobs", body: `{"steps":2}`}, 401)
	r.must(call{method: "POST", path: "/api/v1/_test/jobs", token: tok, body: `{"steps":0}`}, 422)
	r.must(call{method: "POST", path: "/api/v1/_test/events", token: tok, body: `{"type":"test.hello","data":{"a":1}}`}, 202)
	r.must(call{method: "POST", path: "/api/v1/_test/events", body: `{"type":"test.hello","data":{}}`}, 401)
	r.must(call{method: "POST", path: "/api/v1/_test/events", token: tok, body: `{"type":"test.hello","data":{},"user_id":"` + uuid.NewString() + `"}`}, 403)
	r.must(call{method: "POST", path: "/api/v1/_test/events", token: tok, body: `{"type":"SAI ĐỊNH DẠNG","data":{}}`}, 422)

	// cổng LLM (fake): chỉ ADMIN.
	chat := `{"task":"CHAT","prompt":"xin chào"}`
	r.must(call{method: "POST", path: "/api/v1/_test/llm/chat", token: admin, body: chat}, 200)
	r.must(call{method: "POST", path: "/api/v1/_test/llm/chat", body: chat}, 401)
	r.must(call{method: "POST", path: "/api/v1/_test/llm/chat", token: tok, body: chat}, 403)
	r.must(call{method: "POST", path: "/api/v1/_test/llm/chat", token: admin, body: `{"task":"CHAT"}`}, 422)
	r.must(call{method: "GET", path: "/api/v1/_test/llm/stats", token: admin}, 200)
	r.must(call{method: "GET", path: "/api/v1/_test/llm/stats?lanes=1", token: admin}, 200)
	r.must(call{method: "GET", path: "/api/v1/_test/llm/stats"}, 401)
	r.must(call{method: "GET", path: "/api/v1/_test/llm/stats", token: teacher}, 403)
	r.must(call{method: "POST", path: "/api/v1/_test/llm/fake", token: admin, body: `{"error_rate":1,"error_kind":"AUTH"}`}, 200)
	// mọi nhà cung cấp lỗi: CHAT (INTERACTIVE) suy giảm 200; làn NEAR_REALTIME trả 503 LLM_UNAVAILABLE
	r.must(call{method: "POST", path: "/api/v1/_test/llm/chat", token: admin, body: chat}, 200)
	r.must(call{method: "POST", path: "/api/v1/_test/llm/chat", token: admin, body: `{"task":"UTILITY","prompt":"xin chào"}`}, 503)
	r.must(call{method: "POST", path: "/api/v1/_test/llm/fake", token: admin, body: `{"error_rate":0,"error_kind":"SERVER"}`}, 200)
	r.must(call{method: "POST", path: "/api/v1/_test/llm/fake", body: `{}`}, 401)
	r.must(call{method: "POST", path: "/api/v1/_test/llm/fake", token: tok, body: `{}`}, 403)
	r.must(call{method: "POST", path: "/api/v1/_test/llm/fake", token: admin, body: `{"error_rate":2}`}, 422)
}

// passwordScenarios: forgot / reset / tokens-preview / đổi mật khẩu / thiết bị (US-P2-04) với mọi status đã khai báo.
func (r *runner) passwordScenarios() {
	const password = "Edupilot#2026-demo"
	ctx := context.Background()
	hash, err := auth.HashPassword(password, 4)
	if err != nil {
		r.t.Fatalf("băm mật khẩu: %v", err)
	}
	email := "ct-pw-" + uuid.NewString()[:8] + "@example.test"
	var uid uuid.UUID
	if err := r.rig.deps.DB.QueryRow(ctx,
		`insert into users (email, full_name, role, status, password_hash) values ($1, 'Người Thử', 'STUDENT', 'ACTIVE', $2) returning id`, email, hash).Scan(&uid); err != nil {
		r.t.Fatalf("tạo người dùng: %v", err)
	}
	issue := func() string {
		tx, err := r.rig.deps.DB.Begin(ctx)
		if err != nil {
			r.t.Fatal(err)
		}
		tok, err := auth.Tokens{Clock: r.rig.deps.Clock}.Issue(ctx, tx, uid, auth.TokenResetPassword, time.Hour, nil)
		if err != nil || tx.Commit(ctx) != nil {
			r.t.Fatalf("phát token đặt lại: %v", err)
		}
		return tok
	}

	r.must(call{method: "POST", path: "/api/v1/auth/forgot-password", body: `{"email":""}`}, 422)
	for range 3 {
		r.must(call{method: "POST", path: "/api/v1/auth/forgot-password", body: `{"email":"` + email + `"}`}, 202)
	}
	r.must(call{method: "POST", path: "/api/v1/auth/forgot-password", body: `{"email":"` + email + `"}`}, 429)

	tok := issue()
	r.must(call{method: "POST", path: "/api/v1/auth/reset-password", body: `{"token":"` + tok + `","new_password":"ngan"}`}, 422)
	r.must(call{method: "POST", path: "/api/v1/auth/reset-password", body: `{"token":"` + tok + `","new_password":"Mat-khau-moi-2026"}`}, 200)
	r.must(call{method: "POST", path: "/api/v1/auth/reset-password", body: `{"token":"` + tok + `","new_password":"Mat-khau-moi-2026"}`}, 410)

	const pw = "Mat-khau-moi-2026"
	prev := func(kind, token string, want int) {
		r.must(call{method: "POST", path: "/api/v1/auth/tokens/preview", body: `{"kind":"` + kind + `","token":"` + token + `"}`}, want)
	}
	r.rig.clearPreviewLimit(r.t)
	prev("RESET_PASSWORD", issue(), 200)
	prev("VERIFY_EMAIL", "x", 422)
	prev("INVITE", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", 410)
	for range 25 { // 20 lần / phút / IP; bộ đếm sống 2 phút nên có thể đã có sẵn lần chạy trước ⇒ lặp tới khi chạm trần
		if st, _, _ := r.do(call{method: "POST", path: "/api/v1/auth/tokens/preview", body: `{"kind":"INVITE","token":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}`}); st == 429 {
			break
		}
	}
	prev("INVITE", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", 429)

	login := func() (access, sid string) {
		_, b := r.must(call{method: "POST", path: "/api/v1/auth/login", headers: map[string]string{"Origin": "https://localhost"}, body: `{"email":"` + email + `","password":"` + pw + `"}`}, 200)
		var m struct {
			AccessToken string `json:"access_token"`
		}
		if err := json.Unmarshal(b, &m); err != nil {
			r.t.Fatal(err)
		}
		parts := strings.Split(m.AccessToken, ".")
		raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
		var c struct {
			Sid string `json:"sid"`
		}
		_ = json.Unmarshal(raw, &c)
		return m.AccessToken, c.Sid
	}
	a, _ := login()
	_, bSid := login()
	for _, c := range []call{
		{method: "GET", path: "/api/v1/me/sessions"}, {method: "DELETE", path: "/api/v1/me/sessions"},
		{method: "DELETE", path: "/api/v1/me/sessions/" + bSid}, {method: "POST", path: "/api/v1/me/password", body: `{}`},
	} {
		r.must(c, 401)
	}
	r.must(call{method: "GET", path: "/api/v1/me/sessions", token: a}, 200)
	r.must(call{method: "DELETE", path: "/api/v1/me/sessions/" + bSid, token: a}, 204)
	r.must(call{method: "DELETE", path: "/api/v1/me/sessions/" + bSid, token: a}, 404)
	r.must(call{method: "DELETE", path: "/api/v1/me/sessions", token: a}, 200)
	r.must(call{method: "POST", path: "/api/v1/me/password", token: a, body: `{"current_password":"` + pw + `","new_password":"Mat-khau-lan-hai-2026"}`}, 204)
	r.must(call{method: "POST", path: "/api/v1/me/password", token: a, body: `{"current_password":"` + pw + `","new_password":"Mat-khau-lan-ba-2026"}`}, 422) // mật khẩu hiện tại giờ là mật khẩu lần hai
	for range 4 {
		r.must(call{method: "POST", path: "/api/v1/me/password", token: a, body: `{"current_password":"sai-mat-khau-1","new_password":"Mat-khau-lan-ba-2026"}`}, 422)
	}
	r.must(call{method: "POST", path: "/api/v1/me/password", token: a, body: `{"current_password":"sai-mat-khau-1","new_password":"Mat-khau-lan-ba-2026"}`}, 429)
}

// clearPreviewLimit xoá bộ đếm 20 lần / phút / IP của tokens/preview: Redis dùng chung giữa các test và các lần chạy (TTL 2 phút),
// IP của rig luôn là 127.0.0.1 nên bộ đếm của lần trước sẽ làm lần sau bị 429 sớm.
func (rg *rig) clearPreviewLimit(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	keys, err := rg.deps.Redis.Keys(ctx, "ep:rl:auth:token:ip:*").Result()
	if err != nil {
		t.Fatalf("liệt kê bộ đếm: %v", err)
	}
	if len(keys) > 0 {
		if err := rg.deps.Redis.Del(ctx, keys...).Err(); err != nil {
			t.Fatalf("xoá bộ đếm: %v", err)
		}
	}
}
