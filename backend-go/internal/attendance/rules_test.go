package attendance_test

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/attendance"
)

// Sáu test của SRS FEAT-attendance-selfcheck §11 «Chưa có test» (P-11-43, DEF-R10.8-02).

func (w world) openWith(i int, body map[string]any) resp {
	return w.r.do(req{path: w.path(i, "/checkin-windows"), bearer: w.T.access, hdr: map[string]string{"Idempotency-Key": uuid.NewString()}, body: body})
}

func (w world) closeWindow(wid string) resp {
	return w.r.do(req{path: w.cpath("/checkin-windows/" + wid + "/close"), bearer: w.T.access})
}

// US-AS-02 AC1–3, AC8.
func TestWindowOpenRules(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	w.enable("")

	// minutes ngoài [1, 60] → 422; trong khoảng → 201 đúng thời lượng.
	for _, m := range []int{0, -5, 61, 600} {
		res := w.openWith(0, map[string]any{"minutes": m})
		require.Equal(t, http.StatusUnprocessableEntity, res.code, "minutes=%d: %s", m, res.body)
	}
	require.Zero(t, r.count(`select count(*) from checkin_windows where session_id = $1`, w.ses[0]), "422 không mở cửa sổ")
	edge := w.openWith(0, map[string]any{"minutes": 60})
	require.Equal(t, http.StatusCreated, edge.code, string(edge.body))
	o, c := parse(t, w, 0, edge.json()["id"].(string))
	require.Equal(t, 60*time.Minute, c.Sub(o).Round(time.Second))
	require.Equal(t, http.StatusOK, w.closeWindow(edge.json()["id"].(string)).code)

	// Mặc định: vòng 2 (cửa sổ này) 3 phút; vòng 1 mặc định 10 phút ở buổi khác.
	r2 := w.open(0)
	require.Equal(t, http.StatusCreated, r2.code, string(r2.body))
	require.EqualValues(t, 2, r2.json()["round"])
	o, c = parse(t, w, 0, r2.json()["id"].(string))
	require.Equal(t, 3*time.Minute, c.Sub(o).Round(time.Second))
	// Mở vòng ≥ 2 không sinh sự kiện outbox nào của lớp (không thông báo tới sinh viên).
	require.Zero(t, r.count(`select count(*) from outbox where payload->>'course_id' = $1`, w.course.String()))
	require.Equal(t, http.StatusOK, w.closeWindow(r2.json()["id"].(string)).code)
	r1 := w.open(1)
	require.Equal(t, http.StatusCreated, r1.code, string(r1.body))
	o, c = parse(t, w, 1, r1.json()["id"].(string))
	require.Equal(t, 10*time.Minute, c.Sub(o).Round(time.Second))
	require.Equal(t, http.StatusOK, w.closeWindow(r1.json()["id"].(string)).code)

	// Hai yêu cầu mở song song (khác Idempotency-Key) trên buổi chưa có cửa sổ → đúng một 201 và một 409.
	before := r.count(`select count(*) from outbox where payload->>'course_id' = $1`, w.course.String())
	var wg sync.WaitGroup
	codes := make([]int, 2)
	errs := make([]string, 2)
	start := make(chan struct{})
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			res := w.openWith(1, map[string]any{})
			codes[i], errs[i] = res.code, res.errCode()
		}()
	}
	close(start)
	wg.Wait()
	require.ElementsMatch(t, []int{http.StatusCreated, http.StatusConflict}, codes, "%v %v", codes, errs)
	for i, c := range codes {
		if c == http.StatusConflict {
			require.Equal(t, "WINDOW_ALREADY_OPEN", errs[i])
		}
	}
	require.Equal(t, before, r.count(`select count(*) from outbox where payload->>'course_id' = $1`, w.course.String()), "vòng ≥ 2: không outbox")
	require.Equal(t, 1, r.count(`select count(*) from checkin_windows where session_id = $1 and closed_at is null`, w.ses[1]))
}

// US-AS-02 AC6 + US-AS-07 AC1.
func TestOverHeadcount(t *testing.T) {
	// công thức > headcount + max(2, ⌈5 %·headcount⌉), cả hai phía biên
	for _, c := range []struct {
		head, checked int
		over          bool
	}{
		{0, 2, false}, {0, 3, true}, // dung sai tối thiểu 2
		{10, 12, false}, {10, 13, true},
		{40, 42, false}, {40, 43, true}, // 5 % = 2,0 → 2
		{41, 44, false}, {41, 45, true}, // 5 % = 2,05 → 3
		{100, 105, false}, {100, 106, true},
		{2000, 2100, false}, {2000, 2101, true},
	} {
		h := c.head
		require.Equal(t, c.over, attendance.OverHeadcount(c.checked, &h), "headcount %d, điểm danh %d", c.head, c.checked)
	}
	require.False(t, attendance.OverHeadcount(9999, nil), "chưa nhập số đếm → không bao giờ vượt")

	r := newRig(t)
	w := r.world(13)
	w.enable("")
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	patch := func(s session, body any) resp {
		return r.call(http.MethodPatch, w.cpath("/checkin-windows/"+wid), s, body)
	}
	overOf := func() bool {
		for _, it := range items(r.call(http.MethodGet, w.path(0, "/checkin-windows"), w.T, nil), "items") {
			if it["id"] == wid {
				return it["over_headcount"].(bool)
			}
		}
		t.Fatal("cửa sổ không có trong danh sách")
		return false
	}
	// biên PATCH: 0…2000 hợp lệ; ngoài → 422; null xoá; sinh viên → 403
	for _, bad := range []int{-1, 2001} {
		require.Equal(t, http.StatusUnprocessableEntity, patch(w.T, map[string]any{"headcount": bad}).code, "headcount=%d", bad)
	}
	require.Equal(t, http.StatusForbidden, patch(w.sv(0), map[string]any{"headcount": 10}).code)
	for _, ok := range []int{0, 2000} {
		require.Equal(t, http.StatusOK, patch(w.T, map[string]any{"headcount": ok}).code)
	}
	require.Equal(t, http.StatusOK, patch(w.T, map[string]any{"headcount": nil}).code)
	require.False(t, overOf())
	require.Equal(t, http.StatusOK, patch(w.A, map[string]any{"headcount": 10}).code, "TA cũng đặt được")
	require.Equal(t, 4, r.count(`select count(*) from audit_log where action = 'checkin_window.headcount' and entity_id = $1`, wid), "mỗi lần đổi một dòng audit: 0, 2000, null, 10")

	// 12 người điểm danh, đếm 10: 12 > 12 sai; người thứ 13 → vượt
	for i := range 12 {
		require.Equal(t, http.StatusCreated, w.checkin(w.sv(i), map[string]any{"token": tok}).code)
	}
	require.False(t, overOf(), "12 = 10 + 2: chưa vượt")
	require.Equal(t, http.StatusCreated, w.checkin(w.sv(12), map[string]any{"token": tok}).code)
	require.True(t, overOf(), "13 > 12")
	res := patch(w.T, map[string]any{"headcount": 11})
	require.Equal(t, http.StatusOK, res.code)
	require.False(t, overOf(), "13 = 11 + 2")
}

// US-AS-02 AC4 + AC8.
func TestDisplayNoStore(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	w.enable("")
	wid := w.open(0).json()["id"].(string)
	d := w.display(w.T, 0, wid)
	require.Equal(t, http.StatusOK, d.code, string(d.body))
	require.Contains(t, d.hdr.Get("Cache-Control"), "no-store")
	require.Equal(t, "no-referrer", d.hdr.Get("Referrer-Policy"))
	require.Equal(t, http.StatusForbidden, w.display(w.sv(0), 0, wid).code)
	require.NotContains(t, w.display(w.sv(0), 0, wid).hdr.Get("Cache-Control"), "public")

	// quá 30 lần / phút / người → 429 (người khác không bị ảnh hưởng)
	cfgN := 30 // DISPLAY_RATE_PER_MIN mặc định
	for i := 2; i <= cfgN; i++ { // đã dùng 1 lần ở trên (403 của sinh viên tính cho sinh viên, không cho GV)
		require.Equal(t, http.StatusOK, w.display(w.T, 0, wid).code, "lần %d", i)
	}
	over := w.display(w.T, 0, wid)
	require.Equal(t, http.StatusTooManyRequests, over.code)
	require.Equal(t, "RATE_LIMITED", over.errCode())
	require.NotEmpty(t, over.hdr.Get("Retry-After"))
	require.Equal(t, http.StatusOK, w.display(w.A, 0, wid).code, "hạn mức theo người")

	// cửa sổ đã đóng → 409 WINDOW_CLOSED (người khác để không dính hạn mức)
	require.Equal(t, http.StatusOK, w.closeWindow(wid).code)
	closed := w.display(w.A, 0, wid)
	require.Equal(t, http.StatusConflict, closed.code)
	require.Equal(t, "WINDOW_CLOSED", closed.errCode())
}

// US-AS-04: hai sinh viên cùng khoá, đồng thời → cả hai lượt mang DEVICE_SHARED.
func TestDeviceSharedRace(t *testing.T) {
	r := newRig(t)
	const n = 6
	w := r.world(n)
	w.enable("")
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	k := newDevkey(t)
	sess := make([]session, n)
	for i := range sess {
		sess[i] = w.sv(i)
	}
	codes := make([]int, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			codes[i] = w.checkin(sess[i], map[string]any{"token": tok, "device": k.device(t, tok, "", w.students[i].ID)}).code
		}()
	}
	close(start)
	wg.Wait()
	for i, c := range codes {
		require.Equal(t, http.StatusCreated, c, "sv %d", i)
	}
	for i := range n {
		fl, _ := r.flagsOf(wid, w.students[i].ID)
		require.Contains(t, fl, attendance.FlagDeviceShared, "sv %d mất cờ khi điểm danh đồng thời", i)
	}
}

// US-AS-10 AC mạng.
func TestFlagCampus(t *testing.T) {
	r := newRig(t)
	w := r.world(4)
	sess := []session{w.sv(0), w.sv(1), w.sv(2), w.sv(3)} // đăng nhập xong rồi mới đổi IP
	inNet, outNet := "198.51.100.7", "203.0.113.9"

	// chưa cấu hình dải mạng: không cờ, in_campus rỗng
	w.enable("")
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	r.ip = outNet
	require.Equal(t, http.StatusCreated, w.checkin(sess[0], map[string]any{"token": tok}).code)
	fl, _ := r.flagsOf(wid, w.students[0].ID)
	require.NotContains(t, fl, attendance.FlagOffCampus)
	require.Zero(t, r.count(`select count(*) from attendance_checkins where window_id = $1 and in_campus is not null`, wid))
	require.Equal(t, http.StatusOK, w.closeWindow(wid).code)

	// có dải mạng: trong dải → in_campus=true, không cờ; ngoài dải → OFF_CAMPUS, in_campus=false
	w.enable(`, "campus_cidrs": ["198.51.100.0/24"]`)
	wid2 := w.open(0).json()["id"].(string)
	tok2, _ := w.token(0, wid2)
	r.ip = inNet
	require.Equal(t, http.StatusCreated, w.checkin(sess[1], map[string]any{"token": tok2}).code)
	r.ip = outNet
	require.Equal(t, http.StatusCreated, w.checkin(sess[2], map[string]any{"token": tok2}).code)
	f1, _ := r.flagsOf(wid2, w.students[1].ID)
	f2, _ := r.flagsOf(wid2, w.students[2].ID)
	require.NotContains(t, f1, attendance.FlagOffCampus)
	require.Contains(t, f2, attendance.FlagOffCampus)
	require.Equal(t, "true", r.scalar(`select in_campus::text from attendance_checkins where window_id = $1 and student_id = $2`, wid2, w.students[1].ID))
	require.Equal(t, "false", r.scalar(`select in_campus::text from attendance_checkins where window_id = $1 and student_id = $2`, wid2, w.students[2].ID))
	// nhân viên thấy in_campus + nhãn trong danh sách lượt; sinh viên không thấy cờ
	r.ip = "203.0.113.200"
	list := r.do(req{method: http.MethodGet, path: w.path(0, "/checkins"), bearer: w.A.access})
	require.Equal(t, http.StatusOK, list.code, string(list.body))
	require.Contains(t, string(list.body), `"in_campus":false`)
	require.Contains(t, string(list.body), attendance.FlagOffCampus)
}

// US-AS-10: không toạ độ và không IP trong DB, log, audit.
func TestNoCoordsOrIPStored(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	sess := []session{w.sv(0), w.sv(1)} // đăng nhập (ghi IP của rig vào phiên) TRƯỚC khi đổi sang IP dùng riêng cho điểm danh
	w.enable(`, "campus_cidrs": ["198.51.100.0/24"], "geofence": {"lat": 21.0050, "lng": 105.8430, "radius_m": 100}`)
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	const ip, ip2 = "198.51.100.77", "203.0.113.77"
	r.logs.Reset()
	r.ip = ip
	k := newDevkey(t)
	require.Equal(t, http.StatusCreated, w.checkin(sess[0], map[string]any{"token": tok, "device": k.device(t, tok, "", w.students[0].ID), "geo": map[string]any{"lat": 21.00517, "lng": 105.84317, "accuracy_m": 20}}).code)
	r.ip = ip2
	require.Equal(t, http.StatusCreated, w.checkin(sess[1], map[string]any{"token": tok, "geo": map[string]any{"lat": 21.00531, "lng": 105.84329, "accuracy_m": 20}}).code)

	secrets := []string{ip, ip2, "21.00517", "105.84317", "21.00531", "105.84329"}
	for _, s := range secrets {
		require.NotContains(t, r.logs.String(), s, "log")
	}
	// mọi bảng của lược đồ public: không dòng nào chứa IP hay toạ độ thô (toàn dòng cast sang text)
	rows, err := r.pool.Query(t.Context(), `select table_name from information_schema.tables where table_schema = 'public' and table_type = 'BASE TABLE' order by 1`)
	require.NoError(t, err)
	var tables []string
	for rows.Next() {
		var n string
		require.NoError(t, rows.Scan(&n))
		tables = append(tables, n)
	}
	rows.Close()
	require.NotEmpty(t, tables)
	for _, tb := range tables {
		for _, s := range secrets {
			n := r.count(fmt.Sprintf(`select count(*) from %q as t where t::text like $1`, tb), "%"+strings.ReplaceAll(s, "%", "")+"%")
			require.Zero(t, n, "bảng %s chứa %q", tb, s)
		}
	}
	// chỉ còn khoảng cách + sai số + in_campus (đã tính, không lưu IP)
	require.Equal(t, 2, r.count(`select count(*) from attendance_checkins where window_id = $1 and geo_distance_m is not null and geo_accuracy_m is not null and in_campus is not null`, wid))
	require.Equal(t, 0, r.count(`select count(*) from attendance_checkins where geo_distance_m < 0`))
}
