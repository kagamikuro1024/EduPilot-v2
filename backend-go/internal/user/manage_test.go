package user_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

// staff tạo một giảng viên ACTIVE và trả (user, version=1).
func (r *rig) staff(role store.UserRole) store.User {
	return r.addUser(uniq("nv"), role, store.UserStatusACTIVE)
}

// AC5.
func TestDisableRevokesSessions(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	gv := r.staff(store.UserRoleTEACHER)
	s1, s2 := r.mustLogin(gv.Email), r.mustLogin(gv.Email)
	require.Equal(t, http.StatusNotFound, r.api(s1.access).code)
	res := r.patch(a, gv.ID, map[string]any{"status": "DISABLED", "version": 1})
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, "DISABLED", res.json()["status"])
	require.EqualValues(t, 2, res.json()["version"])
	for _, s := range []session{s1, s2} {
		got := r.api(s.access)
		require.Equal(t, http.StatusUnauthorized, got.code)
		require.Equal(t, "SESSION_REVOKED", got.errCode())
		require.Equal(t, "account_disabled", got.details()["reason"])
	}
	require.Equal(t, "2", r.scalar(`select count(*)::text from auth_sessions where user_id = $1 and revoked_reason = 'ACCOUNT_DISABLED'`, gv.ID))
	require.Equal(t, "1", r.scalar(`select count(*)::text from audit_log where entity_id = $1 and action = 'user_disabled' and actor_id is not null`, gv.ID.String()))
}

func TestDisabledLoginUniform(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	gv := r.staff(store.UserRoleTA)
	require.Equal(t, http.StatusOK, r.patch(a, gv.ID, map[string]any{"status": "DISABLED", "version": 1}).code)
	wrong := r.login(gv.Email, "sai-mat-khau-1")
	require.Equal(t, http.StatusUnauthorized, wrong.code, "sai mật khẩu vẫn 401: không lộ trạng thái")
	require.Equal(t, "INVALID_CREDENTIALS", wrong.errCode())
}

func TestDisabledCorrectPassword403(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	gv := r.staff(store.UserRoleTEACHER)
	require.Equal(t, http.StatusOK, r.patch(a, gv.ID, map[string]any{"status": "DISABLED", "version": 1}).code)
	res := r.login(gv.Email, rigPassword)
	require.Equal(t, http.StatusForbidden, res.code)
	require.Equal(t, "ACCOUNT_DISABLED", res.errCode())
}

func TestEnableRestores(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	gv := r.staff(store.UserRoleTEACHER)
	require.Equal(t, http.StatusOK, r.patch(a, gv.ID, map[string]any{"status": "DISABLED", "version": 1}).code)
	res := r.patch(a, gv.ID, map[string]any{"status": "ACTIVE", "version": 2})
	require.Equal(t, http.StatusOK, res.code)
	require.Equal(t, "ACTIVE", res.json()["status"])
	require.Equal(t, http.StatusOK, r.login(gv.Email, rigPassword).code)
	// chưa nhận lời mời (không có mật khẩu) ⇒ mở khoá về INVITED
	inv := r.invite(a, uniq("chua.nhan"), "Chưa Nhận", "TA").json()
	id := inv["id"].(string)
	require.Equal(t, http.StatusOK, r.patch(a, id, map[string]any{"status": "DISABLED", "version": 1}).code)
	back := r.patch(a, id, map[string]any{"status": "ACTIVE", "version": 2})
	require.Equal(t, http.StatusOK, back.code)
	require.Equal(t, "INVITED", back.json()["status"])
}

func TestDisableVersionConflict(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	gv := r.staff(store.UserRoleTEACHER)
	res := r.patch(a, gv.ID, map[string]any{"status": "DISABLED", "version": 7})
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "VERSION_CONFLICT", res.errCode())
	require.EqualValues(t, 1, res.details()["current_version"])
	require.Equal(t, "ACTIVE", r.scalar(`select status::text from users where id = $1`, gv.ID), "không đổi gì khi lỗi")
	require.Equal(t, http.StatusUnprocessableEntity, r.patch(a, gv.ID, map[string]any{"status": "DISABLED"}).code, "thiếu version")
}

func TestDisableKeepsEnrollments(t *testing.T) {
	r := newRig(t)
	adm, a := r.admin()
	gv := r.staff(store.UserRoleTEACHER)
	var course string
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by)
		values ('INT1006', 'LOP-'||left(gen_random_uuid()::text, 6), 'Lớp thử', '2026-2027-HK1', 'AN7K2MQ', $1) returning id::text`, adm.ID).Scan(&course))
	_, err := r.pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TEACHER', 'ACTIVE', 'ADMIN')`, course, gv.ID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.patch(a, gv.ID, map[string]any{"status": "DISABLED", "version": 1}).code)
	require.Equal(t, http.StatusOK, r.patch(a, gv.ID, map[string]any{"role": "TA", "version": 2}).code)
	require.Equal(t, "1", r.scalar(`select count(*)::text from enrollments where user_id = $1 and status = 'ACTIVE' and role_in_course = 'TEACHER'`, gv.ID), "đổi vai / khoá không đụng enrollments")
}

// AC6.
func TestRoleChangeAllowedOnlyStaff(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	gv := r.staff(store.UserRoleTEACHER)
	res := r.patch(a, gv.ID, map[string]any{"role": "TA", "version": 1})
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, "TA", res.json()["role"])
	back := r.patch(a, gv.ID, map[string]any{"role": "TEACHER", "version": 2})
	require.Equal(t, "TEACHER", back.json()["role"])
	for _, role := range []string{"ADMIN", "STUDENT", "ROOT", ""} {
		got := r.patch(a, gv.ID, map[string]any{"role": role, "version": 3})
		require.Equal(t, http.StatusUnprocessableEntity, got.code, role)
		require.Equal(t, "VALIDATION_FAILED", got.errCode())
	}
}

func TestRoleChangeRevokesSessions(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	gv := r.staff(store.UserRoleTEACHER)
	s := r.mustLogin(gv.Email)
	require.Equal(t, http.StatusOK, r.patch(a, gv.ID, map[string]any{"role": "TA", "version": 1}).code)
	got := r.api(s.access)
	require.Equal(t, http.StatusUnauthorized, got.code)
	require.Equal(t, "SESSION_REVOKED", got.errCode())
	require.Equal(t, "role_changed", got.details()["reason"])
	s2 := r.mustLogin(gv.Email)
	require.Equal(t, "TA", claimsOf(t, s2.access)["role"], "token mới mang vai mới")
}

func TestCannotChangeSelf(t *testing.T) {
	r := newRig(t)
	adm, a := r.admin()
	_ = r.addUser(uniq("admin.khac"), store.UserRoleADMIN, store.UserStatusACTIVE) // có quản trị viên khác nên không phải "cuối cùng"
	for _, body := range []map[string]any{{"status": "DISABLED", "version": 1}, {"role": "TA", "version": 1}} {
		res := r.patch(a, adm.ID, body)
		require.Equal(t, http.StatusConflict, res.code, fmt.Sprint(body))
		require.Equal(t, "CONFLICT", res.errCode())
		require.Equal(t, "self", res.details()["reason"])
	}
	require.Equal(t, http.StatusOK, r.patch(a, adm.ID, map[string]any{"full_name": "Đổi Tên Mình", "version": 1}).code, "đổi tên mình thì được")
}

func TestCannotDisableLastAdmin(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	c := r.addUser(uniq("admin.c"), store.UserRoleADMIN, store.UserStatusACTIVE)
	// C là ADMIN ACTIVE duy nhất: mọi ADMIN khác (kể cả người đang thao tác) bị khoá thẳng ở DB; token của A vẫn còn hạn.
	_, err := r.pool.Exec(t.Context(), `update users set status = 'DISABLED' where role = 'ADMIN' and id <> $1`, c.ID)
	require.NoError(t, err)
	last := r.patch(a, c.ID, map[string]any{"status": "DISABLED", "version": 1})
	require.Equal(t, http.StatusConflict, last.code, string(last.body))
	require.Equal(t, "CONFLICT", last.errCode())
	require.Equal(t, "last_admin", last.details()["reason"])
	require.Equal(t, "ACTIVE", r.scalar(`select status::text from users where id = $1`, c.ID))
	// còn một ADMIN ACTIVE khác ⇒ khoá được
	d := r.addUser(uniq("admin.d"), store.UserRoleADMIN, store.UserStatusACTIVE)
	require.Equal(t, http.StatusOK, r.patch(a, c.ID, map[string]any{"status": "DISABLED", "version": 1}).code)
	_ = d
}

func TestCannotTouchAdminOrStudentRole(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	other := r.addUser(uniq("admin.khac"), store.UserRoleADMIN, store.UserStatusACTIVE)
	sv := r.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	for _, target := range []store.User{other, sv} {
		for _, role := range []string{"TEACHER", "TA", "ADMIN"} {
			res := r.patch(a, target.ID, map[string]any{"role": role, "version": 1})
			require.Equal(t, http.StatusUnprocessableEntity, res.code, "%s → %s", target.Role, role)
		}
		require.Equal(t, string(target.Role), r.scalar(`select role::text from users where id = $1`, target.ID))
	}
	// khoá sinh viên thì được (không đổi vai)
	require.Equal(t, http.StatusOK, r.patch(a, sv.ID, map[string]any{"status": "DISABLED", "version": 1}).code)
	require.Equal(t, http.StatusNotFound, r.patch(a, "00000000-0000-7000-8000-000000000001", map[string]any{"status": "DISABLED", "version": 1}).code)
}

// AC7.
func TestAdminUsersRBACMatrix(t *testing.T) {
	r := newRig(t)
	_, adminS := r.admin()
	target := r.addUser(uniq("muc.tieu"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	inv := r.invite(adminS, uniq("moi"), "Được Mời", "TA").json()["id"].(string)
	roles := map[string]store.UserRole{"STUDENT": store.UserRoleSTUDENT, "TA": store.UserRoleTA, "TEACHER": store.UserRoleTEACHER, "ADMIN": store.UserRoleADMIN}
	type op struct {
		name, method, path string
		body               any
	}
	ops := []op{
		{"list", http.MethodGet, "/admin/users", nil},
		{"invite", http.MethodPost, "/admin/users", map[string]string{"email": "", "full_name": "X", "role": "TA"}},
		{"patch", http.MethodPatch, "/admin/users/" + target.ID.String(), map[string]any{"full_name": "Đổi", "version": 1}},
		{"resend", http.MethodPost, "/admin/users/" + inv + "/resend-invite", nil},
	}
	cases := 0
	for name, role := range roles {
		u := r.addUser(uniq("rbac."+strings.ToLower(name)), role, store.UserStatusACTIVE)
		s := r.mustLogin(u.Email)
		for _, o := range ops {
			res := r.do(req{method: o.method, path: o.path, body: o.body, bearer: s.access, hdr: idem()})
			if role == store.UserRoleADMIN {
				require.NotEqual(t, http.StatusForbidden, res.code, "%s %s", name, o.name)
				require.NotEqual(t, http.StatusUnauthorized, res.code, "%s %s", name, o.name)
			} else {
				require.Equal(t, http.StatusForbidden, res.code, "%s %s (kể cả GET)", name, o.name)
				require.Equal(t, "FORBIDDEN", res.errCode())
				require.Equal(t, "role", res.details()["reason"])
			}
			cases++
		}
	}
	for _, o := range ops { // không JWT
		res := r.do(req{method: o.method, path: o.path, body: o.body, hdr: idem()})
		require.Equal(t, http.StatusUnauthorized, res.code, o.name)
		require.Equal(t, "UNAUTHENTICATED", res.errCode())
		cases++
	}
	require.GreaterOrEqual(t, cases, 20)
	// vai trong JWT thắng DB: DB nói ADMIN nhưng token mang vai TEACHER ⇒ 403
	t2 := r.addUser(uniq("claim"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	s := r.mustLogin(t2.Email)
	_, err := r.pool.Exec(t.Context(), `update users set role = 'ADMIN' where id = $1`, t2.ID)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, r.do(req{method: http.MethodGet, path: "/admin/users", bearer: s.access}).code)
}
