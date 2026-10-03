package integration_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

// student: sinh viên ACTIVE đã xác minh email, tuỳ chọn có MSSV tự khai.
func (r *rig) student(code string) (store.User, session) {
	r.t.Helper()
	u := r.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	_, err := r.pool.Exec(r.t.Context(), `update users set email_verified_at = now() where id = $1`, u.ID)
	require.NoError(r.t, err)
	if code != "" {
		_, err = r.pool.Exec(r.t.Context(), `update users set student_code = $2 where id = $1`, u.ID, code)
		require.NoError(r.t, err)
	}
	return u, r.mustLogin(u.Email)
}

func (r *rig) joinWith(s session, code string) resp {
	return r.do(req{method: http.MethodPost, path: "/courses/join", bearer: s.access, body: map[string]any{"code": code}})
}

func (r *rig) codeOf(courseID string) string {
	var c string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select join_code from courses where id = $1`, courseID).Scan(&c))
	return strings.TrimSpace(c)
}

func (r *rig) memberOp(s session, op, courseID string, uid uuid.UUID, body map[string]any) resp {
	return r.do(req{method: http.MethodPost, path: "/courses/" + courseID + "/members/" + uid.String() + "/" + op, bearer: s.access, body: body})
}

func (r *rig) enrollmentOf(courseID string, uid uuid.UUID) (status, warning string) {
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select status::text, coalesce(warning,'') from enrollments where course_id = $1 and user_id = $2`, courseID, uid).Scan(&status, &warning))
	return
}

func (r *rig) notesOf(s session, typ string) []map[string]any {
	var out []map[string]any
	for _, it := range r.notifications(s) {
		if m := it.(map[string]any); m["type"] == typ {
			out = append(out, m)
		}
	}
	return out
}

// AC10: MSSV tự khai trùng ảnh chụp của người khác ⇒ PENDING + EMAIL_MISMATCH bất kể lớp có bật duyệt hay không; không đọc được gì của lớp.
func TestJoinStudentCodeCollisionPending(t *testing.T) {
	r := newRig(t)
	r.worker()
	x := r.f2()
	id := r.openCourse(x.A, map[string]any{"teacher_id": x.gv.ID, "ta_ids": []string{x.ta.ID.String()}})
	code := r.codeOf(id)
	b, sb := r.student("20229002")
	require.Equal(t, "ACTIVE", r.joinWith(sb, code).json()["status"])

	xx, sx := r.student("20229002") // trùng MSSV với B, email đã xác minh của riêng X
	res := r.joinWith(sx, code)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, "PENDING", res.json()["status"], "lớp không bật duyệt nhưng vẫn phải chờ")
	st, warn := r.enrollmentOf(id, xx.ID)
	require.Equal(t, "PENDING", st)
	require.Equal(t, "EMAIL_MISMATCH", warn)
	require.Equal(t, http.StatusForbidden, r.do(req{method: http.MethodGet, path: "/courses/" + id, bearer: sx.access}).code, "X không đọc được gì của lớp")
	// B không bị ảnh hưởng
	bst, bwarn := r.enrollmentOf(id, b.ID)
	require.Equal(t, "ACTIVE", bst)
	require.Empty(t, bwarn)

	waitFor(t, 10*time.Second, "JOIN_REQUEST cho giảng viên và TA", func() bool {
		return len(r.notesOf(x.G, "JOIN_REQUEST")) == 1 && len(r.notesOf(x.T, "JOIN_REQUEST")) == 1
	})
	// Chỉ yêu cầu của X có cảnh báo; B vào bình thường nên không sinh JOIN_REQUEST (lớp không bật duyệt).
	for _, n := range r.notesOf(x.G, "JOIN_REQUEST") {
		require.True(t, strings.HasSuffix(n["body"].(string), "— email chưa khớp MSSV"), "giảng viên nhận cảnh báo: %v", n["body"])
	}
	for _, n := range r.notesOf(x.T, "JOIN_REQUEST") {
		body, _ := n["body"].(string)
		require.NotContains(t, body, "MSSV", "TA không biết lý do")
		require.NotContains(t, body, "chưa khớp")
	}
}

func TestMismatchApprovalNeedsTeacher(t *testing.T) {
	r := newRig(t)
	x := r.f2()
	id := r.openCourse(x.A, map[string]any{"teacher_id": x.gv.ID, "ta_ids": []string{x.ta.ID.String()}})
	code := r.codeOf(id)
	_, sb := r.student("20229002")
	r.joinWith(sb, code)
	u1, s1 := r.student("20229002")
	r.joinWith(s1, code)
	// giảng viên: thiếu cờ ⇒ 422; có cờ ⇒ duyệt được
	missing := r.memberOp(x.G, "approve", id, u1.ID, nil)
	require.Equal(t, http.StatusUnprocessableEntity, missing.code, string(missing.body))
	require.Contains(t, string(missing.body), "confirm_mismatch")
	st, _ := r.enrollmentOf(id, u1.ID)
	require.Equal(t, "PENDING", st)
	ok := r.memberOp(x.G, "approve", id, u1.ID, map[string]any{"confirm_mismatch": true})
	require.Equal(t, http.StatusOK, ok.code, string(ok.body))
	require.Equal(t, "ACTIVE", ok.json()["status"])
	// Admin cũng duyệt được khi có cờ
	u2, s2 := r.student("20229002")
	r.joinWith(s2, code)
	require.Equal(t, http.StatusOK, r.memberOp(x.A, "approve", id, u2.ID, map[string]any{"confirm_mismatch": true}).code)
}

func TestMismatchTAForbidden(t *testing.T) {
	r := newRig(t)
	x := r.f2()
	id := r.openCourse(x.A, map[string]any{"teacher_id": x.gv.ID, "ta_ids": []string{x.ta.ID.String()}})
	code := r.codeOf(id)
	_, sb := r.student("20229002")
	r.joinWith(sb, code)
	u, su := r.student("20229002")
	r.joinWith(su, code)
	for _, body := range []map[string]any{nil, {"confirm_mismatch": true}} {
		res := r.memberOp(x.T, "approve", id, u.ID, body)
		require.Equal(t, http.StatusForbidden, res.code, string(res.body))
		require.Equal(t, "mismatch_needs_teacher", res.details()["reason"])
	}
	st, _ := r.enrollmentOf(id, u.ID)
	require.Equal(t, "PENDING", st)
	// TA vẫn từ chối được
	require.Equal(t, http.StatusOK, r.memberOp(x.T, "reject", id, u.ID, nil).code)
}

// Thứ tự ngược: X vào trước (chưa ai giữ MSSV) ⇒ ACTIVE; B đến sau ⇒ B bị gắn cảnh báo; X không bị đổi gì.
func TestMismatchReverseOrder(t *testing.T) {
	r := newRig(t)
	x := r.f2()
	id := r.openCourse(x.A, map[string]any{"teacher_id": x.gv.ID})
	code := r.codeOf(id)
	ux, sx := r.student("20229002")
	require.Equal(t, "ACTIVE", r.joinWith(sx, code).json()["status"])
	ub, sb := r.student("20229002")
	res := r.joinWith(sb, code)
	require.Equal(t, "PENDING", res.json()["status"])
	st, warn := r.enrollmentOf(id, ub.ID)
	require.Equal(t, "PENDING", st)
	require.Equal(t, "EMAIL_MISMATCH", warn)
	xs, xw := r.enrollmentOf(id, ux.ID)
	require.Equal(t, "ACTIVE", xs, "người vào trước không bị gỡ quyền hay đổi trạng thái")
	require.Empty(t, xw)
	require.Equal(t, http.StatusOK, r.do(req{method: http.MethodGet, path: "/courses/" + id, bearer: sx.access}).code)
	require.Equal(t, http.StatusForbidden, r.do(req{method: http.MethodGet, path: "/courses/" + id, bearer: sb.access}).code)
}

// AC2 (cấp hệ thống, worker thật): 10 yêu cầu song song cùng mã của một sinh viên ⇒ đúng một dòng.
func TestJoinConcurrentTenRequests(t *testing.T) {
	r := newRig(t)
	x := r.f2()
	id := r.openCourse(x.A, map[string]any{"teacher_id": x.gv.ID})
	code := r.codeOf(id)
	u, s := r.student("")
	done := make(chan resp, 10)
	for range 10 {
		go func() { done <- r.joinWith(s, code) }()
	}
	for range 10 {
		res := <-done
		require.Equal(t, http.StatusOK, res.code, string(res.body))
	}
	var n int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from enrollments where course_id = $1 and user_id = $2`, id, u.ID).Scan(&n))
	require.Equal(t, 1, n)
}
