package today_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

type cls struct {
	id   uuid.UUID
	code string
}

func (c cls) sid() string { return c.id.String() }

// world: một Admin + một lớp do Admin mở. Người dùng thêm bằng r.person / r.staffOf / r.joined.
func (r *rig) klass(code string) cls {
	r.t.Helper()
	adm := r.addUser(uniq("adm"), store.UserRoleADMIN, store.UserStatusACTIVE)
	return cls{id: r.course(adm.ID, code), code: code}
}

func (r *rig) person(role store.UserRole, verified bool) (store.User, session) {
	r.t.Helper()
	u := r.addUser(uniq(strings.ToLower(string(role))), role, store.UserStatusACTIVE)
	if verified {
		_, err := r.pool.Exec(r.t.Context(), `update users set email_verified_at = now() where id = $1`, u.ID)
		require.NoError(r.t, err)
	}
	return u, r.mustLogin(u.Email)
}

// enrollAt ghi danh với status_changed_at = (đồng hồ giả − age) và cảnh báo tuỳ chọn.
func (r *rig) enrollAt(c cls, u store.User, role, status, warning string, age time.Duration) {
	r.t.Helper()
	var w any
	if warning != "" {
		w = warning
	}
	var snap any
	if role == "STUDENT" {
		snap = "B20DC" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", "")[:6])
	}
	var removed any
	if status == "REMOVED" {
		removed = r.clk.Now().Add(-age)
	}
	_, err := r.pool.Exec(r.t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, warning, student_code_snapshot, status_changed_at, removed_at)
		values ($1, $2, $3::enrollment_role, $4::enrollment_status, 'CODE', $5, $6, $7, $8)`, c.id, u.ID, role, status, w, snap, r.clk.Now().Add(-age), removed)
	require.NoError(r.t, err)
}

// tget gọi GET sau khi xoá cache "Hôm nay" của CHÍNH người này (mỗi test thấy dữ liệu mới nhất). tgetC giữ nguyên cache.
func (r *rig) tget(s session, path string) resp {
	r.t.Helper()
	r.dropCache(s)
	return r.tgetC(s, path)
}

func (r *rig) tgetC(s session, path string) resp {
	return r.do(req{method: http.MethodGet, path: path, bearer: s.access})
}

// dropCache xoá `ep:today:{sub}:*` (sub lấy từ JWT; không đụng khoá của người khác dùng chung Redis).
func (r *rig) dropCache(s session) {
	r.t.Helper()
	parts := strings.Split(s.access, ".")
	if len(parts) != 3 {
		return
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(r.t, err)
	var claims struct {
		Sub string `json:"sub"`
	}
	require.NoError(r.t, json.Unmarshal(raw, &claims))
	keys, err := r.rdb.Keys(r.t.Context(), "ep:today:"+claims.Sub+":*").Result()
	require.NoError(r.t, err)
	if len(keys) > 0 {
		require.NoError(r.t, r.rdb.Del(r.t.Context(), keys...).Err())
	}
}

func kinds(actions []any) []string {
	out := make([]string, len(actions))
	for i, a := range actions {
		out[i] = a.(map[string]any)["kind"].(string)
	}
	return out
}

func actionsOf(res resp) []any {
	a, _ := res.json()["actions"].([]any)
	return a
}

func (r *rig) session(c cls, no int, start time.Time, dur time.Duration, room string) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), `insert into class_sessions (course_id, session_no, starts_at, ends_at, room) values ($1, $2, $3, $4, $5)`, c.id, no, start, start.Add(dur), room)
	require.NoError(r.t, err)
}

// ===== Sinh viên (AC3) =====

func TestStudentTodayNoCourse(t *testing.T) {
	r := newRig(t)
	_, s := r.person(store.UserRoleSTUDENT, true)
	res := r.tget(s, "/me/today")
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	j := res.json()
	require.Equal(t, true, j["no_course"])
	require.Equal(t, true, j["email_verified"])
	rec := j["recommended"].(map[string]any)
	require.Equal(t, "JOIN_CODE", rec["kind"])
	require.Equal(t, "Nhập mã tham gia lớp", rec["title"])
	require.Equal(t, "Bạn chưa vào lớp nào. Nhập mã do giảng viên cung cấp để bắt đầu.", rec["reason"])
	require.Equal(t, "/join", rec["href"])
	require.Equal(t, float64(1), rec["estimate_minutes"])
	require.Empty(t, j["timeline"])
	require.Empty(t, j["continue"])
}

// Q-QC-P211-2: vừa chưa xác minh vừa chưa có lớp ⇒ VERIFY_EMAIL đứng trước, no_course vẫn true.
func TestStudentTodayUnverified(t *testing.T) {
	r := newRig(t)
	u, s := r.person(store.UserRoleSTUDENT, false)
	j := r.tget(s, "/me/today").json()
	require.Equal(t, false, j["email_verified"])
	require.Equal(t, true, j["no_course"])
	rec := j["recommended"].(map[string]any)
	require.Equal(t, "VERIFY_EMAIL", rec["kind"])
	require.Equal(t, "/verify-email", rec["href"])
	local, domain, _ := strings.Cut(u.Email, "@")
	require.Equal(t, "Chưa xác minh email thì chưa vào được lớp. Kiểm tra hộp thư "+local[:1]+"***@"+domain+".", rec["reason"])
	require.NotContains(t, rec["reason"], local, "email được che")
}

func TestStudentTodayPendingOnly(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	u, s := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c, u, "STUDENT", "PENDING", "", 2*time.Hour)
	j := r.tget(s, "/me/today").json()
	rec := j["recommended"].(map[string]any)
	require.Equal(t, "JOIN_PENDING", rec["kind"])
	require.Equal(t, "Chờ giảng viên duyệt", rec["title"])
	require.Equal(t, "Yêu cầu vào lớp 761988 đã gửi 2 giờ trước.", rec["reason"])
	require.Equal(t, "", rec["href"])
	require.Equal(t, true, j["no_course"], "chưa có lớp ACTIVE")
}

func TestStudentTodayNoUrgent(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	u, s := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c, u, "STUDENT", "ACTIVE", "", time.Hour)
	j := r.tget(s, "/me/today").json()
	require.Nil(t, j["recommended"])
	require.Equal(t, false, j["no_course"])
	require.Empty(t, j["timeline"])
}

func TestStudentTodayTimeline(t *testing.T) {
	r := newRig(t)
	r.clk.Set(time.Date(2026, 10, 12, 4, 0, 0, 0, time.UTC)) // 11:00 ICT; đặt TRƯỚC khi đăng nhập để JWT cùng đồng hồ
	c := r.klass("761988")
	u, s := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c, u, "STUDENT", "ACTIVE", "", time.Hour)
	now := r.clk.Now()
	r.session(c, 9, now.Add(-24*time.Hour), 2*time.Hour, "P.301")    // hôm qua: không hiện
	r.session(c, 10, now.Add(-3*time.Hour), 2*time.Hour, "P.302")    // 08:00–10:00 ICT hôm nay: DONE
	r.session(c, 11, now.Add(-30*time.Minute), 2*time.Hour, "P.302") // đang diễn ra: NOW
	r.session(c, 12, now.Add(3*time.Hour), 2*time.Hour, "P.303")     // chiều nay: NEXT
	r.session(c, 13, now.Add(5*24*time.Hour), 2*time.Hour, "")       // 5 ngày nữa: NEXT
	r.session(c, 14, now.Add(9*24*time.Hour), 2*time.Hour, "P.304")  // ngoài 7 ngày: không hiện
	tl := r.tget(s, "/courses/"+c.sid()+"/today").json()["timeline"].([]any)
	var got []string
	for _, e := range tl {
		m := e.(map[string]any)
		got = append(got, fmt.Sprintf("%s|%s|%s", m["title"], m["place"], m["state"]))
		require.Equal(t, "761988", m["course"].(map[string]any)["class_code"])
	}
	require.Equal(t, []string{"Buổi 10 · An ninh mạng|P.302|DONE", "Buổi 11 · An ninh mạng|P.302|NOW", "Buổi 12 · An ninh mạng|P.303|NEXT", "Buổi 13 · An ninh mạng||NEXT"}, got)
}

func TestStudentTodayAtMostOneRecommended(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	u, s := r.person(store.UserRoleSTUDENT, false) // chưa xác minh + chờ duyệt
	r.enrollAt(c, u, "STUDENT", "PENDING", "", time.Hour)
	j := r.tget(s, "/me/today").json()
	rec, ok := j["recommended"].(map[string]any)
	require.True(t, ok, "recommended là MỘT đối tượng, không phải mảng")
	require.Equal(t, "VERIFY_EMAIL", rec["kind"])
}

// ===== Cách ly (AC7) =====

func TestStudentTodayOnlyOwnData(t *testing.T) {
	r := newRig(t)
	c1, c2 := r.klass("761987"), r.klass("761988")
	a, sa := r.person(store.UserRoleSTUDENT, true)
	b, sb := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c1, a, "STUDENT", "ACTIVE", "", time.Hour)
	r.enrollAt(c2, b, "STUDENT", "ACTIVE", "", time.Hour)
	r.session(c1, 1, r.clk.Now().Add(time.Hour), time.Hour, "P.101")
	r.session(c2, 1, r.clk.Now().Add(time.Hour), time.Hour, "P.202")
	other := r.addUser(uniq("khac"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	r.enrollAt(c1, other, "STUDENT", "PENDING", "", time.Hour)
	ra, rb := string(r.tget(sa, "/me/today").body), string(r.tget(sb, "/me/today").body)
	require.Contains(t, ra, "761987")
	require.NotContains(t, ra, "761988")
	require.NotContains(t, ra, "P.202")
	require.Contains(t, rb, "761988")
	require.NotContains(t, rb, "761987")
	for _, body := range []string{ra, rb} {
		for _, leak := range []string{a.Email, b.Email, other.Email, other.ID.String(), "user_id", "student_code", "join_code", "MSSV"} {
			require.NotContains(t, body, leak)
		}
	}
	require.NotContains(t, ra, b.ID.String())
}

func TestStaffKindsNeverInStudentResponse(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	u, s := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c, u, "STUDENT", "ACTIVE", "", time.Hour)
	// Lớp có đủ mọi việc của staff: yêu cầu chờ duyệt, email chưa khớp, chưa thiết lập.
	for range 3 {
		p := r.addUser(uniq("p"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
		r.enrollAt(c, p, "STUDENT", "PENDING", "", 50*time.Hour)
	}
	m := r.addUser(uniq("m"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	r.enrollAt(c, m, "STUDENT", "PENDING", "EMAIL_MISMATCH", time.Hour)
	for _, path := range []string{"/me/today", "/courses/" + c.sid() + "/today"} {
		body := string(r.tget(s, path).body)
		for _, k := range []string{"JOIN_REQUEST", "EMAIL_MISMATCH", "COURSE_SETUP", "actions", "upcoming", "LLM_"} {
			require.NotContains(t, body, k, path)
		}
	}
}

func TestTodayOutsider403(t *testing.T) {
	r := newRig(t)
	c1, c2 := r.klass("761987"), r.klass("761988")
	a, sa := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c1, a, "STUDENT", "ACTIVE", "", time.Hour)
	pend, sp := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c1, pend, "STUDENT", "PENDING", "", time.Hour)
	rem, sr := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c1, rem, "STUDENT", "REMOVED", "", time.Hour)
	require.Equal(t, http.StatusForbidden, r.tget(sa, "/courses/"+c2.sid()+"/today").code, "sinh viên lớp khác")
	require.Equal(t, http.StatusForbidden, r.tget(sp, "/courses/"+c1.sid()+"/today").code, "PENDING")
	require.Equal(t, http.StatusForbidden, r.tget(sr, "/courses/"+c1.sid()+"/today").code, "REMOVED")
	require.Equal(t, http.StatusForbidden, r.tget(sa, "/courses/"+uuid.NewString()+"/today").code, "lớp không có")
}

// AC7 fuzz: 200 cặp (người, lớp) ngẫu nhiên ⇒ mọi course.id trong kết quả ∈ lớp ACTIVE của chính người đó.
func TestTodayFuzzCourseScope(t *testing.T) {
	r := newRig(t)
	var cs []cls
	for i := range 4 {
		c := r.klass(fmt.Sprintf("FZ%d", i))
		cs = append(cs, c)
		r.session(c, 1, r.clk.Now().Add(time.Hour), time.Hour, "P.1")
	}
	type person struct {
		s    session
		own  map[string]bool
		pend map[string]bool
	}
	var ps []person
	for i := range 5 {
		role := []store.UserRole{store.UserRoleSTUDENT, store.UserRoleSTUDENT, store.UserRoleTA, store.UserRoleTEACHER, store.UserRoleSTUDENT}[i]
		u, s := r.person(role, true)
		p := person{s: s, own: map[string]bool{}, pend: map[string]bool{}}
		er := map[store.UserRole]string{store.UserRoleSTUDENT: "STUDENT", store.UserRoleTA: "TA", store.UserRoleTEACHER: "TEACHER"}[role]
		for j, c := range cs {
			if (i+j)%2 == 0 {
				r.enrollAt(c, u, er, "ACTIVE", "", time.Hour)
				p.own[c.sid()] = true
			} else if er == "STUDENT" && j%3 == 0 { // chờ duyệt: không phải lớp của viewer
				r.enrollAt(c, u, er, "PENDING", "", time.Hour)
				p.pend[c.sid()] = true
			}
		}
		ps = append(ps, p)
	}
	for n := range 200 {
		p, c := ps[n%len(ps)], cs[(n*7+n/5)%len(cs)]
		res := r.tget(p.s, "/courses/"+c.sid()+"/today")
		if !p.own[c.sid()] {
			require.Equal(t, http.StatusForbidden, res.code)
			continue
		}
		require.Equal(t, http.StatusOK, res.code, string(res.body))
		for _, id := range uuidsIn(string(res.body)) {
			require.True(t, id == c.sid(), "course.id %s ngoài lớp %s", id, c.sid())
		}
		all := r.tget(p.s, "/me/today")
		for _, id := range uuidsIn(string(all.body)) {
			require.True(t, p.own[id] || p.pend[id] || !isCourse(cs, id), "me/today lộ lớp %s", id)
		}
	}
}

func isCourse(cs []cls, id string) bool {
	for _, c := range cs {
		if c.sid() == id {
			return true
		}
	}
	return false
}

// uuidsIn trả mọi UUID trong thân JSON (id mục việc dạng KIND:uuid cũng được tách).
func uuidsIn(body string) []string {
	var out []string
	for i := 0; i+36 <= len(body); i++ {
		if _, err := uuid.Parse(body[i : i+36]); err == nil && body[i+8] == '-' {
			out = append(out, body[i:i+36])
			i += 35
		}
	}
	return out
}
