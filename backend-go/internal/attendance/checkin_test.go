package attendance_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/attendance"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/today"
)

// enable bật điểm danh tự phục vụ cho lớp (settings.attendance), thêm `extra` JSON vào chính sách.
func (w world) enable(extra string) {
	body := `{"self_checkin": true, "rotate_seconds": 15, "late_after_min": 15, "require": "WEB"` + extra + `}`
	w.r.exec(`update courses set settings = settings || jsonb_build_object('attendance', $2::jsonb) where id = $1`, w.course, body)
}

func (w world) cpath(tail string) string { return "/courses/" + w.course.String() + tail }

func (w world) open(i int) resp {
	return w.r.do(req{path: w.path(i, "/checkin-windows"), bearer: w.T.access, hdr: map[string]string{"Idempotency-Key": uuid.NewString()}, body: map[string]any{}})
}

func (w world) display(s session, i int, wid string) resp {
	return w.r.call(http.MethodGet, w.path(i, "/checkin-windows/"+wid+"/display"), s, nil)
}

func (w world) checkin(s session, body any) resp {
	return w.r.call(http.MethodPost, w.cpath("/checkins"), s, body)
}

// token đọc token đang hiệu lực của cửa sổ từ khung hiển thị của giảng viên.
func (w world) token(i int, wid string) (tok, code string) {
	d := w.display(w.T, i, wid)
	require.Equal(w.r.t, http.StatusOK, d.code, string(d.body))
	j := d.json()
	_, tok, _ = strings.Cut(j["qr_url"].(string), "#")
	return tok, strings.ReplaceAll(j["short_code"].(string), " ", "")
}

func (w world) sv(i int) session { return w.r.mustLogin(w.students[i].Email) }

type devkey struct {
	k   *ecdsa.PrivateKey
	pub string
}

func newDevkey(t *testing.T) devkey {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	spki, err := x509.MarshalPKIXPublicKey(&k.PublicKey)
	require.NoError(t, err)
	return devkey{k, base64.StdEncoding.EncodeToString(spki)}
}

// device ký lượt của sinh viên `uid` (đường token nếu code == "").
func (d devkey) device(t *testing.T, token, code string, uid uuid.UUID) map[string]any {
	norm := code
	h := sha256.Sum256(attendance.SignedMessage(token, norm, uid))
	sig, err := ecdsa.SignASN1(rand.Reader, d.k, h[:])
	require.NoError(t, err)
	return map[string]any{"kind": "WEB_KEY", "public_key": d.pub, "signature": base64.StdEncoding.EncodeToString(sig)}
}

func TestCheckinFlow(t *testing.T) {
	r := newRig(t)
	w := r.world(3)
	s1, s2 := w.sv(0), w.sv(1)

	// Lớp chưa bật → SELF_CHECKIN_DISABLED; bật nhưng chưa mở cửa sổ → NO_WINDOW.
	require.Equal(t, "SELF_CHECKIN_DISABLED", w.checkin(s1, map[string]any{"code": "AAAAAA"}).errCode())
	w.enable("")
	require.Equal(t, "CHECKIN_TOKEN_INVALID", w.checkin(s1, map[string]any{"code": "AAAAAA"}).errCode(), "chưa có cửa sổ nào khớp mã")

	o := w.open(0)
	require.Equal(t, http.StatusCreated, o.code, string(o.body))
	wid := o.json()["id"].(string)
	dup := w.open(0)
	require.Equal(t, "WINDOW_ALREADY_OPEN", dup.errCode())
	require.Equal(t, wid, dup.details()["window_id"])

	// Sinh viên không đọc được khung hiển thị; TA / GV gọi đường của sinh viên → 403.
	require.Equal(t, http.StatusForbidden, w.display(s1, 0, wid).code)
	require.Equal(t, http.StatusForbidden, w.checkin(w.T, map[string]any{"code": "AAAAAA"}).code)
	require.Equal(t, http.StatusForbidden, w.checkin(w.OUT, map[string]any{"code": "AAAAAA"}).code)
	// Chưa ghi danh ACTIVE (PENDING) / đã bị mời ra: guard chặn trước handler — 403 FORBIDDEN, details.reason = course (không có mã NOT_IN_COURSE riêng; P-108-11).
	pend := r.mustLogin(w.pending.Email)
	pc := w.checkin(pend, map[string]any{"code": "AAAAAA"})
	require.Equal(t, http.StatusForbidden, pc.code)
	require.Equal(t, "FORBIDDEN", pc.errCode())
	require.Equal(t, "course", pc.details()["reason"])
	gone := r.addUserRemoved(w)
	require.Equal(t, http.StatusForbidden, w.checkin(r.mustLogin(gone.Email), map[string]any{"code": "AAAAAA"}).code)

	tok, code := w.token(0, wid)
	k := newDevkey(t)
	first := w.checkin(s1, map[string]any{"token": tok, "device": k.device(t, tok, "", w.students[0].ID)})
	require.Equal(t, http.StatusCreated, first.code, string(first.body))
	require.Len(t, first.json(), 2, "chỉ checked_in_at + round — không cờ, không mức tin cậy")
	require.Contains(t, first.hdr.Get("Cache-Control"), "no-store")
	again := w.checkin(s1, map[string]any{"token": tok})
	require.Equal(t, http.StatusOK, again.code)
	require.Equal(t, first.json()["checked_in_at"], again.json()["checked_in_at"])

	// Đường mã ngắn; mã sai → CHECKIN_TOKEN_INVALID kiểu mã; token rác → CHECKIN_TOKEN_INVALID.
	require.Equal(t, http.StatusCreated, w.checkin(s2, map[string]any{"code": code}).code)
	require.NotEmpty(t, w.checkin(s2, map[string]any{"code": "ZZZZZZ"}).errCode())
	require.NotEmpty(t, w.checkin(w.sv(2), map[string]any{"token": "x.y.z"}).errCode())
	require.Equal(t, 2, r.count(`select count(*) from attendance_checkins where window_id = $1`, wid))

	// Token cũ hơn một bước vẫn nhận; hai bước thì CHECKIN_TOKEN_EXPIRED.
	r.clk.Advance(5 * time.Second)
	require.Equal(t, http.StatusCreated, w.checkin(w.sv(2), map[string]any{"token": tok}).code)
	r.clk.Advance(35 * time.Second)
	require.Equal(t, "CHECKIN_TOKEN_EXPIRED", w.checkin(w.sv(2), map[string]any{"token": tok}).errCode())
}

func TestCheckinFlags(t *testing.T) {
	r := newRig(t)
	w := r.world(3)
	w.enable("")
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	k := newDevkey(t)
	for _, i := range []int{0, 1} { // cùng một khoá thiết bị, hai tài khoản
		require.Equal(t, http.StatusCreated, w.checkin(w.sv(i), map[string]any{"token": tok, "device": k.device(t, tok, "", w.students[i].ID)}).code)
	}
	require.Equal(t, http.StatusCreated, w.checkin(w.sv(2), map[string]any{"token": tok}).code) // không khoá

	flags := func(i int) string {
		return r.scalar(`select array_to_string(flags, ',') from attendance_checkins where window_id = $1 and student_id = $2`, wid, w.students[i].ID)
	}
	require.Contains(t, flags(1), attendance.FlagDeviceShared)
	require.Contains(t, flags(0), attendance.FlagDeviceShared, "cờ gắn cả lượt trước của người dùng chung máy")
	require.NotContains(t, flags(0), attendance.FlagDeviceNew, "lượt đầu với khoá mới không bị DEVICE_NEW")
	require.Contains(t, flags(2), attendance.FlagNoDeviceKey)

	// Từ lượt thứ hai, khoá khác với khoá đã dùng → DEVICE_NEW.
	wid2 := w.open(1).json()["id"].(string)
	tok2, _ := w.token(1, wid2)
	k2 := newDevkey(t)
	require.Equal(t, http.StatusCreated, w.checkin(w.sv(0), map[string]any{"token": tok2, "device": k2.device(t, tok2, "", w.students[0].ID)}).code)
	require.Contains(t, r.scalar(`select array_to_string(flags, ',') from attendance_checkins where window_id = $1 and student_id = $2`, wid2, w.students[0].ID), attendance.FlagDeviceNew)

	list := r.do(req{method: http.MethodGet, path: w.path(0, "/checkins"), bearer: w.A.access})
	require.Equal(t, http.StatusOK, list.code, string(list.body))
	require.Contains(t, string(list.body), "Cùng máy với bạn khác")

	// Staff bỏ cờ: ghi người bỏ + giờ; audit không có toạ độ.
	id := r.scalar(`select id::text from attendance_checkins where window_id = $1 and student_id = $2`, wid, w.students[1].ID)
	c := r.do(req{path: w.cpath("/checkins/" + id + "/clear-flags"), bearer: w.A.access})
	require.Equal(t, http.StatusOK, c.code, string(c.body))
	require.Equal(t, 1, r.count(`select count(*) from attendance_checkins where id = $1 and flags_cleared_by is not null`, id))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where action = 'checkin.clear_flags'`))
}

func TestCloseAbsentComplete(t *testing.T) {
	r := newRig(t)
	w := r.world(4)
	w.enable("")
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	for _, i := range []int{0, 1} {
		require.Equal(t, http.StatusCreated, w.checkin(w.sv(i), map[string]any{"token": tok}).code)
	}
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": uuid.NewString()} }
	require.Equal(t, "WINDOW_OPEN", r.do(req{path: w.path(0, "/absent-unchecked"), bearer: w.T.access, hdr: idem()}).errCode())
	require.Equal(t, "NO_WINDOW", r.do(req{path: w.path(1, "/absent-unchecked"), bearer: w.T.access, hdr: idem()}).errCode())
	// Đóng: 2 dòng PRESENT; đóng lần hai không ghi thêm.
	cl := r.do(req{path: w.cpath("/checkin-windows/" + wid + "/close"), bearer: w.T.access})
	require.Equal(t, http.StatusOK, cl.code, string(cl.body))
	require.EqualValues(t, 2, cl.json()["unchecked"])
	require.Equal(t, 2, r.count(`select count(*) from attendance_records where session_id = $1 and status = 'PRESENT'`, w.ses[0]))
	require.Equal(t, http.StatusOK, r.do(req{path: w.cpath("/checkin-windows/" + wid + "/close"), bearer: w.T.access}).code)
	require.Equal(t, 2, r.count(`select count(*) from attendance_records where session_id = $1`, w.ses[0]))
	require.Equal(t, "WINDOW_CLOSED", w.checkin(w.sv(2), map[string]any{"token": tok}).errCode())

	// Hoàn tất khi còn 2 người chưa có dòng → 422 CHECKIN_UNRESOLVED {count: 2}, không ghi gì.
	g := w.put(w.T, 0, nil, true)
	require.Equal(t, http.StatusUnprocessableEntity, g.code)
	require.Equal(t, "CHECKIN_UNRESOLVED", g.errCode())
	require.EqualValues(t, 2, g.details()["count"])
	require.Equal(t, 0, r.count(`select count(*) from class_sessions where id = $1 and attendance_done_at is not null`, w.ses[0]))

	// Đánh vắng người chưa điểm danh → hoàn tất được.
	a := r.do(req{path: w.path(0, "/absent-unchecked"), bearer: w.T.access, hdr: map[string]string{"Idempotency-Key": uuid.NewString()}})
	require.Equal(t, http.StatusOK, a.code, string(a.body))
	require.Equal(t, 2, r.count(`select count(*) from attendance_records where session_id = $1 and status = 'ABSENT'`, w.ses[0]))
	require.Equal(t, http.StatusOK, w.put(w.T, 0, nil, true).code)

	// Buổi không có cửa sổ: hoàn tất vẫn điền PRESENT như cũ.
	require.Equal(t, http.StatusOK, w.put(w.T, 1, nil, true).code)
	require.Equal(t, 4, r.count(`select count(*) from attendance_records where session_id = $1 and status = 'PRESENT'`, w.ses[1]))
}

func TestSweepAutoCloseAndLate(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	w.enable("")
	svc := &attendance.Service{Pool: r.pool, Redis: r.rdb, Clock: r.clk, Cfg: r.cfg.Attendance}
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	require.Equal(t, http.StatusCreated, w.checkin(w.sv(0), map[string]any{"token": tok}).code)

	r.clk.Advance(9 * time.Minute)
	n, err := svc.Sweep(t.Context())
	require.NoError(t, err)
	require.Zero(t, n, "chưa tới closes_at")
	r.clk.Advance(2 * time.Minute)
	n, err = svc.Sweep(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, 1, r.count(`select count(*) from checkin_windows where id = $1 and closed_at is not null and closed_by is null`, wid))
	require.Equal(t, 1, r.count(`select count(*) from attendance_records where session_id = $1 and status = 'PRESENT'`, w.ses[0]))

	// Vòng 2 → lượt đầu sau starts_at + late_after → LATE. Buổi bắt đầu 20 phút trước.
	r.exec(`update class_sessions set starts_at = $2 where id = $1`, w.ses[0], r.clk.Now().Add(-20*time.Minute))
	w2 := w.open(0)
	require.Equal(t, http.StatusCreated, w2.code, string(w2.body))
	require.EqualValues(t, 2, w2.json()["round"])
	wid2 := w2.json()["id"].(string)
	tok2, _ := w.token(0, wid2)
	require.Equal(t, http.StatusCreated, w.checkin(w.sv(1), map[string]any{"token": tok2}).code)
	require.Equal(t, http.StatusOK, r.do(req{path: w.cpath("/checkin-windows/" + wid2 + "/close"), bearer: w.T.access}).code)
	require.Equal(t, "LATE", r.scalar(`select status::text from attendance_records where session_id = $1 and student_id = $2`, w.ses[0], w.students[1].ID))
	require.Equal(t, "PRESENT", r.scalar(`select status::text from attendance_records where session_id = $1 and student_id = $2`, w.ses[0], w.students[0].ID), "dòng đã có không bị vòng sau ghi đè")
}

func TestMeAttendanceCheckedAtAndTodayReview(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	w.enable("")
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	require.Equal(t, http.StatusCreated, w.checkin(w.sv(0), map[string]any{"token": tok}).code) // không khoá → NO_DEVICE_KEY
	require.Equal(t, http.StatusOK, r.do(req{path: w.cpath("/checkin-windows/" + wid + "/close"), bearer: w.T.access}).code)

	me := r.call(http.MethodGet, w.cpath("/me/attendance"), w.sv(0), nil)
	require.Equal(t, http.StatusOK, me.code, string(me.body))
	sess := me.json()["attendance"].(map[string]any)["sessions"].([]any)[0].(map[string]any)
	require.NotNil(t, sess["checked_in_at"])
	require.NotContains(t, string(me.body), "flags")

	td := r.call(http.MethodGet, "/me/today", w.A, nil)
	require.Equal(t, http.StatusOK, td.code, string(td.body))
	require.Contains(t, string(td.body), `"CHECKIN_REVIEW"`)
	require.Contains(t, string(td.body), "lượt điểm danh cần xem")
	sv := r.call(http.MethodGet, "/me/today", w.sv(1), nil)
	require.NotContains(t, string(sv.body), "CHECKIN_REVIEW")

	id := r.scalar(`select id::text from attendance_checkins where window_id = $1`, wid)
	require.Equal(t, http.StatusOK, r.do(req{path: w.cpath("/checkins/" + id + "/clear-flags"), bearer: w.A.access}).code)
	// Bỏ cờ → sự kiện outbox → bộ vô hiệu «Hôm nay» (worker) xoá cache của Staff; ở đây chạy thẳng bộ vô hiệu.
	var payload []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select payload::text::bytea from outbox where topic = 'crm.attendance_saved' order by id desc limit 1`).Scan(&payload))
	require.NoError(t, today.Invalidator{Pool: r.pool, Redis: r.rdb}.Handle(t.Context(), outbox.Message{Topic: "crm.attendance_saved", Payload: payload}))
	require.NotContains(t, string(r.call(http.MethodGet, "/me/today", w.A, nil).body), "CHECKIN_REVIEW")
}

func TestSettingsAttendance(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	get := r.call(http.MethodGet, w.cpath(""), w.T, nil)
	ver := int(get.json()["settings"].(map[string]any)["version"].(float64))
	put := r.call(http.MethodPut, w.cpath("/settings"), w.T, map[string]any{"version": ver, "attendance": map[string]any{
		"self_checkin": true, "rotate_seconds": 20, "geofence": map[string]any{"lat": 21.0, "lng": 105.8, "radius_m": 150}, "campus_cidrs": []string{"10.0.0.0/8"}}})
	require.Equal(t, http.StatusOK, put.code, string(put.body))
	got := put.json()["attendance"].(map[string]any)
	require.Equal(t, true, got["self_checkin"])
	require.EqualValues(t, 20, got["rotate_seconds"])

	// Sinh viên chỉ thấy 3 trường: không vị trí, không CIDR, không chu kỳ.
	st := r.call(http.MethodGet, w.cpath(""), w.sv(0), nil)
	sa := st.json()["attendance"].(map[string]any)
	require.Len(t, sa, 3)
	require.Equal(t, true, sa["geo_check"])
	require.NotContains(t, string(st.body), "campus_cidrs")

	// audit_log không có toạ độ.
	aud := r.scalar(`select coalesce(string_agg(coalesce(before::text,'') || coalesce(after::text,''), ' '), '') from audit_log where action = 'course_attendance_updated'`)
	require.NotEmpty(t, aud)
	require.NotContains(t, aud, "105.8")
	require.NotContains(t, aud, "21.0")

	// Sai miền → 422 theo trường; TA → 403.
	bad := r.call(http.MethodPut, w.cpath("/settings"), w.T, map[string]any{"version": ver + 1, "attendance": map[string]any{"rotate_seconds": 1}})
	require.Equal(t, http.StatusUnprocessableEntity, bad.code, string(bad.body))
	require.Contains(t, string(bad.body), "attendance.rotate_seconds")
	require.Equal(t, http.StatusForbidden, r.call(http.MethodPut, w.cpath("/settings"), w.A, map[string]any{"version": ver + 1, "attendance": map[string]any{"self_checkin": true}}).code)
}

func TestPurgeKeepsRecords(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	w.enable("")
	svc := &attendance.Service{Pool: r.pool, Redis: r.rdb, Clock: r.clk, Cfg: r.cfg.Attendance}
	wid := w.open(0).json()["id"].(string)
	tok, _ := w.token(0, wid)
	k := newDevkey(t)
	require.Equal(t, http.StatusCreated, w.checkin(w.sv(0), map[string]any{"token": tok, "device": k.device(t, tok, "", w.students[0].ID)}).code)
	require.Equal(t, http.StatusOK, r.do(req{path: w.cpath("/checkin-windows/" + wid + "/close"), bearer: w.T.access}).code)

	r.clk.Advance(29 * 24 * time.Hour)
	_, _, err := svc.PurgeCheckins(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from attendance_checkins`))
	r.clk.Advance(2 * 24 * time.Hour)
	_, _, err = svc.PurgeCheckins(t.Context())
	require.NoError(t, err)
	require.Equal(t, 0, r.count(`select count(*) from attendance_checkins where window_id = $1`, wid))
	require.Equal(t, 1, r.count(`select count(*) from checkin_windows where id = $1`, wid), "cửa sổ (sĩ số, số lượt) giữ lại; chỉ lượt thô bị xoá")
	require.Equal(t, 1, r.count(`select count(*) from attendance_records where session_id = $1`, w.ses[0]), "dòng điểm danh chính thức giữ nguyên")
	require.Equal(t, 0, r.count(`select count(*) from user_devices`), "thiết bị không còn lượt nào bị xoá")
}
