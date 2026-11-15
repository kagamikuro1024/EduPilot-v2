package attendance_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/attendance"
	"github.com/edupilot/backend-go/internal/seedtool"
	"github.com/edupilot/backend-go/internal/store"
)

func (r *rig) flagsOf(wid string, student uuid.UUID) (flags string, assurance int) {
	r.t.Helper()
	var a int
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select array_to_string(flags, ','), assurance from attendance_checkins where window_id = $1 and student_id = $2`, wid, student).Scan(&flags, &a))
	return flags, a
}

func TestGeoFlagsAndAssurance(t *testing.T) {
	r := newRig(t)
	w := r.world(5)
	w.enable(`, "geofence": {"lat": 21.0050, "lng": 105.8430, "radius_m": 100}`)
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	geo := func(lat, lng, acc float64, extra map[string]any) map[string]any {
		g := map[string]any{"lat": lat, "lng": lng, "accuracy_m": acc}
		for k, v := range extra {
			g[k] = v
		}
		return g
	}
	for i, g := range []map[string]any{
		geo(21.0050, 105.8430, 20, nil),                          // trong vùng
		geo(21.0300, 105.8430, 20, nil),                          // ~2,7 km
		geo(21.0050, 105.8430, 500, nil),                         // sai số lớn
		geo(0, 0, 0, map[string]any{"denied": true}),             // từ chối quyền
		geo(21.0050, 105.8430, 20, map[string]any{"mock": true}), // vị trí giả
	} {
		k := newDevkey(t)
		res := w.checkin(w.sv(i), map[string]any{"token": tok, "device": k.device(t, tok, "", w.students[i].ID), "geo": g})
		require.Equal(t, http.StatusCreated, res.code, string(res.body))
	}
	for i, want := range []struct {
		flag      string
		assurance int
	}{{"", 2}, {attendance.FlagGeoOut, 1}, {attendance.FlagGeoWeak, 1}, {attendance.FlagGeoDenied, 1}, {attendance.FlagGeoMock, 2}} {
		fl, a := r.flagsOf(wid, w.students[i].ID)
		if want.flag == "" {
			require.NotContains(t, fl, "GEO", "sv %d", i)
		} else {
			require.Contains(t, fl, want.flag, "sv %d", i)
		}
		require.Equal(t, want.assurance, a, "sv %d: vị trí lỗi hạ về 1; vị trí giả chỉ hạ mức 3 xuống 2", i)
	}
	// Toạ độ không vào DB (chỉ khoảng cách + sai số) và không vào log.
	require.Equal(t, 0, r.count(`select count(*) from attendance_checkins where geo_distance_m is not null and geo_distance_m < 0`))
	require.NotContains(t, r.logs.String(), "105.8430")
	require.NotContains(t, r.logs.String(), "21.0300")
}

func TestRoundsRecheckAndLimit(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	w.enable("")
	closeW := func(wid string) {
		require.Equal(t, http.StatusOK, r.do(req{path: w.cpath("/checkin-windows/" + wid + "/close"), bearer: w.T.access}).code)
	}
	w1 := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, w1)
	for i := range 2 {
		require.Equal(t, http.StatusCreated, w.checkin(w.sv(i), map[string]any{"token": tok}).code)
	}
	closeW(w1)

	// Vòng 2: chỉ sv 0 có mặt → lượt vòng 1 của sv 1 mang RECHECK_MISS; thời lượng mặc định 3 phút.
	o2 := w.open(0)
	require.Equal(t, http.StatusCreated, o2.code, string(o2.body))
	require.EqualValues(t, 2, o2.json()["round"])
	w2 := o2.json()["id"].(string)
	opened, closes := parse(t, w, 0, w2)
	require.Equal(t, 3*time.Minute, closes.Sub(opened).Round(time.Second))
	tok2, _ := w.token(0, w2)
	require.Equal(t, http.StatusCreated, w.checkin(w.sv(0), map[string]any{"token": tok2}).code)
	closeW(w2)
	fl0, _ := r.flagsOf(w1, w.students[0].ID)
	fl1, _ := r.flagsOf(w1, w.students[1].ID)
	require.NotContains(t, fl0, attendance.FlagRecheckMiss)
	require.Contains(t, fl1, attendance.FlagRecheckMiss)

	// Vòng 3 rồi hết: vòng 4 → 409 ROUND_LIMIT.
	w3 := w.open(0)
	require.Equal(t, http.StatusCreated, w3.code)
	closeW(w3.json()["id"].(string))
	lim := w.open(0)
	require.Equal(t, http.StatusConflict, lim.code)
	require.Equal(t, "ROUND_LIMIT", lim.errCode())
	require.Equal(t, 3, r.count(`select count(*) from checkin_windows where session_id = $1`, w.ses[0]))
}

func parse(t *testing.T, w world, i int, wid string) (opened, closes time.Time) {
	list := w.r.call(http.MethodGet, w.path(i, "/checkin-windows"), w.T, nil)
	for _, it := range items(list, "items") {
		if it["id"] == wid {
			var err error
			opened, err = time.Parse(time.RFC3339, it["opened_at"].(string))
			require.NoError(t, err)
			closes, err = time.Parse(time.RFC3339, it["closes_at"].(string))
			require.NoError(t, err)
		}
	}
	return opened, closes
}

func TestSpotCheckPrefersFlagged(t *testing.T) {
	r := newRig(t)
	w := r.world(4)
	w.enable("")
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	k := newDevkey(t)
	for i := range 4 { // sv 0 + 1 cùng khoá (cùng cờ DEVICE_SHARED); sv 2, 3 không khoá (cờ NO_DEVICE_KEY)
		body := map[string]any{"token": tok}
		if i < 2 {
			body["device"] = k.device(t, tok, "", w.students[i].ID)
		}
		require.Equal(t, http.StatusCreated, w.checkin(w.sv(i), body).code)
	}
	// Bỏ cờ của sv 2, 3 → chỉ còn sv 0, 1 còn cờ chưa bỏ.
	for _, i := range []int{2, 3} {
		id := r.scalar(`select id::text from attendance_checkins where window_id = $1 and student_id = $2`, wid, w.students[i].ID)
		require.Equal(t, http.StatusOK, r.do(req{path: w.cpath("/checkins/" + id + "/clear-flags"), bearer: w.T.access}).code)
	}
	spot := func(n int) []map[string]any {
		x := r.do(req{path: w.cpath("/checkin-windows/" + wid + "/spot-check"), bearer: w.T.access, body: map[string]any{"n": n}})
		require.Equal(t, http.StatusOK, x.code, string(x.body))
		return items(x, "students")
	}
	got := spot(2)
	require.Len(t, got, 2)
	for _, g := range got {
		require.Equal(t, "FLAGGED", g["reason"])
	}
	all := spot(10)
	require.Len(t, all, 4, "n lớn hơn số lượt: trả hết, không trùng")
	seen := map[any]bool{}
	for _, g := range all {
		require.False(t, seen[g["student_id"]])
		seen[g["student_id"]] = true
	}
	require.Equal(t, 2, countReason(all, "FLAGGED"))
	require.Equal(t, 2, countReason(all, "RANDOM"))
	require.Equal(t, 0, r.count(`select count(*) from audit_log where action like 'checkin.spot%'`), "không ghi audit")
}

func countReason(xs []map[string]any, reason string) int {
	n := 0
	for _, x := range xs {
		if x["reason"] == reason {
			n++
		}
	}
	return n
}

func TestCheckinsListCursor(t *testing.T) {
	r := newRig(t)
	w := r.world(5)
	w.enable("")
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	for i := range 5 {
		require.Equal(t, http.StatusCreated, w.checkin(w.sv(i), map[string]any{"token": tok}).code)
		r.clk.Advance(time.Second)
	}
	var ids []string
	cursor := ""
	for page := 0; page < 4; page++ {
		path := w.path(0, "/checkins?limit=2") + cursor
		x := r.call(http.MethodGet, path, w.A, nil)
		require.Equal(t, http.StatusOK, x.code, string(x.body))
		require.Contains(t, x.hdr.Get("Cache-Control"), "no-store")
		for _, it := range items(x, "items") {
			ids = append(ids, it["id"].(string))
			require.NotEmpty(t, it["flags"])
		}
		nc, _ := x.json()["next_cursor"].(string)
		if nc == "" {
			break
		}
		cursor = "&cursor=" + nc
	}
	require.Len(t, ids, 5)
	require.Len(t, uniq2(ids), 5, "không trang nào lặp lượt")
	require.Equal(t, http.StatusUnprocessableEntity, r.call(http.MethodGet, w.path(0, "/checkins?limit=101"), w.A, nil).code)
	// Sinh viên không bao giờ thấy danh sách này.
	require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, w.path(0, "/checkins"), w.sv(0), nil).code)
}

func uniq2(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func TestForgetStudentKeepsOthers(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	w.enable("")
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	k := newDevkey(t)
	for i := range 2 {
		require.Equal(t, http.StatusCreated, w.checkin(w.sv(i), map[string]any{"token": tok, "device": k.device(t, tok, "", w.students[i].ID)}).code)
	}
	n, err := attendance.ForgetStudent(t.Context(), store.New(r.pool), w.students[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	require.Equal(t, 0, r.count(`select count(*) from attendance_checkins where student_id = $1`, w.students[0].ID))
	require.Equal(t, 1, r.count(`select count(*) from attendance_checkins where student_id = $1`, w.students[1].ID))
	require.Equal(t, 1, r.count(`select count(*) from user_devices`), "máy còn lượt của người khác thì giữ")
}

func TestLogsNoCheckinSecrets(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	w.enable(`, "geofence": {"lat": 21.0050, "lng": 105.8430, "radius_m": 100}`)
	wid := w.open(0).json()["id"].(string)
	tok, code := w.token(0, wid)
	k := newDevkey(t)
	dev := k.device(t, tok, "", w.students[0].ID)
	require.Equal(t, http.StatusCreated, w.checkin(w.sv(0), map[string]any{"token": tok, "device": dev, "geo": map[string]any{"lat": 21.0051, "lng": 105.8431, "accuracy_m": 20}}).code)
	w.checkin(w.sv(1), map[string]any{"code": "ZZZZZZ"}) // mã sai
	w.checkin(w.sv(1), map[string]any{"token": "x.y.z"})
	logs := r.logs.String()
	for name, secret := range map[string]string{"token": tok, "mã ngắn": code, "chữ ký": dev["signature"].(string), "khoá công khai": k.pub, "kinh độ": "105.8431", "vĩ độ": "21.0051"} {
		require.False(t, strings.Contains(logs, secret), "log không được chứa %s", name)
	}
	aud := r.scalar(`select coalesce(string_agg(coalesce(before::text,'') || coalesce(after::text,''), ' '), '') from audit_log`)
	require.NotContains(t, aud, tok)
	require.NotContains(t, aud, "105.84")
}

// Lệnh dev seed-checkin-demo (US-AS-14 AC1): hai cửa sổ đã đóng ở buổi đầu; vòng 1 đủ ca cờ; vòng 2 + RECHECK_MISS; over_headcount; idempotent; vẫn phải đánh vắng mới hoàn tất.
func TestSeedCheckinDemo(t *testing.T) {
	r := newRig(t)
	w := r.world(24)
	n, err := seedtool.SeedCheckinDemo(t.Context(), r.pool, w.course)
	require.NoError(t, err)
	require.EqualValues(t, 22+21, n) // vòng 1: 22 lượt; vòng 2: mọi người trừ người cuối
	count := func(q string) string { return r.scalar(strings.ReplaceAll(q, "$C", "'"+w.course.String()+"'")) }
	require.Equal(t, "2", count(`select count(*) from checkin_windows where course_id = $C and closed_at is not null`))
	require.Equal(t, "22", count(`select count(*) from attendance_checkins c join checkin_windows w on w.id = c.window_id where c.course_id = $C and w.round = 1`))
	for flag, want := range map[string]string{"DEVICE_SHARED": "2", "DEVICE_NEW": "1", "NO_DEVICE_KEY": "1", "TOKEN_LATE": "1", "RECHECK_MISS": "1"} {
		require.Equal(t, want, count(`select count(*) from attendance_checkins where course_id = $C and '`+flag+`' = any(flags)`), flag)
	}
	require.Equal(t, "1", count(`select count(*) from attendance_records where course_id = $C and status = 'LATE'`))
	require.Equal(t, "1", count(`select count(*) from checkin_windows where course_id = $C and headcount < (select count(*) from attendance_checkins c where c.window_id = checkin_windows.id)`), "over_headcount")
	require.Equal(t, "true", count(`select (settings->'attendance'->>'self_checkin') from courses where id = $C`))

	again, err := seedtool.SeedCheckinDemo(t.Context(), r.pool, w.course) // chạy lần hai: số hàng không đổi
	require.NoError(t, err)
	require.EqualValues(t, 0, again)
	require.Equal(t, "43", count(`select count(*) from attendance_checkins where course_id = $C`))

	svc := &attendance.Service{Pool: r.pool, Redis: r.rdb, Clock: r.clk, Cfg: r.cfg.Attendance}
	review, err := svc.ReviewSessions(t.Context(), []uuid.UUID{w.course})
	require.NoError(t, err)
	require.Len(t, review, 1)
	require.Equal(t, 6, review[0].Flagged)
	g := w.put(w.T, 0, nil, true)
	require.Equal(t, "CHECKIN_UNRESOLVED", g.errCode())
	require.EqualValues(t, 2, g.details()["count"]) // 24 sinh viên, 22 có lượt
}
