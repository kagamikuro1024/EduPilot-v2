package exam_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/platform/clock"
)

const guardSecret = "0123456789abcdef0123456789abcdef"

type fixedResolver struct{ m auth.Membership }

func (f fixedResolver) Resolve(context.Context, auth.Principal, uuid.UUID) (auth.Membership, error) {
	return f.m, nil
}

type enr struct {
	name string
	m    auth.Membership
}

func enrolments() []enr {
	return []enr{
		{"không có ghi danh", auth.Membership{}},
		{"ACTIVE GV", auth.Membership{Found: true, Role: auth.RoleTeacher, Status: "ACTIVE"}},
		{"ACTIVE TA", auth.Membership{Found: true, Role: auth.RoleTA, Status: "ACTIVE"}},
		{"ACTIVE SV", auth.Membership{Found: true, Role: auth.RoleStudent, Status: "ACTIVE"}},
		{"PENDING SV", auth.Membership{Found: true, Role: auth.RoleStudent, Status: "PENDING"}},
		{"REMOVED SV", auth.Membership{Found: true, Role: auth.RoleStudent, Status: "REMOVED"}},
		{"REMOVED GV", auth.Membership{Found: true, Role: auth.RoleTeacher, Status: "REMOVED"}},
	}
}

// serve gọi route thật của bảng `exam.Routes()` qua CourseAccessGuard thật với một Membership cố định.
func serve(t *testing.T, rt exam.Route, jwtRole auth.Role, m auth.Membership) (int, string) {
	t.Helper()
	clk := clock.NewFake(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
	tok, err := auth.NewIssuer(guardSecret, 0, clk).Issue("00000000-0000-7000-8000-000000000001", jwtRole, "qc@example.test")
	require.NoError(t, err)
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(auth.NewVerifier(guardSecret, clk)))
		r.With(auth.CourseAccessGuard(fixedResolver{m}, rt.Mode)).MethodFunc(rt.Method, "/courses/{courseId}"+rt.Path, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	})
	path := "/courses/" + uuid.NewString() + strings.NewReplacer("{qid}", uuid.NewString(), "{tid}", uuid.NewString(), "{eid}", uuid.NewString(), "{aid}", uuid.NewString(),
		"{itemId}", uuid.NewString(), "{runId}", uuid.NewString(), "{sid}", uuid.NewString(), "{id}", uuid.NewString()).Replace(rt.Path)
	req := httptest.NewRequest(rt.Method, path, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var body struct {
		Details struct {
			Reason string `json:"reason"`
		} `json:"details"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body.Details.Reason
}

// allowedRoles là tập vai TRONG LỚP được vào theo chế độ của route (SRS 6.2).
func allowedRoles(mode auth.GuardMode) map[auth.Role]bool {
	switch mode {
	case auth.StaffRole:
		return map[auth.Role]bool{auth.RoleTeacher: true, auth.RoleTA: true}
	case auth.TeacherRole:
		return map[auth.Role]bool{auth.RoleTeacher: true}
	case auth.StudentRole:
		return map[auth.Role]bool{auth.RoleStudent: true}
	default: // MemberRole
		return map[auth.Role]bool{auth.RoleTeacher: true, auth.RoleTA: true, auth.RoleStudent: true}
	}
}

// TestExamGuardMatrix — US-PE-01 AC4: mọi route × vai JWT × tình trạng ghi danh (55 route × 4 × 7 > 1.500 ca).
func TestExamGuardMatrix(t *testing.T) {
	t.Parallel()
	n := 0
	for _, rt := range exam.Routes() {
		if rt.No == exam.MeExamLockNo {
			continue
		}
		for _, jwt := range []auth.Role{auth.RoleAdmin, auth.RoleTeacher, auth.RoleTA, auth.RoleStudent} {
			for _, e := range enrolments() {
				code, reason := serve(t, rt, jwt, e.m)
				active := e.m.Found && e.m.Status == "ACTIVE" && jwt != auth.RoleAdmin
				want := active && allowedRoles(rt.Mode)[e.m.Role]
				name := rt.Method + " " + rt.Path + " · " + string(jwt) + " · " + e.name
				if want {
					require.Equal(t, http.StatusNoContent, code, name)
				} else {
					require.Equal(t, http.StatusForbidden, code, name)
					wantReason := "course"
					if active {
						wantReason = "role" // thành viên ACTIVE sai vai
					}
					require.Equal(t, wantReason, reason, name)
				}
				n++
			}
		}
	}
	require.Greater(t, n, 150)
}

// TestExamAdminDenied — ADMIN không bao giờ qua route nào của PE (không đọc nội dung bài thi), kể cả khi có ghi danh giả.
func TestExamAdminDenied(t *testing.T) {
	t.Parallel()
	for _, rt := range exam.Routes() {
		if rt.No == exam.MeExamLockNo {
			continue
		}
		for _, e := range enrolments() {
			code, _ := serve(t, rt, auth.RoleAdmin, e.m)
			require.Equal(t, http.StatusForbidden, code, rt.Path+" · "+e.name)
		}
	}
}

// TestExamStudentRouteRoleOnly — route của sinh viên: GV / TA ACTIVE → 403 reason="role"; người ngoài lớp / PENDING / REMOVED → "course".
func TestExamStudentRouteRoleOnly(t *testing.T) {
	t.Parallel()
	for _, rt := range exam.Routes() {
		if rt.Mode != auth.StudentRole {
			continue
		}
		for _, role := range []auth.Role{auth.RoleTeacher, auth.RoleTA} {
			code, reason := serve(t, rt, role, auth.Membership{Found: true, Role: role, Status: "ACTIVE"})
			require.Equal(t, http.StatusForbidden, code, rt.Path)
			require.Equal(t, "role", reason, rt.Path)
		}
		for _, m := range []auth.Membership{{}, {Found: true, Role: auth.RoleStudent, Status: "PENDING"}, {Found: true, Role: auth.RoleStudent, Status: "REMOVED"}} {
			code, reason := serve(t, rt, auth.RoleStudent, m)
			require.Equal(t, http.StatusForbidden, code)
			require.Equal(t, "course", reason)
		}
		code, _ := serve(t, rt, auth.RoleStudent, auth.Membership{Found: true, Role: auth.RoleStudent, Status: "ACTIVE"})
		require.Equal(t, http.StatusNoContent, code, rt.Path)
	}
}

// TestExamRoutesTable — 57 thao tác (56 của SRS + #57 chi tiết cặp độ giống, đề xuất #17), số thứ tự 1…57 liền nhau, không trùng (method, path).
func TestExamRoutesTable(t *testing.T) {
	t.Parallel()
	rs := exam.Routes()
	require.Len(t, rs, 57)
	seen := map[string]bool{}
	for i, rt := range rs {
		require.Equal(t, i+1, rt.No)
		k := rt.Method + " " + rt.Path
		require.False(t, seen[k], k)
		seen[k] = true
	}
}
