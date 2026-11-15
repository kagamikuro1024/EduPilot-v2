package assignment_test

import (
	"bytes"
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
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

const asgSecret = "assignment-http-secret-0123456789abcdef-dev"

// httpEnv dựng router gateway thật trên CSDL của env (kiểm quyền qua HTTP).
type httpEnv struct {
	*env
	h              http.Handler
	ta, other, adm uuid.UUID
}

func newHTTPEnv(t *testing.T) *httpEnv {
	t.Helper()
	e := newEnv(t)
	vals := map[string]string{"DATABASE_URL": "postgres://x", "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": asgSecret, "BLOB_ENDPOINT": "127.0.0.1:9", "BLOB_BUCKET": "x", "BLOB_ACCESS_KEY": "x", "BLOB_SECRET_KEY": "x",
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test", "RATE_LIMIT_IP_PER_MIN": "1000000", "RATE_LIMIT_USER_PER_MIN": "1000000"}
	cfg, err := config.Load(func(k string) string { return vals[k] }, config.Gateway)
	require.NoError(t, err)
	rdb, err := appredis.New(t.Context(), cfg.RedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	store, err := blob.New(t.Context(), blob.Config{Endpoint: "127.0.0.1:9", Bucket: "x", AccessKey: "x", SecretKey: "x"})
	require.NoError(t, err)
	h := &httpEnv{env: e}
	h.h = httpapi.NewRouter(httpapi.Deps{Cfg: cfg, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: e.pool, Redis: rdb, Clock: clock.Real{}, State: httpapi.NewState(), Jobs: jobs.NewService(e.pool), Blob: store})
	h.ta, h.other, h.adm = e.user("TA"), e.user("TEACHER"), e.user("ADMIN")
	e.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TA', 'ACTIVE', 'ADMIN')`, e.course, h.ta)
	return h
}

func (h *httpEnv) call(method, path string, uid uuid.UUID, role auth.Role, body string) int {
	h.t.Helper()
	tok, err := auth.NewIssuer(asgSecret, time.Hour, clock.Real{}).Issue(uid.String(), role, "")
	require.NoError(h.t, err)
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	if method != http.MethodGet {
		req.Header.Set("Idempotency-Key", uuid.NewString())
	}
	w := httptest.NewRecorder()
	h.h.ServeHTTP(w, req)
	return w.Code
}

// TestPublishRBAC — AC9: công bố chỉ TEACHER của lớp; TA, STUDENT, ADMIN, giảng viên lớp khác 403.
func TestPublishRBAC(t *testing.T) {
	h := newHTTPEnv(t)
	a := h.create("BT03")
	r := h.reviewed(a)
	path := "/api/v1/courses/" + h.course.String() + "/grading-results/publish"
	body := `{"result_ids":["` + r.res.String() + `"]}`
	stu := h.studentOf(r)
	for name, c := range map[string]struct {
		id   uuid.UUID
		role auth.Role
	}{"TA": {h.ta, auth.RoleTA}, "sinh viên": {stu, auth.RoleStudent}, "admin": {h.adm, auth.RoleAdmin}, "giảng viên lớp khác": {h.other, auth.RoleTeacher}} {
		require.Equal(t, http.StatusForbidden, h.call(http.MethodPost, path, c.id, c.role, body), name)
	}
	require.Equal(t, "REVIEWED", h.status(r.res))
	require.Equal(t, http.StatusOK, h.call(http.MethodPost, path, h.teacher, auth.RoleTeacher, body))
	require.Equal(t, "PUBLISHED", h.status(r.res))
}

// TestStatsRBAC — AC9: số liệu bài: TA, TEACHER 200; còn lại 403.
func TestStatsRBAC(t *testing.T) {
	h := newHTTPEnv(t)
	a := h.create("BT03")
	path := "/api/v1/courses/" + h.course.String() + "/assignments/" + a.ID.String() + "/stats"
	require.Equal(t, http.StatusOK, h.call(http.MethodGet, path, h.teacher, auth.RoleTeacher, ""))
	require.Equal(t, http.StatusOK, h.call(http.MethodGet, path, h.ta, auth.RoleTA, ""))
	for name, c := range map[string]struct {
		id   uuid.UUID
		role auth.Role
	}{"sinh viên": {h.students[0], auth.RoleStudent}, "admin": {h.adm, auth.RoleAdmin}, "giảng viên lớp khác": {h.other, auth.RoleTeacher}} {
		require.Equal(t, http.StatusForbidden, h.call(http.MethodGet, path, c.id, c.role, ""), name)
	}
}

// TestResultIDOR — AC9: sinh viên A đoán id kết quả / bài nộp của B → 403 ở mọi route kết quả; không đường nào cho sinh viên.
func TestResultIDOR(t *testing.T) {
	h := newHTTPEnv(t)
	a := h.create("BT03")
	rb := h.reviewed(a)
	sa := h.students[0]
	base := "/api/v1/courses/" + h.course.String()
	for _, c := range []struct{ method, path, body string }{
		{http.MethodGet, base + "/grading-results/" + rb.res.String(), ""},
		{http.MethodGet, base + "/grading-results/" + rb.sub.String(), ""},
		{http.MethodPut, base + "/grading-results/" + rb.res.String(), `{"version":1}`},
		{http.MethodPost, base + "/grading-results/" + rb.res.String() + "/review", `{"version":1}`},
		{http.MethodPost, base + "/grading-results/" + rb.res.String() + "/retry", ``},
		{http.MethodGet, base + "/submissions/" + rb.sub.String() + "/file", ""},
	} {
		require.Equal(t, http.StatusForbidden, h.call(c.method, c.path, sa, auth.RoleStudent, c.body), c.method+" "+c.path)
	}
}

// TestGuidanceRBAC — FEAT-grading-guidance US-GD-04 AC6 / US-GD-11: cờ chỉ TEACHER + ADMIN đọc; duyệt / bỏ / áp dụng / rút CHỈ giảng viên của lớp (TA, SV, ADMIN, giảng viên lớp khác 403);
// xem tổng quan, số đo, chấm lại: TA được (cờ tắt → chấm lại 409 chứ không 403); sinh viên và ADMIN không.
func TestGuidanceRBAC(t *testing.T) {
	h := newHTTPEnv(t)
	a := h.create("BT03")
	base := "/api/v1/courses/" + h.course.String()
	asg := base + "/assignments/" + a.ID.String()
	stu := h.students[0]
	type who struct {
		id   uuid.UUID
		role auth.Role
	}
	teacher, ta := who{h.teacher, auth.RoleTeacher}, who{h.ta, auth.RoleTA}
	sv, adm, other := who{stu, auth.RoleStudent}, who{h.adm, auth.RoleAdmin}, who{h.other, auth.RoleTeacher}
	item := uuid.NewString()
	for _, c := range []struct {
		name, method, path, body string
		ok                       map[string]int // người → mã; còn lại 403
		def                      int
	}{
		{"cờ đọc", http.MethodGet, base + "/guidance-flags", "", map[string]int{"giảng viên": 200, "admin": 200}, 403},
		{"tổng quan", http.MethodGet, asg + "/guidance", "", map[string]int{"giảng viên": 200, "TA": 200}, 403},
		{"số đo", http.MethodGet, asg + "/guidance/stats", "", map[string]int{"giảng viên": 200, "TA": 200}, 403},
		{"rút lưu ý", http.MethodPost, asg + "/guidance/reflect", "", map[string]int{"giảng viên": 409}, 403}, // cờ tắt → 409 GUIDANCE_DISABLED
		{"duyệt / bỏ", http.MethodPost, asg + "/guidance/items/" + item + "/decide", `{"decision":"APPROVE","version":1}`, map[string]int{"giảng viên": 409}, 403},
		{"áp dụng", http.MethodPost, asg + "/guidance/apply", `{"expected_version":0}`, map[string]int{"giảng viên": 409}, 403},
		{"chấm lại", http.MethodPost, asg + "/regrade-stale", "", map[string]int{"giảng viên": 409, "TA": 409}, 403},
	} {
		for name, w := range map[string]who{"giảng viên": teacher, "TA": ta, "sinh viên": sv, "admin": adm, "giảng viên lớp khác": other} {
			want := c.def
			if v, ok := c.ok[name]; ok {
				want = v
			}
			require.Equal(t, want, h.call(c.method, c.path, w.id, w.role, c.body), c.name+" · "+name)
		}
	}
	// cờ: PUT — REGRESSION_CAPTURE chỉ ADMIN; TA / SV 403
	put := base + "/guidance-flags/REGRESSION_CAPTURE"
	require.Equal(t, 403, h.call(http.MethodPut, put, h.teacher, auth.RoleTeacher, `{"enabled":true,"version":1}`))
	require.Equal(t, 403, h.call(http.MethodPut, base+"/guidance-flags/GRADING_GUIDANCE", h.ta, auth.RoleTA, `{"enabled":true,"version":1}`))
	require.Equal(t, 200, h.call(http.MethodPut, put, h.adm, auth.RoleAdmin, `{"enabled":true,"version":1}`))
}
