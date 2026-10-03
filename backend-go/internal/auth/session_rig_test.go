package auth_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

const (
	rigSecret   = "session-test-secret-0123456789abcdef-dev"
	rigPassword = "Edupilot#2026-demo"
	rigOrigin   = "https://localhost"
)

// sessRig là gateway thật (router đầy đủ) trên Postgres + Redis thật, với đồng hồ giả.
type sessRig struct {
	t    *testing.T
	h    http.Handler
	pool *pgxpool.Pool
	rdb  *appredis.Client
	clk  *clock.Fake
	logs *bytes.Buffer
}

type rigOpt func(env map[string]string)

func newSessRig(t *testing.T, opts ...rigOpt) *sessRig {
	t.Helper()
	testutil.RequireContainers(t)
	env := map[string]string{
		"DATABASE_URL": testutil.MigratedPostgresURL(t), "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": rigSecret,
		"BLOB_ENDPOINT": "127.0.0.1:9", "BLOB_BUCKET": "x", "BLOB_ACCESS_KEY": "x", "BLOB_SECRET_KEY": "x",
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test", "BCRYPT_COST": "4",
		"RATE_LIMIT_IP_PER_MIN": "1000000", "RATE_LIMIT_USER_PER_MIN": "1000000", "LOG_LEVEL": "debug",
	}
	for _, o := range opts {
		o(env)
	}
	cfg, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	require.NoError(t, err)

	pool, err := pgxpool.New(t.Context(), cfg.DatabaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	rdb, err := appredis.New(t.Context(), cfg.RedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	logs := &bytes.Buffer{}
	clk := clock.NewFake(time.Now().UTC().Truncate(time.Second))
	d := httpapi.Deps{
		Cfg: cfg, Log: slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
		DB: pool, Redis: rdb, Clock: clk, State: httpapi.NewState(),
	}
	return &sessRig{t: t, h: httpapi.NewRouter(d), pool: pool, rdb: rdb, clk: clk, logs: logs}
}

// user chèn một người dùng có mật khẩu rigPassword.
func (r *sessRig) user(email string, role store.UserRole, status store.UserStatus) store.User {
	r.t.Helper()
	hash, err := auth.HashPassword(rigPassword, 4)
	require.NoError(r.t, err)
	u, err := store.New(r.pool).InsertUser(r.t.Context(), store.InsertUserParams{
		Email: email, FullName: "Người Dùng", Role: role, Status: status, PasswordHash: &hash,
	})
	require.NoError(r.t, err)
	return u
}

func uniq(prefix string) string { return prefix + "." + uuid.NewString()[:8] + "@example.test" }

type resp struct {
	code int
	hdr  http.Header
	body []byte
}

func (x resp) json() map[string]any {
	var m map[string]any
	_ = json.Unmarshal(x.body, &m)
	return m
}

func (x resp) errCode() string { s, _ := x.json()["code"].(string); return s }

// cookie trả giá trị Set-Cookie của tên cho trước ("" nếu không có) và cho biết có phải lệnh xoá không.
func (x resp) cookie(name string) *http.Cookie {
	for _, c := range (&http.Response{Header: x.hdr}).Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

type req struct {
	method, path string
	body         any
	hdr          map[string]string
	rt           string // giá trị cookie ep_rt gửi kèm
	bearer       string
}

func (r *sessRig) do(q req) resp {
	r.t.Helper()
	var rd io.Reader
	if q.body != nil {
		b, err := json.Marshal(q.body)
		require.NoError(r.t, err)
		rd = bytes.NewReader(b)
	}
	if q.method == "" {
		q.method = http.MethodPost
	}
	hr := httptest.NewRequest(q.method, "/api/v1"+q.path, rd)
	hr.RemoteAddr = "203.0.113.7:4444"
	if q.body != nil {
		hr.Header.Set("Content-Type", "application/json")
	}
	hr.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0 Safari/537.36")
	for k, v := range q.hdr {
		if v == "" {
			hr.Header.Del(k)
		} else {
			hr.Header.Set(k, v)
		}
	}
	if q.rt != "" {
		hr.AddCookie(&http.Cookie{Name: "ep_rt", Value: q.rt})
	}
	if q.bearer != "" {
		hr.Header.Set("Authorization", "Bearer "+q.bearer)
	}
	w := httptest.NewRecorder()
	r.h.ServeHTTP(w, hr)
	return resp{code: w.Code, hdr: w.Header(), body: w.Body.Bytes()}
}

func (r *sessRig) login(email, password string) resp {
	return r.do(req{path: "/auth/login", body: map[string]string{"email": email, "password": password}, hdr: map[string]string{"Origin": rigOrigin}})
}

func (r *sessRig) refresh(rt string) resp {
	return r.do(req{path: "/auth/refresh", rt: rt, hdr: map[string]string{"Origin": rigOrigin}})
}

func (r *sessRig) logout(rt string) resp {
	return r.do(req{path: "/auth/logout", rt: rt, hdr: map[string]string{"Origin": rigOrigin}})
}

// api gọi một API cần Bearer (GET /jobs/{uuid}: 404 nếu đã xác thực, 401 nếu không).
func (r *sessRig) api(access string) resp {
	return r.do(req{method: http.MethodGet, path: "/jobs/" + uuid.NewString(), bearer: access})
}

// session là phiên đã đăng nhập: access + refresh hiện tại.
type session struct{ access, rt, sid string }

func (r *sessRig) mustLogin(email string) session {
	r.t.Helper()
	res := r.login(email, rigPassword)
	require.Equal(r.t, http.StatusOK, res.code, string(res.body))
	return sessionOf(r.t, res)
}

func sessionOf(t *testing.T, res resp) session {
	t.Helper()
	c := res.cookie("ep_rt")
	require.NotNil(t, c)
	acc, _ := res.json()["access_token"].(string)
	return session{access: acc, rt: c.Value, sid: claims(t, acc)["sid"].(string)}
}

func claims(t *testing.T, jwt string) map[string]any {
	t.Helper()
	parts := strings.Split(jwt, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}

func (r *sessRig) devToken(sub string) string {
	r.t.Helper()
	tok, err := auth.NewIssuer(rigSecret, time.Hour, r.clk).Issue(sub, auth.RoleStudent, "dev@edupilot.local")
	require.NoError(r.t, err)
	return tok
}

func authHash(cost int) (string, error) { return auth.HashPassword(rigPassword, cost) }

func hashOf(s string) string { return auth.HashToken(s) }
