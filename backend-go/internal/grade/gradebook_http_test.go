package grade_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
)

// rbacRig: lớp có công thức CONFIRMED, một sinh viên có điểm; đủ vai để chạy ma trận quyền.
type rbacRig struct {
	*httpRig
	course, student, weekly uuid.UUID
	tok                     map[string]string
}

func newRBACRig(t *testing.T) *rbacRig {
	t.Helper()
	r := &rbacRig{httpRig: newHTTPRig(t)}
	var gv uuid.UUID
	r.course, gv, r.student, r.weekly, _ = r.sheet()
	ta := r.user("TA", "TA")
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TA', 'ACTIVE', 'ADMIN')`, r.course, ta)
	sv := r.student2()
	r.tok = map[string]string{
		"GV": r.token(gv, auth.RoleTeacher), "TA": r.token(ta, auth.RoleTA), "SV": r.token(sv, auth.RoleStudent),
		"ADMIN": r.token(r.user("ADMIN", "AD"), auth.RoleAdmin), "GV khác": r.token(r.user("TEACHER", "GV khác"), auth.RoleTeacher), "": "",
	}
	return r
}

func (r *rbacRig) student2() uuid.UUID { return r.tdb.student(r.course, "SV", "20220001") }

// TestRBACMatrix — các route của US-P6-06: vai × route (SRS 7.5). Đọc: TEACHER + TA; ghi: chỉ TEACHER; STUDENT / ADMIN / giảng viên lớp khác 403; chưa đăng nhập 401.
func TestRBACMatrix(t *testing.T) {
	r := newRBACRig(t)
	base := "/api/v1/courses/" + r.course.String()
	entry := map[string]any{"item_id": r.weekly, "student_id": r.student, "score": "7", "version": nil}
	routes := []struct {
		name   string
		method string
		path   string
		body   any
		write  bool
	}{
		{"gradebook", http.MethodGet, base + "/gradebook", nil, false},
		{"preview", http.MethodGet, base + "/gradebook/preview", nil, false},
		{"explain", http.MethodGet, base + "/gradebook/students/" + r.student.String() + "/explain", nil, false},
		{"history", http.MethodGet, base + "/gradebook/students/" + r.student.String() + "/history", nil, false},
		{"put entry", http.MethodPut, base + "/grade-entries", entry, true},
		{"publish", http.MethodPost, base + "/grade-items/" + r.weekly.String() + "/publish", map[string]any{"published": true}, true},
	}
	for _, rt := range routes {
		for who, tok := range r.tok {
			st, _ := r.do(rt.method, rt.path, tok, "", rt.body)
			switch {
			case who == "":
				require.Equal(t, http.StatusUnauthorized, st, rt.name)
			case who == "GV":
				require.Equal(t, http.StatusOK, st, rt.name)
			case who == "TA" && !rt.write:
				require.Equal(t, http.StatusOK, st, rt.name)
			default:
				require.Equal(t, http.StatusForbidden, st, "%s / %s", rt.name, who)
			}
		}
		st, _ := r.do(rt.method, "/api/v1/courses/khong-phai-uuid"+rt.path[len(base):], r.tok["GV"], "", rt.body)
		require.Equal(t, http.StatusNotFound, st, rt.name)
	}
}

// TestGradebookRBAC — AC10: TA đọc bảng / giải trình / lịch sử, KHÔNG ghi điểm; sinh viên / admin / lớp khác không đọc được.
func TestGradebookRBAC(t *testing.T) {
	r := newRBACRig(t)
	base := "/api/v1/courses/" + r.course.String()
	st, _ := r.do(http.MethodPut, base+"/grade-entries", r.tok["TA"], "", map[string]any{"item_id": r.weekly, "student_id": r.student, "score": "9", "version": nil})
	require.Equal(t, http.StatusForbidden, st)
	require.Zero(t, r.countRows(`select count(*) from grade_entries`), "TA không ghi được điểm")
	for who, tok := range map[string]string{"SV": r.tok["SV"], "ADMIN": r.tok["ADMIN"], "GV khác": r.tok["GV khác"]} {
		st, out := r.do(http.MethodGet, base+"/gradebook", tok, "", nil)
		require.Equal(t, http.StatusForbidden, st, who)
		require.NotContains(t, out, "rows", who)
	}
}

// TestPreviewRBAC — xem trước chỉ cho Staff của lớp (không lộ điểm dự kiến cho sinh viên / admin / lớp khác).
func TestPreviewRBAC(t *testing.T) {
	r := newRBACRig(t)
	path := "/api/v1/courses/" + r.course.String() + "/gradebook/preview"
	for who, want := range map[string]int{"GV": 200, "TA": 200, "SV": 403, "ADMIN": 403, "GV khác": 403, "": 401} {
		st, _ := r.do(http.MethodGet, path, r.tok[who], "", nil)
		require.Equal(t, want, st, who)
	}
}
