package course_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func (r *rig) generate(s session, courseID string, body map[string]any, hdr map[string]string) resp {
	return r.post(s, "/courses/"+courseID+"/sessions/generate", body, hdr)
}

// thứ Năm hằng tuần 09:00–11:30 P.302, 15 buổi: 2026-08-27 → 2026-12-03.
func weekly(extra map[string]any) map[string]any {
	b := map[string]any{"weekdays": []int{4}, "start_time": "09:00", "end_time": "11:30", "room": "P.302", "from": "2026-08-27", "to": "2026-12-03"}
	for k, v := range extra {
		b[k] = v
	}
	return b
}

func TestSessionsGenerate(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	res := r.generate(k.a.G, k.id, weekly(nil), idemKey())
	require.Equal(t, http.StatusCreated, res.code, string(res.body))
	j := res.json()
	require.Equal(t, []float64{15, 1, 15, 0}, []float64{j["created"].(float64), j["first_session_no"].(float64), j["last_session_no"].(float64), j["skipped"].(float64)})
	require.Equal(t, 15, r.count(`select count(*) from class_sessions where course_id = $1`, k.id))
	require.Equal(t, "1|15|15", r.scalar(`select min(session_no)||'|'||max(session_no)||'|'||count(distinct session_no) from class_sessions where course_id = $1`, k.id), "session_no liên tục")
	// Giờ theo Asia/Ho_Chi_Minh: 09:00 ICT = 02:00 UTC, kéo dài 2 giờ 30.
	require.Equal(t, "02:00|2.5|P.302", r.scalar(`select to_char(starts_at at time zone 'UTC', 'HH24:MI')||'|'||(extract(epoch from ends_at - starts_at)/3600)::numeric(4,1)||'|'||room from class_sessions where course_id = $1 and session_no = 1`, k.id))
	require.Equal(t, "4", r.scalar(`select distinct extract(isodow from starts_at at time zone 'Asia/Ho_Chi_Minh')::text from class_sessions where course_id = $1`, k.id), "mọi buổi là thứ Năm")

	list := r.get(k.a.S, "/courses/"+k.id+"/sessions")
	require.Equal(t, http.StatusForbidden, list.code, "sinh viên ngoài lớp")
	u, su := r.student("")
	r.enroll(k.course, u.ID, "STUDENT", "ACTIVE")
	page := r.get(su, "/courses/"+k.id+"/sessions?limit=10")
	require.Equal(t, http.StatusOK, page.code, string(page.body))
	items := page.json()["items"].([]any)
	require.Len(t, items, 10)
	require.NotEmpty(t, page.json()["next_cursor"])
	require.Equal(t, float64(1), items[0].(map[string]any)["session_no"])
	next := r.get(su, "/courses/"+k.id+"/sessions?limit=10&cursor="+page.json()["next_cursor"].(string)).json()
	require.Len(t, next["items"], 5)
	require.Nil(t, next["next_cursor"])
	require.Equal(t, float64(11), next["items"].([]any)[0].(map[string]any)["session_no"], "tiếp nối đúng, không lặp")
}

func TestSessionsGenerateNoDuplicate(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.generate(k.a.G, k.id, weekly(nil), idemKey()).code)
	again := r.generate(k.a.G, k.id, weekly(nil), idemKey())
	require.Equal(t, http.StatusCreated, again.code)
	require.Equal(t, float64(0), again.json()["created"])
	require.Equal(t, float64(15), again.json()["skipped"])
	require.Equal(t, 15, r.count(`select count(*) from class_sessions where course_id = $1`, k.id))
	// Mở rộng khoảng ngày: chỉ tạo phần mới, số buổi nối tiếp 16, 17.
	more := r.generate(k.a.G, k.id, weekly(map[string]any{"to": "2026-12-17"}), idemKey())
	require.Equal(t, []float64{2, 16, 17, 15}, []float64{more.json()["created"].(float64), more.json()["first_session_no"].(float64), more.json()["last_session_no"].(float64), more.json()["skipped"].(float64)})
	// Cùng Idempotency-Key ⇒ phát lại đúng phản hồi cũ.
	key := idemKey()
	a := r.generate(k.a.G, k.id, weekly(map[string]any{"from": "2027-01-07", "to": "2027-01-14"}), key)
	b := r.generate(k.a.G, k.id, weekly(map[string]any{"from": "2027-01-07", "to": "2027-01-14"}), key)
	require.JSONEq(t, string(a.body), string(b.body))
	require.Equal(t, 19, r.count(`select count(*) from class_sessions where course_id = $1`, k.id))
}

func TestSessionsGenerateLimit60(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	daily := func(to string) map[string]any {
		return map[string]any{"weekdays": []int{1, 2, 3, 4, 5, 6, 7}, "start_time": "08:00", "end_time": "09:00", "from": "2026-09-01", "to": to}
	}
	res := r.generate(k.a.G, k.id, daily("2026-09-30"), idemKey()) // 30 ngày
	require.Equal(t, http.StatusCreated, res.code)
	again := r.generate(k.a.G, k.id, daily("2026-10-30"), idemKey()).json() // đúng 60 ngày: 30 mới + 30 đã có
	require.Equal(t, []float64{30, 30}, []float64{again["created"].(float64), again["skipped"].(float64)})
	over := r.generate(k.a.G, k.id, daily("2026-11-30"), idemKey()) // 91 ngày
	require.Equal(t, http.StatusUnprocessableEntity, over.code, string(over.body))
	require.Contains(t, string(over.body), "TOO_MANY_SESSIONS")
	require.Equal(t, 60, r.count(`select count(*) from class_sessions where course_id = $1`, k.id), "vượt 60 ⇒ không ghi gì")
	// Đúng 60 buổi (từ 1/9 đến 30/10) thì được; 61 thì không.
	r2 := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.generate(r2.a.G, r2.id, daily("2026-10-30"), idemKey()).code)
	require.Equal(t, http.StatusUnprocessableEntity, r.generate(r2.a.G, r2.id, daily("2026-10-31"), idemKey()).code)
}

func TestSessionsGenerateExcludeDates(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	res := r.generate(k.a.G, k.id, weekly(map[string]any{"exclude_dates": []string{"2026-09-03", "2026-10-01", "2026-12-25"}}), idemKey())
	require.Equal(t, http.StatusCreated, res.code, string(res.body))
	require.Equal(t, float64(13), res.json()["created"])
	require.Equal(t, 0, r.count(`select count(*) from class_sessions where course_id = $1 and (starts_at at time zone 'Asia/Ho_Chi_Minh')::date in ('2026-09-03', '2026-10-01')`, k.id))
	require.Equal(t, "13", r.scalar(`select max(session_no)::text from class_sessions where course_id = $1`, k.id), "số buổi vẫn liên tục sau khi bỏ ngày")
	// Chọn số buổi đầu: first_session_no = 10 ⇒ 10, 11…; trùng số đã có ⇒ 422.
	k2 := r.klass(nil)
	one := r.generate(k2.a.G, k2.id, weekly(map[string]any{"to": "2026-09-10", "first_session_no": 10}), idemKey())
	require.Equal(t, []float64{10, 12}, []float64{one.json()["first_session_no"].(float64), one.json()["last_session_no"].(float64)})
	clash := r.generate(k2.a.G, k2.id, weekly(map[string]any{"from": "2026-10-01", "to": "2026-10-08", "first_session_no": 11}), idemKey())
	require.Equal(t, http.StatusUnprocessableEntity, clash.code)
	require.Contains(t, string(clash.body), "SESSION_NO_TAKEN")
}

func TestSessionsGenerateValidation(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	for name, body := range map[string]map[string]any{
		"thiếu thứ":          weekly(map[string]any{"weekdays": []int{}}),
		"thứ 8":              weekly(map[string]any{"weekdays": []int{8}}),
		"giờ sai":            weekly(map[string]any{"start_time": "9:00"}),
		"kết thúc trước":     weekly(map[string]any{"end_time": "08:00"}),
		"ngày sai":           weekly(map[string]any{"from": "27/08/2026"}),
		"to trước from":      weekly(map[string]any{"to": "2026-08-01"}),
		"khoảng quá dài":     weekly(map[string]any{"to": "2028-12-03"}),
		"loại trừ sai":       weekly(map[string]any{"exclude_dates": []string{"hôm qua"}}),
		"phòng quá dài":      weekly(map[string]any{"room": "P.0123456789012345678901234567890123456789"}),
		"first_session_no 0": weekly(map[string]any{"first_session_no": 0}),
	} {
		res := r.generate(k.a.G, k.id, body, idemKey())
		require.Equal(t, http.StatusUnprocessableEntity, res.code, name+": "+string(res.body))
	}
	require.Equal(t, 0, r.count(`select count(*) from class_sessions where course_id = $1`, k.id))
}

func TestSessionsGenerateRBAC(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	_, member := r.student("")
	for name, tc := range map[string]struct {
		s    session
		code int
	}{
		"giảng viên": {k.a.G, 201}, "trợ giảng": {k.a.T, 201}, "ADMIN": {k.a.A, 403}, "sinh viên ngoài lớp": {k.a.S, 403},
		"sinh viên khác": {member, 403}, "giảng viên lớp khác": {k.a.G2, 403}, "không JWT": {session{}, 401},
	} {
		res := r.generate(tc.s, k.id, weekly(map[string]any{"from": "2026-09-01", "to": "2026-09-08"}), idemKey())
		require.Equal(t, tc.code, res.code, name+": "+string(res.body))
	}
	require.Equal(t, http.StatusUnprocessableEntity, r.generate(k.a.G, k.id, weekly(nil), nil).code, "thiếu Idempotency-Key")
	require.Equal(t, http.StatusForbidden, r.generate(k.a.A, k.id, weekly(nil), nil).code, "403 đến trước 422")
	require.Equal(t, http.StatusNotFound, r.generate(k.a.G, "khong-phai-uuid", weekly(nil), idemKey()).code)
	require.Equal(t, http.StatusOK, r.post(k.a.A, "/admin/courses/"+k.id+"/archive", nil, nil).code)
	res := r.generate(k.a.G, k.id, weekly(map[string]any{"from": "2026-12-01", "to": "2026-12-08"}), idemKey())
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "COURSE_ARCHIVED", res.errCode())
	require.Equal(t, http.StatusForbidden, r.get(k.a.A, "/courses/"+k.id+"/sessions").code, "Admin không đọc lịch lớp")
}
