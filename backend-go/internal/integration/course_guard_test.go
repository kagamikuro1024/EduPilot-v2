package integration_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

var courseParam = regexp.MustCompile(`\{(courseId|id)\}`)

type courseRoute struct{ method, path string }

// courseRoutes quét bảng route thật của gateway: mọi route có {courseId}, hoặc {id} dưới /courses/.
func courseRoutes(t *testing.T, h http.Handler) []courseRoute {
	t.Helper()
	mux, ok := h.(chi.Routes)
	require.True(t, ok, "router phải là chi.Routes để quét bằng chi.Walk")
	var out []courseRoute
	require.NoError(t, chi.Walk(mux, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		route = strings.TrimPrefix(strings.ReplaceAll(route, "/*/", "/"), "/api/v1") // r.do tự thêm tiền tố
		if strings.HasPrefix(route, "/admin/") {                                     // quản trị: RequireRole(ADMIN) và test ma trận riêng (TestAdminCoursesRBACMatrix), không phải route của một lớp
			return nil
		}
		if strings.Contains(route, "{courseId}") || strings.HasPrefix(route, "/courses/{id}") {
			out = append(out, courseRoute{method, route})
		}
		return nil
	}))
	return out
}

// concrete thay mọi tham số đường dẫn bằng uuid; tham số lớp bằng courseID.
func (cr courseRoute) concrete(courseID uuid.UUID) string {
	p := courseParam.ReplaceAllString(cr.path, courseID.String())
	return regexp.MustCompile(`\{[^}]+\}`).ReplaceAllStringFunc(p, func(string) string { return uuid.NewString() })
}

// callAs gọi một route bằng token của người dùng, kèm Idempotency-Key để không bị chặn sớm ở bước khác.
func (r *rig) callAs(s session, cr courseRoute, courseID uuid.UUID) resp {
	var body any
	if cr.method != http.MethodGet && cr.method != http.MethodDelete {
		body = map[string]any{}
	}
	return r.do(req{method: cr.method, path: cr.concrete(courseID), bearer: s.access, body: body, hdr: map[string]string{"Idempotency-Key": uuid.NewString()}})
}

// adminOnly là các route lớp của sprint trước mang hợp đồng ADMIN-only bất biến (RequireRole, 422 khi id không phải uuid), không qua
// CourseAccessGuard. Danh sách đóng: thêm route vào đây phải có lý do trong proposals; chúng vẫn bị kiểm 403 `reason=role` với người không phải ADMIN.
var adminOnly = map[string]bool{"GET /courses/{id}/llm-budget": true, "PUT /courses/{id}/llm-budget": true}

// AC5: mọi route lớp đều có guard — người ngoài lớp (token hợp lệ) không nhận 2xx; một route quên guard thì bị lộ.
func TestAllCourseRoutesGuarded(t *testing.T) {
	r := newRig(t)
	adm := r.addUser(uniq("adm"), store.UserRoleADMIN, store.UserStatusACTIVE)
	course := r.course(adm.ID, "GRD-"+uuid.NewString()[:6])
	routes := courseRoutes(t, r.h)
	require.NotEmpty(t, routes, "bảng route phải có ít nhất một route lớp")
	for _, role := range []store.UserRole{store.UserRoleSTUDENT, store.UserRoleTA, store.UserRoleTEACHER} {
		outsider := r.mustLogin(r.addUser(uniq("out"), role, store.UserStatusACTIVE).Email)
		for _, cr := range routes {
			res := r.callAs(outsider, cr, course)
			require.Falsef(t, res.code >= 200 && res.code < 300, "%s %s: người ngoài lớp (%s) nhận %d", cr.method, cr.path, role, res.code)
			require.Equalf(t, http.StatusForbidden, res.code, "%s %s (%s): thiếu CourseAccessGuard? status=%d %s", cr.method, cr.path, role, res.code, res.body)
			want := "course"
			if adminOnly[cr.method+" "+cr.path] {
				want = "role"
			}
			require.Equal(t, want, res.details()["reason"], "%s %s", cr.method, cr.path)
		}
	}
}

// Bằng chứng bộ quét bắt được route quên guard: route thường (không guard) với người ngoài lớp KHÔNG ra 403.
func TestAllCourseRoutesGuardedDetectsMissing(t *testing.T) {
	mux := chi.NewRouter()
	mux.Get("/courses/{id}/oops", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	found := courseRoutes(t, mux)
	require.Len(t, found, 1)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, found[0].concrete(uuid.New()), nil))
	require.NotEqual(t, http.StatusForbidden, rec.Code, "route không guard phải làm TestAllCourseRoutesGuarded đỏ")
}

// AC5: sinh viên chỉ ở lớp 1 gọi MỌI route lớp 2 → 403, và không thấy lớp 2 ở me/courses.
func TestCourseIsolation(t *testing.T) {
	r := newRig(t)
	adm := r.addUser(uniq("adm"), store.UserRoleADMIN, store.UserStatusACTIVE)
	sv := r.addUser(uniq("sv1"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	tag := uuid.NewString()[:6]
	c1, c2 := r.course(adm.ID, "ISO1-"+tag), r.course(adm.ID, "ISO2-"+tag)
	r.enroll(c1, sv.ID, "STUDENT", "ACTIVE")
	s := r.mustLogin(sv.Email)
	for _, cr := range courseRoutes(t, r.h) {
		res := r.callAs(s, cr, c2)
		require.Equalf(t, http.StatusForbidden, res.code, "%s %s: lớp 2 phải 403, được %d", cr.method, cr.path, res.code)
	}
	require.Equal(t, http.StatusOK, r.callAs(s, courseRoute{http.MethodGet, "/courses/{id}"}, c1).code, "lớp của chính mình vẫn mở")
	list := r.do(req{method: http.MethodGet, path: "/me/courses", bearer: s.access})
	require.Equal(t, http.StatusOK, list.code)
	require.Contains(t, string(list.body), "ISO1-"+tag)
	require.NotContains(t, string(list.body), "ISO2-"+tag)
	require.NotContains(t, string(list.body), c2.String())
}

// AC6: truy vấn nào chạm content_chunks cũng phải có điều kiện course_ids (chặn hàm đọc chunk "không lọc lớp").
func TestNoUnscopedChunkQuery(t *testing.T) {
	files, err := filepath.Glob("../store/queries/*.sql")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	reads := 0
	for _, f := range files {
		raw, err := os.ReadFile(f)
		require.NoError(t, err)
		for _, block := range strings.Split(string(raw), "-- name:")[1:] {
			lower := strings.ToLower(block)
			if !strings.Contains(lower, "content_chunks") {
				continue
			}
			head, _, _ := strings.Cut(block, "\n")
			name := strings.TrimSpace(strings.SplitN(strings.TrimSpace(head), " ", 2)[0])
			if strings.Contains(lower, "insert into content_chunks") || strings.Contains(lower, "delete from content_chunks") {
				continue // ghi theo document_id (việc của nạp tài liệu), không phải đường ĐỌC
			}
			reads++
			require.Containsf(t, lower, "course_ids", "%s (%s): đọc content_chunks mà không lọc theo course_ids", name, filepath.Base(f))
		}
	}
	require.Equal(t, 1, reads, "P2 chỉ có MỘT đường đọc chunk: ChunksForCourse")
}
