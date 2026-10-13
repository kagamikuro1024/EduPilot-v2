package activity_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

// US-P5-03 AC9: đọc thời gian học — chỉ đúng người.

func (r *rig) seed(course, uid any, daysAgo int, area string, secs int) {
	r.exec(`insert into study_time_daily (course_id, student_id, date, area, active_seconds) values ($1, $2, (now() at time zone 'Asia/Ho_Chi_Minh')::date - $3::int, $4, $5)
		on conflict (course_id, student_id, date, area) do update set active_seconds = excluded.active_seconds`, course, uid, daysAgo, area, secs)
}

func TestMeStudyTimeOwnOnly(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	a, b := w.student(true), w.student(true)
	r.clk.Set(time.Now().UTC())
	r.seed(w.course, a.u.ID, 0, "CHAT", 190)
	r.seed(w.course, a.u.ID, 0, "THREADS", 130)
	r.seed(w.course, b.u.ID, 0, "CHAT", 3000)
	res := r.call(http.MethodGet, "/courses/"+w.course.String()+"/me/study-time", a.s, nil)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	j := res.json()
	require.EqualValues(t, 5, j["this_week_minutes"], "floor((190 + 130) / 60)")
	weeks := j["weeks"].([]any)
	require.Len(t, weeks, 1)
	wk := weeks[0].(map[string]any)
	require.EqualValues(t, 5, wk["minutes"])
	require.Equal(t, map[string]any{"CHAT": float64(3), "THREADS": float64(2)}, wk["by_area"], "mỗi khu vực làm tròn xuống phút")
	require.Equal(t, map[string]any{"enabled": true, "locked": false}, j["tracking"])
	require.NotContains(t, j, "tracked")
	// không có tham số student: query ?student= / ?uid= bị bỏ qua, vẫn là của chính mình
	res = r.call(http.MethodGet, "/courses/"+w.course.String()+"/me/study-time?student="+b.u.ID.String(), a.s, nil)
	require.EqualValues(t, 5, res.json()["this_week_minutes"])
	// chưa có dữ liệu → mảng rỗng, không lỗi
	c := w.student(true)
	res = r.call(http.MethodGet, "/courses/"+w.course.String()+"/me/study-time", c.s, nil)
	require.Equal(t, http.StatusOK, res.code)
	require.Equal(t, []any{}, res.json()["weeks"])
	require.EqualValues(t, 0, res.json()["this_week_minutes"])
}

func TestStudyTimeReadRBAC(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	r.seed(w.course, s.u.ID, 0, "CHAT", 600)
	own := "/courses/" + w.course.String() + "/students/" + s.u.ID.String() + "/study-time"
	for name, sess := range map[string]session{"giảng viên": w.T, "trợ giảng": w.A} {
		res := r.call(http.MethodGet, own, sess, nil)
		require.Equal(t, http.StatusOK, res.code, name)
		require.EqualValues(t, 10, res.json()["this_week_minutes"], name)
		require.Equal(t, true, res.json()["tracked"], name)
		require.NotContains(t, res.json(), "tracking", name)
	}
	require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, own, s.s, nil).code, "sinh viên kể cả uid của mình")
	require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, own, w.ADM, nil).code, "admin")
	other := w.student(true)
	require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, own, other.s, nil).code)
	// sinh viên ngoài lớp / không ACTIVE
	outU := r.addUser(uniq("out"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	out := r.mustLogin(outU.Email)
	require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, "/courses/"+w.course.String()+"/me/study-time", out, nil).code, "không ACTIVE ở lớp")
	require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, "/courses/"+w.course.String()+"/me/study-time", w.T, nil).code, "giảng viên không có 'của mình'")
	// uid không thuộc lớp → 404
	require.Equal(t, http.StatusNotFound, r.call(http.MethodGet, "/courses/"+w.course.String()+"/students/"+outU.ID.String()+"/study-time", w.T, nil).code)
	require.Equal(t, http.StatusUnauthorized, r.do(req{method: http.MethodGet, path: own}).code)
}

func TestStudyTimeWeeksRange(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	base := "/courses/" + w.course.String() + "/me/study-time"
	for _, q := range []string{"0", "21", "-1", "abc", "1.5"} {
		require.Equal(t, http.StatusUnprocessableEntity, r.call(http.MethodGet, base+"?weeks="+q, s.s, nil).code, q)
	}
	for _, q := range []string{"1", "8", "20", ""} {
		require.Equal(t, http.StatusOK, r.call(http.MethodGet, base+"?weeks="+q, s.s, nil).code, q)
	}
	// weeks giới hạn cửa sổ: dữ liệu cách 3 tuần không có ở weeks=1
	r.clk.Set(time.Now().UTC())
	r.seed(w.course, s.u.ID, 21, "CHAT", 120)
	res := r.call(http.MethodGet, base+"?weeks=1", s.s, nil)
	require.Equal(t, []any{}, res.json()["weeks"])
	res = r.call(http.MethodGet, base+"?weeks=8", s.s, nil)
	require.Len(t, res.json()["weeks"], 1)
}

// TestMeStudyTimeTrackingFlags — tracking {enabled, locked} theo từng lớp (AC13): X cho tắt + đã tắt, Y không cho tắt.
func TestMeStudyTimeTrackingFlags(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	y := r.courseWith(w.adm.ID, fmt.Sprintf("Y%06d", time.Now().UnixNano()%1000000), false)
	s := w.student(true)
	r.enroll(y, s.u.ID, "STUDENT", "ACTIVE")
	require.Equal(t, http.StatusOK, r.call(http.MethodPut, "/me/settings", s.s, map[string]any{"track_study_time": false, "version": 1}).code)
	x := r.call(http.MethodGet, "/courses/"+w.course.String()+"/me/study-time", s.s, nil).json()["tracking"]
	yy := r.call(http.MethodGet, "/courses/"+y.String()+"/me/study-time", s.s, nil).json()["tracking"]
	require.Equal(t, map[string]any{"enabled": false, "locked": false}, x)
	require.Equal(t, map[string]any{"enabled": true, "locked": true}, yy)
	// GET /me/settings: study_time.courses mỗi lớp ACTIVE một dòng
	set := r.call(http.MethodGet, "/me/settings", s.s, nil).json()
	require.Equal(t, false, set["track_study_time"])
	rows := set["study_time"].(map[string]any)["courses"].([]any)
	require.Len(t, rows, 2)
	byID := map[string]bool{}
	for _, c := range rows {
		m := c.(map[string]any)
		byID[m["course_id"].(string)] = m["locked"].(bool)
	}
	require.Equal(t, map[string]bool{w.course.String(): false, y.String(): true}, byID)
}

// TestMeCoursesTrackingNotice — hộp thoại minh bạch đi cùng `GET /me/courses` (không thêm yêu cầu): chưa ack → false; có lớp cho tắt → opt_out_available.
func TestMeCoursesTrackingNotice(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(false)
	get := func(sess session) map[string]any {
		res := r.call(http.MethodGet, "/me/courses", sess, nil)
		require.Equal(t, http.StatusOK, res.code)
		return res.json()
	}
	j := get(s.s)
	require.Equal(t, false, j["tracking_notice_acked"])
	require.Equal(t, true, j["study_time_opt_out_available"])
	r.ack(s)
	require.Equal(t, true, get(s.s)["tracking_notice_acked"])
	strict := r.world(false)
	s2 := strict.student(false)
	require.Equal(t, false, get(s2.s)["study_time_opt_out_available"], "lớp không cho tắt: không có dòng 'Bạn có thể tắt'")
	// chưa có lớp nào: không có gì để đo, không hộp thoại
	lone := r.addUser(uniq("lone"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	require.Equal(t, true, get(r.mustLogin(lone.Email))["tracking_notice_acked"])
}
