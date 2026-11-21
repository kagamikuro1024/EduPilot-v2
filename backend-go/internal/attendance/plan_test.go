package attendance_test

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/attendance"
)

// Các test kế hoạch của SRS FEAT-attendance-selfcheck §9 (P-11-43): mỗi test một hành vi, không gộp vào TestCheckinFlow.

// P-108-13 / US-AS-03 AC9: thân có khoá lạ → 422 VALIDATION_FAILED, details[].code = unknown; danh tính chỉ từ JWT (không có bản ghi nào được tạo).
func TestCheckinRejectsUnknownBody(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	w.enable("")
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	for _, extra := range []string{"student_id", "user_id", "student_code", "khoa_la"} {
		res := w.checkin(w.sv(0), map[string]any{"token": tok, extra: w.students[1].ID.String()})
		require.Equal(t, http.StatusUnprocessableEntity, res.code, extra+": "+string(res.body))
		require.Equal(t, "VALIDATION_FAILED", res.errCode(), extra)
		var found bool
		for _, d := range res.json()["details"].([]any) {
			m := d.(map[string]any)
			found = found || (m["field"] == extra && m["code"] == "unknown")
		}
		require.True(t, found, "details phải nêu %q là unknown: %s", extra, res.body)
	}
	require.Equal(t, 0, r.count(`select count(*) from attendance_checkins where window_id = $1`, wid), "thân bị từ chối thì không ghi gì")
}

// FR-12: lượt dùng token cũ hơn một bước và quá hạn nới (TOKEN_LATE_EXTRA_MS = 5 s) mang cờ TOKEN_LATE; lượt dùng token còn mới thì không.
func TestFlagTokenLate(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	w.enable("") // rotate_seconds = 15
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	flags := func(i int) string {
		return r.scalar(`select array_to_string(flags, ',') from attendance_checkins where window_id = $1 and student_id = $2`, wid, w.students[i].ID)
	}
	step := r.clk.Now().Unix() / 15
	require.Equal(t, http.StatusCreated, w.checkin(w.sv(0), map[string]any{"token": tok}).code)
	require.NotContains(t, flags(0), attendance.FlagTokenLate, "lượt đầu bước: tuổi token < 15 s")

	// Sang bước kế, 8 s sau mốc đổi bước ⇒ tuổi token = 15 + 8 = 23 s > 15 + 5 s, vẫn trong «cũ hơn một bước».
	r.clk.Advance(time.Unix((step+1)*15+8, 0).Sub(r.clk.Now()))
	res := w.checkin(w.sv(1), map[string]any{"token": tok})
	require.Equal(t, http.StatusCreated, res.code, string(res.body))
	require.Contains(t, flags(1), attendance.FlagTokenLate)
}

// US-AS-03: F5 / hai tab / yêu cầu song song cùng lượt → đúng MỘT bản ghi, mọi phản hồi cùng một giờ điểm danh.
func TestCheckinRace(t *testing.T) {
	r := newRig(t, func(env map[string]string) { env["CHECKIN_RATE_PER_MIN"] = "1000" }) // chặn tốc độ có test riêng; ở đây chỉ đo tranh chấp ghi
	w := r.world(1)
	w.enable("")
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	s := w.sv(0)
	const n = 12
	var wg sync.WaitGroup
	start := make(chan struct{})
	res := make([]resp, n)
	for i := range res {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			res[i] = w.checkin(s, map[string]any{"token": tok})
		}()
	}
	close(start)
	wg.Wait()
	created := 0
	for i, x := range res {
		require.Contains(t, []int{http.StatusCreated, http.StatusOK}, x.code, "lượt %d: %s", i, x.body)
		if x.code == http.StatusCreated {
			created++
		}
		require.Equal(t, res[0].json()["checked_in_at"], x.json()["checked_in_at"], "lượt %d", i)
	}
	require.Equal(t, 1, created, "đúng một lượt được tạo (201), còn lại phát lại (200)")
	require.Equal(t, 1, r.count(`select count(*) from attendance_checkins where window_id = $1 and student_id = $2`, wid, w.students[0].ID))
}

// US-AS-05: đóng cửa sổ đồng thời với lưu lưới điểm danh (cùng buổi) — không mất dòng, không ghi đôi, cả hai 200, thứ tự nào cũng cùng kết quả cuối.
func TestCloseVsGridRace(t *testing.T) {
	r := newRig(t)
	for round := 0; round < 6; round++ {
		w := r.world(4)
		w.enable("")
		wid := w.open(0).json()["id"].(string)
		tok, _ := w.token(0, wid)
		for _, i := range []int{0, 1} {
			require.Equal(t, http.StatusCreated, w.checkin(w.sv(i), map[string]any{"token": tok}).code)
		}
		grid := []rec{
			{"student_id": w.students[0].ID.String(), "status": "PRESENT"}, // trùng với dòng do «Đóng» ghi
			{"student_id": w.students[2].ID.String(), "status": "ABSENT"},
			{"student_id": w.students[3].ID.String(), "status": "EXCUSED"},
		}
		var cl, pu resp
		var wg sync.WaitGroup
		start := make(chan struct{})
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			cl = r.do(req{path: w.cpath("/checkin-windows/" + wid + "/close"), bearer: w.T.access})
		}()
		go func() {
			defer wg.Done()
			<-start
			pu = w.put(w.T, 0, grid, false)
		}()
		close(start)
		wg.Wait()
		require.Equal(t, http.StatusOK, cl.code, "vòng %d đóng: %s", round, cl.body)
		require.Equal(t, http.StatusOK, pu.code, "vòng %d lưu lưới: %s", round, pu.body)
		status := func(i int) string {
			return r.scalar(`select status::text from attendance_records where session_id = $1 and student_id = $2`, w.ses[0], w.students[i].ID)
		}
		require.Equal(t, 4, r.count(`select count(*) from attendance_records where session_id = $1`, w.ses[0]), "vòng %d: mỗi sinh viên đúng một dòng", round)
		require.Equal(t, []string{"PRESENT", "PRESENT", "ABSENT", "EXCUSED"}, []string{status(0), status(1), status(2), status(3)}, "vòng %d", round)
	}
}

// US-AS-13: purge không xoá lượt thô của cửa sổ còn mở, dù đã quá hạn giữ dữ liệu; chỉ cửa sổ đã đóng quá hạn mới bị xoá.
func TestPurgeSkipsOpenWindow(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	w.enable("")
	svc := &attendance.Service{Pool: r.pool, Redis: r.rdb, Clock: r.clk, Cfg: r.cfg.Attendance}
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	require.Equal(t, http.StatusCreated, w.checkin(w.sv(0), map[string]any{"token": tok}).code)

	r.clk.Advance(60 * 24 * time.Hour)
	_, _, err := svc.PurgeCheckins(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from attendance_checkins where window_id = $1`, wid), "cửa sổ chưa đóng: giữ lượt")

	teacher := r.mustLogin(w.teacher.Email) // đồng hồ đã nhảy 60 ngày: phiên cũ hết hạn
	require.Equal(t, http.StatusOK, r.do(req{path: w.cpath("/checkin-windows/" + wid + "/close"), bearer: teacher.access}).code)
	r.clk.Advance(31 * 24 * time.Hour)
	_, _, err = svc.PurgeCheckins(t.Context())
	require.NoError(t, err)
	require.Equal(t, 0, r.count(`select count(*) from attendance_checkins where window_id = $1`, wid), "đóng rồi và quá hạn: xoá")
}

// US-AS-03: mã ngắn gắn với cửa sổ của ĐÚNG lớp — mã lớp A không điểm danh được ở lớp B; mã vòng cũ (cửa sổ đã đóng) → WINDOW_CLOSED; mã hết hạn → CHECKIN_TOKEN_INVALID.
func TestCheckinByCodeIsolation(t *testing.T) {
	r := newRig(t)
	a, b := r.world(2), r.world(2)
	a.enable("")
	b.enable("")
	widA := a.open(0).json()["id"].(string)
	widB := b.open(0).json()["id"].(string)
	_, codeA := a.token(0, widA)
	_, codeB := b.token(0, widB)
	require.NotEqual(t, codeA, codeB)

	// Mã của A ở lớp B (sinh viên của B): không lẫn.
	res := b.checkin(b.sv(0), map[string]any{"code": codeA})
	require.Equal(t, "CHECKIN_TOKEN_INVALID", res.errCode(), string(res.body))
	require.Equal(t, 0, r.count(`select count(*) from attendance_checkins where window_id = $1`, widB))
	require.Equal(t, 0, r.count(`select count(*) from attendance_checkins where window_id = $1`, widA))
	// Mã đúng lớp dùng được.
	require.Equal(t, http.StatusCreated, b.checkin(b.sv(0), map[string]any{"code": codeB}).code)
	require.Equal(t, http.StatusCreated, a.checkin(a.sv(0), map[string]any{"code": codeA}).code)

	// Đóng cửa sổ A rồi mở vòng 2: mã vòng 1 → WINDOW_CLOSED; mã vòng 2 vào đúng cửa sổ mới.
	require.Equal(t, http.StatusOK, r.do(req{path: a.cpath("/checkin-windows/" + widA + "/close"), bearer: a.T.access}).code)
	o2 := a.open(0)
	require.Equal(t, http.StatusCreated, o2.code, string(o2.body))
	wid2 := o2.json()["id"].(string)
	_, code2 := a.token(0, wid2)
	require.Equal(t, "WINDOW_CLOSED", a.checkin(a.sv(1), map[string]any{"code": codeA}).errCode())
	require.Equal(t, http.StatusCreated, a.checkin(a.sv(1), map[string]any{"code": code2}).code)
	require.Equal(t, 1, r.count(`select count(*) from attendance_checkins where window_id = $1 and student_id = $2`, wid2, a.students[1].ID))

	// Mã quá hai bước → không còn khớp.
	r.clk.Advance(50 * time.Second)
	require.Equal(t, "CHECKIN_TOKEN_INVALID", b.checkin(b.sv(1), map[string]any{"code": codeB}).errCode())
}
