package llmconfig_test

// Kiểm thử API cấu hình LLM qua router THẬT (httpapi.NewRouter) trên Postgres + Redis thật và provider `fake`
// (US-P1-04). Một gateway dùng chung cho cả gói; mỗi test dọn bảng cấu hình trước khi chạy và KHÔNG chạy song song.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/llmconfig"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appdb "github.com/edupilot/backend-go/internal/platform/db"
	applog "github.com/edupilot/backend-go/internal/platform/log"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

const (
	apiSecret = "llmconfig-api-test-secret-0123456789abcdef"
	goodKey   = "good-key"
)

// syncBuf là bộ đệm log an toàn cho nhiều goroutine.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

type apiRig struct {
	h    http.Handler
	rt   *llmrt.Runtime
	pool *pgxpool.Pool
	rdb  *goredis.Client
	logs *syncBuf
	cfg  config.Config
}

var (
	apiOnce sync.Once
	apiVal  *apiRig
	apiErr  error
)

func getAPI(t *testing.T) *apiRig {
	t.Helper()
	testutil.RequireContainers(t)
	apiOnce.Do(func() { apiVal, apiErr = buildAPI(t) })
	require.NoError(t, apiErr)
	apiVal.reset(t)
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		for _, l := range strings.Split(apiVal.logs.String(), "\n") {
			if strings.Contains(l, `"level":"ERROR"`) {
				t.Log(l)
			}
		}
	})
	return apiVal
}

func buildAPI(t *testing.T) (*apiRig, error) {
	env := map[string]string{
		"DATABASE_URL": testutil.MigratedPostgresURL(t), "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": apiSecret,
		"BLOB_ENDPOINT": "127.0.0.1:9", "BLOB_BUCKET": "t", "BLOB_ACCESS_KEY": "t-dev", "BLOB_SECRET_KEY": "t-dev-secret",
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test", "REQUEST_TIMEOUT": "20s",
		"MAX_BODY_BYTES": "65536", "RATE_LIMIT_IP_PER_MIN": "1000000", "RATE_LIMIT_USER_PER_MIN": "1000000",
		"INSTANCE_ID": "llmcfg-gw", "DB_MAX_CONNS": "8", "LOG_LEVEL": "debug", "LLM_PROVIDER": "fake", "FAKE_LLM_VALID_KEY": goodKey,
		"FAKE_LLM_LATENCY_MIN": "1ms", "FAKE_LLM_LATENCY_MAX": "2ms",
	}
	cfg, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	if err != nil {
		return nil, err
	}
	logs := &syncBuf{}
	log := applog.NewTo(logs, cfg, "gateway")
	ctx := context.Background()
	pool, err := appdb.NewPool(ctx, cfg, log)
	if err != nil {
		return nil, err
	}
	rdb, err := appredis.New(ctx, cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	rt, err := llmrt.New(ctx, cfg, pool, rdb.Client, log)
	if err != nil {
		return nil, err
	}
	deps := httpapi.Deps{Cfg: cfg, Log: log, DB: pool, Redis: rdb, Clock: clock.Real{}, State: httpapi.NewState(), Jobs: jobs.NewService(pool), LLM: rt}
	return &apiRig{h: httpapi.NewRouter(deps), rt: rt, pool: pool, rdb: rdb.Client, logs: logs, cfg: cfg}, nil
}

// reset dọn cấu hình LLM giữa các test (audit_log append-only nên đếm tương đối).
func (r *apiRig) reset(t *testing.T) {
	t.Helper()
	_, err := r.pool.Exec(t.Context(), `truncate llm_task_routes, llm_models, llm_providers, llm_audit; delete from llm_budgets`)
	require.NoError(t, err)
	keys, err := r.rdb.Keys(t.Context(), "ep:llm:budget*").Result() // chi phí đã đếm của test trước
	require.NoError(t, err)
	if len(keys) > 0 {
		require.NoError(t, r.rdb.Del(t.Context(), keys...).Err())
	}
	r.rt.Budget.Invalidate()
	r.rt.Registry.Fake().Set(fake.Settings{ValidKey: goodKey, LatencyMin: time.Millisecond, LatencyMax: 2 * time.Millisecond})
	require.NoError(t, r.rt.Registry.Reload(t.Context()))
}

func (r *apiRig) token(t *testing.T, role auth.Role) string {
	t.Helper()
	return r.tokenFor(t, uuid.NewString(), role, clock.Real{}, time.Hour)
}

type shiftClock struct{ d time.Duration }

func (c shiftClock) Now() time.Time { return time.Now().Add(c.d) }

func (r *apiRig) tokenFor(t *testing.T, sub string, role auth.Role, clk clock.Clock, ttl time.Duration) string {
	t.Helper()
	tok, err := auth.NewIssuer(apiSecret, ttl, clk).Issue(sub, role, "")
	require.NoError(t, err)
	return tok
}

type resp struct {
	status int
	hdr    http.Header
	body   []byte
}

func (x resp) obj(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(x.body, &m), string(x.body))
	return m
}

func (x resp) code(t *testing.T) string { t.Helper(); s, _ := x.obj(t)["code"].(string); return s }

// do gọi router; body là map/struct (JSON hoá) hoặc []byte nguyên văn. hdr = cặp khoá, giá trị.
func (r *apiRig) do(t *testing.T, tok, method, path string, body any, hdr ...string) resp {
	t.Helper()
	var rd io.Reader
	switch b := body.(type) {
	case nil:
	case []byte:
		rd = bytes.NewReader(b)
	default:
		raw, err := json.Marshal(b)
		require.NoError(t, err)
		rd = bytes.NewReader(raw)
	}
	req := httptest.NewRequestWithContext(t.Context(), method, "/api/v1"+path, rd)
	if rd != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	rec := httptest.NewRecorder()
	r.h.ServeHTTP(rec, req)
	return resp{status: rec.Code, hdr: rec.Header(), body: rec.Body.Bytes()}
}

func idem() string { return "idem-" + uuid.NewString() }

func fakeModels() []map[string]any {
	return []map[string]any{{"model": "fake-chat", "kind": "chat", "price_in": "0", "price_out": "0"}}
}

// mk tạo nhà cung cấp fake qua API (khoá đúng), đòi 201.
func (r *apiRig) mk(t *testing.T, tok, name string, models ...map[string]any) map[string]any {
	t.Helper()
	if len(models) == 0 {
		models = fakeModels()
	}
	x := r.do(t, tok, "POST", "/admin/llm/providers", map[string]any{"type": "fake", "name": name, "api_key": goodKey, "models": models}, "Idempotency-Key", idem())
	require.Equal(t, http.StatusCreated, x.status, string(x.body))
	return x.obj(t)
}

func count(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(t.Context(), sql, args...).Scan(&n))
	return n
}

// ---- AC1: ma trận phân quyền ----

func TestRBACMatrix(t *testing.T) {
	r := getAPI(t)
	rid := uuid.NewString()
	type op struct {
		method, path string
		body         any
		read         bool // TEACHER được phép
		admin        bool // chỉ ADMIN
	}
	ops := []op{
		{"GET", "/admin/llm/providers", nil, true, false},
		{"POST", "/admin/llm/providers", map[string]any{}, false, true},
		{"PUT", "/admin/llm/providers/" + rid, map[string]any{}, false, true},
		{"DELETE", "/admin/llm/providers/" + rid, nil, false, true},
		{"POST", "/admin/llm/providers/" + rid + "/test", nil, false, true},
		{"POST", "/admin/llm/providers/test", map[string]any{}, false, true},
		{"GET", "/admin/llm/routes", nil, true, false},
		{"PUT", "/admin/llm/routes", map[string]any{}, false, true},
		{"GET", "/admin/llm/usage", nil, true, false},
		{"GET", "/admin/llm/budget", nil, true, false},
		{"PUT", "/admin/llm/budget", map[string]any{}, false, true},
		{"GET", "/courses/" + rid + "/llm-budget", nil, false, true},
		{"PUT", "/courses/" + rid + "/llm-budget", map[string]any{}, false, true},
	}
	require.Len(t, ops, 13)
	cases := 0
	for _, o := range ops {
		for _, role := range []auth.Role{auth.RoleAdmin, auth.RoleTeacher, auth.RoleTA, auth.RoleStudent} {
			allowed := role == auth.RoleAdmin || (role == auth.RoleTeacher && o.read)
			x := r.do(t, r.token(t, role), o.method, o.path, o.body)
			name := fmt.Sprintf("%s %s %s", role, o.method, o.path)
			if allowed {
				require.NotContains(t, []int{401, 403}, x.status, name+" "+string(x.body))
			} else {
				require.Equal(t, 403, x.status, name)
				require.Equal(t, "FORBIDDEN", x.code(t), name)
				require.Equal(t, "role", x.obj(t)["details"].(map[string]any)["reason"], name)
			}
			cases++
		}
		x := r.do(t, "", o.method, o.path, o.body)
		require.Equal(t, 401, x.status, o.method+" "+o.path)
		require.Equal(t, "UNAUTHENTICATED", x.code(t))
		cases++
		exp := r.tokenFor(t, uuid.NewString(), auth.RoleAdmin, shiftClock{-2 * time.Hour}, time.Hour)
		x = r.do(t, exp, o.method, o.path, o.body)
		require.Equal(t, 401, x.status, o.method+" "+o.path)
		require.Equal(t, "TOKEN_EXPIRED", x.code(t))
		cases++
	}
	require.GreaterOrEqual(t, cases, 60)
	// chặn TRƯỚC khi chạm nhà cung cấp: TEACHER bị 403 mà fake không bị gọi.
	before := r.rt.Registry.Fake().Calls()
	r.do(t, r.token(t, auth.RoleTeacher), "POST", "/admin/llm/providers/test", map[string]any{"type": "fake", "api_key": goodKey, "model": "fake-chat"})
	require.Equal(t, before, r.rt.Registry.Fake().Calls())
}

// ---- AC2: Test ----

func TestProviderTestEndpoint(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	nProv := count(t, r.pool, `select count(*) from llm_providers`)
	audits := count(t, r.pool, `select count(*) from audit_log where action = 'test'`)

	bad := r.do(t, adm, "POST", "/admin/llm/providers/test", map[string]any{"type": "fake", "api_key": canary, "model": "fake-chat"})
	require.Equal(t, 200, bad.status, string(bad.body))
	o := bad.obj(t)
	require.Equal(t, false, o["ok"])
	require.Equal(t, "AUTH", o["error_kind"])
	require.Equal(t, "Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá.", o["message"])
	require.Contains(t, o, "latency_ms")
	require.NotContains(t, string(bad.body), canary)

	good := r.do(t, adm, "POST", "/admin/llm/providers/test", map[string]any{"type": "fake", "api_key": goodKey, "model": "fake-chat"})
	require.Equal(t, 200, good.status)
	g := good.obj(t)
	require.Equal(t, true, g["ok"])
	require.NotContains(t, g, "error_kind")

	require.Equal(t, nProv, count(t, r.pool, `select count(*) from llm_providers`), "Test không được ghi bản ghi")
	require.Equal(t, audits+2, count(t, r.pool, `select count(*) from audit_log where action = 'test'`), "mỗi lần Test một dòng audit_log")
	require.Zero(t, count(t, r.pool, `select count(*) from audit_log where after::text like '%'||$1||'%'`, canary))
	require.NotContains(t, r.logs.String(), canary)

	// Mỗi loại lỗi có câu tiếng Việt riêng (bảy loại).
	want := map[provider.Kind]string{
		provider.KindNetwork: "NETWORK", provider.KindTimeout: "TIMEOUT", provider.KindRateLimit: "RATE_LIMIT",
		provider.KindModelNotFound: "MODEL_NOT_FOUND", provider.KindBadResponse: "BAD_RESPONSE", provider.KindServer: "NETWORK",
	}
	msgs := map[string]bool{"Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá.": true}
	for k, ek := range want {
		r.rt.Registry.Fake().Set(fake.Settings{ValidKey: goodKey, ErrorRate: 1, ErrorKind: k})
		x := r.do(t, r.token(t, auth.RoleAdmin), "POST", "/admin/llm/providers/test", map[string]any{"type": "fake", "api_key": goodKey, "model": "fake-chat"})
		require.Equal(t, 200, x.status, string(x.body))
		require.Equal(t, ek, x.obj(t)["error_kind"], string(k))
		if ek != "NETWORK" || k == provider.KindNetwork {
			require.False(t, msgs[x.obj(t)["message"].(string)], "câu bị trùng: %s", x.obj(t)["message"])
			msgs[x.obj(t)["message"].(string)] = true
		}
	}
	r.rt.Registry.Fake().Set(fake.Settings{ValidKey: goodKey})
	// DIMS_MISMATCH: mô hình nhúng chỉ trả vectơ ≠ 1536.
	r.rt.Registry.Fake().Set(fake.Settings{ValidKey: goodKey, ErrorRate: 1, ErrorKind: provider.KindDimsMismatch})
	x := r.do(t, r.token(t, auth.RoleAdmin), "POST", "/admin/llm/providers/test", map[string]any{"type": "fake", "api_key": goodKey, "model": "fake-embed", "kind": "embedding"})
	require.Equal(t, "DIMS_MISMATCH", x.obj(t)["error_kind"])
	require.False(t, msgs[x.obj(t)["message"].(string)])

	// Bản ghi đã lưu: kết quả vào last_test; ghi đè api_key thì KHÔNG vào last_test.
	r.rt.Registry.Fake().Set(fake.Settings{ValidKey: goodKey})
	p := r.mk(t, adm, "A")
	id := p["id"].(string)
	x = r.do(t, adm, "POST", "/admin/llm/providers/"+id+"/test", nil)
	require.Equal(t, true, x.obj(t)["ok"], string(x.body))
	list := r.do(t, adm, "GET", "/admin/llm/providers", nil).obj(t)["items"].([]any)[0].(map[string]any)
	lt := list["last_test"].(map[string]any)
	require.Equal(t, true, lt["ok"])
	x = r.do(t, adm, "POST", "/admin/llm/providers/"+id+"/test", map[string]any{"api_key": "wrong"})
	require.Equal(t, "AUTH", x.obj(t)["error_kind"])
	list = r.do(t, adm, "GET", "/admin/llm/providers", nil).obj(t)["items"].([]any)[0].(map[string]any)
	require.Equal(t, true, list["last_test"].(map[string]any)["ok"], "Test có ghi đè không được đổi last_test")
	require.Equal(t, 404, r.do(t, adm, "POST", "/admin/llm/providers/"+uuid.NewString()+"/test", nil).status)
}

func TestTestRateLimit(t *testing.T) {
	r := getAPI(t)
	sub := uuid.NewString()
	tok := r.tokenFor(t, sub, auth.RoleAdmin, clock.Real{}, time.Hour)
	body := map[string]any{"type": "fake", "api_key": goodKey, "model": "fake-chat"}
	for i := range 10 {
		require.Equal(t, 200, r.do(t, tok, "POST", "/admin/llm/providers/test", body).status, "lần %d", i+1)
	}
	x := r.do(t, tok, "POST", "/admin/llm/providers/test", body)
	require.Equal(t, 429, x.status)
	require.Equal(t, "RATE_LIMITED", x.code(t))
	require.NotEmpty(t, x.hdr.Get("Retry-After"))
	// người khác không bị ảnh hưởng; các thao tác khác của cùng người không bị chặn
	require.Equal(t, 200, r.do(t, r.token(t, auth.RoleAdmin), "POST", "/admin/llm/providers/test", body).status)
	require.Equal(t, 200, r.do(t, tok, "GET", "/admin/llm/providers", nil).status)
}

// ---- AC3: lưu kiểm tra trước ----

func TestSaveVerifiesFirst(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	x := r.do(t, adm, "POST", "/admin/llm/providers", map[string]any{"type": "fake", "name": "F", "api_key": "bad-key", "models": fakeModels()}, "Idempotency-Key", idem())
	require.Equal(t, 422, x.status, string(x.body))
	require.Equal(t, "VALIDATION_FAILED", x.code(t))
	d := x.obj(t)["details"].([]any)[0].(map[string]any)
	require.Equal(t, "api_key", d["field"])
	require.Equal(t, "PROVIDER_AUTH_FAILED", d["code"])
	require.Zero(t, count(t, r.pool, `select count(*) from llm_providers`), "khoá sai không được lưu")
	require.NotContains(t, string(x.body), "bad-key")

	// không tới được máy chủ → PROVIDER_UNREACHABLE
	r.rt.Registry.Fake().Set(fake.Settings{ValidKey: goodKey, ErrorRate: 1, ErrorKind: provider.KindNetwork})
	x = r.do(t, adm, "POST", "/admin/llm/providers", map[string]any{"type": "fake", "name": "F", "api_key": goodKey, "models": fakeModels()}, "Idempotency-Key", idem())
	require.Equal(t, 422, x.status)
	require.Equal(t, "PROVIDER_UNREACHABLE", x.obj(t)["details"].([]any)[0].(map[string]any)["code"])
	r.rt.Registry.Fake().Set(fake.Settings{ValidKey: goodKey})

	// PUT khoá sai: 422, khoá cũ giữ nguyên, version không đổi.
	p := r.mk(t, adm, "A")
	id := uuid.MustParse(p["id"].(string))
	put := map[string]any{"type": "fake", "name": "A", "api_key": "bad-key", "models": fakeModels(), "version": p["version"]}
	x = r.do(t, adm, "PUT", "/admin/llm/providers/"+id.String(), put)
	require.Equal(t, 422, x.status, string(x.body))
	require.Equal(t, "PROVIDER_AUTH_FAILED", x.obj(t)["details"].([]any)[0].(map[string]any)["code"])
	k, err := r.rt.Resolver.DecryptKey(t.Context(), id)
	require.NoError(t, err)
	require.Equal(t, goodKey, k)
	cur := r.do(t, adm, "GET", "/admin/llm/providers", nil).obj(t)["items"].([]any)[0].(map[string]any)
	require.EqualValues(t, p["version"], cur["version"])

	// lưu đúng: last_test.ok = true
	require.Equal(t, true, p["last_test"].(map[string]any)["ok"])
}

func TestSaveSkipVerify(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	before := count(t, r.pool, `select count(*) from audit_log where entity = 'llm_provider' and after->>'skip_verify' = 'true'`)
	x := r.do(t, adm, "POST", "/admin/llm/providers", map[string]any{"type": "fake", "name": "S", "api_key": "bad-key", "skip_verify": true, "models": fakeModels()}, "Idempotency-Key", idem())
	require.Equal(t, 201, x.status, string(x.body))
	o := x.obj(t)
	lt := o["last_test"].(map[string]any)
	require.Nil(t, lt["ok"], "skip_verify → last_test.ok = null")
	require.Equal(t, true, o["has_key"])
	require.Equal(t, before+1, count(t, r.pool, `select count(*) from audit_log where entity = 'llm_provider' and after->>'skip_verify' = 'true'`))
	require.NotContains(t, string(x.body), "bad-key")
	// TEACHER không có skip_verify (chặn ở RBAC): đã phủ ở TestRBACMatrix.
}

func TestUpdateWithoutKeyNoVerify(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	p := r.mk(t, adm, "A")
	r.rt.Registry.Fake().Set(fake.Settings{ValidKey: goodKey, ErrorRate: 1, ErrorKind: provider.KindNetwork}) // nếu có kiểm tra thì sẽ hỏng
	calls := r.rt.Registry.Fake().Calls()
	models := []map[string]any{{"model": "fake-chat", "kind": "chat", "price_in": "1.5", "price_out": "2.5", "enabled": true}}
	x := r.do(t, adm, "PUT", "/admin/llm/providers/"+p["id"].(string), map[string]any{"type": "fake", "name": "B", "enabled": false, "models": models, "version": p["version"]})
	require.Equal(t, 200, x.status, string(x.body))
	o := x.obj(t)
	require.Equal(t, "B", o["name"])
	require.Equal(t, false, o["enabled"])
	require.EqualValues(t, 2, o["version"])
	require.Equal(t, "W/\"v2\"", x.hdr.Get("ETag"))
	require.Equal(t, "1.5000", o["models"].([]any)[0].(map[string]any)["price_in"])
	require.Equal(t, true, o["last_test"].(map[string]any)["ok"], "đổi tên/giá không xoá kết quả Test")
	require.Equal(t, calls, r.rt.Registry.Fake().Calls(), "không được gọi nhà cung cấp")
}

// ---- AC5: CRUD ----

func TestProviderCRUD(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	p := r.mk(t, adm, "OpenAI")
	for _, k := range []string{"id", "type", "name", "base_url", "enabled", "has_key", "key_status", "rpm_limit", "tpm_limit", "last_test", "circuit", "models", "version"} {
		require.Contains(t, p, k)
	}
	require.Equal(t, "ok", p["key_status"])
	require.Equal(t, "closed", p["circuit"])
	require.Equal(t, true, p["has_key"])
	require.NotContains(t, p, "api_key")
	list := r.do(t, adm, "GET", "/admin/llm/providers", nil)
	require.Equal(t, 200, list.status)
	require.NotEmpty(t, list.hdr.Get("ETag"))
	lo := list.obj(t)
	require.Len(t, lo["items"], 1)
	fb := lo["env_fallback"].(map[string]any)
	require.Equal(t, true, fb["active"], "chưa có tuyến trong DB → vẫn chạy bằng env dự phòng")
	require.Equal(t, []any{"fake"}, fb["providers"])
	require.Equal(t, 304, r.do(t, adm, "GET", "/admin/llm/providers", nil, "If-None-Match", list.hdr.Get("ETag")).status)

	del := r.do(t, adm, "DELETE", "/admin/llm/providers/"+p["id"].(string), nil)
	require.Equal(t, 204, del.status)
	require.Equal(t, 404, r.do(t, adm, "DELETE", "/admin/llm/providers/"+p["id"].(string), nil).status)
	require.Equal(t, 404, r.do(t, adm, "DELETE", "/admin/llm/providers/not-a-uuid", nil).status)
	require.Empty(t, r.do(t, adm, "GET", "/admin/llm/providers", nil).obj(t)["items"])
}

func TestProviderVersionConflict(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	p := r.mk(t, adm, "A")
	id := p["id"].(string)
	put := func(name string, version int) resp {
		return r.do(t, adm, "PUT", "/admin/llm/providers/"+id, map[string]any{"type": "fake", "name": name, "version": version})
	}
	require.Equal(t, 200, put("A2", 1).status)
	x := put("A3", 1)
	require.Equal(t, 409, x.status)
	require.Equal(t, "VERSION_CONFLICT", x.code(t))
	require.EqualValues(t, 2, x.obj(t)["details"].(map[string]any)["current_version"])
	require.Equal(t, "W/\"v2\"", x.hdr.Get("ETag"))
	// If-Match thay cho version trong thân; thiếu cả hai → 422
	require.Equal(t, 200, r.do(t, adm, "PUT", "/admin/llm/providers/"+id, map[string]any{"type": "fake", "name": "A4"}, "If-Match", `W/"v2"`).status)
	require.Equal(t, 422, r.do(t, adm, "PUT", "/admin/llm/providers/"+id, map[string]any{"type": "fake", "name": "A5"}).status)
	// trùng tên → 409 CONFLICT
	r.mk(t, adm, "Khac")
	dup := r.do(t, adm, "PUT", "/admin/llm/providers/"+id, map[string]any{"type": "fake", "name": "Khac", "version": 3})
	require.Equal(t, 409, dup.status)
	require.Equal(t, "CONFLICT", dup.code(t))
	dup = r.do(t, adm, "POST", "/admin/llm/providers", map[string]any{"type": "fake", "name": "Khac", "api_key": goodKey, "models": fakeModels()}, "Idempotency-Key", idem())
	require.Equal(t, "CONFLICT", dup.code(t))
}

func TestProviderInUse(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	p := r.mk(t, adm, "A")
	mid := p["models"].([]any)[0].(map[string]any)["id"].(string)
	put := r.do(t, adm, "PUT", "/admin/llm/routes", map[string]any{"task": "CHAT", "chain": []string{mid}, "version": 0})
	require.Equal(t, 200, put.status, string(put.body))
	x := r.do(t, adm, "DELETE", "/admin/llm/providers/"+p["id"].(string), nil)
	require.Equal(t, 409, x.status)
	require.Equal(t, "PROVIDER_IN_USE", x.code(t))
	require.Equal(t, []any{"CHAT"}, x.obj(t)["details"].(map[string]any)["tasks"])
	// bỏ mô hình đang được dùng khỏi danh sách cũng bị chặn
	y := r.do(t, adm, "PUT", "/admin/llm/providers/"+p["id"].(string), map[string]any{"type": "fake", "name": "A", "models": []map[string]any{{"model": "khac", "kind": "chat", "price_in": "0", "price_out": "0"}}, "skip_verify": true, "version": 1})
	require.Equal(t, 409, y.status, string(y.body))
	require.Equal(t, "PROVIDER_IN_USE", y.code(t))
}

func TestProviderIdempotent(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	key := idem()
	body := map[string]any{"type": "fake", "name": "I", "api_key": goodKey, "models": fakeModels()}
	a := r.do(t, adm, "POST", "/admin/llm/providers", body, "Idempotency-Key", key)
	require.Equal(t, 201, a.status, string(a.body))
	b := r.do(t, adm, "POST", "/admin/llm/providers", body, "Idempotency-Key", key)
	require.Equal(t, 201, b.status)
	require.Equal(t, "true", b.hdr.Get("Idempotent-Replayed"))
	require.JSONEq(t, string(a.body), string(b.body))
	require.Equal(t, 1, count(t, r.pool, `select count(*) from llm_providers`))
	nokey := r.do(t, adm, "POST", "/admin/llm/providers", body)
	require.Equal(t, 422, nokey.status)
	require.Equal(t, "IDEMPOTENCY_KEY_REQUIRED", nokey.code(t))
}

func TestProviderLimit(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	for i := range llmconfig.MaxProviders {
		r.mk(t, adm, fmt.Sprintf("P%02d", i))
	}
	x := r.do(t, adm, "POST", "/admin/llm/providers", map[string]any{"type": "fake", "name": "Du", "api_key": goodKey, "models": fakeModels()}, "Idempotency-Key", idem())
	require.Equal(t, 422, x.status, string(x.body))
	require.Equal(t, "VALIDATION_FAILED", x.code(t))
	require.Contains(t, x.obj(t)["details"].([]any)[0].(map[string]any)["message"], "tối đa 20")
	require.Equal(t, llmconfig.MaxProviders, count(t, r.pool, `select count(*) from llm_providers`))
}

// ---- AC13: base_url ----

func TestBaseURLValidation(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	bad := []string{"file:///etc/passwd", "ftp://h/x", "javascript:alert(1)", "http://u:p@h/v1", "http://", "http://h/#x", "https://" + strings.Repeat("a", 300) + ".example", "", "not a url", "//h/v1"}
	for _, u := range bad {
		in := map[string]any{"type": "openai_compatible", "base_url": u, "api_key": "x", "model": "m"}
		x := r.do(t, adm, "POST", "/admin/llm/providers/test", in)
		require.Equal(t, 422, x.status, "test %q: %s", u, x.body)
		require.Equal(t, "VALIDATION_FAILED", x.code(t))
		code := x.obj(t)["details"].([]any)[0].(map[string]any)["code"]
		if u != "" {
			require.Equal(t, "INVALID_BASE_URL", code, u)
		}
		y := r.do(t, adm, "POST", "/admin/llm/providers", map[string]any{"type": "openai_compatible", "name": "U", "base_url": u, "api_key": "x", "skip_verify": true, "models": fakeModels()}, "Idempotency-Key", idem())
		require.Equal(t, 422, y.status, "save %q: %s", u, y.body)
	}
	require.Zero(t, count(t, r.pool, `select count(*) from llm_providers`))
	// PUT cũng kiểm
	p := r.mk(t, adm, "A")
	x := r.do(t, adm, "PUT", "/admin/llm/providers/"+p["id"].(string), map[string]any{"type": "fake", "name": "A", "base_url": "http://u:p@h", "version": 1})
	require.Equal(t, 422, x.status)
	require.Equal(t, "INVALID_BASE_URL", x.obj(t)["details"].([]any)[0].(map[string]any)["code"])
}

func TestBaseURLInternalAllowed(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	for _, u := range []string{"http://localhost:9/v1", "http://127.0.0.1:9/v1", "http://169.254.169.254/v1", "http://host.docker.internal:9999/v1", "http://gateway/v1", "https://10.0.0.5:9/v1"} {
		x := r.do(t, adm, "POST", "/admin/llm/providers/test", map[string]any{"type": "openai_compatible", "base_url": u, "api_key": "x", "model": "m"})
		require.Equal(t, 200, x.status, "%s: %s", u, x.body)
		o := x.obj(t)
		require.Equal(t, false, o["ok"], u) // không có máy chủ ở đó, nhưng địa chỉ được CHẤP NHẬN
		require.Contains(t, []any{"NETWORK", "TIMEOUT", "AUTH", "BAD_RESPONSE", "MODEL_NOT_FOUND"}, o["error_kind"], u)
	}
}

// Máy chủ nhà cung cấp trả 302 → không theo; lỗi BAD_RESPONSE; địa chỉ đích không bị chạm.
func TestProviderNoRedirectFollow(t *testing.T) {
	r := getAPI(t)
	hits := 0
	var mu sync.Mutex
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { mu.Lock(); hits++; mu.Unlock() }))
	defer target.Close()
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		http.Redirect(w, rq, target.URL+"/steal", http.StatusFound)
	}))
	defer redir.Close()
	x := r.do(t, r.token(t, auth.RoleAdmin), "POST", "/admin/llm/providers/test", map[string]any{"type": "openai_compatible", "base_url": redir.URL + "/v1", "api_key": "x", "model": "m"})
	require.Equal(t, 200, x.status, string(x.body))
	require.Equal(t, false, x.obj(t)["ok"])
	require.Equal(t, "BAD_RESPONSE", x.obj(t)["error_kind"])
	mu.Lock()
	defer mu.Unlock()
	require.Zero(t, hits, "không được theo chuyển hướng")
}

// Thân lỗi của nhà cung cấp không bao giờ ra ngoài (kể cả khi Test thất bại).
func TestTestEndpointNeverEchoesProviderBody(t *testing.T) {
	r := getAPI(t)
	const leak = "SECRET-PROVIDER-BODY-sk-live-abcdef123456"
	for _, st := range []int{400, 401, 404, 429, 500, 503} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(st)
			_, _ = w.Write([]byte(`{"error":{"message":"` + leak + `","type":"x"}}`))
		}))
		x := r.do(t, r.token(t, auth.RoleAdmin), "POST", "/admin/llm/providers/test", map[string]any{"type": "openai_compatible", "base_url": srv.URL + "/v1", "api_key": "x", "model": "m"})
		srv.Close()
		require.Equal(t, 200, x.status)
		require.NotContains(t, string(x.body), leak, "HTTP %d", st)
		require.NotContains(t, string(x.body), "abcdef123456")
		require.Equal(t, false, x.obj(t)["ok"])
	}
	require.NotContains(t, r.logs.String(), "abcdef123456", "thân lỗi nhà cung cấp không được vào log")
}

// Gemini trả 400 nhắc "API key" khi khoá sai → AUTH — chỉ ở Test.
func TestGeminiBadKeyIsAuthOnlyInTest(t *testing.T) {
	r := getAPI(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":{"message":"API key not valid. Please pass a valid API key.","status":"INVALID_ARGUMENT"}}`))
	}))
	defer srv.Close()
	x := r.do(t, r.token(t, auth.RoleAdmin), "POST", "/admin/llm/providers/test", map[string]any{"type": "gemini", "base_url": srv.URL + "/v1beta/openai/", "api_key": "x", "model": "gemini-3.6-flash"})
	require.Equal(t, "AUTH", x.obj(t)["error_kind"], string(x.body))
}

// ---- AC6: tuyến ----

func TestRoutesGetShape(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	x := r.do(t, adm, "GET", "/admin/llm/routes", nil)
	require.Equal(t, 200, x.status, string(x.body))
	o := x.obj(t)
	items := o["items"].([]any)
	require.Len(t, items, 7)
	tasks := []string{"CHAT", "CLASSIFY", "UTILITY", "GRADING", "QUESTION_GEN", "INSIGHT", "EMBEDDING"}
	for i, it := range items {
		m := it.(map[string]any)
		require.Equal(t, tasks[i], m["task"])
		for _, k := range []string{"lane", "chain", "params", "version"} {
			require.Contains(t, m, k)
		}
	}
	require.Equal(t, "INTERACTIVE", items[0].(map[string]any)["lane"])
	require.Nil(t, o["embedding"])

	p := r.mk(t, adm, "A", fakeModels()[0], map[string]any{"model": "fake-embed", "kind": "embedding", "dims": 1536, "price_in": "0", "price_out": "0"})
	ms := p["models"].([]any)
	var chat, emb string
	for _, m := range ms {
		mm := m.(map[string]any)
		if mm["kind"] == "chat" {
			chat = mm["id"].(string)
		} else {
			emb = mm["id"].(string)
		}
	}
	require.Equal(t, 200, r.do(t, adm, "PUT", "/admin/llm/routes", map[string]any{"task": "CHAT", "chain": []string{chat}, "params": map[string]any{"temperature": 0.3}, "version": 0}).status)
	require.Equal(t, 200, r.do(t, adm, "PUT", "/admin/llm/routes", map[string]any{"task": "EMBEDDING", "chain": []string{emb}, "version": 0}).status)
	x = r.do(t, adm, "GET", "/admin/llm/routes", nil)
	o = x.obj(t)
	c := o["items"].([]any)[0].(map[string]any)
	require.Equal(t, "fake-chat", c["chain"].([]any)[0].(map[string]any)["model"])
	require.Equal(t, "A", c["chain"].([]any)[0].(map[string]any)["provider_name"])
	require.EqualValues(t, 0.3, c["params"].(map[string]any)["temperature"])
	e := o["embedding"].(map[string]any)
	require.EqualValues(t, 1536, e["dims"])
	require.Equal(t, false, e["reindex_required"])
	require.Contains(t, e, "indexed_chunks")
	require.Nil(t, e["indexed_chunks"])
	// TEACHER đọc được
	require.Equal(t, 200, r.do(t, r.token(t, auth.RoleTeacher), "GET", "/admin/llm/routes", nil).status)
}

func TestRoutesPutRules(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	p := r.mk(t, adm, "A", fakeModels()[0], map[string]any{"model": "fake-embed", "kind": "embedding", "dims": 1536, "price_in": "0", "price_out": "0"})
	var chat, emb string
	for _, m := range p["models"].([]any) {
		mm := m.(map[string]any)
		if mm["kind"] == "chat" {
			chat = mm["id"].(string)
		} else {
			emb = mm["id"].(string)
		}
	}
	put := func(body map[string]any) resp { return r.do(t, adm, "PUT", "/admin/llm/routes", body) }
	rule := func(x resp) string {
		require.Equal(t, 422, x.status, string(x.body))
		require.Equal(t, "ROUTE_INVALID", x.code(t), string(x.body))
		return x.obj(t)["details"].(map[string]any)["rule"].(string)
	}
	require.Equal(t, "chain_empty", rule(put(map[string]any{"task": "CHAT", "chain": []string{}, "version": 0})))
	require.Equal(t, "kind_mismatch", rule(put(map[string]any{"task": "CHAT", "chain": []string{emb}, "version": 0})))
	require.Equal(t, "kind_mismatch", rule(put(map[string]any{"task": "EMBEDDING", "chain": []string{chat}, "version": 0})))
	require.Equal(t, "duplicate_model", rule(put(map[string]any{"task": "CHAT", "chain": []string{chat, chat}, "version": 0})))
	require.Equal(t, "params_out_of_range", rule(put(map[string]any{"task": "CHAT", "chain": []string{chat}, "params": map[string]any{"temperature": 5}, "version": 0})))
	require.Equal(t, "embedding_single", rule(put(map[string]any{"task": "EMBEDDING", "chain": []string{emb, emb}, "version": 0})))
	bad := put(map[string]any{"task": "CHAT", "chain": []string{"not-a-uuid"}, "version": 0})
	require.Equal(t, 422, bad.status)
	require.Equal(t, 422, put(map[string]any{"task": "NOPE", "chain": []string{chat}, "version": 0}).status)
	ok := put(map[string]any{"task": "CHAT", "chain": []string{chat}, "version": 0})
	require.Equal(t, 200, ok.status, string(ok.body))
	require.Equal(t, "W/\"v1\"", ok.hdr.Get("ETag"))
	st := put(map[string]any{"task": "CHAT", "chain": []string{chat}, "version": 0})
	require.Equal(t, 409, st.status)
	require.Equal(t, "VERSION_CONFLICT", st.code(t))
	// nhà đang tắt không được đứng trong tuyến
	q := r.mk(t, adm, "Tat")
	off := r.do(t, adm, "PUT", "/admin/llm/providers/"+q["id"].(string), map[string]any{"type": "fake", "name": "Tat", "enabled": false, "version": 1})
	require.Equal(t, 200, off.status, string(off.body))
	qm := q["models"].([]any)[0].(map[string]any)["id"].(string)
	require.Equal(t, "provider_disabled", rule(put(map[string]any{"task": "CLASSIFY", "chain": []string{qm}, "version": 0})))
}

func TestRoutesHotReload(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	a := r.mk(t, adm, "A", map[string]any{"model": "fake-chat", "kind": "chat", "price_in": "0", "price_out": "0"})
	b := r.mk(t, adm, "B", map[string]any{"model": "fake-chat-2", "kind": "chat", "price_in": "0", "price_out": "0"})
	id := func(p map[string]any) string { return p["models"].([]any)[0].(map[string]any)["id"].(string) }
	require.Equal(t, 200, r.do(t, adm, "PUT", "/admin/llm/routes", map[string]any{"task": "CHAT", "chain": []string{id(a)}, "version": 0}).status)
	cur := func() string {
		rt, err := r.rt.Registry.Route(llm.TaskChat)
		require.NoError(t, err)
		return rt.Targets[0].Model
	}
	require.Eventually(t, func() bool { return cur() == "fake-chat" }, time.Second, 20*time.Millisecond)
	start := time.Now()
	require.Equal(t, 200, r.do(t, adm, "PUT", "/admin/llm/routes", map[string]any{"task": "CHAT", "chain": []string{id(b)}, "version": 1}).status)
	require.Eventually(t, func() bool { return cur() == "fake-chat-2" }, time.Second, 10*time.Millisecond)
	require.Less(t, time.Since(start), time.Second, "lời gọi kế tiếp dùng mô hình mới ≤ 1 s")
}

func TestEmbeddingChangeFlagsReindex(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	em := func(name string) map[string]any {
		return map[string]any{"model": name, "kind": "embedding", "dims": 1536, "price_in": "0", "price_out": "0"}
	}
	a := r.mk(t, adm, "A", em("fake-embed"))
	b := r.mk(t, adm, "B", em("fake-embed-2"))
	id := func(p map[string]any) string { return p["models"].([]any)[0].(map[string]any)["id"].(string) }
	x := r.do(t, adm, "PUT", "/admin/llm/routes", map[string]any{"task": "EMBEDDING", "chain": []string{id(a)}, "version": 0})
	require.Equal(t, 200, x.status, string(x.body))
	same := r.do(t, adm, "PUT", "/admin/llm/routes", map[string]any{"task": "EMBEDDING", "chain": []string{id(a)}, "version": 1})
	require.Equal(t, false, same.obj(t)["reindex_required"])
	y := r.do(t, adm, "PUT", "/admin/llm/routes", map[string]any{"task": "EMBEDDING", "chain": []string{id(b)}, "version": 2})
	require.Equal(t, 200, y.status, string(y.body))
	require.Equal(t, true, y.obj(t)["reindex_required"])
	// mô hình nhúng khác 1536 chiều bị từ chối
	c := r.mk(t, adm, "C", em("khac-768"))
	bad := c["models"].([]any)[0].(map[string]any)["id"].(string)
	_, err := r.pool.Exec(t.Context(), `update llm_models set dims = 768 where id = $1`, bad)
	require.NoError(t, err)
	z := r.do(t, adm, "PUT", "/admin/llm/routes", map[string]any{"task": "EMBEDDING", "chain": []string{bad}, "version": 3})
	require.Equal(t, 422, z.status, string(z.body))
	require.Equal(t, "MODEL_DIMS_MISMATCH", z.code(t))
}

// ---- AC7: mức dùng ----

func TestUsageAggregates(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	_, err := r.pool.Exec(t.Context(), `
insert into llm_audit (task, lane, provider, model, tokens_in, tokens_out, latency_ms, cost_est, status, degraded, trace_id, created_at)
select 'CHAT', 'INTERACTIVE', 'p', 'm', 100, 40, i, 0.5,
       case when i % 100 = 0 then 'error' else 'ok' end, i % 50 = 0, 't', now() - interval '1 hour'
  from generate_series(1, 999) i`)
	require.NoError(t, err)
	x := r.do(t, adm, "GET", "/admin/llm/usage", nil)
	require.Equal(t, 200, x.status, string(x.body))
	o := x.obj(t)
	require.Equal(t, "task", o["group"])
	row := o["items"].([]any)[0].(map[string]any)
	require.Equal(t, "CHAT", row["key"])
	require.EqualValues(t, 999, row["calls"])
	require.EqualValues(t, 99900, row["tokens_in"])
	require.EqualValues(t, 39960, row["tokens_out"])
	require.Equal(t, "499.5000", row["cost_est"])
	require.EqualValues(t, 500, row["latency_p50_ms"])
	require.EqualValues(t, 949, row["latency_p95_ms"])
	require.EqualValues(t, 9, row["errors"])
	require.EqualValues(t, 19, row["degraded"])
	d := r.do(t, adm, "GET", "/admin/llm/usage?group=day", nil).obj(t)["items"].([]any)
	require.Len(t, d, 1)
	require.EqualValues(t, 999, d[0].(map[string]any)["calls"])
	// TEACHER xem được tổng hệ thống
	require.Equal(t, 200, r.do(t, r.token(t, auth.RoleTeacher), "GET", "/admin/llm/usage", nil).status)
}

func TestUsageWindowLimit(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	x := r.do(t, adm, "GET", "/admin/llm/usage?from=2026-01-01&to=2026-06-01", nil)
	require.Equal(t, 422, x.status, string(x.body))
	require.Equal(t, "VALIDATION_FAILED", x.code(t))
	require.Equal(t, 200, r.do(t, adm, "GET", "/admin/llm/usage?from=2026-01-01&to=2026-03-31", nil).status)
	require.Equal(t, 422, r.do(t, adm, "GET", "/admin/llm/usage?from=nope", nil).status)
	require.Equal(t, 422, r.do(t, adm, "GET", "/admin/llm/usage?group=week", nil).status)
	require.Equal(t, 422, r.do(t, adm, "GET", "/admin/llm/usage?course_id=zzz", nil).status)
	require.Equal(t, 422, r.do(t, adm, "GET", "/admin/llm/usage?from=2026-02-01&to=2026-01-01", nil).status)
}

func TestUsageTeacherNoCourseFilter(t *testing.T) {
	r := getAPI(t)
	tch := r.token(t, auth.RoleTeacher)
	x := r.do(t, tch, "GET", "/admin/llm/usage?course_id="+uuid.NewString(), nil)
	require.Equal(t, 403, x.status)
	require.Equal(t, "FORBIDDEN", x.code(t))
	require.Equal(t, 200, r.do(t, r.token(t, auth.RoleAdmin), "GET", "/admin/llm/usage?course_id="+uuid.NewString(), nil).status)
}

// ---- AC8: ngân sách ----

func TestBudgetGetShape(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	x := r.do(t, adm, "GET", "/admin/llm/budget", nil)
	require.Equal(t, 200, x.status, string(x.body))
	o := x.obj(t)
	for _, k := range []string{"scope", "daily_limit", "monthly_limit", "spent_today", "spent_month", "pct_today", "pct_month", "state", "version"} {
		require.Contains(t, o, k)
	}
	require.Equal(t, "system", o["scope"])
	require.Nil(t, o["daily_limit"])
	require.Nil(t, o["pct_today"])
	require.Equal(t, "ok", o["state"])
	require.Equal(t, "0.0000", o["spent_today"])
	require.Equal(t, `W/"v0"`, x.hdr.Get("ETag"))
	require.Equal(t, 200, r.do(t, r.token(t, auth.RoleTeacher), "GET", "/admin/llm/budget", nil).status)
}

func TestBudgetPutRules(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	put := func(b map[string]any) resp { return r.do(t, adm, "PUT", "/admin/llm/budget", b) }
	ok := put(map[string]any{"daily_limit": "100000", "monthly_limit": "2000000", "version": 0})
	require.Equal(t, 200, ok.status, string(ok.body))
	o := ok.obj(t)
	require.Equal(t, "100000.00", o["daily_limit"])
	require.Equal(t, "2000000.00", o["monthly_limit"])
	require.EqualValues(t, 1, o["version"])
	require.Equal(t, `W/"v1"`, ok.hdr.Get("ETag"))
	require.Equal(t, 422, put(map[string]any{"daily_limit": "-1", "version": 1}).status)
	require.Equal(t, 422, put(map[string]any{"daily_limit": "3000000", "monthly_limit": "2000000", "version": 1}).status)
	st := put(map[string]any{"daily_limit": "1", "monthly_limit": "2", "version": 0})
	require.Equal(t, 409, st.status)
	require.Equal(t, "VERSION_CONFLICT", st.code(t))
	nul := put(map[string]any{"daily_limit": nil, "monthly_limit": nil, "version": 1})
	require.Equal(t, 200, nul.status, string(nul.body))
	require.Nil(t, nul.obj(t)["daily_limit"])
	// course
	cid := uuid.NewString()
	c := r.do(t, adm, "PUT", "/courses/"+cid+"/llm-budget", map[string]any{"daily_limit": "5000", "monthly_limit": nil, "version": 0})
	require.Equal(t, 200, c.status, string(c.body))
	require.Equal(t, "course", c.obj(t)["scope"])
	require.Equal(t, cid, c.obj(t)["course_id"])
	g := r.do(t, adm, "GET", "/courses/"+cid+"/llm-budget", nil)
	require.Equal(t, "5000.00", g.obj(t)["daily_limit"])
	require.Equal(t, 422, r.do(t, adm, "PUT", "/courses/not-a-uuid/llm-budget", map[string]any{"version": 0}).status)
	require.Equal(t, 422, r.do(t, adm, "GET", "/courses/not-a-uuid/llm-budget", nil).status)
}

func TestBudgetTakesEffect(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	// chi phí đã chi vào bộ đếm Redis; hạn mức mới đổi trạng thái NGAY (không đợi nạp lại).
	r.rt.Budget.Charge(t.Context(), nil, decimal.RequireFromString("850"))
	put := r.do(t, adm, "PUT", "/admin/llm/budget", map[string]any{"daily_limit": "1000", "monthly_limit": "100000000", "version": 0})
	require.Equal(t, 200, put.status, string(put.body))
	o := put.obj(t)
	require.Equal(t, "warn", o["state"])
	require.EqualValues(t, 85, o["pct_today"])
	require.Equal(t, "850.0000", o["spent_today"])
	put = r.do(t, adm, "PUT", "/admin/llm/budget", map[string]any{"daily_limit": "800", "monthly_limit": "100000000", "version": 1})
	require.Equal(t, "exhausted", put.obj(t)["state"])
	require.Equal(t, int(r.rt.Budget.State(t.Context(), nil)), 2, "Scheduler thấy hạn mức mới ngay")
	put = r.do(t, adm, "PUT", "/admin/llm/budget", map[string]any{"daily_limit": nil, "monthly_limit": nil, "version": 2})
	require.Equal(t, "ok", put.obj(t)["state"])
	require.Zero(t, int(r.rt.Budget.State(t.Context(), nil)))
}

func TestCourseBudgetAdminOnly(t *testing.T) {
	r := getAPI(t)
	cid := uuid.NewString()
	for _, role := range []auth.Role{auth.RoleTeacher, auth.RoleTA, auth.RoleStudent} {
		require.Equal(t, 403, r.do(t, r.token(t, role), "GET", "/courses/"+cid+"/llm-budget", nil).status, string(role))
		require.Equal(t, 403, r.do(t, r.token(t, role), "PUT", "/courses/"+cid+"/llm-budget", map[string]any{"version": 0}).status, string(role))
	}
	require.Equal(t, 200, r.do(t, r.token(t, auth.RoleAdmin), "GET", "/courses/"+cid+"/llm-budget", nil).status)
}

// ---- AC4: khoá không bao giờ quay về ----

func TestCanaryNeverEchoed(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	var seen []string
	rec := func(x resp) resp { seen = append(seen, string(x.body)); return x }
	tail := "do-not-leak"

	// tạo (khoá sai → 422), tạo với skip_verify, sai định dạng, đổi khoá, Test (cả hai loại), mọi thao tác còn lại
	rec(r.do(t, adm, "POST", "/admin/llm/providers", map[string]any{"type": "fake", "name": "C0", "api_key": canary, "models": fakeModels()}, "Idempotency-Key", idem()))
	made := rec(r.do(t, adm, "POST", "/admin/llm/providers", map[string]any{"type": "fake", "name": "C1", "api_key": canary, "skip_verify": true, "models": fakeModels()}, "Idempotency-Key", idem()))
	require.Equal(t, 201, made.status, string(made.body))
	p := made.obj(t)
	id := p["id"].(string)
	rec(r.do(t, adm, "POST", "/admin/llm/providers", map[string]any{"type": "nope", "name": "C2", "api_key": canary, "models": fakeModels()}, "Idempotency-Key", idem()))
	rec(r.do(t, adm, "POST", "/admin/llm/providers", []byte(`{"type":"fake","api_key":"`+canary+`",`), "Idempotency-Key", idem()))
	rec(r.do(t, adm, "PUT", "/admin/llm/providers/"+id, map[string]any{"type": "fake", "name": "C1", "api_key": canary + "2", "skip_verify": true, "version": 1}))
	rec(r.do(t, adm, "PUT", "/admin/llm/providers/"+id, map[string]any{"type": "fake", "name": "C1", "api_key": canary + "3", "version": 2}))
	rec(r.do(t, adm, "PUT", "/admin/llm/providers/"+id, map[string]any{"type": "fake", "name": "C1", "api_key": canary, "version": 99}))
	rec(r.do(t, adm, "POST", "/admin/llm/providers/test", map[string]any{"type": "fake", "api_key": canary, "model": "fake-chat"}))
	rec(r.do(t, adm, "POST", "/admin/llm/providers/"+id+"/test", map[string]any{"api_key": canary}))
	rec(r.do(t, adm, "POST", "/admin/llm/providers/"+id+"/test", nil))
	rec(r.do(t, adm, "GET", "/admin/llm/providers", nil))
	rec(r.do(t, r.token(t, auth.RoleTeacher), "GET", "/admin/llm/providers", nil))
	rec(r.do(t, adm, "GET", "/admin/llm/routes", nil))
	rec(r.do(t, adm, "GET", "/admin/llm/usage", nil))
	rec(r.do(t, adm, "GET", "/admin/llm/budget", nil))
	rec(r.do(t, adm, "PUT", "/admin/llm/budget", map[string]any{"daily_limit": "1", "version": 0}))
	rec(r.do(t, adm, "DELETE", "/admin/llm/providers/"+id, nil))
	// 500 giả lập: Redis/DB đều ổn nên dùng thân cực lớn → 413 mang khoá ở đầu thân
	rec(r.do(t, adm, "POST", "/admin/llm/providers", append([]byte(`{"api_key":"`+canary+`","pad":"`), bytes.Repeat([]byte("x"), 70000)...), "Idempotency-Key", idem()))

	for i, s := range seen {
		require.NotContains(t, s, tail, "phản hồi #%d", i)
	}
	logs := r.logs.String()
	require.NotContains(t, logs, tail, "log")
	for _, q := range []string{
		`select count(*) from audit_log where before::text like '%'||$1||'%' or after::text like '%'||$1||'%'`,
		`select count(*) from outbox where payload::text like '%'||$1||'%'`,
	} {
		require.Zero(t, count(t, r.pool, q, tail), q)
	}
	var n int
	err := r.pool.QueryRow(t.Context(), `select count(*) from llm_audit where provider like '%'||$1||'%' or model like '%'||$1||'%' or error_kind like '%'||$1||'%'`, tail).Scan(&n)
	require.NoError(t, err)
	require.Zero(t, n)
	// bản mã trong DB không chứa khoá rõ
	var raw []byte
	err = r.pool.QueryRow(t.Context(), `select api_key_enc from llm_providers limit 1`).Scan(&raw)
	if err != nil {
		require.ErrorIs(t, err, pgx.ErrNoRows)
	} else {
		require.NotContains(t, string(raw), tail)
	}
}

// ---- AC11: quy tắc chung ----

func TestErrorFormat(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	p := r.mk(t, adm, "A")
	cases := []resp{
		r.do(t, "", "GET", "/admin/llm/providers", nil),
		r.do(t, r.token(t, auth.RoleStudent), "GET", "/admin/llm/providers", nil),
		r.do(t, adm, "DELETE", "/admin/llm/providers/"+uuid.NewString(), nil),
		r.do(t, adm, "PUT", "/admin/llm/providers/"+p["id"].(string), map[string]any{"type": "fake", "name": "A", "version": 9}),
		r.do(t, adm, "POST", "/admin/llm/providers", map[string]any{}, "Idempotency-Key", idem()),
		r.do(t, adm, "POST", "/admin/llm/providers", []byte(`{"type":`), "Idempotency-Key", idem()),
		r.do(t, adm, "PUT", "/admin/llm/routes", map[string]any{"task": "CHAT", "chain": []string{}, "version": 0}),
	}
	for i, x := range cases {
		require.GreaterOrEqual(t, x.status, 400, "ca %d", i)
		require.Contains(t, x.hdr.Get("Content-Type"), "application/json", "ca %d", i)
		o := x.obj(t)
		require.NotEmpty(t, o["code"], "ca %d: %s", i, x.body)
		require.NotEmpty(t, o["message"], "ca %d", i)
		require.NotEmpty(t, o["trace_id"], "ca %d", i)
	}
	// Content-Type sai → 415
	req := httptest.NewRequestWithContext(t.Context(), "PUT", "/api/v1/admin/llm/budget", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+adm)
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	r.h.ServeHTTP(rec, req)
	require.Equal(t, 415, rec.Code)
}

func TestBodyLimit(t *testing.T) {
	r := getAPI(t)
	adm := r.token(t, auth.RoleAdmin)
	big := append([]byte(`{"type":"fake","name":"x","pad":"`), bytes.Repeat([]byte("x"), int(r.cfg.MaxBodyBytes)+10)...)
	big = append(big, []byte(`"}`)...)
	for _, c := range []struct{ method, path string }{
		{"POST", "/admin/llm/providers"}, {"PUT", "/admin/llm/providers/" + uuid.NewString()}, {"POST", "/admin/llm/providers/test"},
		{"PUT", "/admin/llm/routes"}, {"PUT", "/admin/llm/budget"},
	} {
		x := r.do(t, adm, c.method, c.path, big, "Idempotency-Key", idem())
		require.Equal(t, 413, x.status, c.method+" "+c.path+" "+string(x.body))
		require.Equal(t, "PAYLOAD_TOO_LARGE", x.code(t))
	}
}

// qCounter đếm số truy vấn gửi tới Postgres.
type qCounter struct {
	mu sync.Mutex
	n  int
}

func (c *qCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
	return ctx
}
func (*qCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func TestNoNPlusOne(t *testing.T) {
	r := newRig(t)
	for i := range llmconfig.MaxProviders {
		ms := make([]llmconfig.ModelInput, 0, 10)
		for j := range 10 {
			ms = append(ms, chatModel(fmt.Sprintf("m%d", j)))
		}
		r.provider(t, fmt.Sprintf("P%02d", i), "k", ms...)
	}
	cnt := &qCounter{}
	pcfg := r.pool.Config()
	pcfg.ConnConfig.Tracer = cnt
	pool, err := pgxpool.NewWithConfig(t.Context(), pcfg)
	require.NoError(t, err)
	defer pool.Close()
	svc := llmconfig.New(pool, r.c)
	ps, err := svc.ListProviders(r.ctx)
	require.NoError(t, err)
	require.Len(t, ps, 20)
	require.Len(t, ps[0].Models, 10)
	cnt.mu.Lock()
	defer cnt.mu.Unlock()
	require.LessOrEqual(t, cnt.n, 3, "GET providers: số truy vấn với 20 nhà × 10 mô hình")
}
