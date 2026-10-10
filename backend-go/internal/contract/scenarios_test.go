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
	xff   string            // IP khách giả cho các lời gọi /auth/* (gateway tin X-Forwarded-For từ 127.0.0.1)
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
	if r.xff != "" && (strings.HasPrefix(c.path, "/api/v1/auth/") || strings.HasPrefix(c.path, "/api/v1/courses/join")) { // /courses/join*: bộ đếm đoán mã theo IP
		req.Header.Set("X-Forwarded-For", r.xff)
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
	// `text/csv` (US-PE-08 `results.csv`, phân cách `;`): bộ giải mã CSV của kin-openapi chỉ biết dấu phẩy nên không kiểm thân bằng schema; nội dung do `exam.TestResultsCSV` kiểm.
	csv := strings.HasPrefix(h.Get("Content-Type"), "text/csv")
	// `text/event-stream` (chat riêng, US-P3-05): kin-openapi không có bộ giải mã SSE; khung SSE do `internal/chat` kiểm (TestStreamEventOrder, TestChatSSENotBuffered).
	sse := strings.HasPrefix(h.Get("Content-Type"), "text/event-stream")
	o.Err = spec.ValidateResponse(context.Background(), req, status, h, body, ValidateOpts{SkipBody: skipBody || csv || sse})
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
// freshIP đổi IP khách giả: bộ đếm giới hạn theo IP ở Redis dùng chung giữa các lần chạy (cửa sổ 1 phút – 1 giờ) nên mỗi nhóm kịch bản
// dùng một IP riêng để kết quả không phụ thuộc thứ tự hay lần chạy trước.
func (r *runner) freshIP() {
	id := uuid.New()
	r.xff = fmt.Sprintf("198.18.%d.%d", id[0], id[1]) // 198.18.0.0/15: dải kiểm thử mạng (RFC 2544)
}

// exhaust gọi c (với IP riêng) tới khi gặp 429; fail nếu 80 lần mà chưa thấy. Trả số lần gọi.
func (r *runner) exhaust(c call) int {
	r.t.Helper()
	saved := r.xff
	defer func() { r.xff = saved }()
	r.freshIP()
	for i := 1; i <= 80; i++ {
		if st, _, _ := r.do(c); st == 429 {
			return i
		}
	}
	r.t.Fatalf("%s %s: 80 lần vẫn chưa 429", c.method, c.path)
	return 0
}

func (r *runner) authScenarios() {
	r.freshIP()
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
	r.adminUserScenarios()
	r.courseScenarios()
	r.courseAdminScenarios()
	r.courseJoinScenarios()
	r.examScenarios()
}

// accountScenarios: register / verify-email / resend-verification (US-P2-03) với mọi status đã khai báo.
func (r *runner) accountScenarios() {
	r.freshIP()
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
	r.must(call{method: "GET", path: "/api/v1/_test/chat-gate", token: tok}, 200)
	r.must(call{method: "GET", path: "/api/v1/_test/chat-gate"}, 401)
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
	r.must(call{method: "GET", path: "/api/v1/_test/llm/payloads", token: admin}, 200)
	r.must(call{method: "GET", path: "/api/v1/_test/llm/payloads"}, 401)
	r.must(call{method: "GET", path: "/api/v1/_test/llm/payloads", token: teacher}, 403)
	r.must(call{method: "POST", path: "/api/v1/_test/llm/payloads/reset", token: admin}, 204)
	r.must(call{method: "POST", path: "/api/v1/_test/llm/payloads/reset"}, 401)
	r.must(call{method: "POST", path: "/api/v1/_test/llm/payloads/reset", token: teacher}, 403)
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
	r.freshIP()
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

	// 429 theo IP/hành động (SRS 4.2.2): mỗi hành động một IP riêng, gọi tới khi chạm trần mặc định.
	badLogin := `{"email":"ct-throttle-` + uuid.NewString()[:8] + `@example.test","password":"sai-mat-khau-1"}`
	r.exhaust(call{method: "POST", path: "/api/v1/auth/login", headers: map[string]string{"Origin": "https://localhost"}, body: badLogin})
	r.exhaust(call{method: "POST", path: "/api/v1/auth/register", body: `{"email":"a@b","password":"x","full_name":""}`})
	r.exhaust(call{method: "POST", path: "/api/v1/auth/verify-email", body: `{"token":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}`})
	r.exhaust(call{method: "POST", path: "/api/v1/auth/reset-password", body: `{"token":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","new_password":"Mat-khau-moi-2026"}`})
	r.exhaust(call{method: "POST", path: "/api/v1/auth/refresh", headers: map[string]string{"Origin": "https://localhost"}})

	// khoá đăng nhập: 10 lần sai ⇒ LOGIN_THROTTLED (cùng mã 429 với giới hạn IP)
	r.freshIP()
	locked := "ct-lock-" + uuid.NewString()[:8] + "@example.test"
	if _, err := r.rig.deps.DB.Exec(ctx, `insert into users (email, full_name, role, status, password_hash) values ($1, 'Người Thử', 'STUDENT', 'ACTIVE', $2)`, locked, hash); err != nil {
		r.t.Fatalf("tạo người dùng: %v", err)
	}
	for range 5 {
		r.must(call{method: "POST", path: "/api/v1/auth/login", headers: map[string]string{"Origin": "https://localhost"}, body: `{"email":"` + locked + `","password":"sai-mat-khau-1"}`}, 401)
	}
	r.must(call{method: "POST", path: "/api/v1/auth/login", headers: map[string]string{"Origin": "https://localhost"}, body: `{"email":"` + locked + `","password":"` + password + `"}`}, 429)
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

// adminUserScenarios: /admin/users (4 thao tác) + /auth/accept-invite (US-P2-06) với mọi status đã khai báo.
func (r *runner) adminUserScenarios() {
	r.freshIP()
	ctx := context.Background()
	var adminID uuid.UUID
	if err := r.rig.deps.DB.QueryRow(ctx,
		`insert into users (email, full_name, role, status, password_hash) values ($1, 'Quản Trị Thử', 'ADMIN', 'ACTIVE', 'x') returning id`,
		"ct-admin-"+uuid.NewString()[:8]+"@example.test").Scan(&adminID); err != nil {
		r.t.Fatalf("tạo admin: %v", err)
	}
	admin := r.rig.token(r.t, adminID.String(), auth.RoleAdmin)
	student := r.rig.token(r.t, uuid.NewString(), auth.RoleStudent)
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": "ct-" + uuid.NewString()} }
	const users = "/api/v1/admin/users"

	r.must(call{method: "GET", path: users, token: admin}, 200)
	r.must(call{method: "GET", path: users + "?limit=0", token: admin}, 422)
	r.must(call{method: "GET", path: users}, 401)
	r.must(call{method: "GET", path: users, token: student}, 403)

	email := "ct-gv-" + uuid.NewString()[:8] + "@example.test"
	invite := `{"email":"` + email + `","full_name":"Giảng Viên Thử","role":"TEACHER"}`
	_, b := r.must(call{method: "POST", path: users, token: admin, headers: idem(), body: invite}, 201)
	var created struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	if err := json.Unmarshal(b, &created); err != nil {
		r.t.Fatal(err)
	}
	r.must(call{method: "POST", path: users, token: admin, headers: idem(), body: invite}, 409)
	r.must(call{method: "POST", path: users, token: admin, headers: idem(), body: `{"email":"x@y.zz","full_name":"X","role":"STUDENT"}`}, 422)
	r.must(call{method: "POST", path: users, headers: idem(), body: invite}, 401)
	r.must(call{method: "POST", path: users, token: student, headers: idem(), body: invite}, 403)

	one := users + "/" + created.ID
	r.must(call{method: "PATCH", path: one, token: admin, body: `{"full_name":"Giảng Viên Đã Đổi Tên","version":1}`}, 200)
	r.must(call{method: "PATCH", path: one, token: admin, body: `{"full_name":"Lỗi Phiên Bản","version":1}`}, 409)                    // VERSION_CONFLICT
	r.must(call{method: "PATCH", path: users + "/" + adminID.String(), token: admin, body: `{"status":"DISABLED","version":1}`}, 409) // CONFLICT self
	r.must(call{method: "PATCH", path: one, token: admin, body: `{"role":"ADMIN","version":2}`}, 422)
	r.must(call{method: "PATCH", path: users + "/" + uuid.NewString(), token: admin, body: `{"status":"DISABLED","version":1}`}, 404)
	r.must(call{method: "PATCH", path: one, body: `{"status":"DISABLED","version":2}`}, 401)
	r.must(call{method: "PATCH", path: one, token: student, body: `{"status":"DISABLED","version":2}`}, 403)

	resend := one + "/resend-invite"
	r.must(call{method: "POST", path: resend, token: admin}, 200)
	r.must(call{method: "POST", path: resend, token: admin}, 429)
	r.must(call{method: "POST", path: users + "/" + adminID.String() + "/resend-invite", token: admin}, 409) // không ở trạng thái INVITED
	r.must(call{method: "POST", path: users + "/" + uuid.NewString() + "/resend-invite", token: admin}, 404)
	r.must(call{method: "POST", path: resend}, 401)
	r.must(call{method: "POST", path: resend, token: student}, 403)

	// nhận lời mời: token phát trực tiếp (thư do worker gửi, ở đây chỉ cần liên kết)
	var invitedID uuid.UUID
	if err := r.rig.deps.DB.QueryRow(ctx, `select id from users where email = $1`, email).Scan(&invitedID); err != nil {
		r.t.Fatal(err)
	}
	tx, err := r.rig.deps.DB.Begin(ctx)
	if err != nil {
		r.t.Fatal(err)
	}
	tok, err := auth.Tokens{Clock: r.rig.deps.Clock}.Issue(ctx, tx, invitedID, auth.TokenInvite, time.Hour, nil)
	if err != nil || tx.Commit(ctx) != nil {
		r.t.Fatalf("phát token mời: %v", err)
	}
	accept := func(token, pw string, want int) {
		r.must(call{method: "POST", path: "/api/v1/auth/accept-invite", body: `{"token":"` + token + `","password":"` + pw + `"}`}, want)
	}
	accept(tok, "ngan", 422)
	accept(tok, "Mat-khau-nhan-moi-2026", 200)
	accept(tok, "Mat-khau-nhan-moi-2026", 410)
	r.exhaust(call{method: "POST", path: "/api/v1/auth/accept-invite", body: `{"token":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","password":"Mat-khau-nhan-moi-2026"}`})
}

// courseScenarios: /me/courses, /courses/{id}, /me/profile, /me/settings (US-P2-07) với mọi status đã khai báo (trừ 503 — xem exempt.go).
func (r *runner) courseScenarios() {
	r.freshIP()
	ctx := context.Background()
	db := r.rig.deps.DB
	mkUser := func(role string) uuid.UUID {
		var id uuid.UUID
		if err := db.QueryRow(ctx, `insert into users (email, full_name, role, status, password_hash) values ($1, 'Người Thử', $2::user_role, 'ACTIVE', 'x') returning id`,
			"ct-co-"+uuid.NewString()[:8]+"@example.test", role).Scan(&id); err != nil {
			r.t.Fatalf("tạo người dùng: %v", err)
		}
		return id
	}
	adminID, svID, outID := mkUser("ADMIN"), mkUser("STUDENT"), mkUser("STUDENT")
	var courseID uuid.UUID
	if err := db.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by)
		values ('INT1006', $1, 'An ninh mạng', '2026-2027-HK1', $2, $3) returning id`, "CT-"+uuid.NewString()[:6], "CT"+strings.ToUpper(strings.NewReplacer("0", "X", "1", "Y", "O", "Z", "I", "W", "L", "V").Replace(uuid.NewString()[:5])), adminID).Scan(&courseID); err != nil {
		r.t.Fatalf("tạo lớp: %v", err)
	}
	if _, err := db.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, courseID, svID); err != nil {
		r.t.Fatal(err)
	}
	sv := r.rig.token(r.t, svID.String(), auth.RoleStudent)
	out := r.rig.token(r.t, outID.String(), auth.RoleStudent)

	h, _ := r.must(call{method: "GET", path: "/api/v1/me/courses", token: sv}, 200)
	r.must(call{method: "GET", path: "/api/v1/me/courses", token: sv, headers: map[string]string{"If-None-Match": h.Get("ETag")}}, 304)
	r.must(call{method: "GET", path: "/api/v1/me/courses?limit=0", token: sv}, 422)
	r.must(call{method: "GET", path: "/api/v1/me/courses"}, 401)

	one := "/api/v1/courses/" + courseID.String()
	r.must(call{method: "GET", path: one, token: sv}, 200)
	r.must(call{method: "GET", path: one}, 401)
	r.must(call{method: "GET", path: one, token: out}, 403)
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid", token: sv}, 404)
	_ = adminID

	const profile, settings = "/api/v1/me/profile", "/api/v1/me/settings"
	h, _ = r.must(call{method: "GET", path: profile, token: sv}, 200)
	r.must(call{method: "GET", path: profile, token: sv, headers: map[string]string{"If-None-Match": h.Get("ETag")}}, 304)
	r.must(call{method: "GET", path: profile}, 401)
	r.must(call{method: "PUT", path: profile, token: sv, body: `{"full_name":"Tên Mới","version":1}`}, 200)
	r.must(call{method: "PUT", path: profile, token: sv, body: `{"full_name":"Lỗi Phiên Bản","version":1}`}, 409)
	r.must(call{method: "PUT", path: profile, token: sv, body: `{"role":"ADMIN","version":2}`}, 422)
	r.must(call{method: "PUT", path: profile, body: `{"full_name":"X","version":2}`}, 401)

	h, _ = r.must(call{method: "GET", path: settings, token: sv}, 200)
	r.must(call{method: "GET", path: settings, token: sv, headers: map[string]string{"If-None-Match": h.Get("ETag")}}, 304)
	r.must(call{method: "GET", path: settings}, 401)
	r.must(call{method: "PUT", path: settings, token: sv, body: `{"notify_ticket_by_mail":false,"version":1}`}, 200)
	r.must(call{method: "PUT", path: settings, token: sv, body: `{"notify_ticket_by_mail":true,"version":1}`}, 409)
	r.must(call{method: "PUT", path: settings, token: sv, body: `{"theme":"dark","version":2}`}, 422)
	r.must(call{method: "PUT", path: settings, body: `{"version":2}`}, 401)
}

// courseAdminScenarios: /admin/courses (5 thao tác), assistants, assistant-candidates, notifications (US-P2-08) với mọi status đã khai báo.
func (r *runner) courseAdminScenarios() {
	r.freshIP()
	ctx := context.Background()
	db := r.rig.deps.DB
	mkUser := func(role, status string) uuid.UUID {
		var id uuid.UUID
		if err := db.QueryRow(ctx, `insert into users (email, full_name, role, status, password_hash) values ($1, 'Người Thử', $2::user_role, $3::user_status, 'x') returning id`,
			"ct-ad-"+uuid.NewString()[:8]+"@example.test", role, status).Scan(&id); err != nil {
			r.t.Fatalf("tạo người dùng: %v", err)
		}
		return id
	}
	adminID, gvID, gv2ID, taID, svID := mkUser("ADMIN", "ACTIVE"), mkUser("TEACHER", "ACTIVE"), mkUser("TEACHER", "ACTIVE"), mkUser("TA", "ACTIVE"), mkUser("STUDENT", "ACTIVE")
	admin := r.rig.token(r.t, adminID.String(), auth.RoleAdmin)
	gv := r.rig.token(r.t, gvID.String(), auth.RoleTeacher)
	student := r.rig.token(r.t, svID.String(), auth.RoleStudent)
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": "ct-" + uuid.NewString()} }
	const courses = "/api/v1/admin/courses"
	code := "CT-" + uuid.NewString()[:8]
	body := func(class string) string {
		return `{"subject_code":"INT1006","class_code":"` + class + `","name":"An ninh mạng","semester":"2026-2027-HK1"}`
	}

	r.must(call{method: "GET", path: courses, token: admin}, 200)
	r.must(call{method: "GET", path: courses + "?limit=0", token: admin}, 422)
	r.must(call{method: "GET", path: courses}, 401)
	r.must(call{method: "GET", path: courses, token: student}, 403)

	_, b := r.must(call{method: "POST", path: courses, token: admin, headers: idem(), body: body(code)}, 201)
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(b, &created); err != nil {
		r.t.Fatal(err)
	}
	r.must(call{method: "POST", path: courses, token: admin, headers: idem(), body: body(code)}, 409)
	r.must(call{method: "POST", path: courses, token: admin, headers: idem(), body: `{"class_code":"x"}`}, 422)
	r.must(call{method: "POST", path: courses, headers: idem(), body: body(code)}, 401)
	r.must(call{method: "POST", path: courses, token: student, headers: idem(), body: body(code)}, 403)

	one := courses + "/" + created.ID
	r.must(call{method: "PUT", path: one, token: admin, body: `{"name":"Tên Đã Đổi","version":1}`}, 200)
	r.must(call{method: "PUT", path: one, token: admin, body: `{"name":"Lỗi Phiên Bản","version":1}`}, 409)
	r.must(call{method: "PUT", path: one, token: admin, body: `{"class_code":"x","version":2}`}, 422)
	r.must(call{method: "PUT", path: courses + "/" + uuid.NewString(), token: admin, body: `{"name":"X","version":1}`}, 404)
	r.must(call{method: "PUT", path: one, body: `{"name":"X","version":2}`}, 401)
	r.must(call{method: "PUT", path: one, token: student, body: `{"name":"X","version":2}`}, 403)

	assign := one + "/assign"
	r.must(call{method: "POST", path: assign, token: admin, body: `{"teacher_id":"` + gvID.String() + `","ta_ids":["` + taID.String() + `"]}`}, 200)
	r.must(call{method: "POST", path: assign, token: admin, body: `{"teacher_id":"` + svID.String() + `"}`}, 422)
	r.must(call{method: "POST", path: courses + "/" + uuid.NewString() + "/assign", token: admin, body: `{}`}, 404)
	r.must(call{method: "POST", path: assign, body: `{}`}, 401)
	r.must(call{method: "POST", path: assign, token: student, body: `{}`}, 403)

	// giảng viên của lớp tự quản lý trợ giảng
	assistants := "/api/v1/courses/" + created.ID + "/assistants"
	r.must(call{method: "PUT", path: assistants, token: gv, body: `{"ta_ids":["` + taID.String() + `"]}`}, 200)
	r.must(call{method: "PUT", path: assistants, token: gv, body: `{"ta_ids":["` + svID.String() + `"]}`}, 422)
	r.must(call{method: "PUT", path: assistants, body: `{"ta_ids":[]}`}, 401)
	r.must(call{method: "PUT", path: assistants, token: r.rig.token(r.t, gv2ID.String(), auth.RoleTeacher), body: `{"ta_ids":[]}`}, 403)
	r.must(call{method: "PUT", path: "/api/v1/courses/khong-phai-uuid/assistants", token: gv, body: `{"ta_ids":[]}`}, 404)
	cands := "/api/v1/courses/" + created.ID + "/assistant-candidates"
	r.must(call{method: "GET", path: cands + "?q=nguoi", token: gv}, 200)
	r.must(call{method: "GET", path: cands + "?q=" + strings.Repeat("a", 101), token: gv}, 422)
	r.must(call{method: "GET", path: cands}, 401)
	r.must(call{method: "GET", path: cands, token: student}, 403)
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/assistant-candidates", token: gv}, 404)

	// lưu trữ rồi mọi thao tác ghi → 409 COURSE_ARCHIVED
	archive := one + "/archive"
	r.must(call{method: "POST", path: archive, token: admin}, 200)
	r.must(call{method: "POST", path: archive, token: admin}, 200)
	r.must(call{method: "POST", path: archive}, 401)
	r.must(call{method: "POST", path: archive, token: student}, 403)
	r.must(call{method: "POST", path: courses + "/" + uuid.NewString() + "/archive", token: admin}, 404)
	r.must(call{method: "PUT", path: one, token: admin, body: `{"name":"Sau Lưu Trữ","version":3}`}, 409)
	r.must(call{method: "POST", path: assign, token: admin, body: `{"teacher_id":"` + gv2ID.String() + `"}`}, 409)
	r.must(call{method: "PUT", path: assistants, token: gv, body: `{"ta_ids":[]}`}, 409)

	// thông báo
	if _, err := db.Exec(ctx, `insert into notifications (user_id, type, title, link) values ($1, 'COURSE_ASSIGNED', 'Thử', '/class/members')`, gvID); err != nil {
		r.t.Fatal(err)
	}
	_, b = r.must(call{method: "GET", path: "/api/v1/notifications", token: gv}, 200)
	var page struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(b, &page); err != nil || len(page.Items) == 0 {
		r.t.Fatalf("thông báo: %v %s", err, b)
	}
	r.must(call{method: "GET", path: "/api/v1/notifications?limit=0", token: gv}, 422)
	r.must(call{method: "GET", path: "/api/v1/notifications"}, 401)
	read := "/api/v1/notifications/" + page.Items[0].ID + "/read"
	r.must(call{method: "POST", path: read, token: gv}, 204)
	r.must(call{method: "POST", path: read}, 401)
	r.must(call{method: "POST", path: read, token: student}, 404)
}

// courseJoinScenarios: vào lớp bằng mã, mã và cài đặt tham gia, thành viên (US-P2-09) với mọi status đã khai báo.
func (r *runner) courseJoinScenarios() {
	r.freshIP()
	ctx := context.Background()
	db := r.rig.deps.DB
	mkUser := func(role string, verified bool) uuid.UUID {
		var id uuid.UUID
		if err := db.QueryRow(ctx, `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, 'Người Thử', $2::user_role, 'ACTIVE', 'x', case when $3 then now() end) returning id`,
			"ct-jn-"+uuid.NewString()[:8]+"@example.test", role, verified).Scan(&id); err != nil {
			r.t.Fatalf("tạo người dùng: %v", err)
		}
		return id
	}
	adminID, gvID, gv2ID, taID := mkUser("ADMIN", true), mkUser("TEACHER", true), mkUser("TEACHER", true), mkUser("TA", true)
	admin, gv := r.rig.token(r.t, adminID.String(), auth.RoleAdmin), r.rig.token(r.t, gvID.String(), auth.RoleTeacher)
	gv2, ta := r.rig.token(r.t, gv2ID.String(), auth.RoleTeacher), r.rig.token(r.t, taID.String(), auth.RoleTA)
	student := func() (uuid.UUID, string) {
		id := mkUser("STUDENT", true)
		return id, r.rig.token(r.t, id.String(), auth.RoleStudent)
	}
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": "ct-" + uuid.NewString()} }
	openCourse := func(extra string) (id, code string) {
		body := `{"subject_code":"INT1006","class_code":"` + "JN-" + uuid.NewString()[:8] + `","name":"An ninh mạng","semester":"2026-2027-HK1","teacher_id":"` + gvID.String() + `","ta_ids":["` + taID.String() + `"]` + extra + `}`
		_, b := r.must(call{method: "POST", path: "/api/v1/admin/courses", token: admin, headers: idem(), body: body}, 201)
		var c struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(b, &c); err != nil {
			r.t.Fatal(err)
		}
		if err := db.QueryRow(context.Background(), `select trim(join_code) from courses where id = $1`, c.ID).Scan(&code); err != nil {
			r.t.Fatal(err)
		}
		return c.ID, code
	}
	cid, code := openCourse("")
	const preview, join = "/api/v1/courses/join/preview", "/api/v1/courses/join"
	body := func(c string) string { return `{"code":"` + c + `"}` }

	// xem trước
	_, sv1 := student()
	r.must(call{method: "POST", path: preview, token: sv1, body: body(strings.ToLower(code))}, 200)
	r.must(call{method: "POST", path: preview, token: sv1, body: body("ZZZZZZZ")}, 404)
	r.must(call{method: "POST", path: preview, token: sv1, body: `{"x":1}`}, 422)
	r.must(call{method: "POST", path: preview, body: body(code)}, 401)
	r.must(call{method: "POST", path: preview, token: gv, body: body(code)}, 403)
	_, spammer := student()
	for range 5 {
		r.must(call{method: "POST", path: preview, token: spammer, body: body("ZZZZZZZ")}, 404)
	}
	r.must(call{method: "POST", path: preview, token: spammer, body: body(code)}, 429)
	r.freshIP()

	// vào lớp
	r.must(call{method: "POST", path: join, token: sv1, body: body(code)}, 200)
	r.must(call{method: "POST", path: join, token: sv1, body: body(code)}, 200)
	r.must(call{method: "POST", path: join, token: sv1, body: `{"x":1}`}, 422)
	r.must(call{method: "POST", path: join, body: body(code)}, 401)
	r.must(call{method: "POST", path: join, token: ta, body: body(code)}, 403)
	_, sv2 := student()
	r.must(call{method: "POST", path: join, token: sv2, body: body("ZZZZZZZ")}, 404)
	_, spammer2 := student()
	for range 5 {
		r.must(call{method: "POST", path: join, token: spammer2, body: body("ZZZZZZZ")}, 404)
	}
	r.must(call{method: "POST", path: join, token: spammer2, body: body(code)}, 429)
	r.freshIP()
	fullID, fullCode := openCourse(`,"capacity":1`)
	_, a := student()
	_, b := student()
	r.must(call{method: "POST", path: join, token: a, body: body(fullCode)}, 200)
	r.must(call{method: "POST", path: join, token: b, body: body(fullCode)}, 409)
	_ = fullID

	// mã và cài đặt tham gia
	jc := "/api/v1/courses/" + cid + "/join-code"
	r.must(call{method: "GET", path: jc, token: ta}, 200)
	r.must(call{method: "GET", path: jc}, 401)
	r.must(call{method: "GET", path: jc, token: sv1}, 403)
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/join-code", token: gv}, 404)
	js := "/api/v1/courses/" + cid + "/join-settings"
	r.must(call{method: "PUT", path: js, token: gv, body: `{"require_approval":true,"version":1}`}, 200)
	r.must(call{method: "PUT", path: js, token: gv, body: `{"require_approval":false,"version":1}`}, 409)
	r.must(call{method: "PUT", path: js, token: gv, body: `{"capacity":1001,"version":2}`}, 422)
	r.must(call{method: "PUT", path: js, body: `{"version":2}`}, 401)
	r.must(call{method: "PUT", path: js, token: ta, body: `{"version":2}`}, 403)
	r.must(call{method: "PUT", path: "/api/v1/courses/khong-phai-uuid/join-settings", token: gv, body: `{"version":2}`}, 404)

	// thành viên: hai sinh viên chờ duyệt
	u3, s3 := student()
	u4, s4 := student()
	r.must(call{method: "POST", path: join, token: s3, body: body(code)}, 200)
	r.must(call{method: "POST", path: join, token: s4, body: body(code)}, 200)
	mem := "/api/v1/courses/" + cid + "/members"
	r.must(call{method: "GET", path: mem, token: ta}, 200)
	r.must(call{method: "GET", path: mem + "?limit=0", token: ta}, 422)
	r.must(call{method: "GET", path: mem}, 401)
	r.must(call{method: "GET", path: mem, token: sv1}, 403)
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/members", token: ta}, 404)
	one := func(uid uuid.UUID, op string) string { return mem + "/" + uid.String() + op }
	r.must(call{method: "POST", path: one(u3, "/approve"), token: ta, body: `{}`}, 200)
	r.must(call{method: "POST", path: one(u3, "/approve"), token: ta, body: `{}`}, 409)
	r.must(call{method: "POST", path: one(gvID, "/approve"), token: gv, body: `{}`}, 422)
	r.must(call{method: "POST", path: one(uuid.New(), "/approve"), token: gv, body: `{}`}, 404)
	r.must(call{method: "POST", path: one(u4, "/approve")}, 401)
	r.must(call{method: "POST", path: one(u4, "/approve"), token: s4, body: `{}`}, 403)
	r.must(call{method: "POST", path: one(u4, "/reject"), token: gv}, 200)
	r.must(call{method: "POST", path: one(u4, "/reject"), token: gv}, 409)
	r.must(call{method: "POST", path: one(uuid.New(), "/reject"), token: gv}, 404)
	r.must(call{method: "POST", path: one(u4, "/reject")}, 401)
	r.must(call{method: "POST", path: one(u4, "/reject"), token: sv1}, 403)
	r.must(call{method: "POST", path: one(u4, "/undo"), token: ta}, 200)
	r.must(call{method: "POST", path: one(u4, "/undo"), token: ta}, 409)
	r.must(call{method: "POST", path: one(uuid.New(), "/undo"), token: ta}, 404)
	r.must(call{method: "POST", path: one(u4, "/undo")}, 401)
	r.must(call{method: "POST", path: one(u4, "/undo"), token: sv1}, 403)
	r.must(call{method: "DELETE", path: one(u3, ""), token: gv}, 200)
	r.must(call{method: "DELETE", path: one(u3, ""), token: gv}, 409)
	r.must(call{method: "DELETE", path: one(gvID, ""), token: admin}, 422)
	r.must(call{method: "DELETE", path: one(uuid.New(), ""), token: gv}, 404)
	r.must(call{method: "DELETE", path: one(u3, "")}, 401)
	r.must(call{method: "DELETE", path: one(u3, ""), token: ta}, 403)
	r.must(call{method: "DELETE", path: one(u3, ""), token: gv2}, 403)

	// tạo lại mã, rồi lưu trữ ⇒ 409
	rg := jc + "/regenerate"
	r.must(call{method: "POST", path: rg, token: gv}, 200)
	r.must(call{method: "POST", path: rg}, 401)
	r.must(call{method: "POST", path: rg, token: ta}, 403)
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/join-code/regenerate", token: gv}, 404)
	r.must(call{method: "POST", path: "/api/v1/admin/courses/" + cid + "/archive", token: admin}, 200)
	r.must(call{method: "POST", path: rg, token: gv}, 409)

	// roster và chia sẻ (US-P2-10): cid đã lưu trữ; c2 là lớp sống cùng học phần, cùng giảng viên.
	c2, c2code := openCourse("")
	multipart := func(file string) (string, string) {
		const bd = "ctboundary7MA4YWxkTrZu0gW"
		return "multipart/form-data; boundary=" + bd, "--" + bd + "\r\nContent-Disposition: form-data; name=\"file\"; filename=\"r.csv\"\r\nContent-Type: text/csv\r\n\r\n" + file + "\r\n--" + bd + "--\r\n"
	}
	rosterCSV := "Email,Họ và tên,MSSV\nct-ros-" + uuid.NewString()[:8] + "@example.test,Sinh Viên Một,B20DC00001\n"
	up := func(path, query, file string, tok string, h map[string]string) call {
		ct, body := multipart(file)
		return call{method: "POST", path: "/api/v1/courses/" + path + "/roster/import" + query, token: tok, headers: h, ctype: ct, body: body}
	}
	r.must(up(c2, "?dry_run=true", rosterCSV, gv, idem()), 200)
	r.must(up(c2, "", rosterCSV, gv, idem()), 200)
	r.must(up(c2, "", rosterCSV, "", idem()), 401)
	r.must(up(c2, "", rosterCSV, ta, idem()), 403)
	r.must(up("khong-phai-uuid", "", rosterCSV, gv, idem()), 404)
	r.must(up(cid, "", rosterCSV, gv, idem()), 409)
	r.must(up(c2, "", strings.Repeat("x", 3<<20), gv, idem()), 413)
	r.must(up(c2, "", "\x7fELF\x02\x01\x01\x00", gv, idem()), 422)

	ss := "/api/v1/courses/" + c2 + "/share-sources"
	r.must(call{method: "GET", path: ss, token: gv}, 200)
	r.must(call{method: "GET", path: ss}, 401)
	r.must(call{method: "GET", path: ss, token: ta}, 403)
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/share-sources", token: gv}, 404)

	sf := "/api/v1/courses/" + c2 + "/share-from"
	from := func(src, what string) string { return `{"source_course_id":"` + src + `","what":["` + what + `"]}` }
	other, _ := openCourse("")
	r.must(call{method: "POST", path: "/api/v1/courses/" + other + "/share-from", token: gv, headers: idem(), body: from(c2, "documents")}, 200)
	r.must(call{method: "POST", path: sf, headers: idem(), body: from(other, "documents")}, 401)
	r.must(call{method: "POST", path: sf, token: ta, headers: idem(), body: from(other, "documents")}, 403)
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/share-from", token: gv, headers: idem(), body: from(other, "documents")}, 404)
	r.must(call{method: "POST", path: sf, token: gv, headers: idem(), body: from(cid, "documents")}, 409)
	r.must(call{method: "POST", path: sf, token: gv, headers: idem(), body: from(other, "questions")}, 422)

	// "Hôm nay" và bỏ qua thiết lập (US-P2-11): ba dạng phản hồi + mọi status.
	td := "/api/v1/me/today"
	r.must(call{method: "GET", path: td, token: gv}, 200)
	r.must(call{method: "GET", path: td, token: admin}, 200)
	_, svToday := student()
	r.must(call{method: "GET", path: td, token: svToday}, 200)
	r.must(call{method: "GET", path: td}, 401)
	ct := "/api/v1/courses/" + c2 + "/today"
	r.must(call{method: "GET", path: ct, token: gv}, 200)
	r.must(call{method: "GET", path: ct, token: ta}, 200)
	r.must(call{method: "GET", path: ct}, 401)
	r.must(call{method: "GET", path: ct, token: admin}, 403)
	r.must(call{method: "GET", path: ct, token: svToday}, 403)
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/today", token: gv}, 404)
	// sinh viên đã vào lớp: dạng sinh viên có timeline.
	_, svIn := student()
	r.must(call{method: "POST", path: join, token: svIn, body: body(c2code)}, 200)
	r.must(call{method: "GET", path: ct, token: svIn}, 200)
	r.must(call{method: "GET", path: td, token: svIn}, 200)

	// buổi học tối thiểu
	gen := "/api/v1/courses/" + c2 + "/sessions/generate"
	wk := `{"weekdays":[4],"start_time":"09:00","end_time":"11:30","room":"P.302","from":"2026-09-01","to":"2026-09-30"}`
	r.must(call{method: "POST", path: gen, token: gv, headers: idem(), body: wk}, 201)
	r.must(call{method: "POST", path: gen, token: ta, headers: idem(), body: wk}, 201)
	r.must(call{method: "POST", path: gen, headers: idem(), body: wk}, 401)
	r.must(call{method: "POST", path: gen, token: admin, headers: idem(), body: wk}, 403)
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/sessions/generate", token: gv, headers: idem(), body: wk}, 404)
	r.must(call{method: "POST", path: "/api/v1/courses/" + cid + "/sessions/generate", token: gv, headers: idem(), body: wk}, 409)
	r.must(call{method: "POST", path: gen, token: gv, headers: idem(), body: `{"weekdays":[9],"start_time":"09:00","end_time":"11:30","from":"2026-09-01","to":"2026-09-30"}`}, 422)
	ls := "/api/v1/courses/" + c2 + "/sessions"
	r.must(call{method: "GET", path: ls, token: gv}, 200)
	r.must(call{method: "GET", path: ls + "?limit=2", token: svIn}, 200)
	r.must(call{method: "GET", path: ls}, 401)
	r.must(call{method: "GET", path: ls, token: admin}, 403)
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/sessions", token: gv}, 404)
	r.must(call{method: "GET", path: ls + "?limit=0", token: gv}, 422)

	dm := "/api/v1/courses/" + c2 + "/setup/dismiss"
	r.must(call{method: "POST", path: dm, token: gv}, 204)
	r.must(call{method: "POST", path: dm, token: gv}, 204)
	r.must(call{method: "POST", path: dm}, 401)
	r.must(call{method: "POST", path: dm, token: ta}, 403)
	r.must(call{method: "POST", path: dm, token: admin}, 403)
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/setup/dismiss", token: gv}, 404)
	r.must(call{method: "POST", path: "/api/v1/courses/" + cid + "/setup/dismiss", token: gv}, 409)
}
