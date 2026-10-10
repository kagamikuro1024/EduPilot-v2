package calendar_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/calendar"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

const jwtSecret = "0123456789abcdef0123456789abcdef"

// fx là một lớp thật (Postgres + Redis + router thật): giảng viên, trợ giảng, hai sinh viên.
type fx struct {
	t       *testing.T
	pool    *pgxpool.Pool
	rdb     *appredis.Client
	clk     *clock.Fake
	svc     *calendar.Service
	srv     *httptest.Server
	iss     *auth.Issuer
	logs    *bytes.Buffer
	course  uuid.UUID
	teacher uuid.UUID
	ta      uuid.UUID
	sv      uuid.UUID
	sv2     uuid.UUID
	ip      string // IP giả (X-Forwarded-For) riêng từng test: Redis dùng chung giữa các test song song
}

func newFx(t *testing.T) *fx {
	t.Helper()
	testutil.RequireContainers(t)
	ctx := t.Context()
	dburl := testutil.MigratedPostgresURL(t)
	pool := testutil.RuntimePoolAt(t, dburl)
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	f := &fx{t: t, pool: pool, rdb: rdb, clk: clock.NewFake(time.Now()), iss: auth.NewIssuer(jwtSecret, time.Hour, clock.Real{}), logs: &bytes.Buffer{}, ip: fmt.Sprintf("10.%d.%d.%d", rand.IntN(256), rand.IntN(256), rand.IntN(254)+1)}
	f.svc = &calendar.Service{Pool: pool, Redis: rdb, Clock: f.clk, PublicURL: "http://app.test"}
	f.teacher = f.user("GV", "TEACHER", "ACTIVE")
	f.ta = f.user("TA", "TA", "ACTIVE")
	f.sv = f.user("SV1", "STUDENT", "ACTIVE")
	f.sv2 = f.user("SV2", "STUDENT", "ACTIVE")
	f.course = f.newCourse("ACTIVE")
	f.enroll(f.course, f.teacher, "TEACHER", "ACTIVE")
	f.enroll(f.course, f.ta, "TA", "ACTIVE")
	f.enroll(f.course, f.sv, "STUDENT", "ACTIVE")
	f.enroll(f.course, f.sv2, "STUDENT", "ACTIVE")
	env := map[string]string{
		"DATABASE_URL": dburl, "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": jwtSecret,
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test", "BCRYPT_COST": "4",
		"RATE_LIMIT_IP_PER_MIN": "1000000", "RATE_LIMIT_USER_PER_MIN": "1000000", "REQUEST_TIMEOUT": "5s", "APP_PUBLIC_URL": "http://app.test",
		"BLOB_ENDPOINT": testutil.MinIOEndpoint(t), "BLOB_BUCKET": "cal", "BLOB_ACCESS_KEY": testutil.MinIOAccessKey, "BLOB_SECRET_KEY": testutil.MinIOSecretKey,
	}
	cfg, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	require.NoError(t, err)
	log := slog.New(slog.NewTextHandler(f.logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	f.srv = httptest.NewServer(httpapi.NewRouter(httpapi.Deps{Cfg: cfg, Log: log, DB: pool, Redis: rdb, Clock: f.clk, State: httpapi.NewState()}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fx) user(name, role, status string) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	require.NoError(f.t, f.pool.QueryRow(f.t.Context(), `insert into users (email, full_name, role, status, email_verified_at, password_hash) values ($1, $2, $3::user_role, $4::user_status, now(), 'x') returning id`,
		uuid.NewString()+"@example.test", name, role, status).Scan(&id))
	return id
}

func (f *fx) newCourse(status string) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	var c uuid.UUID
	require.NoError(f.t, f.pool.QueryRow(f.t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by, status, archived_at, join_enabled) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3, $4::course_status, case when $4::text = 'ARCHIVED' then now() end, $4::text = 'ACTIVE') returning id`,
		"CL"+id.String()[:8], string(jc), f.teacher, status).Scan(&c))
	return c
}

func (f *fx) enroll(course, user uuid.UUID, role, status string) {
	f.t.Helper()
	_, err := f.pool.Exec(f.t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, removed_at) values ($1, $2, $3::enrollment_role, $4::enrollment_status, 'ADMIN', case when $4::text = 'REMOVED' then now() end)`, course, user, role, status)
	require.NoError(f.t, err)
}

// session chèn một buổi học bắt đầu sau `in`, kéo dài 90 phút.
func (f *fx) session(course uuid.UUID, no int, in time.Duration, topic string) uuid.UUID {
	f.t.Helper()
	start := f.clk.Now().Add(in)
	var id uuid.UUID
	require.NoError(f.t, f.pool.QueryRow(f.t.Context(), `insert into class_sessions (course_id, session_no, starts_at, ends_at, room, topic) values ($1, $2, $3, $4, 'P.301', $5) returning id`,
		course, no, start, start.Add(90*time.Minute), topic).Scan(&id))
	return id
}

// exam chèn một bài thi `status` mở sau `in`, dài 60 phút.
func (f *fx) exam(course uuid.UUID, title, status string, in time.Duration) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	if status == "DRAFT" {
		require.NoError(f.t, f.pool.QueryRow(f.t.Context(), `insert into exams (course_id, title, status, created_by) values ($1, $2, 'DRAFT', $3) returning id`, course, title, f.teacher).Scan(&id))
		return id
	}
	open := f.clk.Now().Add(in)
	require.NoError(f.t, f.pool.QueryRow(f.t.Context(), `insert into exams (course_id, title, status, opens_at, closes_at, duration_minutes, created_by) values ($1, $2, $3::exam_status, $4, $5, 30, $6) returning id`,
		course, title, status, open, open.Add(time.Hour), f.teacher).Scan(&id))
	return id
}

func (f *fx) attempt(course, exam, student uuid.UUID, status string) {
	f.t.Helper()
	q := `insert into exam_attempts (course_id, exam_id, student_id, status, deadline_at) values ($1, $2, $3, 'IN_PROGRESS', now() + interval '30 minutes')`
	if status == "GRADING" {
		q = `insert into exam_attempts (course_id, exam_id, student_id, status, deadline_at, submitted_at, submit_reason) values ($1, $2, $3, 'GRADING', now() + interval '30 minutes', now(), 'MANUAL')`
	}
	_, err := f.pool.Exec(f.t.Context(), q, course, exam, student)
	require.NoError(f.t, err)
}

// event chèn thẳng một calendar_events (không qua API) — dùng cho lớp ARCHIVED và các nguồn nhắc.
func (f *fx) event(course uuid.UUID, typ, title string, in time.Duration) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	require.NoError(f.t, f.pool.QueryRow(f.t.Context(), `insert into calendar_events (course_id, type, title, starts_at, created_by) values ($1, $2::calendar_event_type, $3, $4, $5) returning id`,
		course, typ, title, f.clk.Now().Add(in), f.teacher).Scan(&id))
	return id
}

func (f *fx) tok(u uuid.UUID, role auth.Role) string {
	f.t.Helper()
	tok, err := f.iss.Issue(u.String(), role, "x@example.test")
	require.NoError(f.t, err)
	return tok
}

func (f *fx) do(method, path, token, body string, hdr ...string) (int, http.Header, []byte) {
	f.t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, f.srv.URL+path, rd)
	require.NoError(f.t, err)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("X-Forwarded-For", f.ip)
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(f.t, err)
	defer func() { _ = res.Body.Close() }()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, res.Header, b
}

func (f *fx) calURL(course uuid.UUID, from, to time.Duration) string {
	now := f.clk.Now()
	return "/api/v1/courses/" + course.String() + "/calendar?from=" + url.QueryEscape(now.Add(from).Format(time.RFC3339)) + "&to=" + url.QueryEscape(now.Add(to).Format(time.RFC3339))
}

func (f *fx) items(user uuid.UUID, role auth.Role, from, to time.Duration) []map[string]any {
	f.t.Helper()
	st, _, b := f.do("GET", f.calURL(f.course, from, to), f.tok(user, role), "")
	require.Equal(f.t, http.StatusOK, st, string(b))
	var out struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(f.t, json.Unmarshal(b, &out))
	return out.Items
}

// icsToken tạo token cho u và trả (token thô, URL).
func (f *fx) icsToken(u uuid.UUID, role auth.Role) (string, string) {
	f.t.Helper()
	st, _, b := f.do("POST", "/api/v1/me/calendar/ics-token", f.tok(u, role), "")
	require.Equal(f.t, http.StatusCreated, st, string(b))
	var out struct {
		URL string `json:"url"`
	}
	require.NoError(f.t, json.Unmarshal(b, &out))
	pu, err := url.Parse(out.URL)
	require.NoError(f.t, err)
	return pu.Query().Get("token"), out.URL
}

func (f *fx) feed(token string) (int, http.Header, string) {
	st, h, b := f.do("GET", "/api/v1/calendar/feed.ics?token="+url.QueryEscape(token), "", "")
	return st, h, string(b)
}

func count(f *fx, q string, args ...any) int {
	f.t.Helper()
	var n int
	require.NoError(f.t, f.pool.QueryRow(f.t.Context(), q, args...).Scan(&n))
	return n
}
