package activity_test

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/activity"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

var ict = time.FixedZone("ICT", 7*3600) //nolint:gochecknoglobals // test

func (r *rig) count(sql string, args ...any) int {
	r.t.Helper()
	var n int
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), sql, args...).Scan(&n))
	return n
}

func (r *rig) scalar(sql string, args ...any) string {
	r.t.Helper()
	var s string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), sql, args...).Scan(&s))
	return s
}

func (r *rig) exec(sql string, args ...any) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), sql, args...)
	require.NoError(r.t, err, sql)
}

type stu struct {
	u store.User
	s session
}

// world: một lớp (allow = cho tắt đo) có giảng viên, TA, Admin; sinh viên thêm bằng r.student.
type world struct {
	r                *rig
	course           uuid.UUID
	teacher, ta, adm store.User
	T, A, ADM        session
}

func (r *rig) world(allow bool) world {
	r.t.Helper()
	adm := r.addUser(uniq("adm"), store.UserRoleADMIN, store.UserStatusACTIVE)
	w := world{r: r, adm: adm}
	w.teacher = r.addUser(uniq("gv"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	w.ta = r.addUser(uniq("ta"), store.UserRoleTA, store.UserStatusACTIVE)
	w.T, w.A, w.ADM = r.mustLogin(w.teacher.Email), r.mustLogin(w.ta.Email), r.mustLogin(adm.Email)
	w.course = r.courseWith(adm.ID, "C"+strings.ToUpper(uuid.NewString()[:6]), allow)
	r.enroll(w.course, w.teacher.ID, "TEACHER", "ACTIVE")
	r.enroll(w.course, w.ta.ID, "TA", "ACTIVE")
	return w
}

func (r *rig) courseWith(by uuid.UUID, code string, allow bool) uuid.UUID {
	r.t.Helper()
	id := r.course(by, code)
	r.exec(`update courses set allow_study_time_opt_out = $2 where id = $1`, id, allow)
	return id
}

// student: sinh viên ACTIVE của lớp; ack = đã bấm "Đã hiểu".
func (w world) student(ack bool) stu {
	w.r.t.Helper()
	return w.studentIn(w.course, ack)
}

func (w world) studentIn(course uuid.UUID, ack bool) stu {
	w.r.t.Helper()
	u := w.r.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	w.r.enroll(course, u.ID, "STUDENT", "ACTIVE")
	s := stu{u: u, s: w.r.mustLogin(u.Email)}
	if ack {
		w.r.ack(s)
	}
	return s
}

func (r *rig) ack(s stu) {
	r.t.Helper()
	require.Equal(r.t, http.StatusNoContent, r.call(http.MethodPost, "/me/tracking-notice/ack", s.s, nil).code)
}

func (r *rig) call(method, path string, s session, body any) resp {
	return r.do(req{method: method, path: path, bearer: s.access, body: body})
}

// beat gửi một nhịp qua Service trực tiếp (vai STUDENT) và trả lỗi.
func (r *rig) beat(uid, course uuid.UUID, area string) error {
	return r.svc.Heartbeat(r.t.Context(), uid, "STUDENT", activity.Beat{CourseID: course, Area: area, TabID: uuid.NewString()})
}

// httpBeat gửi một nhịp qua router (JWT thật).
func (r *rig) httpBeat(s session, course uuid.UUID, area string) resp {
	return r.call(http.MethodPost, "/activity/heartbeat", s, map[string]any{"course_id": course, "area": area, "tab_id": uuid.NewString()})
}

func (r *rig) day() string { return r.clk.Now().In(ict).Format("20060102") }

func (r *rig) key(course, uid uuid.UUID, area string) string {
	return appredis.Key("study", course.String(), uid.String(), r.day(), area)
}

// secs đọc giây tích luỹ trong Redis (0 nếu chưa có khoá).
func (r *rig) secs(course, uid uuid.UUID, area string) int {
	r.t.Helper()
	v, err := r.rdb.Get(r.t.Context(), r.key(course, uid, area)).Result()
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(v)
	require.NoError(r.t, err)
	return n
}

// next dịch đồng hồ giả sang khung 30 s kế tiếp.
func (r *rig) next() { r.clk.Advance(30 * time.Second) }

// dbSecs: tổng active_seconds của một sinh viên ở một lớp (mọi ngày, mọi khu vực) trong Postgres.
func (r *rig) dbSecs(course, uid uuid.UUID) int {
	r.t.Helper()
	return r.count(`select coalesce(sum(active_seconds), 0)::int from study_time_daily where course_id = $1 and student_id = $2`, course, uid)
}

func (r *rig) rollup() int {
	r.t.Helper()
	n, err := r.svc.Rollup(context.WithoutCancel(r.t.Context()))
	require.NoError(r.t, err)
	return n
}
