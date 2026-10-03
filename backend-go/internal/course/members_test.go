package course_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// deliver chạy trình xử lý worker cho MỌI tin outbox của lớp thuộc topic đã cho.
func (r *rig) deliver(topic, courseID string) {
	r.t.Helper()
	rows, err := r.pool.Query(r.t.Context(), `select id, payload from outbox where topic = $1 and payload->>'course_id' = $2 order by created_at, id`, topic, courseID)
	require.NoError(r.t, err)
	var msgs []outbox.Message
	for rows.Next() {
		var m outbox.Message
		require.NoError(r.t, rows.Scan(&m.ID, &m.Payload))
		m.Topic = topic
		msgs = append(msgs, m)
	}
	require.NoError(r.t, rows.Err())
	n := r.notifier()
	for _, m := range msgs {
		var err error
		switch topic {
		case course.TopicJoinRequested:
			err = n.HandleJoinRequested(context.Background(), m)
		case course.TopicJoinDecided:
			err = n.HandleJoinDecided(context.Background(), m)
		default:
			err = n.HandleAssigned(context.Background(), m)
		}
		require.NoError(r.t, err)
	}
}

func (r *rig) member(s session, op, courseID string, uid uuid.UUID, body any) resp {
	path := "/courses/" + courseID + "/members/" + uid.String()
	switch op {
	case "delete":
		return r.del(s, path)
	default:
		return r.post(s, path+"/"+op, body, nil)
	}
}

// pendingKlass: lớp bật duyệt + một sinh viên đã gửi yêu cầu.
func (r *rig) pendingKlass() (klass, store.User, session) {
	k := r.klass(nil)
	require.Equal(r.t, http.StatusOK, r.setJoin(k, map[string]any{"require_approval": true}).code)
	u, sv := r.student("")
	require.Equal(r.t, http.StatusOK, r.join(sv, k.code).code)
	return k, u, sv
}

func (r *rig) notesOfType(s session, typ string) []map[string]any {
	r.t.Helper()
	var out []map[string]any
	for _, it := range r.get(s, "/notifications?limit=100").json()["items"].([]any) {
		m := it.(map[string]any)
		if m["type"] == typ {
			out = append(out, m)
		}
	}
	return out
}

// ===== AC7 =====

func TestJoinRequiresApprovalPending(t *testing.T) {
	r := newRig(t)
	k, u, sv := r.pendingKlass()
	st, _, _, _ := r.enrollment(k.id, u.ID)
	require.Equal(t, "PENDING", st)
	again := r.join(sv, k.code)
	require.Equal(t, "PENDING", again.json()["status"])
	require.Equal(t, true, again.json()["already_member"])
}

func TestPendingHasNoAccess(t *testing.T) {
	r := newRig(t)
	k, _, sv := r.pendingKlass()
	for _, p := range []string{"", "/members", "/join-code", "/assistant-candidates"} {
		res := r.get(sv, "/courses/"+k.id+p)
		require.Equal(t, http.StatusForbidden, res.code, p)
		require.Equal(t, "course", res.details()["reason"], p)
	}
	for _, it := range r.get(sv, "/me/courses").json()["items"].([]any) { // chỉ thấy chính yêu cầu của mình, ở trạng thái PENDING
		require.Equal(t, "PENDING", it.(map[string]any)["enrollment_status"])
	}
}

func TestJoinRequestNotifiesStaff(t *testing.T) {
	r := newRig(t)
	k, u, sv := r.pendingKlass()
	_, err := r.pool.Exec(t.Context(), `update users set full_name = 'Nguyễn Văn An' where id = $1`, u.ID)
	require.NoError(t, err)
	r.deliver(course.TopicJoinRequested, k.id)
	r.deliver(course.TopicJoinRequested, k.id) // giao lại: không nhân đôi
	cls := r.scalar(`select class_code from courses where id = $1`, k.id)
	for name, s := range map[string]session{"giảng viên": k.a.G, "trợ giảng": k.a.T} {
		got := r.notesOfType(s, "JOIN_REQUEST")
		require.Len(t, got, 1, name)
		require.Equal(t, "Nguyễn Văn An xin vào lớp "+cls, got[0]["title"], name)
		require.Equal(t, "/class/members?course="+k.id+"&tab=pending", got[0]["link"], name)
	}
	require.Empty(t, r.notesOfType(sv, "JOIN_REQUEST"), "sinh viên không nhận")
	require.Empty(t, r.notesOfType(k.a.G2, "JOIN_REQUEST"), "giảng viên lớp khác không nhận")
}

func TestPendingResubmitNoDuplicate(t *testing.T) {
	r := newRig(t)
	k, u, sv := r.pendingKlass()
	for range 3 {
		require.Equal(t, http.StatusOK, r.join(sv, k.code).code)
	}
	require.Equal(t, 1, r.count(`select count(*) from enrollments where course_id = $1 and user_id = $2`, k.id, u.ID))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'course.join_requested' and payload->>'course_id' = $1`, k.id), "gửi lại không thông báo lại")
}

// ===== AC8 =====

func TestApproveReject(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"require_approval": true}).code)
	var users []store.User
	var sessions []session
	for range 4 {
		u, s := r.student("")
		require.Equal(t, http.StatusOK, r.join(s, k.code).code)
		users, sessions = append(users, u), append(sessions, s)
	}
	// TA duyệt
	res := r.member(k.a.T, "approve", k.id, users[0].ID, nil)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, map[string]any{"user_id": users[0].ID.String(), "status": "ACTIVE", "previous_status": "PENDING", "warning": nil}, res.json())
	require.Equal(t, http.StatusOK, r.get(sessions[0], "/courses/"+k.id).code, "được duyệt ⇒ vào lớp ngay")
	// giảng viên từ chối, Admin duyệt
	res = r.member(k.a.G, "reject", k.id, users[1].ID, nil)
	require.Equal(t, http.StatusOK, res.code)
	require.Equal(t, "REMOVED", res.json()["status"])
	require.Equal(t, http.StatusOK, r.member(k.a.A, "approve", k.id, users[2].ID, nil).code)
	// sinh viên không duyệt được; người của lớp khác không duyệt được
	_, outsider := r.student("")
	for name, s := range map[string]session{"SV trong lớp": sessions[0], "SV chờ": sessions[3], "SV ngoài": outsider, "GV lớp khác": k.a.G2} {
		require.Equal(t, http.StatusForbidden, r.member(s, "approve", k.id, users[3].ID, nil).code, name)
	}
	// không còn chờ ⇒ 409; không có ai ⇒ 404
	again := r.member(k.a.G, "approve", k.id, users[0].ID, nil)
	require.Equal(t, http.StatusConflict, again.code)
	require.Equal(t, "not_pending", again.details()["reason"])
	require.Equal(t, http.StatusNotFound, r.member(k.a.G, "approve", k.id, uuid.New(), nil).code)
	var by, prev string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select status_changed_by::text, previous_status::text from enrollments where course_id = $1 and user_id = $2`, k.id, users[0].ID).Scan(&by, &prev))
	require.Equal(t, k.a.ta.ID.String(), by)
	require.Equal(t, "PENDING", prev)
}

func TestRemoveTeacherOnly(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u, sv := r.student("")
	require.Equal(t, http.StatusOK, r.join(sv, k.code).code)
	res := r.member(k.a.T, "delete", k.id, u.ID, nil)
	require.Equal(t, http.StatusForbidden, res.code, "TA không mời ra")
	require.Equal(t, http.StatusForbidden, r.member(sv, "delete", k.id, u.ID, nil).code)
	res = r.member(k.a.G, "delete", k.id, u.ID, nil)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, "REMOVED", res.json()["status"])
	require.Equal(t, "ACTIVE", res.json()["previous_status"])
	again := r.member(k.a.A, "delete", k.id, u.ID, nil)
	require.Equal(t, http.StatusConflict, again.code)
	require.Equal(t, "not_active", again.details()["reason"])
}

func TestApproveRechecksCapacity(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"require_approval": true, "capacity": 1}).code)
	a, sa := r.student("")
	b, sb := r.student("")
	require.Equal(t, "PENDING", r.join(sa, k.code).json()["status"])
	require.Equal(t, "PENDING", r.join(sb, k.code).json()["status"])
	require.Equal(t, http.StatusOK, r.member(k.a.G, "approve", k.id, a.ID, nil).code)
	res := r.member(k.a.G, "approve", k.id, b.ID, nil)
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "COURSE_FULL", res.errCode())
	st, _, _, _ := r.enrollment(k.id, b.ID)
	require.Equal(t, "PENDING", st)
}

func TestUndoWithin60s(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"require_approval": true}).code)
	u1, s1 := r.student("")
	u2, s2 := r.student("")
	r.join(s1, k.code)
	r.join(s2, k.code)
	// duyệt rồi hoàn tác ⇒ về PENDING, mất quyền
	require.Equal(t, http.StatusOK, r.member(k.a.G, "approve", k.id, u1.ID, nil).code)
	r.clk.Advance(30 * time.Second)
	res := r.member(k.a.T, "undo", k.id, u1.ID, nil) // TA cùng quyền với hành động gốc (duyệt)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, "PENDING", res.json()["status"])
	require.Equal(t, http.StatusForbidden, r.get(s1, "/courses/"+k.id).code)
	// từ chối rồi hoàn tác
	require.Equal(t, http.StatusOK, r.member(k.a.G, "reject", k.id, u2.ID, nil).code)
	require.Equal(t, http.StatusOK, r.member(k.a.G, "undo", k.id, u2.ID, nil).code)
	st, _, _, _ := r.enrollment(k.id, u2.ID)
	require.Equal(t, "PENDING", st)
	require.Equal(t, 0, r.count(`select count(*) from enrollments where user_id = $1 and removed_at is not null`, u2.ID))
	// mời ra rồi hoàn tác: TA không hoàn tác được (hành động gốc chỉ giảng viên)
	require.Equal(t, http.StatusOK, r.member(k.a.G, "approve", k.id, u2.ID, nil).code)
	require.Equal(t, http.StatusOK, r.member(k.a.G, "delete", k.id, u2.ID, nil).code)
	require.Equal(t, http.StatusForbidden, r.member(k.a.T, "undo", k.id, u2.ID, nil).code)
	back := r.member(k.a.G, "undo", k.id, u2.ID, nil)
	require.Equal(t, http.StatusOK, back.code, string(back.body))
	require.Equal(t, "ACTIVE", back.json()["status"])
	require.Equal(t, http.StatusOK, r.get(s2, "/courses/"+k.id).code)
}

func TestUndoExpired(t *testing.T) {
	r := newRig(t)
	k, u, _ := r.pendingKlass()
	require.Equal(t, http.StatusOK, r.member(k.a.G, "approve", k.id, u.ID, nil).code)
	r.clk.Advance(61 * time.Second)
	gv := r.mustLogin(k.a.gv.Email)
	res := r.member(gv, "undo", k.id, u.ID, nil)
	require.Equal(t, http.StatusConflict, res.code, string(res.body))
	require.Equal(t, "CONFLICT", res.errCode())
	require.Equal(t, "undo_expired", res.details()["reason"])
	// không có gì để hoàn tác (lần hai sau khi đã hoàn tác)
	u2, s2 := r.student("")
	r.join(s2, k.code)
	require.Equal(t, http.StatusOK, r.member(gv, "approve", k.id, u2.ID, nil).code)
	require.Equal(t, http.StatusOK, r.member(gv, "undo", k.id, u2.ID, nil).code)
	again := r.member(gv, "undo", k.id, u2.ID, nil)
	require.Equal(t, http.StatusConflict, again.code)
	require.Equal(t, "undo_expired", again.details()["reason"])
}

func TestDecisionNotifiesStudent(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"require_approval": true}).code)
	ua, sa := r.student("")
	ub, sb := r.student("")
	r.join(sa, k.code)
	r.join(sb, k.code)
	require.Equal(t, http.StatusOK, r.member(k.a.G, "approve", k.id, ua.ID, nil).code)
	require.Equal(t, http.StatusOK, r.member(k.a.G, "reject", k.id, ub.ID, nil).code)
	r.deliver(course.TopicJoinDecided, k.id)
	r.deliver(course.TopicJoinDecided, k.id)
	name, cls := r.scalar(`select name from courses where id = $1`, k.id), r.scalar(`select class_code from courses where id = $1`, k.id)
	ok := r.notesOfType(sa, "JOIN_APPROVED")
	require.Len(t, ok, 1)
	require.Equal(t, fmt.Sprintf("Bạn đã được duyệt vào lớp %s – %s", name, cls), ok[0]["title"])
	require.Nil(t, ok[0]["link"], "link rỗng ⇒ chuông mở Hôm nay (CHECK của 00003 không nhận \"/\", đề xuất #9)")
	no := r.notesOfType(sb, "JOIN_REJECTED")
	require.Len(t, no, 1)
	require.Equal(t, "Yêu cầu vào lớp "+cls+" chưa được chấp nhận", no[0]["title"])
	require.Equal(t, "/join", no[0]["link"])
	require.Empty(t, r.notesOfType(sa, "JOIN_REJECTED"))
}

func TestCannotRemoveStaffViaMembers(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	for _, uid := range []uuid.UUID{k.a.gv.ID, k.a.ta.ID} {
		for _, op := range []string{"delete", "approve", "reject"} {
			s := k.a.A
			res := r.member(s, op, k.id, uid, nil)
			require.Equal(t, http.StatusUnprocessableEntity, res.code, op+" "+string(res.body))
		}
	}
	teacher, tas := r.staffOf(k.id)
	require.Equal(t, k.a.gv.ID.String(), teacher)
	require.Equal(t, []string{k.a.ta.ID.String()}, tas)
}

func TestMemberActionsAudit(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"require_approval": true}).code)
	u, s := r.student("B20DCCN321")
	r.join(s, k.code)
	require.Equal(t, http.StatusOK, r.member(k.a.G, "approve", k.id, u.ID, nil).code)
	require.Equal(t, http.StatusOK, r.member(k.a.G, "undo", k.id, u.ID, nil).code)
	require.Equal(t, http.StatusOK, r.member(k.a.G, "reject", k.id, u.ID, nil).code)
	require.Equal(t, http.StatusOK, r.member(k.a.G, "undo", k.id, u.ID, nil).code)
	require.Equal(t, http.StatusOK, r.member(k.a.G, "approve", k.id, u.ID, nil).code)
	require.Equal(t, http.StatusOK, r.member(k.a.G, "delete", k.id, u.ID, nil).code)
	for _, a := range []string{"member_joined", "member_approved", "member_rejected", "member_undo", "member_removed"} {
		require.Positive(t, r.count(`select count(*) from audit_log where entity = 'enrollment' and action = $1 and course_id = $2`, a, k.id), a)
	}
	require.Equal(t, 0, r.count(`select count(*) from audit_log where course_id = $1 and entity = 'enrollment' and (coalesce(before::text,'') || coalesce(after::text,'')) ~ ($2 || '|' || $3)`, k.id, u.Email, "B20DCCN321"), "audit không chứa email hay MSSV")
	require.Equal(t, k.a.gv.ID.String(), r.scalar(`select actor_id::text from audit_log where entity = 'enrollment' and action = 'member_removed' and course_id = $1`, k.id))
}

// ===== AC9 =====

func TestRemovedLosesAccessImmediately(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u, sv := r.student("")
	r.join(sv, k.code)
	require.Equal(t, http.StatusOK, r.get(sv, "/courses/"+k.id).code)
	require.Equal(t, http.StatusOK, r.member(k.a.G, "delete", k.id, u.ID, nil).code)
	res := r.get(sv, "/courses/"+k.id)
	require.Equal(t, http.StatusForbidden, res.code, "ngay yêu cầu kế tiếp, không cache")
	require.Equal(t, "course", res.details()["reason"])
}

func TestRejoinReusesRowPending(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u, sv := r.student("B20DCCN777")
	r.join(sv, k.code)
	var before string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select id::text from enrollments where course_id = $1 and user_id = $2`, k.id, u.ID).Scan(&before))
	require.Equal(t, http.StatusOK, r.member(k.a.G, "delete", k.id, u.ID, nil).code)
	_, err := r.pool.Exec(t.Context(), `update users set student_code = 'B20DCCN888' where id = $1`, u.ID)
	require.NoError(t, err)
	res := r.join(sv, k.code)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, "PENDING", res.json()["status"], "vào lại luôn chờ duyệt (mặc định an toàn), kể cả lớp không bật duyệt")
	require.Equal(t, 1, r.count(`select count(*) from enrollments where course_id = $1 and user_id = $2`, k.id, u.ID))
	var after string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select id::text from enrollments where course_id = $1 and user_id = $2`, k.id, u.ID).Scan(&after))
	require.Equal(t, before, after, "cùng dòng")
	_, _, snap, _ := r.enrollment(k.id, u.ID)
	require.Equal(t, "B20DCCN777", snap, "ảnh chụp MSSV cũ giữ nguyên")
	require.Equal(t, http.StatusForbidden, r.get(sv, "/courses/"+k.id).code)
	require.Equal(t, http.StatusOK, r.member(k.a.G, "approve", k.id, u.ID, nil).code)
	require.Equal(t, http.StatusOK, r.get(sv, "/courses/"+k.id).code)
}

func TestRemovedKeepsLearningData(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u, sv := r.student("")
	r.join(sv, k.code)
	note := r.notify(u.ID, &k.course, "đã có từ trước", time.Now().UTC())
	require.Equal(t, http.StatusOK, r.member(k.a.G, "delete", k.id, u.ID, nil).code)
	require.Equal(t, 1, r.count(`select count(*) from notifications where id = $1`, note), "không xoá gì của người học")
	require.Equal(t, 1, r.count(`select count(*) from enrollments where course_id = $1 and user_id = $2 and status = 'REMOVED'`, k.id, u.ID), "dòng ghi danh còn, chỉ đổi trạng thái")
}

// ===== AC12 =====

func TestMembersAndJoinRBACMatrix(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	other := r.klass(nil)
	member, sm := r.student("")
	r.enroll(k.course, member.ID, "STUDENT", "ACTIVE")
	_, outsider := r.student("")
	_, inOther := r.student("")
	_ = other
	target := uuid.New()
	type op struct {
		name string
		do   func(s session) resp
		// quyền: vai nào qua được guard
		staff, manage bool
	}
	ver := func() any { return 999 }
	ops := []op{
		{"GET join-code", func(s session) resp { return r.get(s, "/courses/"+k.id+"/join-code") }, true, false},
		{"GET members", func(s session) resp { return r.get(s, "/courses/"+k.id+"/members") }, true, false},
		{"POST approve", func(s session) resp { return r.member(s, "approve", k.id, target, nil) }, true, false},
		{"POST reject", func(s session) resp { return r.member(s, "reject", k.id, target, nil) }, true, false},
		{"POST undo", func(s session) resp { return r.member(s, "undo", k.id, target, nil) }, true, false},
		{"DELETE member", func(s session) resp { return r.member(s, "delete", k.id, target, nil) }, false, true},
		{"PUT join-settings", func(s session) resp {
			return r.put(s, "/courses/"+k.id+"/join-settings", map[string]any{"enabled": true, "version": ver()})
		}, false, true},
		{"POST regenerate", func(s session) resp { return r.post(s, "/courses/"+k.id+"/join-code/regenerate", nil, nil) }, false, true},
	}
	type who struct {
		name   string
		s      session
		noJWT  bool
		staff  bool // TA, GV, Admin
		manage bool // GV, Admin
	}
	whos := []who{
		{"SV thành viên", sm, false, false, false},
		{"SV ngoài lớp", outsider, false, false, false},
		{"SV lớp khác", inOther, false, false, false},
		{"TA của lớp", k.a.T, false, true, false},
		{"GV của lớp", k.a.G, false, true, true},
		{"GV lớp khác", k.a.G2, false, false, false},
		{"Admin", k.a.A, false, true, true},
		{"không JWT", session{}, true, false, false},
	}
	cases := 0
	for _, o := range ops {
		for _, w := range whos {
			res := o.do(w.s)
			cases++
			allowed := (o.staff && w.staff) || (o.manage && w.manage)
			switch {
			case w.noJWT:
				require.Equal(t, http.StatusUnauthorized, res.code, o.name+" / "+w.name)
			case allowed:
				require.NotContains(t, []int{http.StatusUnauthorized, http.StatusForbidden}, res.code, "%s / %s: %d %s", o.name, w.name, res.code, res.body)
			default:
				require.Equal(t, http.StatusForbidden, res.code, "%s / %s: %s", o.name, w.name, res.body)
			}
		}
	}
	// preview + join: chỉ sinh viên; người khác 403 reason=role (mã hiện hành: ca "regenerate" ở trên đã đổi mã)
	code := r.joinCodeOf(k.id)
	for _, w := range whos {
		for _, f := range []func(session, string) resp{r.preview, r.join} {
			res := f(w.s, code)
			cases++
			switch w.name {
			case "không JWT":
				require.Equal(t, http.StatusUnauthorized, res.code)
			case "SV thành viên", "SV ngoài lớp", "SV lớp khác":
				require.Equal(t, http.StatusOK, res.code, w.name+" "+string(res.body))
			default:
				require.Equal(t, http.StatusForbidden, res.code, w.name)
				require.Equal(t, "role", res.details()["reason"], w.name)
			}
		}
	}
	require.GreaterOrEqual(t, cases, 60)
}

// ===== AC13 =====

func (r *rig) fillMembers(k klass, n int) []store.User {
	r.t.Helper()
	var out []store.User
	for i := range n {
		u := r.addUser(uniq(fmt.Sprintf("m%02d", i)), store.UserRoleSTUDENT, store.UserStatusACTIVE)
		_, err := r.pool.Exec(r.t.Context(), `update users set full_name = $2 where id = $1`, u.ID, fmt.Sprintf("Sinh Viên %02d", i))
		require.NoError(r.t, err)
		_, err = r.pool.Exec(r.t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot, status_changed_at)
			values ($1, $2, 'STUDENT', $3::enrollment_status, 'CODE', $4, now() - ($5 || ' minutes')::interval)`, k.course, u.ID, map[bool]string{true: "PENDING", false: "ACTIVE"}[i%4 == 0], fmt.Sprintf("B20DC%05d", i), fmt.Sprint(i))
		require.NoError(r.t, err)
		out = append(out, u)
	}
	return out
}

func TestMembersListCursor(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	r.fillMembers(k, 9)
	seen := map[string]bool{}
	var order []string
	cursor := ""
	for range 10 {
		q := "/courses/" + k.id + "/members?limit=4&status=ACTIVE&role=STUDENT"
		if cursor != "" {
			q += "&cursor=" + cursor
		}
		res := r.get(k.a.G, q)
		require.Equal(t, http.StatusOK, res.code, string(res.body))
		for _, it := range res.json()["items"].([]any) {
			id := it.(map[string]any)["user_id"].(string)
			require.False(t, seen[id], "trùng giữa các trang")
			seen[id] = true
			order = append(order, it.(map[string]any)["full_name"].(string))
		}
		next, _ := res.json()["next_cursor"].(string)
		if next == "" {
			break
		}
		cursor = next
	}
	require.Len(t, seen, 6, "9 sinh viên, i%4==0 (3 người) đang chờ nên không thuộc ACTIVE")
	require.Equal(t, []string{"Sinh Viên 01", "Sinh Viên 02", "Sinh Viên 03", "Sinh Viên 05", "Sinh Viên 06", "Sinh Viên 07"}, order, "status_changed_at giảm dần, ổn định qua các trang")
	require.Equal(t, http.StatusUnprocessableEntity, r.get(k.a.G, "/courses/"+k.id+"/members?limit=0").code)
	require.Equal(t, http.StatusUnprocessableEntity, r.get(k.a.G, "/courses/"+k.id+"/members?cursor=hong").code)
}

func TestMembersListFilters(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	users := r.fillMembers(k, 9)
	names := func(q string) []string {
		var out []string
		for _, it := range r.get(k.a.T, "/courses/"+k.id+"/members?limit=100&"+q).json()["items"].([]any) {
			out = append(out, it.(map[string]any)["full_name"].(string))
		}
		return out
	}
	require.Len(t, names("status=PENDING"), 3)
	require.Len(t, names("status=ACTIVE&role=STUDENT"), 6)
	require.Len(t, names(""), 9+2, "mặc định ACTIVE + PENDING: 9 sinh viên + giảng viên + TA")
	require.Equal(t, []string{"Người Thử"}, names("role=TEACHER"))
	require.Equal(t, []string{"Người Thử"}, names("role=TA"))
	require.Equal(t, []string{"Sinh Viên 03"}, names("q=sinh+vien+03"), "tên không dấu")
	require.Equal(t, []string{"Sinh Viên 05"}, names("q=B20DC00005"), "tiền tố MSSV")
	require.Equal(t, []string{"Sinh Viên 02"}, names("q=m02."), "tiền tố email")
	_, err := r.pool.Exec(t.Context(), `update enrollments set status = 'REMOVED', removed_at = now() where course_id = $1 and user_id = $2`, k.id, users[1].ID)
	require.NoError(t, err)
	require.Equal(t, []string{"Sinh Viên 01"}, names("status=REMOVED"))
	require.Equal(t, http.StatusUnprocessableEntity, r.get(k.a.G, "/courses/"+k.id+"/members?status=NOPE").code)
	require.Equal(t, http.StatusUnprocessableEntity, r.get(k.a.G, "/courses/"+k.id+"/members?role=NOPE").code)
}

func TestMembersListMinimalFields(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	r.fillMembers(k, 3)
	res := r.get(k.a.G, "/courses/"+k.id+"/members")
	require.Equal(t, http.StatusOK, res.code)
	m := res.json()
	require.Len(t, m, 3)
	require.EqualValues(t, 2, m["counts"].(map[string]any)["active"])
	require.EqualValues(t, 1, m["counts"].(map[string]any)["pending"])
	item := m["items"].([]any)[0].(map[string]any)
	require.Len(t, item, 9)
	for _, key := range []string{"user_id", "full_name", "email", "student_code", "role_in_course", "status", "joined_via", "warning", "status_changed_at"} {
		require.Contains(t, item, key)
	}
	body := strings.ToLower(string(res.body))
	for _, bad := range []string{"password", "ics_token", "hash", "grade", "attendance", "điểm"} {
		require.NotContains(t, body, bad)
	}
	// trang rỗng vẫn mang số đếm
	empty := r.get(k.a.G, "/courses/"+k.id+"/members?q=khongco")
	require.Empty(t, empty.json()["items"])
	require.EqualValues(t, 2, empty.json()["counts"].(map[string]any)["active"])
}

func TestMembersListNoNPlusOne(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	r.fillMembers(k, 40)
	before := r.qc.n.Load()
	res := r.get(k.a.G, "/courses/"+k.id+"/members?limit=100")
	require.Equal(t, http.StatusOK, res.code)
	require.Len(t, res.json()["items"], 40+2) // 30 ACTIVE + 10 PENDING + giảng viên + TA
	// đúng 2 câu: guard (tra ghi danh) + danh sách kèm số đếm, không phụ thuộc số dòng
	require.EqualValues(t, 2, r.qc.n.Load()-before)
}
