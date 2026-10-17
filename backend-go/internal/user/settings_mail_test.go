package user_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

func (r *rig) putSettings(s session, body map[string]any) resp {
	return r.do(req{method: http.MethodPut, path: "/me/settings", bearer: s.access, body: body})
}

// US-P4-04 AC7: công tắc thư theo vai — TA / TEACHER: "Mail báo ticket mới" (mặc định tắt); STUDENT: "Mail khi giảng viên trả lời" (mặc định bật).
func TestSettingsMailFlagsByRole(t *testing.T) {
	r := newRig(t)
	for _, role := range []store.UserRole{store.UserRoleTA, store.UserRoleTEACHER} {
		s := r.mustLogin(r.addUser(uniq("mf"), role, store.UserStatusACTIVE).Email)
		require.Equal(t, false, r.getSettings(s).json()["notify_ticket_by_mail"], string(role)+" mặc định tắt")
		res := r.putSettings(s, map[string]any{"notify_ticket_by_mail": true, "version": 1})
		require.Equal(t, http.StatusOK, res.code, string(res.body))
		require.Equal(t, true, r.getSettings(s).json()["notify_ticket_by_mail"], string(role))
	}
	s := r.mustLogin(r.addUser(uniq("mf"), store.UserRoleSTUDENT, store.UserStatusACTIVE).Email)
	require.Equal(t, true, r.getSettings(s).json()["notify_answer_by_mail"], "sinh viên mặc định bật")
	require.Equal(t, http.StatusOK, r.putSettings(s, map[string]any{"notify_answer_by_mail": false, "version": 1}).code)
	require.Equal(t, false, r.getSettings(s).json()["notify_answer_by_mail"])
}

func TestSettingsVersionConflict(t *testing.T) {
	r := newRig(t)
	s := r.mustLogin(r.addUser(uniq("vc"), store.UserRoleTEACHER, store.UserStatusACTIVE).Email)
	require.Equal(t, http.StatusOK, r.putSettings(s, map[string]any{"notify_ticket_by_mail": true, "version": 1}).code)
	res := r.putSettings(s, map[string]any{"notify_ticket_by_mail": false, "version": 1})
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "VERSION_CONFLICT", res.errCode())
	require.Equal(t, true, r.getSettings(s).json()["notify_ticket_by_mail"], "bản cũ không ghi đè")
}

// Sinh viên gửi notify_ticket_by_mail vẫn 200 (lưu, giữ hợp đồng cũ) nhưng không có tác dụng: sinh viên không bao giờ là người nhận thư ticket mới.
func TestSettingsStudentTicketFlagStoredNoEffect(t *testing.T) {
	r := newRig(t)
	u := r.addUser(uniq("sf"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(u.Email)
	require.Equal(t, http.StatusOK, r.putSettings(s, map[string]any{"notify_ticket_by_mail": true, "version": 1}).code)
	require.Equal(t, true, r.getSettings(s).json()["notify_ticket_by_mail"], "đã lưu")
	require.Equal(t, "0", r.scalar(`select count(*)::text from enrollments where user_id = $1 and role_in_course in ('TEACHER','TA')`, u.ID), "không phải nhân viên nên không bao giờ thành người nhận")
}
