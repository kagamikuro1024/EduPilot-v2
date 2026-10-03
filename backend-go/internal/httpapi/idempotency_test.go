package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appdb "github.com/edupilot/backend-go/internal/platform/db"
	applog "github.com/edupilot/backend-go/internal/platform/log"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// idemBuf là bộ đệm log an toàn khi nhiều goroutine cùng ghi.
type idemBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *idemBuf) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *idemBuf) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// idemDeps dựng Deps với Postgres đã migrate (bảng idempotency_keys) + Redis thật.
func idemDeps(t *testing.T) (Deps, *idemBuf) {
	t.Helper()
	cfg := config.Config{
		Role: config.Gateway, AppEnv: "test", InstanceID: "gw-test", LogLevel: "debug",
		DBMaxConns: 20, DBSlowQueryMS: 200, RequestTimeout: 30 * time.Second, MaxBodyBytes: 1 << 20,
		DatabaseURL: testutil.MigratedPostgresURL(t), RedisURL: testutil.RedisURL(t),
	}
	buf := &idemBuf{}
	log := applog.NewTo(buf, cfg, "gateway")
	pool, err := appdb.NewPool(t.Context(), cfg, log)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(pool.Close)
	rdb, err := appredis.New(t.Context(), cfg.RedisURL)
	if err != nil {
		t.Fatalf("redis.New: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	return Deps{Cfg: cfg, Log: log, DB: pool, Redis: rdb, Clock: clock.Real{}, State: NewState()}, buf
}

// idemCounter đếm số lần handler thật chạy và trả thân khác nhau mỗi lần (phát lại phải giống byte lần đầu).
type idemCounter struct {
	n      atomic.Int64
	status atomic.Int64
	gate   chan struct{} // nil = không chặn
}

func (c *idemCounter) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		n := c.n.Add(1)
		if c.gate != nil {
			<-c.gate
		}
		status := int(c.status.Load())
		if status == 0 {
			status = http.StatusCreated
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_, _ = fmt.Fprintf(w, `{"id":%q,"run":%d}`, uuid.NewString(), n)
	}
}

// idemRouter dựng router thật (otel → recover → log → timeout → body limit) với hai route bắt buộc khoá.
// Danh tính lấy từ header thử `X-Test-Sub` để một router phục vụ nhiều người dùng.
func idemRouter(d Deps, h http.Handler) http.Handler {
	return newRouterWith(d, func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sub := r.Header.Get("X-Test-Sub")
				if sub == "" {
					next.ServeHTTP(w, r)
					return
				}
				p := auth.Principal{Sub: sub, Role: auth.RoleStudent, JTI: "test"}
				next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
			})
		})
		r.With(RequireIdempotencyKey(d)).Post("/_test/items", h.ServeHTTP)
		r.With(RequireIdempotencyKey(d)).Post("/_test/others", h.ServeHTTP)
	})
}

// idemPost gửi một POST; key rỗng = không gửi header.
func idemPost(h http.Handler, path, sub, key, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Test-Sub", sub)
	if key != "" {
		r.Header.Set(idemHeader, key)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func idemCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("thân không phải JSON: %q", w.Body.String())
	}
	code, _ := m["code"].(string)
	return code
}

// idemRows đếm dòng idempotency_keys của một người dùng.
func idemRows(t *testing.T, d Deps, sub string) int {
	t.Helper()
	var n int
	if err := d.DB.QueryRow(t.Context(), "select count(*) from idempotency_keys where user_id = $1", sub).Scan(&n); err != nil {
		t.Fatalf("đếm idempotency_keys: %v", err)
	}
	return n
}

func TestIdempotency_DoubleSend_OneRecord(t *testing.T) {
	t.Parallel()
	d, _ := idemDeps(t)
	c := &idemCounter{}
	h := idemRouter(d, c.handler())
	sub, key, body := uuid.NewString(), "dbl-"+uuid.NewString(), `{"name":"a"}`

	w1 := idemPost(h, "/api/v1/_test/items", sub, key, body)
	w2 := idemPost(h, "/api/v1/_test/items", sub, key, body)

	if w1.Code != http.StatusCreated || w2.Code != http.StatusCreated {
		t.Fatalf("status = %d, %d; muốn 201, 201", w1.Code, w2.Code)
	}
	if got := c.n.Load(); got != 1 {
		t.Fatalf("handler chạy %d lần, muốn 1", got)
	}
	if got := idemRows(t, d, sub); got != 1 {
		t.Fatalf("idempotency_keys có %d dòng, muốn 1", got)
	}
	var row struct {
		endpoint string
		hash     string
		status   int16
	}
	if err := d.DB.QueryRow(t.Context(),
		"select endpoint, request_hash, status_code from idempotency_keys where user_id = $1", sub).
		Scan(&row.endpoint, &row.hash, &row.status); err != nil {
		t.Fatalf("đọc idempotency_keys: %v", err)
	}
	if row.endpoint != "POST /api/v1/_test/items" {
		t.Fatalf("endpoint = %q, muốn mẫu route đầy đủ", row.endpoint)
	}
	if len(row.hash) != 64 || row.status != http.StatusCreated {
		t.Fatalf("request_hash = %q, status_code = %d", row.hash, row.status)
	}
}

func TestIdempotency_ReplayIdentical(t *testing.T) {
	t.Parallel()
	d, _ := idemDeps(t)
	c := &idemCounter{}
	h := idemRouter(d, c.handler())
	sub, key, body := uuid.NewString(), "rep-"+uuid.NewString(), `{"name":"a"}`

	w1 := idemPost(h, "/api/v1/_test/items", sub, key, body)
	w2 := idemPost(h, "/api/v1/_test/items", sub, key, body)

	if w1.Header().Get("Idempotent-Replayed") != "" {
		t.Fatal("lần đầu không được có Idempotent-Replayed")
	}
	if got := w2.Header().Get("Idempotent-Replayed"); got != "true" {
		t.Fatalf("Idempotent-Replayed = %q, muốn true", got)
	}
	if w1.Code != w2.Code {
		t.Fatalf("status %d ≠ %d", w1.Code, w2.Code)
	}
	if w1.Header().Get("Content-Type") != w2.Header().Get("Content-Type") {
		t.Fatalf("Content-Type %q ≠ %q", w1.Header().Get("Content-Type"), w2.Header().Get("Content-Type"))
	}
	if !bytes.Equal(w1.Body.Bytes(), w2.Body.Bytes()) {
		t.Fatalf("thân khác nhau:\n%s\n%s", w1.Body, w2.Body)
	}
}

func TestIdempotency_KeyReused(t *testing.T) {
	t.Parallel()
	d, _ := idemDeps(t)
	c := &idemCounter{}
	h := idemRouter(d, c.handler())
	sub, key := uuid.NewString(), "reuse-"+uuid.NewString()

	if w := idemPost(h, "/api/v1/_test/items", sub, key, `{"name":"a"}`); w.Code != http.StatusCreated {
		t.Fatalf("lần đầu status = %d", w.Code)
	}
	w := idemPost(h, "/api/v1/_test/items", sub, key, `{"name":"b"}`)
	if w.Code != http.StatusUnprocessableEntity || idemCode(t, w) != "IDEMPOTENCY_KEY_REUSED" {
		t.Fatalf("status = %d, code = %q; muốn 422 IDEMPOTENCY_KEY_REUSED", w.Code, idemCode(t, w))
	}
	if got := c.n.Load(); got != 1 {
		t.Fatalf("handler chạy %d lần, muốn 1", got)
	}
}

func TestIdempotency_Required(t *testing.T) {
	t.Parallel()
	d, _ := idemDeps(t)
	c := &idemCounter{}
	h := idemRouter(d, c.handler())

	w := idemPost(h, "/api/v1/_test/items", uuid.NewString(), "", `{"name":"a"}`)
	if w.Code != http.StatusUnprocessableEntity || idemCode(t, w) != "IDEMPOTENCY_KEY_REQUIRED" {
		t.Fatalf("status = %d, code = %q; muốn 422 IDEMPOTENCY_KEY_REQUIRED", w.Code, idemCode(t, w))
	}
	if got := c.n.Load(); got != 0 {
		t.Fatalf("handler chạy %d lần, muốn 0", got)
	}
}

func TestIdempotency_BadKey(t *testing.T) {
	t.Parallel()
	d, _ := idemDeps(t)
	c := &idemCounter{}
	h := idemRouter(d, c.handler())
	sub := uuid.NewString()

	bad := []string{strings.Repeat("k", 7), strings.Repeat("k", 129), "bad key!", "khoaä-0001", "key/with/slash"}
	for _, k := range bad {
		w := idemPost(h, "/api/v1/_test/items", sub, k, `{"name":"a"}`)
		if w.Code != http.StatusUnprocessableEntity || idemCode(t, w) != "VALIDATION_FAILED" {
			t.Fatalf("khoá %q: status = %d, code = %q; muốn 422 VALIDATION_FAILED", k, w.Code, idemCode(t, w))
		}
		var body struct {
			Details []struct{ Field string } `json:"details"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || len(body.Details) != 1 ||
			body.Details[0].Field != idemHeader {
			t.Fatalf("khoá %q: details = %s", k, w.Body)
		}
	}
	for _, k := range []string{strings.Repeat("a", 8), strings.Repeat("b", 128), "a.b_x:-09"} {
		if w := idemPost(h, "/api/v1/_test/items", sub, k, `{"name":"a"}`); w.Code != http.StatusCreated {
			t.Fatalf("khoá hợp lệ %q: status = %d, muốn 201", k, w.Code)
		}
	}
	if got := c.n.Load(); got != 3 {
		t.Fatalf("handler chạy %d lần, muốn 3 (chỉ khoá hợp lệ)", got)
	}
}

func TestIdempotency_Concurrent50(t *testing.T) {
	t.Parallel()
	d, _ := idemDeps(t)
	c := &idemCounter{gate: make(chan struct{})}
	h := idemRouter(d, c.handler())
	sub, key, body := uuid.NewString(), "conc-"+uuid.NewString(), `{"name":"a"}`

	const n = 50
	out := make([]*httptest.ResponseRecorder, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func() {
			defer wg.Done()
			out[i] = idemPost(h, "/api/v1/_test/items", sub, key, body)
		}()
	}
	// Giữ handler đang chạy một nhịp để các request còn lại chắc chắn gặp khoá "đang chạy".
	time.Sleep(200 * time.Millisecond)
	close(c.gate)
	wg.Wait()

	if got := c.n.Load(); got != 1 {
		t.Fatalf("handler chạy %d lần, muốn đúng 1", got)
	}
	var first []byte
	n201, n409, orig := 0, 0, 0
	for i, w := range out {
		switch w.Code {
		case http.StatusCreated:
			n201++
			if w.Header().Get("Idempotent-Replayed") != "true" {
				orig++
			}
			if first == nil {
				first = w.Body.Bytes()
			} else if !bytes.Equal(first, w.Body.Bytes()) {
				t.Fatalf("#%d: thân phát lại khác lần đầu", i)
			}
		case http.StatusConflict:
			n409++
			if idemCode(t, w) != "IDEMPOTENCY_IN_PROGRESS" || w.Header().Get("Retry-After") != "1" {
				t.Fatalf("#%d: 409 thiếu code/Retry-After: %s %q", i, w.Body, w.Header().Get("Retry-After"))
			}
		default:
			t.Fatalf("#%d: status = %d (chỉ được 201 phát lại hoặc 409)", i, w.Code)
		}
	}
	if n201 < 1 || n201+n409 != n {
		t.Fatalf("201 = %d, 409 = %d", n201, n409)
	}
	if orig != 1 {
		t.Fatalf("%d phản hồi 2xx không mang Idempotent-Replayed, muốn đúng 1 bản gốc", orig)
	}
	if got := idemRows(t, d, sub); got != 1 {
		t.Fatalf("idempotency_keys có %d dòng, muốn 1", got)
	}
}

func TestIdempotency_TTL(t *testing.T) {
	t.Parallel()
	d, _ := idemDeps(t)
	c := &idemCounter{gate: make(chan struct{})}
	h := idemRouter(d, c.handler())
	sub, key := uuid.NewString(), "ttl-"+uuid.NewString()

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- idemPost(h, "/api/v1/_test/items", sub, key, `{"name":"a"}`) }()

	// Trong lúc handler chạy: khoá …:lock tồn tại với TTL ≤ 30 s.
	lockKey := ""
	for range 100 {
		keys, err := d.Redis.Keys(t.Context(), "ep:idem:"+sub+":*:lock").Result()
		if err != nil {
			t.Fatalf("Keys: %v", err)
		}
		if len(keys) == 1 {
			lockKey = keys[0]
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if lockKey == "" {
		t.Fatal("không thấy khoá …:lock khi handler đang chạy")
	}
	lockTTL, err := d.Redis.TTL(t.Context(), lockKey).Result()
	if err != nil {
		t.Fatalf("TTL lock: %v", err)
	}
	if lockTTL <= 0 || lockTTL > 30*time.Second {
		t.Fatalf("TTL lock = %s, muốn trong (0, 30s]", lockTTL)
	}
	close(c.gate)
	<-done

	respKey := "ep:idem:" + sub + ":" + strings.TrimSuffix(strings.TrimPrefix(lockKey, "ep:idem:"+sub+":"), ":lock")
	ttl, err := d.Redis.TTL(t.Context(), respKey).Result()
	if err != nil {
		t.Fatalf("TTL phản hồi: %v", err)
	}
	if ttl < 86300*time.Second || ttl > 24*time.Hour {
		t.Fatalf("TTL phản hồi = %s, muốn 86300–86400 s", ttl)
	}
	if n, err := d.Redis.Exists(t.Context(), lockKey).Result(); err != nil || n != 0 {
		t.Fatalf("khoá …:lock chưa được nhả (exists = %d, err = %v)", n, err)
	}
}

func TestIdempotency_5xxNotStored(t *testing.T) {
	t.Parallel()
	d, _ := idemDeps(t)
	c := &idemCounter{}
	h := idemRouter(d, c.handler())
	sub := uuid.NewString()

	for _, status := range []int{http.StatusInternalServerError, http.StatusTooManyRequests} {
		c.status.Store(int64(status))
		key := fmt.Sprintf("st%d-%s", status, uuid.NewString())
		before := c.n.Load()
		if w := idemPost(h, "/api/v1/_test/items", sub, key, `{"name":"a"}`); w.Code != status {
			t.Fatalf("status = %d, muốn %d", w.Code, status)
		}
		if w := idemPost(h, "/api/v1/_test/items", sub, key, `{"name":"a"}`); w.Header().Get("Idempotent-Replayed") != "" {
			t.Fatalf("phản hồi %d không được phát lại", status)
		}
		if got := c.n.Load() - before; got != 2 {
			t.Fatalf("status %d: handler chạy %d lần, muốn 2 (không lưu)", status, got)
		}
	}
	if got := idemRows(t, d, sub); got != 0 {
		t.Fatalf("idempotency_keys có %d dòng, muốn 0", got)
	}
}

func TestIdempotency_RedisDown_FailClosed(t *testing.T) {
	t.Parallel()
	d, _ := idemDeps(t)
	down, err := appredis.New(t.Context(), "redis://127.0.0.1:1/0") // cổng không ai nghe
	if err != nil {
		t.Fatalf("redis.New: %v", err)
	}
	t.Cleanup(func() { _ = down.Close() })
	d.Redis = down

	c := &idemCounter{}
	h := idemRouter(d, c.handler())
	sub := uuid.NewString()
	w := idemPost(h, "/api/v1/_test/items", sub, "down-"+uuid.NewString(), `{"name":"a"}`)
	if w.Code != http.StatusServiceUnavailable || idemCode(t, w) != "SERVICE_UNAVAILABLE" {
		t.Fatalf("status = %d, code = %q; muốn 503 SERVICE_UNAVAILABLE", w.Code, idemCode(t, w))
	}
	if got := c.n.Load(); got != 0 {
		t.Fatalf("handler chạy %d lần khi Redis chết, muốn 0 (fail-closed)", got)
	}
}

func TestIdempotency_RedisFlushedFallsBackToTable(t *testing.T) {
	t.Parallel()
	d, _ := idemDeps(t)
	c := &idemCounter{}
	h := idemRouter(d, c.handler())
	sub, key, body := uuid.NewString(), "flush-"+uuid.NewString(), `{"name":"a"}`

	w1 := idemPost(h, "/api/v1/_test/items", sub, key, body)
	if w1.Code != http.StatusCreated {
		t.Fatalf("lần đầu status = %d", w1.Code)
	}
	// Xoá mọi khoá idempotency của người dùng này (tương đương FLUSHALL, nhưng không đụng test khác).
	keys, err := d.Redis.Keys(t.Context(), "ep:idem:"+sub+":*").Result()
	if err != nil {
		t.Fatalf("Keys: %v", err)
	}
	if len(keys) == 0 {
		t.Fatal("không có khoá Redis nào để xoá")
	}
	if err := d.Redis.Del(t.Context(), keys...).Err(); err != nil {
		t.Fatalf("Del: %v", err)
	}

	w2 := idemPost(h, "/api/v1/_test/items", sub, key, body)
	if w2.Code != w1.Code || w2.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatalf("status = %d, replayed = %q; muốn %d + true", w2.Code, w2.Header().Get("Idempotent-Replayed"), w1.Code)
	}
	var j1, j2 map[string]any
	if err := json.Unmarshal(w1.Body.Bytes(), &j1); err != nil {
		t.Fatalf("thân 1: %v", err)
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &j2); err != nil {
		t.Fatalf("thân 2: %v", err)
	}
	if fmt.Sprint(j1) != fmt.Sprint(j2) {
		t.Fatalf("thân khôi phục khác: %v ≠ %v", j1, j2)
	}
	if got := c.n.Load(); got != 1 {
		t.Fatalf("handler chạy %d lần, muốn 1", got)
	}
	// Đã khôi phục lại vào Redis.
	if n, err := d.Redis.Exists(t.Context(), keys...).Result(); err != nil || n == 0 {
		t.Fatalf("phản hồi chưa được khôi phục vào Redis (exists = %d, err = %v)", n, err)
	}
}

func TestIdempotency_ScopedByUserAndEndpoint(t *testing.T) {
	t.Parallel()
	d, _ := idemDeps(t)
	c := &idemCounter{}
	h := idemRouter(d, c.handler())
	u1, u2, key, body := uuid.NewString(), uuid.NewString(), "scope-"+uuid.NewString(), `{"name":"a"}`

	w1 := idemPost(h, "/api/v1/_test/items", u1, key, body)
	w2 := idemPost(h, "/api/v1/_test/items", u2, key, body)
	w3 := idemPost(h, "/api/v1/_test/others", u1, key, body)

	for i, w := range []*httptest.ResponseRecorder{w1, w2, w3} {
		if w.Code != http.StatusCreated {
			t.Fatalf("#%d status = %d, muốn 201", i, w.Code)
		}
		if w.Header().Get("Idempotent-Replayed") != "" {
			t.Fatalf("#%d không được là phát lại", i)
		}
	}
	if got := c.n.Load(); got != 3 {
		t.Fatalf("handler chạy %d lần, muốn 3 (khoá độc lập theo user + endpoint)", got)
	}
	if got := idemRows(t, d, u1); got != 2 {
		t.Fatalf("u1 có %d dòng, muốn 2", got)
	}
}

func TestIdempotency_NotLogged(t *testing.T) {
	t.Parallel()
	d, logs := idemDeps(t)
	c := &idemCounter{}
	h := idemRouter(d, c.handler())
	sub, key := uuid.NewString(), "canary-key-"+uuid.NewString()

	idemPost(h, "/api/v1/_test/items", sub, key, `{"name":"pii-canary"}`)
	idemPost(h, "/api/v1/_test/items", sub, key, `{"name":"pii-canary"}`)         // phát lại
	idemPost(h, "/api/v1/_test/items", sub, key, `{"name":"pii-canary-khac"}`)    // 422 dùng lại
	idemPost(h, "/api/v1/_test/items", sub, strings.Repeat("x", 7), `{"name":1}`) // 422 khoá sai

	out := logs.String()
	if out == "" {
		t.Fatal("không có dòng log nào để kiểm")
	}
	if strings.Contains(out, key) {
		t.Fatalf("khoá Idempotency-Key lọt vào log:\n%s", out)
	}
	if strings.Contains(out, "pii-canary") {
		t.Fatalf("thân request lọt vào log:\n%s", out)
	}
}
