package course_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

func (r *rig) myCourses(s session, query string, hdr map[string]string) resp {
	return r.do(req{method: http.MethodGet, path: "/me/courses" + query, bearer: s.access, hdr: hdr})
}

func classCodes(res resp) []string {
	var out []string
	for _, it := range res.json()["items"].([]any) {
		out = append(out, it.(map[string]any)["course"].(map[string]any)["class_code"].(string))
	}
	sort.Strings(out)
	return out
}

// seed2 dựng 2 lớp mã duy nhất: SV A học cả hai, SV B chỉ lớp 1, SV D không lớp nào; GV dạy cả hai.
type seed struct {
	adm, gv, ta, svA, svB, svD store.User
	c1, c2                     uuid.UUID
	code1, code2               string
}

func (r *rig) seed2() seed {
	r.t.Helper()
	tag := uuid.NewString()[:6]
	s := seed{code1: "L1-" + tag, code2: "L2-" + tag}
	s.adm = r.addUser(uniq("adm"), store.UserRoleADMIN, store.UserStatusACTIVE)
	s.gv = r.addUser(uniq("gv"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	s.ta = r.addUser(uniq("ta"), store.UserRoleTA, store.UserStatusACTIVE)
	s.svA = r.addUser(uniq("sva"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s.svB = r.addUser(uniq("svb"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s.svD = r.addUser(uniq("svd"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s.c1, s.c2 = r.course(s.adm.ID, s.code1), r.course(s.adm.ID, s.code2)
	r.enroll(s.c1, s.gv.ID, "TEACHER", "ACTIVE")
	r.enroll(s.c2, s.gv.ID, "TEACHER", "ACTIVE")
	r.enroll(s.c1, s.ta.ID, "TA", "ACTIVE")
	r.enroll(s.c1, s.svA.ID, "STUDENT", "ACTIVE")
	r.enroll(s.c2, s.svA.ID, "STUDENT", "ACTIVE")
	r.enroll(s.c1, s.svB.ID, "STUDENT", "ACTIVE")
	return s
}

// AC7.
func TestMeCoursesStudent(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	a := r.myCourses(r.mustLogin(s.svA.Email), "", nil)
	require.Equal(t, http.StatusOK, a.code, string(a.body))
	require.Equal(t, []string{s.code1, s.code2}, classCodes(a))
	first := a.json()["items"].([]any)[0].(map[string]any)
	require.Equal(t, "STUDENT", first["role_in_course"])
	require.Equal(t, "ACTIVE", first["enrollment_status"])
	for _, k := range []string{"id", "class_code", "subject_code", "name", "semester", "status"} {
		require.Contains(t, first["course"], k)
	}
	require.Equal(t, []string{s.code1}, classCodes(r.myCourses(r.mustLogin(s.svB.Email), "", nil)))
	require.Empty(t, classCodes(r.myCourses(r.mustLogin(s.svD.Email), "", nil)))
	// PENDING của chính mình hiện; REMOVED thì không; lớp của người khác không bao giờ
	r.enroll(s.c2, s.svB.ID, "STUDENT", "PENDING")
	b := r.myCourses(r.mustLogin(s.svB.Email), "", nil)
	require.Equal(t, []string{s.code1, s.code2}, classCodes(b))
	for _, it := range b.json()["items"].([]any) {
		m := it.(map[string]any)
		if m["course"].(map[string]any)["class_code"] == s.code2 {
			require.Equal(t, "PENDING", m["enrollment_status"])
		}
	}
	r.enroll(s.c1, s.svD.ID, "STUDENT", "REMOVED")
	require.Empty(t, classCodes(r.myCourses(r.mustLogin(s.svD.Email), "", nil)))
}

func TestMeCoursesStaff(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	gv := r.myCourses(r.mustLogin(s.gv.Email), "", nil)
	require.Equal(t, []string{s.code1, s.code2}, classCodes(gv))
	require.Equal(t, "TEACHER", gv.json()["items"].([]any)[0].(map[string]any)["role_in_course"])
	ta := r.myCourses(r.mustLogin(s.ta.Email), "", nil)
	require.Equal(t, []string{s.code1}, classCodes(ta))
	require.Equal(t, "TA", ta.json()["items"].([]any)[0].(map[string]any)["role_in_course"])
	// TA/GV chỉ thấy lớp ACTIVE: PENDING của staff không tồn tại, REMOVED bị ẩn
	extra := r.addUser(uniq("ta2"), store.UserRoleTA, store.UserStatusACTIVE)
	r.enroll(s.c1, extra.ID, "TA", "REMOVED")
	require.Empty(t, classCodes(r.myCourses(r.mustLogin(extra.Email), "", nil)))
}

func TestMeCoursesAdminEmpty(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	res := r.myCourses(r.mustLogin(s.adm.Email), "", nil)
	require.Equal(t, http.StatusOK, res.code)
	require.Empty(t, res.json()["items"])
	require.Nil(t, res.json()["next_cursor"])
	require.NotNil(t, res.json()["items"], "items luôn là mảng, không phải null")
}

func TestMeCoursesNoJoinCode(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	for _, u := range []store.User{s.svA, s.gv, s.ta} {
		res := r.myCourses(r.mustLogin(u.Email), "?limit=100", nil)
		require.NotContains(t, string(res.body), "join_code")
		require.NotContains(t, string(res.body), "join_url")
	}
	var jc string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select join_code from courses where id = $1`, s.c1).Scan(&jc))
	require.NotContains(t, string(r.myCourses(r.mustLogin(s.svA.Email), "", nil).body), jc)
}

func TestMeCoursesCursor(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	for i := 0; i < 5; i++ {
		c := r.course(s.adm.ID, fmt.Sprintf("P%d-%s", i, uuid.NewString()[:5]))
		r.enroll(c, s.svA.ID, "STUDENT", "ACTIVE")
	}
	sess := r.mustLogin(s.svA.Email)
	seen := map[string]bool{}
	cursor := ""
	for pages := 0; pages < 10; pages++ {
		q := "?limit=3"
		if cursor != "" {
			q += "&cursor=" + url.QueryEscape(cursor)
		}
		res := r.myCourses(sess, q, nil)
		require.Equal(t, http.StatusOK, res.code)
		for _, c := range classCodes(res) {
			require.False(t, seen[c], c)
			seen[c] = true
		}
		next, _ := res.json()["next_cursor"].(string)
		if next == "" {
			break
		}
		cursor = next
	}
	require.Len(t, seen, 7)
	require.Equal(t, http.StatusUnprocessableEntity, r.myCourses(sess, "?limit=101", nil).code)
	require.Equal(t, http.StatusUnprocessableEntity, r.myCourses(sess, "?cursor=hong", nil).code)
}

func TestMeCoursesETag(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	sess := r.mustLogin(s.svA.Email)
	first := r.myCourses(sess, "", nil)
	etag := first.hdr.Get("ETag")
	require.NotEmpty(t, etag)
	again := r.myCourses(sess, "", map[string]string{"If-None-Match": etag})
	require.Equal(t, http.StatusNotModified, again.code)
	require.Empty(t, again.body)
	r.enroll(r.course(s.adm.ID, "NEW-"+uuid.NewString()[:5]), s.svA.ID, "STUDENT", "ACTIVE")
	changed := r.myCourses(sess, "", map[string]string{"If-None-Match": etag})
	require.Equal(t, http.StatusOK, changed.code, "dữ liệu đổi ⇒ ETag đổi")
	require.NotEqual(t, etag, changed.hdr.Get("ETag"))
}

func TestMeCoursesNoNPlusOne(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	for i := 0; i < 20; i++ {
		r.enroll(r.course(s.adm.ID, fmt.Sprintf("N%d-%s", i, uuid.NewString()[:5])), s.svA.ID, "STUDENT", "ACTIVE")
	}
	sess := r.mustLogin(s.svA.Email)
	before := r.qc.n.Load()
	res := r.myCourses(sess, "?limit=100", nil)
	require.Equal(t, http.StatusOK, res.code)
	require.Len(t, res.json()["items"], 22)
	// 1 truy vấn danh sách (+ kiểm thu hồi phiên không chạm DB)
	require.EqualValues(t, 1, r.qc.n.Load()-before, "đúng một câu SQL cho cả trang")
}

// AC8.
func (r *rig) getCourse(s session, id any) resp {
	return r.do(req{method: http.MethodGet, path: fmt.Sprintf("/courses/%v", id), bearer: s.access})
}

func TestGetCourseStudentView(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	res := r.getCourse(r.mustLogin(s.svB.Email), s.c1)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	m := res.json()
	require.Equal(t, s.code1, m["class_code"])
	require.Equal(t, "STUDENT", m["my_role"])
	require.Len(t, m["teachers"], 1)
	require.NotContains(t, string(res.body), "join_code")
	require.NotContains(t, m, "counts", "sinh viên không thấy số liệu / thành viên")
	require.NotContains(t, string(res.body), "members")
	for _, k := range []string{"id", "class_code", "subject_code", "name", "semester", "status", "teachers", "my_role"} {
		require.Contains(t, m, k)
	}
}

func TestGetCourseStaffView(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	r.enroll(s.c1, s.svD.ID, "STUDENT", "PENDING")
	for _, u := range []store.User{s.gv, s.ta} {
		res := r.getCourse(r.mustLogin(u.Email), s.c1)
		require.Equal(t, http.StatusOK, res.code)
		c := res.json()["counts"].(map[string]any)
		require.EqualValues(t, 2, c["students_active"])
		require.EqualValues(t, 1, c["students_pending"])
		require.NotContains(t, string(res.body), "join_code")
	}
}

func TestGetCourseAdminBasic(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	res := r.getCourse(r.mustLogin(s.adm.Email), s.c1)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, "ADMIN", res.json()["my_role"])
	require.NotContains(t, res.json(), "counts", "ADMIN xem bản cơ bản")
	require.Equal(t, http.StatusNotFound, r.getCourse(r.mustLogin(s.adm.Email), uuid.NewString()).code, "ADMIN: lớp không có thật ⇒ 404")
}

func TestGetCourseOutsider403(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	svB := r.mustLogin(s.svB.Email)
	outsider := r.getCourse(svB, s.c2)
	ghost := r.getCourse(svB, uuid.NewString())
	require.Equal(t, http.StatusForbidden, outsider.code)
	require.Equal(t, http.StatusForbidden, ghost.code, "lớp không tồn tại cũng 403, không lộ")
	require.Equal(t, "course", outsider.details()["reason"])
	require.JSONEq(t, stripTrace(outsider), stripTrace(ghost))
	require.Equal(t, http.StatusNotFound, r.getCourse(svB, "khong-phai-uuid").code)
	// PENDING / REMOVED cũng 403
	r.enroll(s.c2, s.svB.ID, "STUDENT", "PENDING")
	require.Equal(t, http.StatusForbidden, r.getCourse(svB, s.c2).code)
	r.enroll(s.c1, s.svD.ID, "STUDENT", "REMOVED")
	require.Equal(t, http.StatusForbidden, r.getCourse(r.mustLogin(s.svD.Email), s.c1).code)
}

func stripTrace(x resp) string {
	m := x.json()
	delete(m, "trace_id")
	b, _ := json.Marshal(m)
	return string(b)
}

// AC14.
func TestMeEndpointsRequireJWT(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	for _, p := range []string{"/me/courses", "/courses/" + s.c1.String()} {
		res := r.do(req{method: http.MethodGet, path: p})
		require.Equal(t, http.StatusUnauthorized, res.code, p)
		require.Equal(t, "UNAUTHENTICATED", res.errCode())
	}
}

func TestMeEndpointsNoUserIDParam(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	a := r.mustLogin(s.svB.Email)
	res := r.myCourses(a, "?user_id="+s.svA.ID.String(), nil)
	require.Equal(t, http.StatusOK, res.code)
	require.Equal(t, []string{s.code1}, classCodes(res), "tham số user_id bị bỏ qua: vẫn là lớp của B")
}

func TestAllRolesGetOwnCourses(t *testing.T) {
	r := newRig(t)
	s := r.seed2()
	for _, u := range []store.User{s.adm, s.gv, s.ta, s.svA} {
		require.Equal(t, http.StatusOK, r.myCourses(r.mustLogin(u.Email), "", nil).code, string(u.Role))
	}
}
