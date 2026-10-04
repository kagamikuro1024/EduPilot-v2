package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// fixedMembership trả một Membership cố định và đếm số lần được gọi.
type fixedMembership struct {
	m     Membership
	err   error
	calls atomic.Int64
}

func (f *fixedMembership) Resolve(context.Context, Principal, uuid.UUID) (Membership, error) {
	f.calls.Add(1)
	return f.m, f.err
}

func guardServe(t *testing.T, res CourseResolver, mode GuardMode, role Role, courseID string) *httptest.ResponseRecorder {
	t.Helper()
	clk := fixedClock()
	tok, err := NewIssuer(testSecret, 0, clk).Issue(testSub, role, testEmail)
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(Middleware(NewVerifier(testSecret, clk)))
		r.With(CourseAccessGuard(res, mode)).Get("/courses/{id}/ping", func(w http.ResponseWriter, r *http.Request) {
			a, _ := CourseFromContext(r.Context())
			_, _ = fmt.Fprintf(w, "%s|%s", a.CourseID, a.CourseRole)
		})
	})
	return do(t, r, "/courses/"+courseID+"/ping", "Bearer "+tok)
}

// US-P2-07 AC4: ma trận vai JWT × tình trạng ghi danh × 6 chế độ khớp SRS FEAT-course-foundation 4.1.
func TestGuardMatrix(t *testing.T) {
	t.Parallel()
	type enr struct {
		name string
		m    Membership
	}
	enrolments := []enr{
		{"không có ghi danh", Membership{}},
		{"ACTIVE vai SV", Membership{Found: true, Role: RoleStudent, Status: "ACTIVE"}},
		{"ACTIVE vai TA", Membership{Found: true, Role: RoleTA, Status: "ACTIVE"}},
		{"ACTIVE vai GV", Membership{Found: true, Role: RoleTeacher, Status: "ACTIVE"}},
		{"PENDING SV", Membership{Found: true, Role: RoleStudent, Status: "PENDING"}},
		{"REMOVED GV", Membership{Found: true, Role: RoleTeacher, Status: "REMOVED"}},
		{"REMOVED TA", Membership{Found: true, Role: RoleTA, Status: "REMOVED"}},
	}
	modes := map[GuardMode]string{Member: "Member", Staff: "Staff", Teacher: "Teacher", StaffOrAdmin: "StaffOrAdmin", Manage: "Manage", MemberOrAdmin: "MemberOrAdmin"}
	// want[mode] = hàm (vai ghi danh đang ACTIVE, JWT) → được qua không
	activeRole := func(m Membership) Role {
		if m.Found && m.Status == "ACTIVE" {
			return m.Role
		}
		return ""
	}
	want := func(mode GuardMode, m Membership, jwt Role) bool {
		r := activeRole(m)
		switch mode {
		case Member:
			return r != ""
		case Staff:
			return r == RoleTeacher || r == RoleTA
		case Teacher:
			return r == RoleTeacher
		case StaffOrAdmin:
			return r == RoleTeacher || r == RoleTA || jwt == RoleAdmin
		case Manage:
			return r == RoleTeacher || jwt == RoleAdmin
		case MemberOrAdmin:
			return r != "" || jwt == RoleAdmin
		}
		return false
	}
	cases := 0
	for _, jwt := range []Role{RoleStudent, RoleTA, RoleTeacher, RoleAdmin} {
		for _, e := range enrolments {
			for mode, mname := range modes {
				cases++
				rec := guardServe(t, &fixedMembership{m: e.m}, mode, jwt, testCourseA)
				if got := rec.Code == http.StatusOK; got != want(mode, e.m, jwt) {
					t.Errorf("JWT %s · %s · %s: status=%d, muốn qua=%v", jwt, e.name, mname, rec.Code, want(mode, e.m, jwt))
					continue
				}
				if rec.Code == http.StatusForbidden {
					body := decodeBody(t, rec)
					d, _ := body["details"].(map[string]any)
					if body["code"] != "FORBIDDEN" || d["reason"] != "course" {
						t.Errorf("JWT %s · %s · %s: thân 403 = %v", jwt, e.name, mname, body)
					}
				}
				if rec.Code == http.StatusOK && e.m.Found && e.m.Status == "ACTIVE" && rec.Body.String() != testCourseA+"|"+string(e.m.Role) {
					t.Errorf("JWT %s · %s · %s: CourseAccess.Role phải lấy từ ghi danh, được %q", jwt, e.name, mname, rec.Body.String())
				}
			}
		}
	}
	if cases < 48 {
		t.Fatalf("chỉ %d ca, cần ≥ 48", cases)
	}
}

func TestGuardNonUUID404(t *testing.T) {
	t.Parallel()
	res := &fixedMembership{m: Membership{Found: true, Role: RoleTeacher, Status: "ACTIVE"}}
	for _, id := range []string{"abc", "123", "00000000-0000-7000-8000-0000000000a", "../etc/passwd"} {
		rec := guardServe(t, res, Member, RoleAdmin, id)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%q: status=%d, muốn 404", id, rec.Code)
		}
	}
	if n := res.calls.Load(); n != 0 {
		t.Fatalf("resolver được gọi %d lần với id sai, muốn 0", n)
	}
}

func TestGuardResolverError503(t *testing.T) {
	t.Parallel()
	res := &fixedMembership{err: fmt.Errorf("enrollments hỏng")}
	for _, mode := range []GuardMode{Member, StaffOrAdmin, MemberOrAdmin} {
		if rec := guardServe(t, res, mode, RoleAdmin, testCourseA); rec.Code != http.StatusServiceUnavailable {
			t.Errorf("mode %d: status=%d, muốn 503 (không cho ADMIN qua khi tra quyền lỗi)", mode, rec.Code)
		}
	}
}

func TestGuardNoCache(t *testing.T) {
	t.Parallel()
	res := &fixedMembership{m: Membership{Found: true, Role: RoleStudent, Status: "ACTIVE"}}
	clk := fixedClock()
	tok, _ := NewIssuer(testSecret, 0, clk).Issue(testSub, RoleStudent, testEmail)
	r := chi.NewRouter()
	r.With(Middleware(NewVerifier(testSecret, clk)), CourseAccessGuard(res, Member)).Get("/courses/{id}/ping", func(w http.ResponseWriter, _ *http.Request) {})
	path := "/courses/" + testCourseA + "/ping"
	if rec := do(t, r, path, "Bearer "+tok); rec.Code != http.StatusOK {
		t.Fatalf("status=%d, muốn 200", rec.Code)
	}
	res.m = Membership{Found: true, Role: RoleStudent, Status: "REMOVED"} // bị mời ra giữa chừng
	if rec := do(t, r, path, "Bearer "+tok); rec.Code != http.StatusForbidden {
		t.Fatalf("sau khi bị mời ra: status=%d, muốn 403 ngay yêu cầu kế (không cache)", rec.Code)
	}
	if n := res.calls.Load(); n != 2 {
		t.Fatalf("resolver được gọi %d lần, muốn 2 (một lần mỗi request)", n)
	}
}

func TestGuardAdminDeniedByDefault(t *testing.T) {
	t.Parallel()
	for _, mode := range []GuardMode{Member, Staff, Teacher} {
		if rec := guardServe(t, DenyAll{}, mode, RoleAdmin, testCourseA); rec.Code != http.StatusForbidden {
			t.Errorf("mode %d: ADMIN status=%d, muốn 403 (ADMIN không đọc nội dung lớp)", mode, rec.Code)
		}
	}
	for _, mode := range []GuardMode{StaffOrAdmin, Manage, MemberOrAdmin} {
		if rec := guardServe(t, DenyAll{}, mode, RoleAdmin, testCourseA); rec.Code != http.StatusOK {
			t.Errorf("mode %d: ADMIN status=%d, muốn 200 (chế độ có \"hoặc ADMIN\")", mode, rec.Code)
		}
	}
}

// Q-QC-P207-1: lớp không tồn tại và lớp có thật mà người gọi không thuộc cho CÙNG thân 403 (không lộ tồn tại).
func TestGuardNonexistentSameAsOutsider(t *testing.T) {
	t.Parallel()
	a := guardServe(t, DenyAll{}, Member, RoleStudent, uuid.NewString())
	b := guardServe(t, DenyAll{}, Member, RoleStudent, testCourseA)
	if a.Code != http.StatusForbidden || b.Code != http.StatusForbidden {
		t.Fatalf("status %d / %d, muốn 403 cả hai", a.Code, b.Code)
	}
	x, y := decodeBody(t, a), decodeBody(t, b)
	delete(x, "trace_id")
	delete(y, "trace_id")
	if fmt.Sprint(x) != fmt.Sprint(y) {
		t.Fatalf("thân khác nhau: %v vs %v", x, y)
	}
}
