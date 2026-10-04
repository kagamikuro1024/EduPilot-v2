package user_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

const (
	rigSecret   = "user-test-secret-0123456789abcdef-dev"
	rigPassword = "Edupilot#2026-demo"
	rigOrigin   = "https://localhost"
)

// qcount đếm câu SQL đi qua pool (kiểm "không N+1").
type qcount struct{ n atomic.Int64 }

func (c *qcount) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	c.n.Add(1)
	return ctx
}
func (c *qcount) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// rig là gateway thật (router đầy đủ) trên Postgres + Redis thật, đồng hồ giả.
type rig struct {
	t    *testing.T
	h    http.Handler
	pool *pgxpool.Pool
	rdb  *appredis.Client
	clk  *clock.Fake
	ip   string
	qc   *qcount
	logs *bytes.Buffer
}

func newRig(t *testing.T, opts ...func(map[string]string)) *rig {
	t.Helper()
	testutil.RequireContainers(t)
	env := map[string]string{
		"DATABASE_URL": testutil.MigratedPostgresURL(t), "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": rigSecret,
		"BLOB_ENDPOINT": "127.0.0.1:9", "BLOB_BUCKET": "x", "BLOB_ACCESS_KEY": "x", "BLOB_SECRET_KEY": "x",
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test", "BCRYPT_COST": "4",
		"RATE_LIMIT_IP_PER_MIN": "1000000", "RATE_LIMIT_USER_PER_MIN": "1000000", "LOG_LEVEL": "debug",
		"AUTH_LOGIN_IP_PER_MIN": "1000000", "AUTH_LOGIN_IP_FAIL_PER_15M": "1000000", "AUTH_TOKEN_IP_PER_MIN": "1000000",
	}
	for _, o := range opts {
		o(env)
	}
	cfg, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	require.NoError(t, err)

	pcfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	require.NoError(t, err)
	qc := &qcount{}
	pcfg.ConnConfig.Tracer = qc
	pool, err := pgxpool.NewWithConfig(t.Context(), pcfg)
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
	id := uuid.New()
	return &rig{t: t, h: httpapi.NewRouter(d), pool: pool, rdb: rdb, clk: clk, qc: qc, logs: logs, ip: fmt.Sprintf("203.0.%d.%d", 1+int(id[0])%254, 1+int(id[1])%254)}
}

func uniq(prefix string) string { return prefix + "." + uuid.NewString()[:8] + "@example.test" }

// addUser chèn người dùng có mật khẩu rigPassword.
func (r *rig) addUser(email string, role store.UserRole, status store.UserStatus) store.User {
	r.t.Helper()
	hash, err := auth.HashPassword(rigPassword, 4)
	require.NoError(r.t, err)
	u, err := store.New(r.pool).InsertUser(r.t.Context(), store.InsertUserParams{Email: email, FullName: "Người Thử", Role: role, Status: status, PasswordHash: &hash})
	require.NoError(r.t, err)
	return u
}

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
func (x resp) details() map[string]any {
	d, _ := x.json()["details"].(map[string]any)
	return d
}

type req struct {
	method, path string
	body         any
	bearer       string
	hdr          map[string]string
}

func (r *rig) do(q req) resp {
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
	hr.RemoteAddr = r.ip + ":4444"
	if q.body != nil {
		hr.Header.Set("Content-Type", "application/json")
	}
	hr.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0 Safari/537.36")
	if q.bearer != "" {
		hr.Header.Set("Authorization", "Bearer "+q.bearer)
	}
	for k, v := range q.hdr {
		hr.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.h.ServeHTTP(w, hr)
	return resp{code: w.Code, hdr: w.Header(), body: w.Body.Bytes()}
}

func (r *rig) login(email, password string) resp {
	return r.do(req{path: "/auth/login", body: map[string]string{"email": email, "password": password}, hdr: map[string]string{"Origin": rigOrigin}})
}

type session struct{ access, rt string }

func (r *rig) mustLogin(email string) session {
	r.t.Helper()
	res := r.login(email, rigPassword)
	require.Equal(r.t, http.StatusOK, res.code, string(res.body))
	acc, _ := res.json()["access_token"].(string)
	var rt string
	for _, c := range (&http.Response{Header: res.hdr}).Cookies() {
		if c.Name == "ep_rt" {
			rt = c.Value
		}
	}
	return session{access: acc, rt: rt}
}

// admin tạo ADMIN ACTIVE và đăng nhập.
func (r *rig) admin() (store.User, session) {
	r.t.Helper()
	u := r.addUser(uniq("admin"), store.UserRoleADMIN, store.UserStatusACTIVE)
	return u, r.mustLogin(u.Email)
}

var idemSeq atomic.Int64

func idem() map[string]string {
	return map[string]string{"Idempotency-Key": fmt.Sprintf("test-key-%d-%s", idemSeq.Add(1), uuid.NewString()[:8])}
}

func (r *rig) invite(s session, email, name, role string) resp {
	return r.do(req{path: "/admin/users", bearer: s.access, hdr: idem(), body: map[string]string{"email": email, "full_name": name, "role": role}})
}

func (r *rig) patch(s session, id any, body map[string]any) resp {
	return r.do(req{method: http.MethodPatch, path: fmt.Sprintf("/admin/users/%v", id), bearer: s.access, body: body})
}

// api gọi một API cần Bearer: GET /jobs/{uuid} ⇒ 404 nếu đã xác thực, 401 nếu không.
func (r *rig) api(access string) resp {
	return r.do(req{method: http.MethodGet, path: "/jobs/" + uuid.NewString(), bearer: access})
}

// --- thư ---

type captureSender struct {
	mu   sync.Mutex
	sent []mail.Mail
}

func (c *captureSender) Send(_ context.Context, m mail.Mail) error {
	c.mu.Lock()
	c.sent = append(c.sent, m)
	c.mu.Unlock()
	return nil
}

var linkTokenRE = regexp.MustCompile(`(?:token=|/invite/)([A-Za-z0-9_-]{43})`)

// deliver chạy consumer thư thật cho mọi thư QUEUED của `to`; trả thư cuối và token rõ trong liên kết.
func (r *rig) deliver(to string) (mail.Mail, string) {
	r.t.Helper()
	rows, err := r.pool.Query(r.t.Context(), `select id from mail_outbox where to_addr = $1 and status = 'QUEUED' order by created_at, id`, to)
	require.NoError(r.t, err)
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	require.NoError(r.t, err)
	require.NotEmpty(r.t, ids, "không có thư QUEUED cho %s", to)
	cs := &captureSender{}
	cfg := config.Config{
		AppPublicURL: "https://localhost", MailFrom: "EduPilot <no-reply@edupilot.local>", MailSendTimeout: time.Second,
		VerifyTokenTTL: 24 * time.Hour, ResetTokenTTL: 30 * time.Minute, InviteTokenTTL: 72 * time.Hour,
		OutboxRetryBackoff: []time.Duration{time.Second, time.Second, time.Second},
	}
	h := &mail.Handler{Pool: r.pool, Clock: r.clk, Sender: cs, Cfg: cfg, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	for _, id := range ids {
		require.NoError(r.t, h.Handle(r.t.Context(), outbox.Message{Topic: mail.Topic, Payload: []byte(`{"mail_id":"` + id.String() + `"}`)}))
	}
	last := cs.sent[len(cs.sent)-1]
	if m := linkTokenRE.FindStringSubmatch(last.Text); m != nil {
		return last, m[1]
	}
	return last, ""
}

func (r *rig) mails(to, template string) int {
	r.t.Helper()
	var n int
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select count(*) from mail_outbox where to_addr = $1 and ($2 = '' or template = $2)`, to, template).Scan(&n))
	return n
}

func (r *rig) scalar(sql string, args ...any) string {
	r.t.Helper()
	var s *string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), sql, args...).Scan(&s))
	if s == nil {
		return ""
	}
	return *s
}

func claimsOf(t *testing.T, jwt string) map[string]any {
	t.Helper()
	parts := strings.Split(jwt, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}
