package grade_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

const httpSecret = "grade-confirm-secret-0123456789abcdef-dev"

type httpRig struct {
	*tdb
	h http.Handler
}

func newHTTPRig(t *testing.T) *httpRig {
	t.Helper()
	testutil.RequireContainers(t)
	url := testutil.MigratedPostgresURL(t)
	pool := testutil.RuntimePoolAt(t, url)
	env := map[string]string{
		"DATABASE_URL": url, "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": httpSecret,
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
	return &httpRig{tdb: &tdb{t: t, ctx: t.Context(), pool: pool, q: store.New(pool)}, h: httpapi.NewRouter(d)}
}

func (r *httpRig) token(id uuid.UUID, role auth.Role) string {
	tok, err := auth.NewIssuer(httpSecret, time.Hour, clock.Real{}).Issue(id.String(), role, "")
	require.NoError(r.t, err)
	return tok
}

func (r *httpRig) do(method, path, token, key string, body any) (int, map[string]any) {
	r.t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
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

// AC5: chỉ TEACHER của lớp xác nhận; TA / STUDENT / ADMIN / giảng viên lớp khác 403; chưa đăng nhập 401; thiếu khoá hoặc thiếu version bị từ chối; không đổi trạng thái khi bị từ chối.
func TestConfirmRBAC(t *testing.T) {
	r := newHTTPRig(t)
	gv := r.user("TEACHER", "GV")
	c := r.course(gv, "L1")
	doc := r.newDraft(c, gv, 1, nil)
	ta := r.user("TA", "TA")
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TA', 'ACTIVE', 'ADMIN')`, c, ta)
	sv := r.student(c, "SV", "20220001")
	admin, other := r.user("ADMIN", "AD"), r.user("TEACHER", "GV khác")
	path := "/api/v1/courses/" + c.String() + "/grade-scheme/confirm"
	body := map[string]any{"version": doc.Version}

	st, _ := r.do(http.MethodPost, path, "", uuid.NewString(), body)
	require.Equal(t, http.StatusUnauthorized, st)
	for name, tok := range map[string]string{"TA": r.token(ta, auth.RoleTA), "STUDENT": r.token(sv, auth.RoleStudent), "ADMIN": r.token(admin, auth.RoleAdmin), "GV khác": r.token(other, auth.RoleTeacher)} {
		st, out := r.do(http.MethodPost, path, tok, uuid.NewString(), body)
		require.Equal(t, http.StatusForbidden, st, name)
		require.Equal(t, "FORBIDDEN", out["code"], name)
	}
	tok := r.token(gv, auth.RoleTeacher)
	st, _ = r.do(http.MethodPost, path, tok, "", body)
	require.GreaterOrEqual(t, st, 400, "thiếu Idempotency-Key")
	require.Less(t, st, 500)
	st, _ = r.do(http.MethodPost, path, tok, uuid.NewString(), map[string]any{})
	require.Equal(t, http.StatusUnprocessableEntity, st, "thiếu version")
	var s string
	r.one(&s, `select status from grade_schemes where id = $1`, doc.ID)
	require.Equal(t, "DRAFT", s)

	st, out := r.do(http.MethodPost, path, tok, uuid.NewString(), body)
	require.Equal(t, http.StatusOK, st)
	require.NotEmpty(t, out["scheme_id"])
}

// AC5: cùng Idempotency-Key gửi lại → cùng phản hồi, đúng một lần xác nhận.
func TestConfirmIdempotent(t *testing.T) {
	r := newHTTPRig(t)
	gv := r.user("TEACHER", "GV")
	c := r.course(gv, "L1")
	doc := r.newDraft(c, gv, 1, nil)
	path := "/api/v1/courses/" + c.String() + "/grade-scheme/confirm"
	tok, key := r.token(gv, auth.RoleTeacher), uuid.NewString()
	st1, out1 := r.do(http.MethodPost, path, tok, key, map[string]any{"version": doc.Version})
	st2, out2 := r.do(http.MethodPost, path, tok, key, map[string]any{"version": doc.Version})
	require.Equal(t, http.StatusOK, st1)
	require.Equal(t, st1, st2)
	require.Equal(t, out1, out2)
	require.Equal(t, 1, r.countRows(`select count(*) from audit_log where entity = 'grade_schemes' and action = 'confirm'`))
	require.Equal(t, 1, r.countRows(`select count(*) from outbox where topic = 'grade.scheme_confirmed'`))
}

// TestConfirmOnlyFromHandler — mã không-test chỉ gọi `ConfirmScheme` từ handler HTTP xác nhận (AI / job / worker không bao giờ tự xác nhận công thức).
func TestConfirmOnlyFromHandler(t *testing.T) {
	var hits []string
	require.NoError(t, filepath.WalkDir("..", func(p string, de os.DirEntry, err error) error {
		if err != nil || de.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(raw), "ConfirmScheme(") {
			hits = append(hits, filepath.ToSlash(p))
		}
		return nil
	}))
	require.ElementsMatch(t, []string{"../grade/confirm.go", "../httpapi/gradehttp/handler.go"}, hits)
}
