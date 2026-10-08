package today_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
	"github.com/edupilot/backend-go/internal/today"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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
	cfg  config.Config
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
	log := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	// Có runtime LLM (nhà cung cấp giả) để bảng route đầy đủ — gồm /courses/{id}/llm-budget — nằm trong bộ quét guard.
	rt, err := llmrt.New(t.Context(), cfg, pool, rdb.Client, log)
	require.NoError(t, err)
	d := httpapi.Deps{
		Cfg: cfg, Log: log, DB: pool, Redis: rdb, Clock: clk, State: httpapi.NewState(), LLM: rt,
	}
	id := uuid.New()
	return &rig{t: t, cfg: cfg, h: httpapi.NewRouter(d), pool: pool, rdb: rdb, clk: clk, qc: qc, logs: logs, ip: fmt.Sprintf("203.0.%d.%d", 1+int(id[0])%254, 1+int(id[1])%254)}
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

// course chèn một lớp (join_code ngẫu nhiên hợp lệ) và trả id + class_code.
func (r *rig) course(createdBy uuid.UUID, classCode string) uuid.UUID {
	r.t.Helper()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	id := uuid.New()
	b := make([]byte, 7)
	for i := range b {
		b[i] = alpha[int(id[i+4])%len(alpha)]
	}
	var out uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by)
		values ('INT1006', $1, 'An ninh mạng', '2026-2027-HK1', $2, $3) returning id`, classCode, string(b), createdBy).Scan(&out))
	return out
}

// worker chạy relay + consumer THẬT (Streams riêng theo test) với handler `course.assigned`, tới hết test.
// hits đếm số lần handler được gọi theo outbox id để kiểm giao lại.
func (r *rig) worker() *sync.Map {
	r.t.Helper()
	p := testutil.TestPrefix(r.t)
	names := outbox.Streams{Dispatch: p + ".dispatch", Dead: p + ".dispatch.dead", Consumer: p + "-c1"}
	cfg := r.cfg
	cfg.InstanceID = p
	cfg.OutboxPollInterval = 20 * time.Millisecond
	cfg.OutboxRetryBackoff = []time.Duration{20 * time.Millisecond, 20 * time.Millisecond, 20 * time.Millisecond}
	od := outbox.Deps{Pool: r.pool, Redis: r.rdb, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Clock: clock.Real{}, Cfg: cfg, Streams: names}

	hits := &sync.Map{}
	n := &course.Notifier{Pool: r.pool, AppPublicURL: "https://localhost"}
	reg := outbox.NewRegistry()
	inv := today.Invalidator{Pool: r.pool, Redis: r.rdb, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	reg.Register(course.TopicAssigned, outbox.Chain(func(ctx context.Context, m outbox.Message) error {
		hits.Store(m.ID, true)
		return n.HandleAssigned(ctx, m)
	}, inv.Handle))
	reg.Register(course.TopicJoinRequested, outbox.Chain(n.HandleJoinRequested, inv.Handle))
	reg.Register(course.TopicJoinDecided, outbox.Chain(n.HandleJoinDecided, inv.Handle))
	for _, t := range []string{course.TopicMemberChanged, course.TopicChanged, course.TopicRosterImport, auth.TopicUserVerified, exam.TopicQuestionReviewed} {
		reg.Register(t, inv.Handle)
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, task := range []interface{ Run(context.Context) error }{outbox.NewRelay(od), outbox.NewConsumer(od, reg)} {
		wg.Add(1)
		go func() { defer wg.Done(); _ = task.Run(ctx) }()
	}
	r.t.Cleanup(func() {
		cancel()
		wg.Wait()
		_ = r.rdb.Del(context.Background(), names.Dispatch, names.Dead).Err()
	})
	return hits
}

// waitFor đợi tối đa d tới khi f() true (poll 25 ms).
func waitFor(t *testing.T, d time.Duration, what string, f func() bool) time.Duration {
	t.Helper()
	start := time.Now()
	for time.Since(start) < d {
		if f() {
			return time.Since(start)
		}
		time.Sleep(25 * time.Millisecond)
	}
	require.Failf(t, "quá hạn", "%s sau %s", what, d)
	return 0
}

// student: sinh viên ACTIVE đã xác minh email, tuỳ chọn có MSSV tự khai.
func (r *rig) student(code string) (store.User, session) {
	r.t.Helper()
	u := r.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	_, err := r.pool.Exec(r.t.Context(), `update users set email_verified_at = now() where id = $1`, u.ID)
	require.NoError(r.t, err)
	if code != "" {
		_, err = r.pool.Exec(r.t.Context(), `update users set student_code = $2 where id = $1`, u.ID, code)
		require.NoError(r.t, err)
	}
	return u, r.mustLogin(u.Email)
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

func (r *rig) count(sql string, args ...any) int {
	r.t.Helper()
	var n int
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), sql, args...).Scan(&n))
	return n
}
