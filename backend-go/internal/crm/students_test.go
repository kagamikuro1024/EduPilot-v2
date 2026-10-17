package crm_test

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func (w world) students_(s session, query string) resp {
	return w.r.call(http.MethodGet, "/courses/"+w.course.String()+"/students"+query, s, nil)
}

func (w world) setRisk(student uuid.UUID, level string, asks, posts int) {
	entered := "null"
	if level == "AT_RISK" {
		entered = "now()"
	}
	w.r.exec(`insert into student_risk (course_id, student_id, level, entered_at, computed_at, last_week_asks, last_week_posts) values ($1, $2, $3::risk_level, `+entered+`, now(), $4, $5)
		on conflict (course_id, student_id) do update set level = excluded.level, entered_at = excluded.entered_at, last_week_asks = excluded.last_week_asks, last_week_posts = excluded.last_week_posts`,
		w.course, student, level, asks, posts)
}

func names(x resp) []string {
	var out []string
	for _, it := range items(x, "items") {
		out = append(out, it["student"].(map[string]any)["full_name"].(string))
	}
	return out
}

func (w world) rename(i int, name string) {
	w.r.exec(`update users set full_name = $1 where id = $2`, name, w.students[i].ID)
}

// TestStudentsListActiveOnly: chỉ ACTIVE; PENDING / REMOVED / người ngoài lớp không có; không có cột điểm, không có câu lý do.
func TestStudentsListActiveOnly(t *testing.T) {
	r := newRig(t)
	w := r.world(3)
	r.addUserRemoved(w)
	res := w.students_(w.T, "")
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, "no-store", res.hdr.Get("Cache-Control"))
	require.Len(t, names(res), 3)
	require.EqualValues(t, 3, res.json()["total"])
	body := string(res.body)
	for _, bad := range []string{"grade", "sentence", "reasons", "score"} {
		require.NotContains(t, body, bad, "danh sách không có điểm / câu lý do")
	}
	first := items(res, "items")[0]
	require.Equal(t, "NONE", first["risk"].(map[string]any)["level"])
	require.Nil(t, first["risk"].(map[string]any)["computed_at"], "job chưa chạy: computed_at rỗng")
}

// TestStudentsListSortOrder: (risk_rank giảm, tên không dấu, id): AT_RISK → WATCH → NONE; trong mỗi nhóm tên không dấu ("Ân" = "an" đứng trước "Anh").
func TestStudentsListSortOrder(t *testing.T) {
	r := newRig(t)
	w := r.world(5)
	for i, n := range []string{"Bình", "Ân", "Anh", "Dũng", "Cường"} {
		w.rename(i, n)
	}
	w.setRisk(w.students[3].ID, "WATCH", 0, 0)   // Dũng
	w.setRisk(w.students[0].ID, "AT_RISK", 0, 0) // Bình
	w.setRisk(w.students[4].ID, "AT_RISK", 0, 0) // Cường
	res := w.students_(w.T, "")
	require.Equal(t, []string{"Bình", "Cường", "Dũng", "Ân", "Anh"}, names(res))
}

// TestStudentsListCursorFull1000: đi hết 1.000 sinh viên bằng con trỏ, không trùng, không sót, đúng thứ tự (trang 100); cursor hỏng → 422.
func TestStudentsListCursorFull1000(t *testing.T) {
	r := newRig(t)
	w := r.world(0)
	tag := uuid.NewString()[:6]
	r.exec(`insert into users (email, full_name, role, status)
		select 'bulk' || g || '.' || $1::text || '@example.test', 'Sv ' || lpad(g::text, 4, '0'), 'STUDENT', 'INVITED' from generate_series(1, 1000) g`, tag)
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via)
		select $1, u.id, 'STUDENT', 'ACTIVE', 'ADMIN' from users u where u.email like 'bulk%.' || $2::text || '@example.test'`, w.course, tag)
	require.Equal(t, 1000, r.count(`select count(*) from enrollments where course_id = $1 and role_in_course = 'STUDENT' and status = 'ACTIVE'`, w.course))
	// một nhóm rủi ro xen vào thứ tự: mỗi 7 người một WATCH, mỗi 50 người một AT_RISK
	r.exec(`insert into student_risk (course_id, student_id, level, entered_at, computed_at)
		select e.course_id, e.user_id, case when n % 50 = 0 then 'AT_RISK' else 'WATCH' end::risk_level, case when n % 50 = 0 then now() end, now()
		from (select course_id, user_id, row_number() over (order by user_id) n from enrollments where course_id = $1 and role_in_course = 'STUDENT') e where n % 7 = 0 or n % 50 = 0`, w.course)
	seen := map[string]bool{}
	var order []string
	cursor := ""
	pages := 0
	for {
		q := "?limit=100"
		if cursor != "" {
			q += "&cursor=" + url.QueryEscape(cursor)
		}
		res := w.students_(w.T, q)
		require.Equal(t, http.StatusOK, res.code, string(res.body))
		require.EqualValues(t, 1000, res.json()["total"])
		its := items(res, "items")
		require.LessOrEqual(t, len(its), 100)
		for _, it := range its {
			id := it["student"].(map[string]any)["id"].(string)
			require.False(t, seen[id], "trùng %s", id)
			seen[id] = true
			order = append(order, fmt.Sprintf("%d|%s", rankOf(it), it["student"].(map[string]any)["full_name"]))
		}
		pages++
		nc, _ := res.json()["next_cursor"].(string)
		if nc == "" {
			break
		}
		cursor = nc
		require.Less(t, pages, 20)
	}
	require.Len(t, seen, 1000)
	require.Equal(t, 10, pages)
	sorted := append([]string(nil), order...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := strings.SplitN(sorted[i], "|", 2), strings.SplitN(sorted[j], "|", 2)
		if a[0] != b[0] {
			return a[0] > b[0]
		}
		return a[1] < b[1]
	})
	require.Equal(t, sorted, order, "thứ tự (hạng giảm, tên) giữ qua các trang")
	require.Equal(t, 422, w.students_(w.T, "?cursor=%25%25").code)
	require.Equal(t, "INVALID_CURSOR", w.students_(w.T, "?cursor=abc").errCode())
	require.Equal(t, 422, w.students_(w.T, "?limit=101").code)
}

func rankOf(it map[string]any) int {
	switch it["risk"].(map[string]any)["level"] {
	case "AT_RISK":
		return 2
	case "WATCH":
		return 1
	}
	return 0
}

func TestStudentsFilterAttention(t *testing.T) {
	r := newRig(t)
	w := r.world(3)
	w.setRisk(w.students[0].ID, "AT_RISK", 0, 0)
	w.setRisk(w.students[1].ID, "WATCH", 0, 0)
	res := w.students_(w.T, "?filter=attention")
	require.Equal(t, []string{"Sv 01"}, names(res))
	require.EqualValues(t, 1, res.json()["total"])
}

func TestStudentsFilterAbsent(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	for n := range 3 { // absent_alert mặc định 3
		w.mark(w.pastSession(1, n+1), w.students[0].ID, "ABSENT")
	}
	for n := range 2 {
		w.mark(w.pastSession(2, n+10), w.students[1].ID, "ABSENT")
	}
	require.Equal(t, []string{"Sv 01"}, names(w.students_(w.T, "?filter=absent")))
	require.Equal(t, http.StatusOK, w.putRisk(w.T, w.version(), map[string]any{"absent_alert": 2}).code)
	require.Equal(t, []string{"Sv 01", "Sv 02"}, names(w.students_(w.T, "?filter=absent")), "ngưỡng theo cài đặt lớp")
}

// TestStudentsFilterInactive: không hỏi / đăng tuần trước VÀ ít phút học 7 ngày.
func TestStudentsFilterInactive(t *testing.T) {
	r := newRig(t)
	w := r.world(3)
	w.setRisk(w.students[0].ID, "NONE", 2, 0)                                                                                                                          // có hỏi
	r.exec(`insert into study_time_daily (course_id, student_id, date, area, active_seconds) values ($1, $2, current_date, 'CHAT', 7200)`, w.course, w.students[1].ID) // 120 phút
	require.Equal(t, []string{"Sv 03"}, names(w.students_(w.T, "?filter=inactive")))
	// giao hai bộ lọc
	w.setRisk(w.students[2].ID, "AT_RISK", 0, 0)
	require.Equal(t, []string{"Sv 03"}, names(w.students_(w.T, "?filter=inactive&filter=attention")))
	require.Equal(t, []string{"Sv 03"}, names(w.students_(w.T, "?filter=inactive,attention")))
	require.Empty(t, names(w.students_(w.T, "?filter=absent&filter=attention")), "giao rỗng → items []")
	require.Equal(t, "[]", fmt.Sprint(w.students_(w.T, "?filter=absent&filter=attention").json()["items"]))
}

func TestStudentsSearchFoldedName(t *testing.T) {
	r := newRig(t)
	w := r.world(3)
	w.rename(0, "Nguyễn Văn Ân")
	w.rename(1, "Trần Thị Bình")
	w.rename(2, "Lê Đức Minh")
	r.exec(`update enrollments set student_code_snapshot = 'B21DCCN001' where course_id = $1 and user_id = $2`, w.course, w.students[1].ID)
	require.Equal(t, []string{"Nguyễn Văn Ân"}, names(w.students_(w.T, "?q=van+an")), "không dấu, không phân biệt hoa thường")
	require.Equal(t, []string{"Lê Đức Minh"}, names(w.students_(w.T, "?q=ĐỨC")))
	require.Equal(t, []string{"Trần Thị Bình"}, names(w.students_(w.T, "?q=B21DCCN")), "MSSV theo tiền tố")
	require.Empty(t, names(w.students_(w.T, "?q=CCN001")), "MSSV không khớp giữa chuỗi")
	require.Len(t, names(w.students_(w.T, "?q=%25")), 0, "ký tự % không phải ký tự đại diện")
}

func TestStudentsFilterInvalid(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	res := w.students_(w.T, "?filter=grade_drop")
	require.Equal(t, http.StatusUnprocessableEntity, res.code)
	require.Equal(t, "VALIDATION_FAILED", res.errCode())
}

// TestStudentsRoutesStudent403: STUDENT (kể cả xem chính mình), ADMIN và người ngoài lớp đều bị chặn.
func TestStudentsRoutesStudent403(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	st := r.mustLogin(w.students[0].Email)
	require.Equal(t, http.StatusForbidden, w.students_(st, "").code)
	require.Equal(t, http.StatusForbidden, w.students_(st, "?q="+url.QueryEscape("Sv 01")).code)
	require.Equal(t, http.StatusForbidden, w.students_(w.ADM, "").code)
	require.Equal(t, http.StatusForbidden, w.students_(w.OUT, "").code)
	require.Equal(t, http.StatusOK, w.students_(w.A, "").code, "TA đọc được")
}

// TestStudentsNewCourseNoDivideByZero: lớp chưa có buổi / dữ liệu → mọi số 0, risk NONE, không lỗi.
func TestStudentsNewCourseNoDivideByZero(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	r.exec(`delete from class_sessions where course_id = $1`, w.course)
	res := w.students_(w.T, "")
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	for _, it := range items(res, "items") {
		a := it["attendance"].(map[string]any)
		require.Equal(t, float64(0), a["recorded"])
		require.Equal(t, float64(0), it["study"].(map[string]any)["minutes_7d"])
		require.Equal(t, "NONE", it["risk"].(map[string]any)["level"])
	}
}

func TestStudentsBeforeFirstRiskRun(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	it := items(w.students_(w.T, ""), "items")[0]
	require.Nil(t, it["risk"].(map[string]any)["computed_at"])
	require.NoError(t, r.svc().RunCourse(t.Context(), w.course, midweek()))
	it = items(w.students_(w.T, ""), "items")[0]
	require.NotNil(t, it["risk"].(map[string]any)["computed_at"])
}

func TestStudentsArchivedReadable(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	r.exec(`update courses set status = 'ARCHIVED', archived_at = now(), join_enabled = false where id = $1`, w.course)
	res := w.students_(w.T, "")
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Len(t, names(res), 2)
}

// TestStudentsTrackedFlag: `tracked` = đã xác nhận thông báo VÀ (lớp không cho tắt đo HOẶC sinh viên chưa tắt).
func TestStudentsTrackedFlag(t *testing.T) {
	r := newRig(t)
	w := r.world(3)
	for _, i := range []int{0, 1} {
		r.exec(`update users set tracking_notice_ack_at = now() where id = $1`, w.students[i].ID)
	}
	r.exec(`insert into user_settings (user_id, track_study_time) values ($1, false)`, w.students[1].ID)
	got := map[string]bool{}
	for _, it := range items(w.students_(w.T, ""), "items") {
		got[it["student"].(map[string]any)["full_name"].(string)] = it["study"].(map[string]any)["tracked"].(bool)
	}
	require.Equal(t, map[string]bool{"Sv 01": true, "Sv 02": false, "Sv 03": false}, got, "Sv 02 đã tắt đo; Sv 03 chưa xác nhận")
}

// TestStudentNeverSeesRisk: không route nào dành cho sinh viên chứa khoá risk / observation / watch (AC9).
func TestStudentNeverSeesRisk(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	w.setRisk(w.students[0].ID, "AT_RISK", 0, 0)
	st := r.mustLogin(w.students[0].Email)
	c := w.course.String()
	forbidden := []string{`"risk`, `"at_risk`, `"observation`, `"watch"`, `AT_RISK`}
	for _, p := range []string{"/me/courses", "/me/settings", "/me/today", "/me", "/notifications", "/chat/sessions", "/courses/" + c, "/courses/" + c + "/threads", "/courses/" + c + "/me/attendance", "/courses/" + c + "/me/participation", "/courses/" + c + "/me/study-time"} {
		res := r.call(http.MethodGet, p, st, nil)
		require.Less(t, res.code, 500, p)
		for _, f := range forbidden {
			require.NotContains(t, string(res.body), f, "%s lộ %s", p, f)
		}
	}
}
