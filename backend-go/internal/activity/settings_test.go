package activity_test

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/store"
)

// US-P5-03 AC11 / AC12 (Q1, Q25): cờ đo của sinh viên (toàn tài khoản) và cờ của lớp (khoá khi có sinh viên đầu tiên).

func TestSettingsTrackStudyTimeVersion(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	put := func(v int, on bool) resp {
		return r.call(http.MethodPut, "/me/settings", s.s, map[string]any{"track_study_time": on, "version": v})
	}
	require.Equal(t, http.StatusOK, put(1, false).code)
	res := put(1, true) // version cũ
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "VERSION_CONFLICT", res.errCode())
	require.Equal(t, false, r.call(http.MethodGet, "/me/settings", s.s, nil).json()["track_study_time"], "409 không đổi gì")
	require.Equal(t, http.StatusOK, put(2, true).code)
	require.Equal(t, true, r.call(http.MethodGet, "/me/settings", s.s, nil).json()["track_study_time"])
	// vai khác vẫn lưu được (không tác dụng)
	require.Equal(t, http.StatusOK, r.call(http.MethodPut, "/me/settings", w.T, map[string]any{"track_study_time": false, "version": 1}).code)
}

func TestSettingsTrackStudyTimeOwnOnly(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	a, b := w.student(true), w.student(true)
	require.Equal(t, http.StatusOK, r.call(http.MethodPut, "/me/settings", a.s, map[string]any{"track_study_time": false, "version": 1}).code)
	require.Equal(t, true, r.call(http.MethodGet, "/me/settings", b.s, nil).json()["track_study_time"], "B không thấy giá trị của A")
	require.Equal(t, true, r.call(http.MethodGet, "/me/settings", w.ADM, nil).json()["track_study_time"], "Admin chỉ đọc của chính mình")
	require.Equal(t, http.StatusUnprocessableEntity, r.call(http.MethodPut, "/me/settings", a.s, map[string]any{"track_study_time": false, "version": 2, "user_id": b.u.ID}).code, "không có tham số danh tính")
	require.Equal(t, 1, r.count(`select count(*) from user_settings where track_study_time = false and user_id = $1`, a.u.ID))
	require.Zero(t, r.count(`select count(*) from user_settings where track_study_time = false and user_id = $1`, b.u.ID))
}

func (w world) putOptOut(sess session, allow bool, version int) resp {
	return w.r.call(http.MethodPut, "/courses/"+w.course.String()+"/settings", sess, map[string]any{"allow_study_time_opt_out": allow, "version": version})
}

func (w world) courseVersion() int {
	return w.r.count(`select version from courses where id = $1`, w.course)
}

func TestOptOutDefaultTrueOnNewCourse(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	good := map[string]any{"subject_code": "INT1006", "class_code": "N" + strings.ToUpper(uuid.NewString()[:6]), "name": "An ninh mạng", "semester": "2026-2027-HK1"}
	res := r.do(req{method: http.MethodPost, path: "/admin/courses", bearer: w.ADM.access, body: good, hdr: map[string]string{"Idempotency-Key": uuid.NewString()}})
	require.Equal(t, http.StatusCreated, res.code, string(res.body))
	id := res.json()["id"].(string)
	require.Equal(t, 1, r.count(`select count(*) from courses where id = $1 and allow_study_time_opt_out`, id), "lớp mới mặc định cho tắt")
	// Admin xem được chi tiết (MemberOrAdmin): cờ có mặt
	got := r.call(http.MethodGet, "/courses/"+id, w.ADM, nil)
	require.Equal(t, http.StatusOK, got.code)
	require.Equal(t, true, got.json()["allow_study_time_opt_out"])
}

func TestAdminCourseRoutesRejectOptOutField(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	bad := map[string]any{"subject_code": "INT1006", "class_code": "M" + strings.ToUpper(uuid.NewString()[:6]), "name": "An ninh mạng", "semester": "2026-2027-HK1", "allow_study_time_opt_out": false}
	res := r.do(req{method: http.MethodPost, path: "/admin/courses", bearer: w.ADM.access, body: bad, hdr: map[string]string{"Idempotency-Key": uuid.NewString()}})
	require.Equal(t, http.StatusUnprocessableEntity, res.code)
	require.Equal(t, "VALIDATION_FAILED", res.errCode())
	require.Equal(t, http.StatusUnprocessableEntity, r.call(http.MethodPut, "/admin/courses/"+w.course.String(), w.ADM, map[string]any{"allow_study_time_opt_out": false, "version": w.courseVersion()}).code)
	require.Equal(t, 1, r.count(`select count(*) from courses where id = $1 and allow_study_time_opt_out`, w.course))
}

func TestOptOutTeacherOnly(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	v := w.courseVersion()
	sv := w.student(true) // sinh viên ACTIVE làm khoá: dùng lớp khác cho ca "đổi được"
	for name, sess := range map[string]session{"trợ giảng": w.A, "sinh viên": sv.s, "admin": w.ADM} {
		require.Equal(t, http.StatusForbidden, w.putOptOut(sess, false, v).code, name)
	}
	w2 := r.world(true)
	v2 := w2.courseVersion()
	res := w2.putOptOut(w2.T, false, v2)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, false, res.json()["allow_study_time_opt_out"])
	require.Equal(t, 1, r.count(`select count(*) from courses where id = $1 and not allow_study_time_opt_out and version = $2`, w2.course, v2+1))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity_id = $1 and action = 'course_study_time_opt_out_updated' and (before->>'allow_study_time_opt_out') = 'true' and (after->>'allow_study_time_opt_out') = 'false'`, w2.course.String()))
	// sai version → 409 VERSION_CONFLICT
	res = w2.putOptOut(w2.T, true, v2)
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "VERSION_CONFLICT", res.errCode())
}

func TestOptOutLockedAfterFirstStudent409(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	pend := r.addUser(uniq("pend"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	r.enroll(w.course, pend.ID, "STUDENT", "PENDING") // PENDING đã đủ khoá
	v := w.courseVersion()
	res := w.putOptOut(w.T, false, v)
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "COURSE_HAS_STUDENTS", res.errCode())
	res = w.putOptOut(w.T, true, v) // đổi cùng giá trị cũ cũng 409
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "COURSE_HAS_STUDENTS", res.errCode())
	require.Equal(t, 1, r.count(`select count(*) from courses where id = $1 and allow_study_time_opt_out and version = $2`, w.course, v), "không đổi gì")
	require.Zero(t, r.count(`select count(*) from audit_log where entity_id = $1 and action = 'course_study_time_opt_out_updated'`, w.course.String()))
}

// TestOptOutLockRaceWithFirstEnrollment — PUT đua với ghi danh sinh viên đầu tiên: chỉ hai kết cục: PUT thắng (200) rồi sinh viên vào lớp dưới luật mới, hoặc sinh viên vào trước rồi PUT nhận 409.
func TestOptOutLockRaceWithFirstEnrollment(t *testing.T) {
	r := newRig(t)
	svc := course.NewService(course.Service{Pool: r.pool, Clock: r.clk, Log: r.svc.Log})
	var wg sync.WaitGroup
	outcomes := make(chan string, 50)
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := r.world(true)
			code := r.scalar(`select join_code from courses where id = $1`, w.course)
			stu := r.addUser(uniq(fmt.Sprintf("race%d", i)), store.UserRoleSTUDENT, store.UserStatusACTIVE)
			r.exec(`update users set email_verified_at = now() where id = $1`, stu.ID)
			var put error
			var joined error
			var inner sync.WaitGroup
			inner.Add(2)
			go func() {
				defer inner.Done()
				_, put = svc.PutSettings(t.Context(), w.teacher.ID, w.course, 1, course.SettingsIn{AllowStudyTimeOptOut: ptr(false)})
			}()
			go func() {
				defer inner.Done()
				_, joined = svc.Join(t.Context(), stu.ID, "203.0.113.9", code)
			}()
			inner.Wait()
			enrolled := r.count(`select count(*) from enrollments where course_id = $1 and user_id = $2`, w.course, stu.ID)
			flag := r.count(`select count(*) from courses where id = $1 and allow_study_time_opt_out`, w.course)
			switch {
			case joined != nil || enrolled != 1:
				outcomes <- fmt.Sprintf("sinh viên không vào được lớp: %v", joined)
			case put == nil && flag == 0:
				outcomes <- "PUT thắng, sinh viên vào dưới luật mới"
			case put != nil && flag == 1 && strings.Contains(put.Error(), "đã có sinh viên"):
				outcomes <- "sinh viên vào trước, PUT 409"
			default:
				outcomes <- fmt.Sprintf("KẾT CỤC SAI: put=%v flag=%d", put, flag)
			}
		}()
	}
	wg.Wait()
	close(outcomes)
	seen := map[string]int{}
	for o := range outcomes {
		seen[o]++
	}
	for o := range seen {
		require.Contains(t, []string{"PUT thắng, sinh viên vào dưới luật mới", "sinh viên vào trước, PUT 409"}, o)
	}
}

func ptr[T any](v T) *T { return &v }
