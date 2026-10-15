package crm_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func (w world) mePath(leaf string) string { return "/courses/" + w.course.String() + "/me/" + leaf }

// TestMeAttendanceOwnOnly: mỗi sinh viên chỉ thấy số của mình; tham số `student_id` bị bỏ qua.
func TestMeAttendanceOwnOnly(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	a, b := w.students[0], w.students[1]
	require.Equal(t, http.StatusOK, w.put(w.T, 0, []rec{row(a.ID, "PRESENT"), row(b.ID, "ABSENT")}, true).code)
	require.Equal(t, http.StatusOK, w.put(w.T, 1, []rec{row(a.ID, "LATE"), row(b.ID, "ABSENT")}, true).code)
	for _, tc := range []struct {
		u                               int
		recorded, present, late, absent int
	}{{0, 2, 1, 1, 0}, {1, 2, 0, 0, 2}} {
		st := r.mustLogin(w.students[tc.u].Email)
		res := r.call(http.MethodGet, w.mePath("attendance")+"?student_id="+w.students[1-tc.u].ID.String(), st, nil)
		require.Equal(t, http.StatusOK, res.code, string(res.body))
		at := res.json()["attendance"].(map[string]any)
		require.Equal(t, float64(tc.recorded), at["recorded"])
		require.Equal(t, float64(tc.present), at["present"])
		require.Equal(t, float64(tc.late), at["late"])
		require.Equal(t, float64(tc.absent), at["absent"])
		require.Len(t, at["sessions"], 2)
	}
}

// TestMeAttendanceNoInternalFields: không note, marked_by, version, id buổi; mới nhất trước; phân trang bằng con trỏ.
func TestMeAttendanceNoInternalFields(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	u := w.students[0]
	require.Equal(t, http.StatusOK, w.put(w.T, 0, []rec{row(u.ID, "ABSENT", "note", "CANARY-NOTE-9Q")}, true).code)
	require.Equal(t, http.StatusOK, w.put(w.T, 1, []rec{row(u.ID, "PRESENT")}, true).code)
	st := r.mustLogin(u.Email)
	res := r.call(http.MethodGet, w.mePath("attendance")+"?limit=1", st, nil)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	for _, bad := range []string{"CANARY-NOTE-9Q", `"note"`, `"marked_by"`, `"version"`, `"marked_at"`, `"id"`} {
		require.NotContains(t, string(res.body), bad)
	}
	sessions := res.json()["attendance"].(map[string]any)["sessions"].([]any)
	require.Len(t, sessions, 1)
	first := sessions[0].(map[string]any)
	require.Len(t, first, 3, "chỉ session_no, starts_at, status")
	require.Equal(t, float64(2), first["session_no"], "mới nhất trước")
	next, _ := res.json()["next_cursor"].(string)
	require.NotEmpty(t, next)
	res = r.call(http.MethodGet, w.mePath("attendance")+"?limit=1&cursor="+next, st, nil)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, float64(1), res.json()["attendance"].(map[string]any)["sessions"].([]any)[0].(map[string]any)["session_no"])
	require.Nil(t, res.json()["next_cursor"])
}

// TestMeAttendanceEmptyCourse: lớp chưa điểm danh → số 0, sessions rỗng, không lỗi.
func TestMeAttendanceEmptyCourse(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	res := r.call(http.MethodGet, w.mePath("attendance"), r.mustLogin(w.students[0].Email), nil)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	at := res.json()["attendance"].(map[string]any)
	require.Equal(t, float64(0), at["recorded"])
	require.Empty(t, at["sessions"])
	res = r.call(http.MethodGet, w.mePath("participation"), r.mustLogin(w.students[0].Email), nil)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	p := res.json()["participation"].(map[string]any)
	require.Equal(t, float64(0), p["speak_count"])
	require.Equal(t, "0.00", p["points_total"])
	require.Empty(t, p["events"])
}

// TestMeParticipationMatchesDB: số lần, tổng điểm (decimal) và sự kiện khớp DB; không lộ id, ghi chú, người ghi.
func TestMeParticipationMatchesDB(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	a, b := w.students[0], w.students[1]
	for range 3 {
		require.Equal(t, http.StatusCreated, w.speak(w.T, a.ID, nil).code)
	}
	require.Equal(t, http.StatusCreated, w.speak(w.T, b.ID, map[string]any{"note": "CANARY-PART-3Z"}).code)
	res := r.call(http.MethodGet, w.mePath("participation"), r.mustLogin(a.Email), nil)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	p := res.json()["participation"].(map[string]any)
	require.Equal(t, float64(3), p["speak_count"])
	require.Equal(t, r.scalar(`select sum(points)::numeric(8,2)::text from participation_events where course_id = $1 and student_id = $2`, w.course, a.ID), p["points_total"])
	require.Equal(t, "0.75", p["points_total"])
	require.Len(t, p["events"], 3)
	for _, bad := range []string{"CANARY-PART-3Z", `"note"`, `"created_by"`, `"student"`, `"id"`, b.ID.String()} {
		require.NotContains(t, string(res.body), bad)
	}
	require.Len(t, p["events"].([]any)[0].(map[string]any), 3, "chỉ date, type, points")
}

// TestMeRoutesStaff403: TA / giảng viên / admin / người ngoài lớp / PENDING đều 403; chưa đăng nhập 401.
func TestMeRoutesStaff403(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	pend := r.mustLogin(w.pending.Email)
	for _, leaf := range []string{"attendance", "participation"} {
		for name, s := range map[string]session{"giảng viên": w.T, "TA": w.A, "admin": w.ADM, "ngoài lớp": w.OUT, "PENDING": pend} {
			require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, w.mePath(leaf), s, nil).code, leaf+" "+name)
		}
		require.Equal(t, http.StatusUnauthorized, r.call(http.MethodGet, w.mePath(leaf), session{}, nil).code, leaf)
	}
}

// TestMeNotCached: giảng viên ghi thêm một lần phát biểu → lần đọc kế tiếp của sinh viên thấy ngay; không cache.
func TestMeNotCached(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	u := w.students[0]
	st := r.mustLogin(u.Email)
	read := func() (string, string) {
		res := r.call(http.MethodGet, w.mePath("participation"), st, nil)
		require.Equal(t, http.StatusOK, res.code, string(res.body))
		return res.json()["participation"].(map[string]any)["points_total"].(string), res.hdr.Get("Cache-Control")
	}
	pts, cc := read()
	require.Equal(t, "0.00", pts)
	require.Equal(t, "no-store", cc)
	require.Equal(t, http.StatusCreated, w.speak(w.T, u.ID, nil).code)
	require.Equal(t, http.StatusCreated, w.speak(w.T, u.ID, nil).code)
	pts, _ = read()
	require.Equal(t, "0.50", pts)
	require.Equal(t, http.StatusCreated, w.speak(w.T, u.ID, nil).code)
	pts, _ = read()
	require.Equal(t, "0.75", pts)
}

// TestMeTwoCoursesIsolated: sinh viên học hai lớp — mỗi route chỉ trả dữ liệu của lớp trong đường dẫn.
func TestMeTwoCoursesIsolated(t *testing.T) {
	r := newRig(t)
	w1, w2 := r.world(1), r.world(1)
	u := w1.students[0]
	r.enroll(w2.course, u.ID, "STUDENT", "ACTIVE")
	require.Equal(t, http.StatusOK, w1.put(w1.T, 0, []rec{row(u.ID, "ABSENT")}, true).code)
	require.Equal(t, http.StatusOK, w2.put(w2.T, 0, []rec{row(u.ID, "PRESENT"), row(w2.students[0].ID, "ABSENT")}, true).code)
	require.Equal(t, http.StatusCreated, w1.speak(w1.T, u.ID, nil).code)
	st := r.mustLogin(u.Email)
	a1 := r.call(http.MethodGet, w1.mePath("attendance"), st, nil).json()["attendance"].(map[string]any)
	a2 := r.call(http.MethodGet, w2.mePath("attendance"), st, nil).json()["attendance"].(map[string]any)
	require.Equal(t, float64(1), a1["absent"])
	require.Equal(t, float64(0), a2["absent"])
	require.Equal(t, float64(1), a2["present"])
	p1 := r.call(http.MethodGet, w1.mePath("participation"), st, nil).json()["participation"].(map[string]any)
	p2 := r.call(http.MethodGet, w2.mePath("participation"), st, nil).json()["participation"].(map[string]any)
	require.Equal(t, float64(1), p1["speak_count"])
	require.Equal(t, float64(0), p2["speak_count"])
}

// TestStudentCrawlNoCanary (US-P5-06 AC2): sinh viên có ghi chú riêng (canary) và nhãn AT_RISK; duyệt mọi route dành cho sinh viên
// (kể cả phát biểu / chuyên cần / thời gian học của mình) không thấy chuỗi canary, câu nhận định hay khoá risk / observation.
func TestStudentCrawlNoCanary(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	b := w.students[0]
	const canary = "CANARY-OBS-4F"
	res := r.call(http.MethodPost, "/courses/"+w.course.String()+"/students/"+b.ID.String()+"/observations", w.T, map[string]any{"tags": []string{"canary"}, "body": canary + " hay vắng, cần nhắn riêng", "kind": "NOTE"})
	require.Equal(t, http.StatusCreated, res.code, string(res.body))
	w.setRisk(b.ID, "AT_RISK", 0, 0)
	require.Equal(t, http.StatusCreated, w.speak(w.T, b.ID, nil).code)
	st := r.mustLogin(b.Email)
	c := w.course.String()
	paths := []string{"/me/courses", "/me/settings", "/me/today", "/notifications", "/chat/sessions", "/courses/" + c, "/courses/" + c + "/today", "/courses/" + c + "/threads", "/courses/" + c + "/calendar",
		w.mePath("attendance"), w.mePath("participation"), w.mePath("study-time")}
	forbidden := []string{canary, "AT_RISK", `"risk`, `"observation`, `"at_risk`, `"watch"`, "cần chú ý", "cần nhắn riêng"}
	for _, p := range paths {
		got := r.call(http.MethodGet, p, st, nil)
		require.Less(t, got.code, 500, p)
		for _, f := range forbidden {
			require.NotContains(t, string(got.body), f, "%s lộ %q", p, f)
		}
	}
	staffOnly403(t, r, w, b.ID, st)
}

// TestStudentNeverSeesObservations: route đọc ghi chú / hồ sơ là của nhân viên — sinh viên (kể cả của chính mình) nhận 403.
func TestStudentNeverSeesObservations(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	staffOnly403(t, r, w, w.students[0].ID, r.mustLogin(w.students[0].Email))
}

func staffOnly403(t *testing.T, r *rig, w world, student uuid.UUID, st session) {
	t.Helper()
	c := w.course.String()
	for _, p := range []string{"/courses/" + c + "/students/" + student.String() + "/observations", "/courses/" + c + "/students/" + student.String() + "/profile"} {
		require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, p, st, nil).code, p)
	}
}

// TestSeedIsolation (US-S7-SEED AC8): sinh viên chỉ đọc dữ liệu của chính mình; sinh viên chưa vào lớp không thấy lớp nào; TA thấy lớp mình nhưng không thấy lớp khác.
func TestSeedIsolation(t *testing.T) {
	r := newRig(t)
	w1, w2 := r.world(2), r.world(1)
	a, b := w1.students[0], w1.students[1]
	require.Equal(t, http.StatusOK, w1.put(w1.T, 0, []rec{row(a.ID, "PRESENT"), row(b.ID, "ABSENT")}, true).code)
	sa, sb := r.mustLogin(a.Email), r.mustLogin(b.Email)
	at := func(s session) map[string]any {
		res := r.call(http.MethodGet, w1.mePath("attendance"), s, nil)
		require.Equal(t, http.StatusOK, res.code)
		return res.json()["attendance"].(map[string]any)
	}
	require.Equal(t, float64(0), at(sa)["absent"])
	require.Equal(t, float64(1), at(sb)["absent"])
	for _, s := range []session{sa, sb} {
		require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, "/courses/"+w1.course.String()+"/students", s, nil).code)
		require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, w2.mePath("attendance"), s, nil).code, "lớp khác")
	}
	// sinh viên D: chưa vào lớp nào
	require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, w1.mePath("attendance"), w1.OUT, nil).code)
	require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, w1.mePath("participation"), w1.OUT, nil).code)
	// TA lớp 1: đọc lớp 1, không đọc lớp 2
	require.Equal(t, http.StatusOK, r.call(http.MethodGet, "/courses/"+w1.course.String()+"/students", w1.A, nil).code)
	require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, "/courses/"+w2.course.String()+"/students", w1.A, nil).code)
	require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, "/courses/"+w2.course.String()+"/students/"+w2.students[0].ID.String()+"/profile", w1.A, nil).code)
}

// TestSeedAdminForbidden: ADMIN không đọc được hồ sơ, danh sách sinh viên, điểm danh (ticket: test của escalation và check-crm-seed.mjs trên stack thật) (dữ liệu cá nhân của sinh viên).
func TestSeedAdminForbidden(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	c := w.course.String()
	for _, p := range []string{"/courses/" + c + "/students", "/courses/" + c + "/students/" + w.students[0].ID.String() + "/profile", w.path(0, "/attendance"), w.mePath("attendance")} {
		require.Equal(t, http.StatusForbidden, r.call(http.MethodGet, p, w.ADM, nil).code, p)
	}
}
