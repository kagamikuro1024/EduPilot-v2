package document_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/testutil"
)

const jwtSecret = "0123456789abcdef0123456789abcdef"

type hrig struct {
	*fx
	srv *httptest.Server
	iss *auth.Issuer
}

// newHRig dựng ROUTER THẬT (auth, CourseAccessGuard thật) cùng CSDL / MinIO của fx.
func newHRig(t *testing.T) *hrig {
	t.Helper()
	f := newFx(t)
	env := map[string]string{
		"DATABASE_URL": f.pool.Config().ConnString(), "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": jwtSecret,
		"BLOB_ENDPOINT": testutil.MinIOEndpoint(t), "BLOB_BUCKET": "x", "BLOB_ACCESS_KEY": testutil.MinIOAccessKey, "BLOB_SECRET_KEY": testutil.MinIOSecretKey,
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test", "BCRYPT_COST": "4",
		"RATE_LIMIT_IP_PER_MIN": "1000000", "RATE_LIMIT_USER_PER_MIN": "1000000", "REQUEST_TIMEOUT": "5s",
	}
	cfg, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	require.NoError(t, err)
	h := httpapi.NewRouter(httpapi.Deps{Cfg: cfg, DB: f.pool, Redis: f.rdb, Clock: clock.Real{}, State: httpapi.NewState(), Blob: f.blob})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &hrig{fx: f, srv: srv, iss: auth.NewIssuer(jwtSecret, time.Hour, clock.Real{})}
}

func (h *hrig) do(t *testing.T, u uuid.UUID, role auth.Role, method, path, body string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, h.srv.URL+"/api/v1"+path, strings.NewReader(body))
	require.NoError(t, err)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if u != uuid.Nil {
		tok, err := h.iss.Issue(u.String(), role, "x@example.test")
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	return res.StatusCode
}

// TestDocumentsMatrix — AC14: TA + TEACHER của lớp quản lý tài liệu (xoá: chỉ TEACHER); STUDENT, ADMIN, ngoài lớp, PENDING / REMOVED → 403; chưa đăng nhập 401.
func TestDocumentsMatrix(t *testing.T) {
	t.Parallel()
	h := newHRig(t)
	sv, admin, outsider, pending := h.user(t, "SV", "STUDENT"), h.user(t, "AD", "ADMIN"), h.user(t, "O", "STUDENT"), h.user(t, "P", "STUDENT")
	_, err := h.pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN'), ($1, $3, 'STUDENT', 'PENDING', 'ADMIN'), ($1, $4, 'TA', 'ACTIVE', 'ADMIN')`, h.course, sv, pending, h.ta)
	require.NoError(t, err)
	doc := h.doc(t, h.course, "Bài", "LECTURE", true, true, 1)
	base := "/courses/" + h.course.String() + "/documents"
	for _, c := range []struct {
		name string
		u    uuid.UUID
		r    auth.Role
		want int
	}{{"teacher", h.teacher, auth.RoleTeacher, 200}, {"ta", h.ta, auth.RoleTA, 200}, {"student", sv, auth.RoleStudent, 403}, {"admin", admin, auth.RoleAdmin, 403}, {"outsider", outsider, auth.RoleStudent, 403},
		{"pending", pending, auth.RoleStudent, 403}, {"anonymous", uuid.Nil, "", 401}} {
		for _, p := range []string{base, base + "/stats", base + "/" + doc.String(), base + "/" + doc.String() + "/chunks"} {
			require.Equal(t, c.want, h.do(t, c.u, c.r, http.MethodGet, p, ""), c.name+" "+p)
		}
	}
	require.Equal(t, 403, h.do(t, sv, auth.RoleStudent, http.MethodPatch, base+"/"+doc.String(), `{"title":"X","version":1}`))
	require.Equal(t, 200, h.do(t, h.ta, auth.RoleTA, http.MethodPatch, base+"/"+doc.String(), `{"title":"X","version":1}`))
}

// TestDeleteTeacherOnly — AC8: TA gọi xoá → 403 (không mất gì); TEACHER → 204; xoá lại → 404.
func TestDeleteTeacherOnly(t *testing.T) {
	t.Parallel()
	h := newHRig(t)
	_, err := h.pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TA', 'ACTIVE', 'ADMIN')`, h.course, h.ta)
	require.NoError(t, err)
	doc := h.doc(t, h.course, "Bài", "LECTURE", true, true, 1)
	p := "/courses/" + h.course.String() + "/documents/" + doc.String()
	require.Equal(t, 403, h.do(t, h.ta, auth.RoleTA, http.MethodDelete, p, ""))
	require.Equal(t, 1, h.count(t, `select count(*) from documents where id=$1`, doc))
	require.Equal(t, 204, h.do(t, h.teacher, auth.RoleTeacher, http.MethodDelete, p, ""))
	require.Equal(t, 404, h.do(t, h.teacher, auth.RoleTeacher, http.MethodDelete, p, ""))
}
