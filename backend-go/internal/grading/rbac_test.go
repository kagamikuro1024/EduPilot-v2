package grading_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

const rbacSecret = "grading-rbac-secret-0123456789abcdef-dev"

type httpRig struct {
	*env
	h http.Handler
}

func newHTTPRig(t *testing.T) *httpRig {
	t.Helper()
	testutil.RequireContainers(t)
	url := testutil.MigratedPostgresURL(t)
	pool := testutil.RuntimePoolAt(t, url)
	env := map[string]string{
		"DATABASE_URL": url, "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": rbacSecret,
		"BLOB_ENDPOINT": "127.0.0.1:9", "BLOB_BUCKET": "x", "BLOB_ACCESS_KEY": "x", "BLOB_SECRET_KEY": "x",
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test", "BCRYPT_COST": "4",
		"RATE_LIMIT_IP_PER_MIN": "1000000", "RATE_LIMIT_USER_PER_MIN": "1000000",
	}
	cfg, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	require.NoError(t, err)
	rdb, err := appredis.New(t.Context(), cfg.RedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	d := httpapi.Deps{Cfg: cfg, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: pool, Redis: rdb, Clock: clock.Real{}, State: httpapi.NewState(), Jobs: jobs.NewService(pool)}
	return &httpRig{env: newEnvPool(t, nil, pool), h: httpapi.NewRouter(d)}
}

func (r *httpRig) token(id uuid.UUID, role auth.Role) string {
	tok, err := auth.NewIssuer(rbacSecret, time.Hour, clock.Real{}).Issue(id.String(), role, "")
	require.NoError(r.t, err)
	return tok
}

func (r *httpRig) extract(course uuid.UUID, token, key string, body any) (int, map[string]any) {
	r.t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+course.String()+"/grade-scheme/extract", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	r.h.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

// TestExtractRBAC — AC10: chỉ TEACHER của lớp gọi được; TA / STUDENT / ADMIN 403; giảng viên lớp khác 403 (guard chung, proposals #11); chưa đăng nhập 401; không job, không lời gọi LLM cho các ca bị từ chối.
func TestExtractRBAC(t *testing.T) {
	r := newHTTPRig(t)
	doc := r.policyDoc("quy-che-lop1.md")
	ta := r.user("TA", "Trợ Giảng")
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TA', 'ACTIVE', 'ADMIN')`, r.course, ta)
	sv := r.user("STUDENT", "Sinh Viên")
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN', '20220001')`, r.course, sv)
	admin := r.user("ADMIN", "Quản Trị")
	other := r.user("TEACHER", "GV Khác")
	body := map[string]any{"document_id": doc}
	key := func() string { return uuid.NewString() }

	st, _ := r.extract(r.course, "", key(), body)
	require.Equal(t, http.StatusUnauthorized, st)
	for name, tok := range map[string]string{"TA": r.token(ta, auth.RoleTA), "STUDENT": r.token(sv, auth.RoleStudent), "ADMIN": r.token(admin, auth.RoleAdmin)} {
		st, out := r.extract(r.course, tok, key(), body)
		require.Equal(t, http.StatusForbidden, st, name)
		require.Equal(t, "FORBIDDEN", out["code"], name)
	}
	// Giảng viên lớp khác: CourseAccessGuard dùng chung trả 403 cho mọi người ngoài lớp (kể cả lớp không tồn tại); spec AC10 ghi 404 → proposals #11.
	st, _ = r.extract(r.course, r.token(other, auth.RoleTeacher), key(), body)
	require.Equal(t, http.StatusForbidden, st)
	var n int
	r.one(&n, `select count(*) from jobs where kind = $1`, "grade_scheme.extract")
	require.Zero(t, n, "ca bị từ chối không xếp việc")

	st, out := r.extract(r.course, r.token(r.teacher, auth.RoleTeacher), key(), body)
	require.Equal(t, http.StatusAccepted, st)
	require.NotEmpty(t, out["job_id"])
	r.one(&n, `select count(*) from jobs where kind = $1`, "grade_scheme.extract")
	require.Equal(t, 1, n)
}

// TestExtractIdempotencyHTTP — AC8: cùng Idempotency-Key gửi hai lần → cùng job_id, đúng một việc; thiếu khoá → 400/422.
func TestExtractIdempotencyHTTP(t *testing.T) {
	r := newHTTPRig(t)
	doc := r.policyDoc("quy-che-lop1.md")
	tok := r.token(r.teacher, auth.RoleTeacher)
	k := uuid.NewString()
	st1, o1 := r.extract(r.course, tok, k, map[string]any{"document_id": doc})
	st2, o2 := r.extract(r.course, tok, k, map[string]any{"document_id": doc})
	require.Equal(t, http.StatusAccepted, st1)
	require.Equal(t, http.StatusAccepted, st2)
	require.Equal(t, o1["job_id"], o2["job_id"])
	var n int
	r.one(&n, `select count(*) from jobs where kind = $1`, "grade_scheme.extract")
	require.Equal(t, 1, n)
	st, _ := r.extract(r.course, tok, "", map[string]any{"document_id": doc})
	require.GreaterOrEqual(t, st, 400)
}

func (r *httpRig) status(course uuid.UUID, token string) (int, map[string]any) {
	r.t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+course.String()+"/grade-scheme/status", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.h.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

// TestStatusRBAC — US-P6-05 AC1: TEACHER và TA của lớp đọc được (không cache); STUDENT / ADMIN / giảng viên lớp khác 403; chưa đăng nhập 401.
func TestStatusRBAC(t *testing.T) {
	r := newHTTPRig(t)
	ta := r.user("TA", "Trợ Giảng")
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TA', 'ACTIVE', 'ADMIN')`, r.course, ta)
	sv := r.user("STUDENT", "Sinh Viên")
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN', '20220001')`, r.course, sv)
	admin, other := r.user("ADMIN", "Quản Trị"), r.user("TEACHER", "GV Khác")

	st, _ := r.status(r.course, "")
	require.Equal(t, http.StatusUnauthorized, st)
	for name, tok := range map[string]string{"STUDENT": r.token(sv, auth.RoleStudent), "ADMIN": r.token(admin, auth.RoleAdmin), "GV lớp khác": r.token(other, auth.RoleTeacher)} {
		st, out := r.status(r.course, tok)
		require.Equal(t, http.StatusForbidden, st, name)
		require.Equal(t, "FORBIDDEN", out["code"], name)
	}
	for name, tok := range map[string]string{"TEACHER": r.token(r.teacher, auth.RoleTeacher), "TA": r.token(ta, auth.RoleTA)} {
		st, out := r.status(r.course, tok)
		require.Equal(t, http.StatusOK, st, name)
		require.Contains(t, []any{"NO_POLICY", "NOT_EXTRACTED"}, out["state"], name)
	}
}
