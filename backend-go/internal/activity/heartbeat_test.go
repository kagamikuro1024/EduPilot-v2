package activity_test

import (
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/activity"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// ---- AC2: chỉ ghi Redis ----

func TestHeartbeatNeverWritesPostgres(t *testing.T) {
	r := newRig(t, func(e map[string]string) { e["HEARTBEAT_MAX_PER_MINUTE"] = "10" })
	w := r.world(true)
	s := w.student(true)
	// làm nóng đệm (ack + ghi danh) rồi đếm: các nhịp kế tiếp không có câu GHI nào, và 60 s đệm không chạm DB
	require.Equal(t, http.StatusNoContent, r.httpBeat(s.s, w.course, "CHAT").code)
	before := r.wr.n.Load()
	for range 3 {
		r.next()
		require.Equal(t, http.StatusNoContent, r.httpBeat(s.s, w.course, "CHAT").code)
	}
	require.Equal(t, before, r.wr.n.Load(), "không INSERT / UPDATE / DELETE nào trong lúc xử lý nhịp")
	require.Zero(t, r.count(`select count(*) from study_time_daily`), "chưa rollup: DB không có hàng")
	require.Equal(t, 120, r.secs(w.course, s.u.ID, "CHAT"), "4 nhịp × 30 s")
}

func TestHeartbeatLatency(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT")) // nóng đệm
	lat := make([]time.Duration, 0, 200)
	for range 200 {
		r.next()
		r.svc.Cfg.MaxPerMinute = 1000
		t0 := time.Now()
		require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
		lat = append(lat, time.Since(t0))
	}
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	if testutil.TimingOn() { // CI chạy nhiều gói song song: cận trên mili-giây chỉ kiểm khi chạy một mình
		require.Less(t, lat[len(lat)*95/100], 20*time.Millisecond, "p95 ≤ 20 ms")
	}
	require.Equal(t, 201*30, r.secs(w.course, s.u.ID, "CHAT"))
}

// ---- AC3: hai tab ----

func TestHeartbeatTwoTabsNoDoubleCount(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT")) // tab 1
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT")) // tab 2 cùng khung 30 s: bỏ
	r.clk.Advance(10 * time.Second)
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT")) // vẫn cùng khung
	require.Equal(t, 30, r.secs(w.course, s.u.ID, "CHAT"))
	// 10 phút, hai tab lệch pha 7 s: mỗi khung đúng một nhịp ⇒ 20 khung × 30 s = 600 (+ nhịp đầu)
	for range 20 {
		r.clk.Advance(23 * time.Second)
		require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
		r.clk.Advance(7 * time.Second)
		require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
	}
	require.LessOrEqual(t, r.secs(w.course, s.u.ID, "CHAT"), 30+10*60+30)
	require.Greater(t, r.secs(w.course, s.u.ID, "CHAT"), 10*60-90)
}

func TestHeartbeatBoundaryTolerance(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	// hai nhịp cách nhau 1 s nhưng vắt qua ranh giới khung: cả hai được tính (sai số một khung do ranh giới)
	r.clk.Advance(29 * time.Second)
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
	r.clk.Advance(time.Second)
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
	require.Equal(t, 60, r.secs(w.course, s.u.ID, "CHAT"))
	// nhịp hợp lệ liên tiếp ở các khung khác nhau đều được tính
	for range 3 {
		r.next()
		require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
	}
	require.Equal(t, 150, r.secs(w.course, s.u.ID, "CHAT"))
}

// ---- AC5: đo đúng ----

// TestMeasureThreeMinutesActive — mở 10 phút, thao tác 3 phút đầu: hook gửi nhịp trong 3 phút thao tác + 2 phút chờ idle rồi dừng ⇒ 300 ± 30 s.
func TestMeasureThreeMinutesActive(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	for range 10 { // 5 phút / 30 s
		require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
		r.next()
	}
	r.clk.Advance(5 * time.Minute) // 5 phút còn lại: hook đã dừng, không có nhịp
	require.InDelta(t, 300, r.secs(w.course, s.u.ID, "CHAT"), 30)
}

// TestHiddenTimeNotCounted — tab ẩn 4 / 10 phút: hook không gửi nhịp khi ẩn ⇒ 6 phút (12 nhịp).
func TestHiddenTimeNotCounted(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	for range 12 {
		require.NoError(t, r.beat(s.u.ID, w.course, "THREADS"))
		r.next()
	}
	r.clk.Advance(4 * time.Minute)
	require.Equal(t, 360, r.secs(w.course, s.u.ID, "THREADS"))
}

// TestOnlyAreasRecorded — chỉ khu vực: khoá Redis chỉ chứa (lớp, người, ngày, khu vực); area lạ → 422; không ghi nội dung / đường dẫn / tiêu đề / tên.
func TestOnlyAreasRecorded(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	for _, a := range activity.Areas {
		r.next()
		require.NoError(t, r.beat(s.u.ID, w.course, a), a)
	}
	for _, bad := range []string{"EXAM", "ME", "chat", "/chat", ""} {
		err := r.beat(s.u.ID, w.course, bad)
		require.Error(t, err, bad)
	}
	var keys []string
	for _, k := range r.rdb.Keys(t.Context(), appredis.Key("study", w.course.String(), s.u.ID.String(), "*")).Val() {
		keys = append(keys, strings.TrimPrefix(k, appredis.Key("study", w.course.String(), s.u.ID.String())+":"+r.day()+":"))
	}
	sort.Strings(keys)
	require.Equal(t, []string{"CALENDAR", "CHAT", "LIBRARY", "PRACTICE", "THREADS"}, keys)
	for _, k := range r.rdb.Keys(t.Context(), appredis.Key("study", "*", s.u.ID.String(), "*")).Val() {
		require.NotContains(t, strings.ToLower(k), strings.ToLower(s.u.FullName))
	}
	require.Equal(t, http.StatusUnprocessableEntity, r.httpBeat(s.s, w.course, "EXAM").code)
}

// ---- AC7: minh bạch lần đầu ----

func TestHeartbeatIgnoredBeforeAck(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(false)
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
	require.Zero(t, r.secs(w.course, s.u.ID, "CHAT"), "chưa ack: 204 không cộng")
	require.Equal(t, http.StatusNoContent, r.httpBeat(s.s, w.course, "CHAT").code)
	require.Zero(t, r.secs(w.course, s.u.ID, "CHAT"))
}

func TestAckIdempotent(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(false)
	r.ack(s)
	first := r.count(`select extract(epoch from tracking_notice_ack_at)::int from users where id = $1`, s.u.ID)
	time.Sleep(1100 * time.Millisecond) // đồng hồ thật: ack_at đổi nếu bị ghi đè
	r.ack(s)
	require.Equal(t, first, r.count(`select extract(epoch from tracking_notice_ack_at)::int from users where id = $1`, s.u.ID), "giữ lần đầu")
}

func TestStaffNotTracked(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	for name, u := range map[string]uuid.UUID{"teacher": w.teacher.ID, "ta": w.ta.ID, "admin": uuid.New()} {
		require.NoError(t, r.svc.Heartbeat(t.Context(), u, strings.ToUpper(name), activity.Beat{CourseID: w.course, Area: "CHAT", TabID: uuid.NewString()}), name)
		require.Zero(t, r.secs(w.course, u, "CHAT"), name)
	}
	require.Equal(t, http.StatusNoContent, r.httpBeat(w.T, w.course, "CHAT").code, "giảng viên: 204 im lặng")
	require.Zero(t, r.secs(w.course, w.teacher.ID, "CHAT"))
}

// TestHeartbeatCountsImmediatelyAfterAck — token cũ vẫn dùng được: ack xong nhịp kế tiếp được đo ngay (không đợi đệm 60 s, không phụ thuộc claim JWT).
func TestHeartbeatCountsImmediatelyAfterAck(t *testing.T) {
	r := newRig(t, func(e map[string]string) { e["HEARTBEAT_MAX_PER_MINUTE"] = "10" })
	w := r.world(true)
	s := w.student(false)
	require.Equal(t, http.StatusNoContent, r.httpBeat(s.s, w.course, "CHAT").code) // đệm ep:trk = chưa ack
	require.Zero(t, r.secs(w.course, s.u.ID, "CHAT"))
	r.ack(s)
	r.next()
	require.Equal(t, http.StatusNoContent, r.httpBeat(s.s, w.course, "CHAT").code) // cùng access token cũ
	require.Equal(t, 30, r.secs(w.course, s.u.ID, "CHAT"))
}

// ---- AC10: nhánh lỗi ----

func TestHeartbeatRedisDown204(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	dead := &activity.Service{Pool: r.pool, Clock: r.clk, Log: r.svc.Log}
	require.NoError(t, dead.Heartbeat(t.Context(), s.u.ID, "STUDENT", activity.Beat{CourseID: w.course, Area: "CHAT", TabID: uuid.NewString()}))
	require.EqualValues(t, 1, dead.Dropped.Load(), "activity_heartbeat_dropped_total tăng")
	require.Zero(t, r.secs(w.course, s.u.ID, "CHAT"))
	require.NotContains(t, r.logs.String(), s.u.Email, "log không PII")
}

// #31 BUG-2: Redis treo (nhận kết nối nhưng không trả lời) → nhịp bị bỏ trong ≤ 1 s, không treo hàng chục giây; vẫn 204 và đếm bỏ.
func TestHeartbeatRedisHangBounded(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { _ = c.Close() }) // giữ mở, không bao giờ trả lời
		}
	}()
	rd, err := appredis.New(t.Context(), "redis://"+ln.Addr().String())
	require.NoError(t, err)
	hung := &activity.Service{Pool: r.pool, Redis: rd, Clock: r.clk, Log: r.svc.Log}
	t0 := time.Now()
	require.NoError(t, hung.Heartbeat(t.Context(), s.u.ID, "STUDENT", activity.Beat{CourseID: w.course, Area: "CHAT", TabID: uuid.NewString()}))
	require.Less(t, time.Since(t0), time.Second, "Redis treo không được giữ nhịp quá 1 s")
	require.EqualValues(t, 1, hung.Dropped.Load())
	require.Contains(t, r.logs.String(), `total=1`, "bộ đếm cộng dồn nằm trong dòng WARN")
}

func TestHeartbeatBadCourse403(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	other := w.r.courseWith(w.adm.ID, "X"+strings.ToUpper(uuid.NewString()[:6]), true)
	require.Equal(t, http.StatusForbidden, r.httpBeat(s.s, other, "CHAT").code, "ngoài lớp")
	require.Equal(t, http.StatusForbidden, r.httpBeat(s.s, uuid.New(), "CHAT").code, "lớp không tồn tại")
}

func TestHeartbeatBadArea422(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	require.Equal(t, http.StatusUnprocessableEntity, r.httpBeat(s.s, w.course, "EXAM").code)
	for name, body := range map[string]any{
		"thiếu course_id": map[string]any{"area": "CHAT", "tab_id": uuid.NewString()},
		"course_id sai":   map[string]any{"course_id": "x", "area": "CHAT", "tab_id": uuid.NewString()},
		"tab_id sai":      map[string]any{"course_id": w.course, "area": "CHAT", "tab_id": "x"},
		"sai kiểu":        map[string]any{"course_id": w.course, "area": 5, "tab_id": uuid.NewString()},
		"trường lạ":       map[string]any{"course_id": w.course, "area": "CHAT", "tab_id": uuid.NewString(), "path": "/chat"},
	} {
		require.Equal(t, http.StatusUnprocessableEntity, r.call(http.MethodPost, "/activity/heartbeat", s.s, body).code, name)
	}
}

func TestHeartbeatDailyCap(t *testing.T) {
	r := newRig(t)
	r.svc.Cfg.DailyCap = 90
	w := r.world(true)
	s := w.student(true)
	for range 6 {
		require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
		r.next()
	}
	require.Equal(t, 90, r.secs(w.course, s.u.ID, "CHAT"), "trần ngày: nhịp thứ 4 trở đi bị bỏ (204, không cộng)")
}

// TestHeartbeatDailyCapAcrossCourses — trần tính trên KHOÁ TỔNG ngày (mọi lớp × khu vực), không theo khoá đếm riêng.
func TestHeartbeatDailyCapAcrossCourses(t *testing.T) {
	r := newRig(t)
	r.svc.Cfg.DailyCap = 120
	w := r.world(true)
	second := w.r.courseWith(w.adm.ID, "Y"+strings.ToUpper(uuid.NewString()[:6]), true)
	s := w.student(true)
	r.enroll(second, s.u.ID, "STUDENT", "ACTIVE")
	type pick struct {
		c uuid.UUID
		a string
	}
	for _, p := range []pick{{w.course, "CHAT"}, {second, "CHAT"}, {w.course, "THREADS"}, {second, "THREADS"}, {w.course, "LIBRARY"}} {
		require.NoError(t, r.beat(s.u.ID, p.c, p.a))
		r.next()
	}
	total := r.secs(w.course, s.u.ID, "CHAT") + r.secs(second, s.u.ID, "CHAT") + r.secs(w.course, s.u.ID, "THREADS") + r.secs(second, s.u.ID, "THREADS") + r.secs(w.course, s.u.ID, "LIBRARY")
	require.Equal(t, 120, total, "nhịp thứ 5 bị bỏ khi tổng hai lớp × hai khu vực chạm trần")
}

func TestHeartbeatBurstIgnored(t *testing.T) {
	r := newRig(t)
	r.svc.Cfg.MaxPerMinute = 4
	w := r.world(true)
	s := w.student(true)
	for range 7 { // mỗi nhịp một khung khác nhau nhưng trong cùng một phút của Redis
		require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
		r.next()
	}
	require.Equal(t, 4*30, r.secs(w.course, s.u.ID, "CHAT"), "quá 4 nhịp / phút bị bỏ (204, không 429)")
}

// ---- AC11 / AC12 / AC13: cấu hình đo ----

func TestHeartbeatRespectsStudyTimeOptOut(t *testing.T) {
	r := newRig(t, func(e map[string]string) { e["HEARTBEAT_MAX_PER_MINUTE"] = "10" })
	w := r.world(true)
	s := w.student(true)
	require.Equal(t, http.StatusNoContent, r.httpBeat(s.s, w.course, "CHAT").code)
	require.Equal(t, 30, r.secs(w.course, s.u.ID, "CHAT"))
	r.next()
	require.Equal(t, http.StatusOK, r.call(http.MethodPut, "/me/settings", s.s, map[string]any{"track_study_time": false, "version": 1}).code)
	require.Equal(t, http.StatusNoContent, r.httpBeat(s.s, w.course, "CHAT").code)
	require.Equal(t, 30, r.secs(w.course, s.u.ID, "CHAT"), "tắt đo: hiệu lực NGAY (DEL ep:trk), không đợi 60 s; dữ liệu cũ giữ")
}

func TestHeartbeatResumesWhenReEnabled(t *testing.T) {
	r := newRig(t, func(e map[string]string) { e["HEARTBEAT_MAX_PER_MINUTE"] = "10" })
	w := r.world(true)
	s := w.student(true)
	require.Equal(t, http.StatusOK, r.call(http.MethodPut, "/me/settings", s.s, map[string]any{"track_study_time": false, "version": 1}).code)
	require.Equal(t, http.StatusNoContent, r.httpBeat(s.s, w.course, "CHAT").code)
	require.Zero(t, r.secs(w.course, s.u.ID, "CHAT"))
	require.Equal(t, http.StatusOK, r.call(http.MethodPut, "/me/settings", s.s, map[string]any{"track_study_time": true, "version": 2}).code)
	r.next()
	require.Equal(t, http.StatusNoContent, r.httpBeat(s.s, w.course, "CHAT").code)
	require.Equal(t, 30, r.secs(w.course, s.u.ID, "CHAT"))
}

func TestHeartbeatLockedCourseIgnoresOptOut(t *testing.T) {
	r := newRig(t)
	w := r.world(false) // lớp không cho tắt
	s := w.student(true)
	r.exec(`insert into user_settings (user_id, track_study_time) values ($1, false) on conflict (user_id) do update set track_study_time = false`, s.u.ID)
	r.svc.InvalidateTracking(t.Context(), s.u.ID)
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
	require.Equal(t, 30, r.secs(w.course, s.u.ID, "CHAT"), "lớp không cho tắt vẫn đo (sau Đã hiểu)")
}

// TestHeartbeatTwoCoursesDifferentConfig / TestDroppedBeatDoesNotTakeWindow — X cho tắt, Y không: tắt đo → X không đo, Y đo; nhịp bị bỏ ở X không chiếm khung 30 s của người đó.
func TestHeartbeatTwoCoursesDifferentConfig(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	y := r.courseWith(w.adm.ID, "Y"+strings.ToUpper(uuid.NewString()[:6]), false)
	s := w.student(true)
	r.enroll(y, s.u.ID, "STUDENT", "ACTIVE")
	r.exec(`insert into user_settings (user_id, track_study_time) values ($1, false) on conflict (user_id) do update set track_study_time = false`, s.u.ID)
	r.svc.InvalidateTracking(t.Context(), s.u.ID)
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT")) // X: bỏ, KHÔNG chiếm khung
	require.NoError(t, r.beat(s.u.ID, y, "CHAT"))        // Y: cùng khung vẫn đo được
	require.Zero(t, r.secs(w.course, s.u.ID, "CHAT"))
	require.Equal(t, 30, r.secs(y, s.u.ID, "CHAT"))
	r.next()
	require.NoError(t, r.beat(s.u.ID, y, "CHAT"))
	r.rollup()
	require.Zero(t, r.count(`select count(*) from study_time_daily where course_id = $1 and student_id = $2`, w.course, s.u.ID), "không có dòng của X")
	require.Equal(t, 60, r.dbSecs(y, s.u.ID))
	// bật lại → X đo tiếp từ nhịp kế
	r.exec(`update user_settings set track_study_time = true where user_id = $1`, s.u.ID)
	r.svc.InvalidateTracking(t.Context(), s.u.ID)
	r.next()
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
	require.Equal(t, 30, r.secs(w.course, s.u.ID, "CHAT"))
}

func TestDroppedBeatDoesNotTakeWindow(t *testing.T) {
	r := newRig(t)
	w := r.world(true)
	s := w.student(true)
	r.exec(`insert into user_settings (user_id, track_study_time) values ($1, false) on conflict (user_id) do update set track_study_time = false`, s.u.ID)
	r.svc.InvalidateTracking(t.Context(), s.u.ID)
	require.NoError(t, r.beat(s.u.ID, w.course, "CHAT"))
	require.Zero(t, r.rdb.Exists(t.Context(), appredis.Key("hb", "win", s.u.ID.String(), r.windowOf())).Val(), "nhịp bị bỏ vì tracked=false không chiếm khung 30 s")
	require.Zero(t, r.rdb.Exists(t.Context(), appredis.Key("hb", "rate", s.u.ID.String())).Val(), "cũng không tính vào quota tốc độ")
}

func (r *rig) windowOf() string {
	return strconv.FormatInt(r.clk.Now().Unix()/activity.HeartbeatSeconds, 10)
}
