package crm_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/crm"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

var ictTest = time.FixedZone("ICT", 7*3600)

// midweek là thứ Tư 12:00 ICT của tuần hiện tại: mọi lần chạy ± 2 ngày vẫn cùng một tuần nên W1 / W2 không trượt giữa test.
func midweek() time.Time {
	n := time.Now().In(ictTest)
	d := time.Date(n.Year(), n.Month(), n.Day(), 12, 0, 0, 0, ictTest)
	return d.AddDate(0, 0, -((int(d.Weekday())+6)%7)+2)
}

func (r *rig) svc() *crm.Service {
	return &crm.Service{Pool: r.pool, Clock: r.clk, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

// pastSession chèn một buổi vào thứ Ba của tuần cách tuần hiện tại `weeksAgo` tuần (1 = W1) và trả id.
func (w world) pastSession(weeksAgo, no int) uuid.UUID {
	w.r.t.Helper()
	start := midweek().AddDate(0, 0, -7*weeksAgo-1).Add(-2*time.Hour + time.Duration(no)*time.Minute) // thứ Ba 10:00 + no phút (starts_at duy nhất mỗi lớp)
	var id uuid.UUID
	require.NoError(w.r.t, w.r.pool.QueryRow(w.r.t.Context(), `insert into class_sessions (course_id, session_no, starts_at, ends_at) values ($1, $2, $3, $4) on conflict (course_id, session_no) do update set session_no = excluded.session_no returning id`,
		w.course, 100+no, start, start.Add(2*time.Hour)).Scan(&id))
	return id
}

func (w world) mark(session, student uuid.UUID, status string) {
	w.r.exec(`insert into attendance_records (course_id, session_id, student_id, status, marked_by) values ($1, $2, $3, $4::attendance_status, $5)`, w.course, session, student, status, w.teacher.ID)
}

// absentTwoWeeks: sinh viên vắng hết các buổi (2 buổi mỗi tuần) ở W1 và W2, không hoạt động nào ⇒ hai tuần yếu (chuyên cần + hoạt động thấp).
func (w world) absentTwoWeeks(s uuid.UUID) {
	n := 0
	for week := 1; week <= 2; week++ {
		for range 2 {
			n++
			w.mark(w.pastSession(week, n), s, "ABSENT")
		}
	}
}

type riskRow struct {
	level               string
	entered, computedAt *time.Time
	asks, posts         int
}

func (w world) risk(s uuid.UUID) (riskRow, bool) {
	w.r.t.Helper()
	rows, err := w.r.pool.Query(w.r.t.Context(), `select level::text, entered_at, computed_at, last_week_asks, last_week_posts from student_risk where course_id = $1 and student_id = $2`, w.course, s)
	require.NoError(w.r.t, err)
	defer rows.Close()
	if !rows.Next() {
		return riskRow{}, false
	}
	var x riskRow
	require.NoError(w.r.t, rows.Scan(&x.level, &x.entered, &x.computedAt, &x.asks, &x.posts))
	return x, true
}

// TestRiskEnteredAtKept: hai đêm liền AT_RISK → entered_at không đổi (computed_at đổi); rời AT_RISK → entered_at NULL (US-P5-04 AC4).
func TestRiskEnteredAtKept(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	s := w.students[0].ID
	w.absentTwoWeeks(s)
	svc := r.svc()
	now := midweek()
	require.NoError(t, svc.RunCourse(t.Context(), w.course, now))
	x, ok := w.risk(s)
	require.True(t, ok)
	require.Equal(t, "AT_RISK", x.level)
	require.NotNil(t, x.entered)
	first := *x.entered
	require.WithinDuration(t, now, first, time.Second)
	other, ok := w.risk(w.students[1].ID)
	require.True(t, ok, "sinh viên không có dữ liệu vẫn có dòng NONE")
	require.Equal(t, "NONE", other.level)
	require.Nil(t, other.entered)

	require.NoError(t, svc.RunCourse(t.Context(), w.course, now.Add(24*time.Hour)))
	x, _ = w.risk(s)
	require.Equal(t, "AT_RISK", x.level)
	require.True(t, first.Equal(*x.entered), "entered_at giữ nguyên: %v ≠ %v", first, *x.entered)
	require.True(t, x.computedAt.After(first), "computed_at là của lần chạy sau")

	r.exec(`update attendance_records set status = 'PRESENT' where course_id = $1 and student_id = $2`, w.course, s)
	require.NoError(t, svc.RunCourse(t.Context(), w.course, now.Add(48*time.Hour)))
	x, _ = w.risk(s)
	require.Equal(t, "NONE", x.level)
	require.Nil(t, x.entered)
}

// TestRiskIdempotentDB: chạy lại cùng dữ liệu → cùng mức, cùng lý do, cùng tuần.
func TestRiskIdempotentDB(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	w.absentTwoWeeks(w.students[0].ID)
	svc := r.svc()
	snap := func() string {
		return r.scalar(`select level::text || reasons::text || weeks::text from student_risk where course_id = $1 and student_id = $2`, w.course, w.students[0].ID)
	}
	require.NoError(t, svc.RunCourse(t.Context(), w.course, midweek()))
	a := snap()
	require.NoError(t, svc.RunCourse(t.Context(), w.course, midweek()))
	require.Equal(t, a, snap())
	require.Contains(t, a, "ATTENDANCE_LOW")
}

// TestRiskJobDoesNotTouchRemoved: sinh viên REMOVED / PENDING không có dòng student_risk; dòng cũ của người vừa bị mời ra bị xoá.
func TestRiskJobDoesNotTouchRemoved(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	svc := r.svc()
	require.NoError(t, svc.RunCourse(t.Context(), w.course, midweek()))
	require.Equal(t, 2, r.count(`select count(*) from student_risk where course_id = $1`, w.course))
	require.Equal(t, 0, r.count(`select count(*) from student_risk where course_id = $1 and student_id = $2`, w.course, w.pending.ID), "PENDING không có dòng")
	r.exec(`update enrollments set status = 'REMOVED', removed_at = now() where course_id = $1 and user_id = $2`, w.course, w.students[1].ID)
	require.NoError(t, svc.RunCourse(t.Context(), w.course, midweek()))
	require.Equal(t, 1, r.count(`select count(*) from student_risk where course_id = $1`, w.course))
	require.Equal(t, 2, r.count(`select count(*) from outbox where topic = 'crm.risk_updated' and payload->>'course_id' = $1`, w.course.String()), "mỗi lượt một sự kiện")
}

// job dựng RiskJob; Redis dùng chung cả gói nên xoá khoá leader và dấu ngày (hôm qua / nay / mai) để các test không ảnh hưởng nhau.
func (r *rig) job(name string, at time.Duration) *crm.RiskJob {
	keys := []string{crm.RiskLeaderKey}
	for d := -1; d <= 1; d++ {
		keys = append(keys, appredis.Key("crm.risk", "done", time.Now().In(ictTest).AddDate(0, 0, d).Format("20060102")))
	}
	require.NoError(r.t, r.rdb.Del(r.t.Context(), keys...).Err())
	return &crm.RiskJob{Svc: r.svc(), Redis: r.rdb, Name: name, At: at, Every: time.Minute}
}

func ictClock(r *rig, h, m int) {
	n := time.Now().In(ictTest)
	r.clk.Set(time.Date(n.Year(), n.Month(), n.Day(), h, m, 0, 0, ictTest))
}

// TestRiskJobRunsOncePerDay: trước 02:00 không chạy; qua 02:00 chạy đúng một lần; tick sau cùng ngày bỏ qua; sang ngày mới chạy lại.
func TestRiskJobRunsOncePerDay(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	j := r.job("a", 2*time.Hour)
	ictClock(r, 1, 59)
	ran, err := j.Once(t.Context())
	require.NoError(t, err)
	require.False(t, ran, "chưa tới 02:00")
	require.Equal(t, 0, r.count(`select count(*) from student_risk where course_id = $1`, w.course))
	ictClock(r, 2, 0)
	ran, err = j.Once(t.Context())
	require.NoError(t, err)
	require.True(t, ran)
	require.Equal(t, 1, r.count(`select count(*) from student_risk where course_id = $1`, w.course))
	r.clk.Advance(5 * time.Minute)
	ran, _ = j.Once(t.Context())
	require.False(t, ran, "cùng ngày chỉ một lần")
	r.clk.Advance(24 * time.Hour)
	ran, _ = j.Once(t.Context())
	require.True(t, ran, "ngày mới chạy lại")
}

// TestRiskJobCatchUpAfterWorkerDown: worker chết qua 02:00 và lên lại 09:00 → chạy bù đúng một lần.
func TestRiskJobCatchUpAfterWorkerDown(t *testing.T) {
	r := newRig(t)
	r.world(1)
	j := r.job("a", 2*time.Hour)
	ictClock(r, 9, 0)
	ran, err := j.Once(t.Context())
	require.NoError(t, err)
	require.True(t, ran)
	ran, _ = j.Once(t.Context())
	require.False(t, ran)
}

// TestRiskJobFailureRetries: lỗi giữa chừng → dấu ngày bị xoá, tick sau chạy lại.
func TestRiskJobFailureRetries(t *testing.T) {
	r := newRig(t)
	r.world(1)
	j := r.job("a", 2*time.Hour)
	ictClock(r, 3, 0)
	good := j.Svc
	bad, err := pgxpool.New(t.Context(), r.pool.Config().ConnString())
	require.NoError(t, err)
	bad.Close() // DB không dùng được: RunAll lỗi giữa chừng
	j.Svc = &crm.Service{Pool: bad, Clock: r.clk}
	_, err = j.Once(t.Context())
	require.Error(t, err)
	j.Svc = good
	r.clk.Advance(time.Minute)
	ran, err := j.Once(t.Context())
	require.NoError(t, err)
	require.True(t, ran, "dấu ngày đã xoá nên tick sau chạy lại")
}

// TestRiskJobLeaderOnce: hai bản worker (tên khác nhau) cùng tick → đúng một lần.
func TestRiskJobLeaderOnce(t *testing.T) {
	r := newRig(t)
	r.world(1)
	ictClock(r, 5, 0)
	jobs := []*crm.RiskJob{r.job("w1", 2*time.Hour), r.job("w2", 2*time.Hour)} // dựng trước: job() xoá khoá Redis dùng chung
	var wg sync.WaitGroup
	var mu sync.Mutex
	ran := 0
	for _, j := range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := j.Once(context.Background())
			require.NoError(t, err)
			mu.Lock()
			defer mu.Unlock()
			if ok {
				ran++
			}
		}()
	}
	wg.Wait()
	require.Equal(t, 1, ran)
}

// ---- ngưỡng trong cài đặt lớp (AC6) ----

func (w world) putRisk(s session, version int, risk map[string]any) resp {
	body := map[string]any{"risk": risk}
	if version >= 0 {
		body["version"] = version
	}
	return w.r.call(http.MethodPut, "/courses/"+w.course.String()+"/settings", s, body)
}

func (w world) version() int {
	return w.r.count(`select version from courses where id = $1`, w.course)
}

func TestSettingsRiskRange(t *testing.T) {
	r := newRig(t)
	w := r.world(0)
	for field, bad := range map[string]any{"attendance_ok_min": "0.49", "activity_min_per_week": 21, "study_min_minutes": 601, "signals_required": 4, "absent_alert": 0} {
		res := w.putRisk(w.T, w.version(), map[string]any{field: bad})
		require.Equal(t, http.StatusUnprocessableEntity, res.code, "%s=%v: %s", field, bad, res.body)
		require.Contains(t, string(res.body), "risk."+field)
	}
	require.Equal(t, http.StatusUnprocessableEntity, w.putRisk(w.T, w.version(), map[string]any{"attendance_ok_min": "0.72"}).code, "bước 0,05")
	require.Equal(t, http.StatusOK, w.putRisk(w.T, w.version(), map[string]any{"attendance_ok_min": "0.50", "activity_min_per_week": 0, "study_min_minutes": 600, "signals_required": 3, "absent_alert": 20}).code, "biên hợp lệ")
}

func TestSettingsRiskRBAC(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	v := w.version()
	require.Equal(t, http.StatusForbidden, w.putRisk(w.A, v, map[string]any{"absent_alert": 4}).code, "TA")
	require.Equal(t, http.StatusForbidden, w.putRisk(w.ADM, v, map[string]any{"absent_alert": 4}).code, "ADMIN")
	st := r.mustLogin(w.students[0].Email)
	require.Equal(t, http.StatusForbidden, w.putRisk(st, v, map[string]any{"absent_alert": 4}).code, "STUDENT")
	require.Equal(t, http.StatusOK, w.putRisk(w.T, v, map[string]any{"absent_alert": 4}).code)
}

func TestSettingsRiskVersion(t *testing.T) {
	r := newRig(t)
	w := r.world(0)
	v := w.version()
	require.Equal(t, http.StatusOK, w.putRisk(w.T, v, map[string]any{"absent_alert": 4}).code)
	res := w.putRisk(w.T, v, map[string]any{"absent_alert": 5})
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "VERSION_CONFLICT", res.errCode())
	require.Equal(t, 1, r.count(`select count(*) from audit_log where course_id = $1 and action = 'course_risk_updated'`, w.course), "chỉ lần thành công có audit (cũ / mới)")
	require.Contains(t, r.scalar(`select before::text || after::text from audit_log where course_id = $1 and action = 'course_risk_updated'`, w.course), `"absent_alert": 4`)
}

// TestRiskDefaultsWhenUnset: chưa đặt → 0,70 / 1 / 60 / 2 / 3; đặt một trường giữ nguyên bốn trường còn lại.
func TestRiskDefaultsWhenUnset(t *testing.T) {
	r := newRig(t)
	w := r.world(0)
	get := func() map[string]any {
		res := r.call(http.MethodGet, "/courses/"+w.course.String(), w.T, nil)
		require.Equal(t, http.StatusOK, res.code, string(res.body))
		return res.json()["settings"].(map[string]any)["risk"].(map[string]any)
	}
	require.Equal(t, map[string]any{"attendance_ok_min": "0.70", "activity_min_per_week": float64(1), "study_min_minutes": float64(60), "signals_required": float64(2), "absent_alert": float64(3)}, get())
	require.Equal(t, http.StatusOK, w.putRisk(w.T, w.version(), map[string]any{"study_min_minutes": 90}).code)
	got := get()
	require.Equal(t, float64(90), got["study_min_minutes"])
	require.Equal(t, "0.70", got["attendance_ok_min"])
}

// TestRiskRecomputedAfterSettings: PUT ngưỡng → outbox `course.settings_changed`; handler tính lại riêng lớp đó theo ngưỡng mới.
func TestRiskRecomputedAfterSettings(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	s := w.students[0].ID
	n := 0
	for week := 1; week <= 2; week++ { // mỗi tuần 4 buổi, vắng 1 ⇒ tỷ lệ 0,75: ≥ 0,70 nên không thấp ở ngưỡng mặc định
		for i := range 4 {
			n++
			st := "PRESENT"
			if i == 0 {
				st = "ABSENT"
			}
			w.mark(w.pastSession(week, n), s, st)
		}
	}
	svc := r.svc()
	require.NoError(t, svc.RunCourse(t.Context(), w.course, midweek()))
	x, _ := w.risk(s)
	require.Equal(t, "NONE", x.level)
	require.Equal(t, http.StatusOK, w.putRisk(w.T, w.version(), map[string]any{"attendance_ok_min": "0.80"}).code)
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'course.settings_changed' and payload->>'course_id' = $1`, w.course.String()))
	other := r.course(w.teacher.ID, "OTH"+uuid.NewString()[:5])
	require.NoError(t, svc.HandleSettingsChanged(t.Context(), settingsMsg(w.course)))
	require.Equal(t, 0, r.count(`select count(*) from student_risk where course_id = $1`, other), "chỉ lớp đổi ngưỡng được tính lại")
	x, _ = w.risk(s)
	require.Equal(t, "AT_RISK", x.level, "0,75 < 0,80 ⇒ hai tuần yếu (chuyên cần + hoạt động)")
}

func settingsMsg(course uuid.UUID) outbox.Message {
	return outbox.Message{Topic: crm.TopicSettingsChanged, Payload: []byte(`{"course_id":"` + course.String() + `"}`)}
}

// studyOf trả `weeks[0].study` của student_risk dưới dạng chữ (NULL khi JSON null: không đo).
func (w world) studyOf(student uuid.UUID) *string {
	w.r.t.Helper()
	var s *string
	require.NoError(w.r.t, w.r.pool.QueryRow(w.r.t.Context(), `select weeks->0->>'study' from student_risk where course_id = $1 and student_id = $2`, w.course, student).Scan(&s))
	return s
}

// lowAttendanceActive: hai tuần chuyên cần thấp (vắng hết) nhưng CÓ hoạt động (bình luận) mỗi tuần ⇒ chỉ chuyên cần thấp; thời gian học quyết định có thành tín hiệu thứ hai không.
func (w world) lowAttendanceActive(s uuid.UUID) {
	w.absentTwoWeeks(s)
	w.r.exec(`insert into forum_threads (course_id, author_id, title, body, created_at) select $1, $2, 't', 'b', $3 from generate_series(1, 2) g`, w.course, s, midweek().AddDate(0, 0, -8))
	w.r.exec(`insert into forum_threads (course_id, author_id, title, body, created_at) select $1, $2, 't', 'b', $3 from generate_series(1, 2) g`, w.course, s, midweek().AddDate(0, 0, -15))
}

// TestRiskStudyNullWhenOptedOut — Q1: sinh viên đã tắt đo ở lớp cho phép tắt ⇒ STUDY = null, KHÔNG bị coi là thấp (chỉ một tín hiệu thấp ⇒ NONE); sinh viên vẫn đo với 0 phút ⇒ STUDY thấp ⇒ AT_RISK.
func TestRiskStudyNullWhenOptedOut(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	off, on := w.students[0].ID, w.students[1].ID
	r.exec(`update users set tracking_notice_ack_at = $1 where id = any($2)`, midweek().AddDate(0, 0, -60), []uuid.UUID{off, on})
	r.exec(`insert into user_settings (user_id, track_study_time) values ($1, false)`, off)
	for _, s := range []uuid.UUID{off, on} {
		w.lowAttendanceActive(s)
	}
	require.NoError(t, r.svc().RunCourse(t.Context(), w.course, midweek()))
	x, _ := w.risk(off)
	require.Equal(t, "NONE", x.level, "tắt đo: STUDY=null không thành tín hiệu thấp thứ hai")
	require.Nil(t, w.studyOf(off))
	y, _ := w.risk(on)
	require.Equal(t, "AT_RISK", y.level, "vẫn đo, 0 phút: chuyên cần + thời gian học thấp")
	require.NotNil(t, w.studyOf(on))
}

// TestRiskStudyPerCourseConfig — cùng một sinh viên tắt đo: lớp X cho tắt ⇒ null; lớp Y không cho tắt ⇒ vẫn đo (0 phút = thấp).
func TestRiskStudyPerCourseConfig(t *testing.T) {
	r := newRig(t)
	wx := r.world(1)
	s := wx.students[0].ID
	y := r.course(wx.teacher.ID, "Y"+uuid.NewString()[:6])
	r.exec(`update courses set allow_study_time_opt_out = false where id = $1`, y)
	r.enroll(y, s, "STUDENT", "ACTIVE")
	wy := wx
	wy.course = y
	r.exec(`update users set tracking_notice_ack_at = $1 where id = $2`, midweek().AddDate(0, 0, -60), s)
	r.exec(`insert into user_settings (user_id, track_study_time) values ($1, false)`, s)
	wx.lowAttendanceActive(s)
	wy.lowAttendanceActive(s)
	svc := r.svc()
	require.NoError(t, svc.RunCourse(t.Context(), wx.course, midweek()))
	require.NoError(t, svc.RunCourse(t.Context(), y, midweek()))
	require.Nil(t, wx.studyOf(s), "lớp X cho tắt: không đo")
	require.NotNil(t, wy.studyOf(s), "lớp Y không cho tắt: cờ của sinh viên bị bỏ qua")
	a, _ := wx.risk(s)
	b, _ := wy.risk(s)
	require.Equal(t, "NONE", a.level)
	require.Equal(t, "AT_RISK", b.level)
}

// TestRiskNotAckedStudyNull — chưa xác nhận thông báo từ trước tuần W ⇒ STUDY = null (không đo ⇒ không thể "thấp").
func TestRiskNotAckedStudyNull(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	w.lowAttendanceActive(w.students[0].ID)
	require.NoError(t, r.svc().RunCourse(t.Context(), w.course, midweek()))
	x, _ := w.risk(w.students[0].ID)
	require.Equal(t, "NONE", x.level)
	require.Nil(t, w.studyOf(w.students[0].ID))
}
