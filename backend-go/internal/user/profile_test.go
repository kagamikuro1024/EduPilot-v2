package user_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

func (r *rig) getProfile(s session) resp {
	return r.do(req{method: http.MethodGet, path: "/me/profile", bearer: s.access})
}

func (r *rig) putProfile(s session, body map[string]any, hdr map[string]string) resp {
	return r.do(req{method: http.MethodPut, path: "/me/profile", bearer: s.access, body: body, hdr: hdr})
}

// AC9.
func TestProfileGetPut(t *testing.T) {
	r := newRig(t)
	u := r.addUser(uniq("pf"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(u.Email)
	got := r.getProfile(s)
	require.Equal(t, http.StatusOK, got.code, string(got.body))
	m := got.json()
	require.Equal(t, u.Email, m["email"])
	require.Equal(t, "STUDENT", m["role"])
	require.Equal(t, false, m["email_verified"])
	require.EqualValues(t, 1, m["version"])
	require.Equal(t, `W/"v1"`, got.hdr.Get("ETag"))
	require.Len(t, m, 7)

	put := r.putProfile(s, map[string]any{"full_name": "  Trần Thu Uyên  ", "student_code": "b20dccn001", "version": 1}, nil)
	require.Equal(t, http.StatusOK, put.code, string(put.body))
	require.Equal(t, "Trần Thu Uyên", put.json()["full_name"])
	require.Equal(t, "B20DCCN001", put.json()["student_code"], "chuẩn hoá chữ hoa")
	require.EqualValues(t, 2, put.json()["version"])
	// xoá MSSV bằng chuỗi rỗng; chỉ sửa tên không đụng MSSV
	require.Equal(t, "B20DCCN001", r.putProfile(s, map[string]any{"full_name": "Tên Khác", "version": 2}, nil).json()["student_code"])
	cleared := r.putProfile(s, map[string]any{"student_code": "", "version": 3}, nil)
	require.Equal(t, http.StatusOK, cleared.code)
	require.Equal(t, "", cleared.json()["student_code"])
	// ràng buộc đầu vào
	bad := r.putProfile(s, map[string]any{"student_code": "ab", "version": 4}, nil)
	require.Equal(t, http.StatusUnprocessableEntity, bad.code)
	require.Equal(t, "STUDENT_CODE_FORMAT", firstCode(bad))
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'a'
	}
	require.Equal(t, http.StatusUnprocessableEntity, r.putProfile(s, map[string]any{"full_name": string(long), "version": 4}, nil).code)
	require.Equal(t, http.StatusUnprocessableEntity, r.putProfile(s, map[string]any{"full_name": "   ", "version": 4}, nil).code)
	// giảng viên không có MSSV
	gv := r.mustLogin(r.addUser(uniq("gv"), store.UserRoleTEACHER, store.UserStatusACTIVE).Email)
	nv := r.putProfile(gv, map[string]any{"student_code": "B20DCCN001", "version": 1}, nil)
	require.Equal(t, http.StatusUnprocessableEntity, nv.code)
	require.Equal(t, http.StatusNotModified, r.do(req{method: http.MethodGet, path: "/me/profile", bearer: gv.access, hdr: map[string]string{"If-None-Match": `W/"v1"`}}).code)
}

func firstCode(x resp) string {
	d, _ := x.json()["details"].([]any)
	if len(d) == 0 {
		return ""
	}
	c, _ := d[0].(map[string]any)["code"].(string)
	return c
}

func TestProfileReadOnlyFields(t *testing.T) {
	r := newRig(t)
	u := r.addUser(uniq("ro"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(u.Email)
	for _, body := range []map[string]any{
		{"role": "ADMIN", "version": 1}, {"email": "khac@example.test", "version": 1}, {"status": "ACTIVE", "version": 1},
		{"full_name": "Tên", "role": "TEACHER", "version": 1}, {"id": uuid.NewString(), "version": 1}, {"email_verified": true, "version": 1},
	} {
		res := r.putProfile(s, body, nil)
		require.Equal(t, http.StatusUnprocessableEntity, res.code, "%v", body)
		require.Equal(t, "VALIDATION_FAILED", res.errCode())
	}
	require.Equal(t, "STUDENT", r.scalar(`select role::text from users where id = $1`, u.ID))
	require.Equal(t, u.Email, r.scalar(`select email from users where id = $1`, u.ID))
	require.Equal(t, "1", r.scalar(`select version::text from users where id = $1`, u.ID), "không đổi gì khi bị từ chối")
	require.Equal(t, http.StatusUnprocessableEntity, r.putProfile(s, map[string]any{"full_name": "Tên"}, nil).code, "thiếu version")
}

func TestProfileVersionConflict(t *testing.T) {
	r := newRig(t)
	u := r.addUser(uniq("vc"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(u.Email)
	require.Equal(t, http.StatusOK, r.putProfile(s, map[string]any{"full_name": "Lần Một", "version": 1}, nil).code)
	res := r.putProfile(s, map[string]any{"full_name": "Lần Hai", "version": 1}, nil)
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "VERSION_CONFLICT", res.errCode())
	require.EqualValues(t, 2, res.details()["current_version"])
	require.Equal(t, `W/"v2"`, res.hdr.Get("ETag"))
	// If-Match thay cho version trong thân
	ok := r.putProfile(s, map[string]any{"full_name": "Qua If-Match"}, map[string]string{"If-Match": `W/"v2"`})
	require.Equal(t, http.StatusOK, ok.code, string(ok.body))
}

// AC9: đổi MSSV không bao giờ đổi ảnh chụp ở lớp (chống đổi MSSV để chui vào dữ liệu người khác).
func TestProfileStudentCodeDoesNotChangeSnapshot(t *testing.T) {
	r := newRig(t)
	adm := r.addUser(uniq("adm"), store.UserRoleADMIN, store.UserStatusACTIVE)
	u := r.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	var course uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by)
		values ('INT1006', 'SNAP-'||left(gen_random_uuid()::text, 6), 'L', '2026-2027-HK1', 'AN7K2MQ', $1) on conflict do nothing returning id`, adm.ID).Scan(&course))
	_, err := r.pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', 'ROSTER', 'B20DCCN001')`, course, u.ID)
	require.NoError(t, err)
	s := r.mustLogin(u.Email)
	require.Equal(t, http.StatusOK, r.putProfile(s, map[string]any{"student_code": "B20DCCN999", "version": 1}, nil).code)
	require.Equal(t, "B20DCCN999", r.scalar(`select student_code from users where id = $1`, u.ID))
	require.Equal(t, "B20DCCN001", r.scalar(`select student_code_snapshot from enrollments where user_id = $1`, u.ID), "ảnh chụp bất biến")
	// và MSSV khai báo không nối tài khoản vào lớp nào
	other := r.addUser(uniq("kk"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	so := r.mustLogin(other.Email)
	require.Equal(t, http.StatusOK, r.putProfile(so, map[string]any{"student_code": "B20DCCN001", "version": 1}, nil).code)
	require.Equal(t, "0", r.scalar(`select count(*)::text from enrollments where user_id = $1`, other.ID))
}

func TestProfileAudit(t *testing.T) {
	r := newRig(t)
	u := r.addUser(uniq("au"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(u.Email)
	require.Equal(t, http.StatusOK, r.putProfile(s, map[string]any{"full_name": "Có Audit", "student_code": "B20DCCN555", "version": 1}, nil).code)
	require.Equal(t, "1", r.scalar(`select count(*)::text from audit_log where entity = 'user' and entity_id = $1 and action = 'profile_updated' and actor_id = $2`, u.ID.String(), u.ID))
	require.Equal(t, "0", r.scalar(`select count(*)::text from audit_log where coalesce(after::text,'') like '%B20DCCN555%' or coalesce(after::text,'') ilike '%password%' or coalesce(after::text,'') like '%$2%'`), "audit không chứa MSSV, mật khẩu")
}

// AC10.
func (r *rig) getSettings(s session) resp {
	return r.do(req{method: http.MethodGet, path: "/me/settings", bearer: s.access})
}

func TestSettingsDefaultsLazyCreate(t *testing.T) {
	r := newRig(t)
	u := r.addUser(uniq("st"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(u.Email)
	require.Equal(t, "0", r.scalar(`select count(*)::text from user_settings where user_id = $1`, u.ID))
	res := r.getSettings(s)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, map[string]any{"notify_ticket_by_mail": true, "notify_answer_by_mail": true, "remind_deadline_by_mail": true, "reminders": map[string]any{"exam": true, "class_session": false, "other": true}, "version": float64(1)}, res.json())
	require.Equal(t, "1", r.scalar(`select count(*)::text from user_settings where user_id = $1`, u.ID))
	require.Equal(t, http.StatusOK, r.getSettings(s).code)
	require.Equal(t, "1", r.scalar(`select count(*)::text from user_settings where user_id = $1`, u.ID), "đọc lần hai không tạo thêm")
}

func TestSettingsPut(t *testing.T) {
	r := newRig(t)
	u := r.addUser(uniq("sp"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	s := r.mustLogin(u.Email)
	put := func(body map[string]any) resp {
		return r.do(req{method: http.MethodPut, path: "/me/settings", bearer: s.access, body: body})
	}
	res := put(map[string]any{"notify_ticket_by_mail": false, "version": 1})
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, false, res.json()["notify_ticket_by_mail"])
	require.Equal(t, true, res.json()["notify_answer_by_mail"], "khoá không gửi ⇒ giữ nguyên")
	require.EqualValues(t, 2, res.json()["version"])
	conflict := put(map[string]any{"remind_deadline_by_mail": false, "version": 1})
	require.Equal(t, http.StatusConflict, conflict.code)
	require.Equal(t, "VERSION_CONFLICT", conflict.errCode())
	require.EqualValues(t, 2, conflict.details()["current_version"])
	require.Equal(t, http.StatusOK, put(map[string]any{"notify_answer_by_mail": false, "remind_deadline_by_mail": false, "version": 2}).code)
	g := r.getSettings(s).json()
	require.Equal(t, false, g["notify_ticket_by_mail"])
	require.Equal(t, false, g["notify_answer_by_mail"])
	require.Equal(t, false, g["remind_deadline_by_mail"])
}

// TestSettingsReminders — US-P8-03 AC12: preferences.reminders gộp từng khoá, mặc định bật / tắt / bật, khoá lạ → 422.
func TestSettingsReminders(t *testing.T) {
	r := newRig(t)
	u := r.addUser(uniq("rm"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(u.Email)
	put := func(body map[string]any) resp {
		return r.do(req{method: http.MethodPut, path: "/me/settings", bearer: s.access, body: body})
	}
	res := put(map[string]any{"reminders": map[string]any{"exam": false}, "version": 1})
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, map[string]any{"exam": false, "class_session": false, "other": true}, res.json()["reminders"])
	res = put(map[string]any{"reminders": map[string]any{"class_session": true}, "version": 2})
	require.Equal(t, map[string]any{"exam": false, "class_session": true, "other": true}, res.json()["reminders"], "khoá không gửi ⇒ giữ nguyên")
	res = put(map[string]any{"notify_ticket_by_mail": false, "version": 3})
	require.Equal(t, map[string]any{"exam": false, "class_session": true, "other": true}, res.json()["reminders"], "PUT không có reminders không đụng tới")
	for _, body := range []map[string]any{{"reminders": map[string]any{"homework": true}, "version": 4}, {"reminders": map[string]any{"exam": "yes"}, "version": 4}, {"reminders": "all", "version": 4}} {
		require.Equal(t, http.StatusUnprocessableEntity, put(body).code, "%v", body)
	}
	require.Equal(t, map[string]any{"exam": false, "class_session": true, "other": true}, r.getSettings(s).json()["reminders"])
}

func TestSettingsUnknownKey422(t *testing.T) {
	r := newRig(t)
	s := r.mustLogin(r.addUser(uniq("uk"), store.UserRoleSTUDENT, store.UserStatusACTIVE).Email)
	for _, body := range []map[string]any{{"theme": "dark", "version": 1}, {"preferences": map[string]any{}, "version": 1}, {"user_id": uuid.NewString(), "version": 1}, {"notify_ticket_by_mail": "yes", "version": 1}} {
		res := r.do(req{method: http.MethodPut, path: "/me/settings", bearer: s.access, body: body})
		require.Equal(t, http.StatusUnprocessableEntity, res.code, "%v", body)
	}
}

func TestSettingsOwnerOnly(t *testing.T) {
	r := newRig(t)
	a := r.addUser(uniq("a"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	b := r.addUser(uniq("b"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	sa, sb := r.mustLogin(a.Email), r.mustLogin(b.Email)
	require.Equal(t, http.StatusOK, r.do(req{method: http.MethodPut, path: "/me/settings", bearer: sa.access, body: map[string]any{"notify_ticket_by_mail": false, "version": 1}}).code)
	got := r.do(req{method: http.MethodGet, path: "/me/settings?user_id=" + a.ID.String(), bearer: sb.access})
	require.Equal(t, http.StatusOK, got.code)
	require.Equal(t, true, got.json()["notify_ticket_by_mail"], "B không thấy tuỳ chọn của A dù truyền user_id")
	require.Equal(t, http.StatusUnauthorized, r.do(req{method: http.MethodGet, path: "/me/settings"}).code)
	require.Equal(t, http.StatusUnauthorized, r.do(req{method: http.MethodGet, path: "/me/profile"}).code)
}

func TestProfileSettingsAllRoles(t *testing.T) {
	r := newRig(t)
	for _, role := range []store.UserRole{store.UserRoleSTUDENT, store.UserRoleTA, store.UserRoleTEACHER, store.UserRoleADMIN} {
		s := r.mustLogin(r.addUser(uniq("rl"), role, store.UserStatusACTIVE).Email)
		require.Equal(t, http.StatusOK, r.getProfile(s).code, string(role))
		require.Equal(t, http.StatusOK, r.getSettings(s).code, string(role))
	}
}
