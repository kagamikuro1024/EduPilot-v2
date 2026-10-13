package thread_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/knowledge"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/thread"
)

// US-P4-06: kiểm duyệt Threads (báo cáo, tự ẩn, xử lý của Staff, sửa / xoá của chủ bài, ghim, tri thức, thông báo gộp).

func (r *rig) student(name string) thread.Actor {
	r.t.Helper()
	a := thread.Actor{UserID: r.user(name, "STUDENT", ""), Role: auth.RoleStudent}
	r.enroll(a.UserID, "STUDENT", "")
	return a
}

func (r *rig) report(a thread.Actor, kind string, id uuid.UUID) error {
	return r.svc.Report(r.t.Context(), a, r.course, kind, id, thread.ReportIn{Reason: "SPAM"})
}

func (r *rig) hidden(id uuid.UUID) bool {
	return r.count(`select count(*) from forum_threads where id=$1 and hidden_at is not null`, id) == 1
}

// comment đăng một bình luận (HUMAN) và trả id bài.
func (r *rig) comment(a thread.Actor, thread_ uuid.UUID) uuid.UUID {
	r.t.Helper()
	p, err := r.svc.Comment(r.t.Context(), a, r.course, thread_, thread.CommentIn{Body: "Bình luận thử"})
	require.NoError(r.t, err)
	return p.ID
}

func (r *rig) notifier() *thread.Notifier {
	return &thread.Notifier{Q: store.New(r.pool), Notify: notify.New(r.pool, nil, nil)}
}

// lastMsg đọc dòng outbox mới nhất của một topic (rig không chạy worker).
func (r *rig) lastMsg(topic string) outbox.Message {
	r.t.Helper()
	m := outbox.Message{Topic: topic}
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select id, payload from outbox where topic = $1 order by created_at desc, id desc limit 1`, topic).Scan(&m.ID, &m.Payload))
	return m
}

// ===== AC1: báo cáo =====

func TestReportOncePerUser(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	require.NoError(t, r.report(r.other, thread.KindThread, id))
	require.Equal(t, "ALREADY_REPORTED", code(r.report(r.other, thread.KindThread, id)))
	require.Equal(t, 1, r.count(`select count(*) from post_reports where target_id=$1`, id))
	require.Equal(t, "VALIDATION_FAILED", code(r.svc.Report(t.Context(), r.ta, r.course, thread.KindThread, id, thread.ReportIn{Reason: "KHONG_CO"})))
	n := "x"
	require.Equal(t, "VALIDATION_FAILED", code(r.svc.Report(t.Context(), r.ta, r.course, thread.KindThread, id, thread.ReportIn{Reason: "SPAM", Note: &n})), "ghi chú chỉ với OTHER")
	require.NoError(t, r.svc.Report(t.Context(), r.ta, r.course, thread.KindThread, id, thread.ReportIn{Reason: "OTHER", Note: &n}))
}

func TestReportOwnPost(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	require.Equal(t, "CANNOT_REPORT_OWN", code(r.report(r.sv, thread.KindThread, id)))
	pid := r.comment(r.other, id)
	require.Equal(t, "CANNOT_REPORT_OWN", code(r.report(r.other, thread.KindPost, pid)))
	require.NoError(t, r.report(r.sv, thread.KindPost, pid))
	// bài AI không có tác giả: ai cũng báo được
	r.rag.hits = nil
	ai := r.aiPost(r.answered())
	require.NoError(t, r.report(r.sv, thread.KindPost, ai))
}

func TestReportCrossCourse(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	other := r.newCourse(r.teacher.UserID)
	err := r.svc.Report(t.Context(), r.other, other, thread.KindThread, id, thread.ReportIn{Reason: "SPAM"})
	require.Equal(t, 404, status(err))
	require.Equal(t, 404, status(r.svc.Report(t.Context(), r.other, r.course, thread.KindThread, uuid.New(), thread.ReportIn{Reason: "SPAM"})))
	require.Zero(t, r.count(`select count(*) from post_reports`))
}

func TestReportAdminForbidden(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	id := h.mustCreate(h.sv, "Hỏi", clean)
	admin := h.user("Quản trị", "ADMIN", "")
	res := h.do(admin, auth.RoleAdmin, http.MethodPost, "/courses/"+h.course.String()+"/threads/"+id.String()+"/report", `{"reason":"SPAM"}`)
	require.Equal(t, http.StatusForbidden, res.StatusCode)
	res = h.do(h.other.UserID, auth.RoleStudent, http.MethodPost, "/courses/"+h.course.String()+"/threads/"+id.String()+"/report", `{"reason":"SPAM"}`)
	require.Equal(t, http.StatusCreated, res.StatusCode)
}

func TestReporterHidden(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	require.NoError(t, r.report(r.other, thread.KindThread, id))
	for _, a := range []thread.Actor{r.sv, r.other} {
		v, err := r.svc.Get(t.Context(), a, r.course, id, pg(30))
		require.NoError(t, err)
		raw, _ := json.Marshal(v)
		for _, k := range []string{"report", "reporter", "SPAM"} {
			require.NotContains(t, string(raw), k, "sinh viên không thấy số / người báo cáo")
		}
	}
}

func TestReportRateLimit(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.svc.Limits = thread.Limits{ReportPerHour: 2}
	var ids []uuid.UUID
	for i := range 3 {
		ids = append(ids, r.mustCreate(r.sv, fmt.Sprintf("Hỏi %d", i), clean))
	}
	require.NoError(t, r.report(r.other, thread.KindThread, ids[0]))
	require.NoError(t, r.report(r.other, thread.KindThread, ids[1]))
	require.Equal(t, 429, status(r.report(r.other, thread.KindThread, ids[2])))
	require.Zero(t, r.count(`select count(*) from post_reports where target_id=$1`, ids[2]))
}

// ===== AC2: tự ẩn =====

// reporters trả n học viên khác chủ bài (không gồm sv).
func (r *rig) reporters(n int) []thread.Actor {
	out := []thread.Actor{r.other}
	for i := 1; i < n; i++ {
		out = append(out, r.student(fmt.Sprintf("Sinh Viên %d", i)))
	}
	return out[:n]
}

func TestAutoHideAt3Distinct(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	rs := r.reporters(3)
	require.NoError(t, r.report(rs[0], thread.KindThread, id))
	require.NoError(t, r.report(rs[1], thread.KindThread, id))
	require.False(t, r.hidden(id), "2 báo cáo chưa đủ")
	require.NoError(t, r.report(rs[2], thread.KindThread, id))
	require.True(t, r.hidden(id))
	require.Equal(t, 1, r.count(`select count(*) from forum_threads where id=$1 and hidden_by is null and hidden_reason='Nhiều báo cáo, chờ duyệt'`, id))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='thread.hidden'`))
	// bình luận: cùng quy tắc
	pid := r.comment(r.sv, r.mustCreate(r.ta, "Khác", clean))
	for _, a := range rs {
		require.NoError(t, r.report(a, thread.KindPost, pid))
	}
	require.Equal(t, 1, r.count(`select count(*) from forum_posts where id=$1 and hidden_at is not null`, pid))
}

func TestAutoHideNotBySameUser(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	for range 3 {
		_ = r.report(r.other, thread.KindThread, id) // lần 2, 3 → ALREADY_REPORTED
	}
	require.Equal(t, 1, r.count(`select count(*) from post_reports where target_id=$1`, id))
	require.False(t, r.hidden(id), "ba báo cáo từ MỘT người không đủ")
}

func TestAutoHideCountsOpenOnly(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	rs := r.reporters(3)
	require.NoError(t, r.report(rs[0], thread.KindThread, id))
	require.NoError(t, r.report(rs[1], thread.KindThread, id))
	_, err := r.svc.DismissReports(t.Context(), r.teacher, r.course, thread.KindThread, id, 1)
	require.NoError(t, err)
	require.NoError(t, r.report(rs[2], thread.KindThread, id))
	require.False(t, r.hidden(id), "hai báo cáo đã DISMISSED không còn tính: chỉ còn 1 OPEN")
}

func TestAutoHiddenVisibleToAuthorAndStaffOnly(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	for _, a := range r.reporters(3) {
		require.NoError(t, r.report(a, thread.KindThread, id))
	}
	_, err := r.svc.Get(t.Context(), r.other, r.course, id, pg(30))
	require.Equal(t, 404, status(err), "người khác: chi tiết 404")
	own, err := r.svc.Get(t.Context(), r.sv, r.course, id, pg(30))
	require.NoError(t, err)
	require.NotNil(t, own.Thread.Hidden)
	require.True(t, own.Thread.Hidden.Auto)
	st, err := r.svc.Get(t.Context(), r.ta, r.course, id, pg(30))
	require.NoError(t, err)
	require.NotNil(t, st.Thread.Hidden)
	list := func(a thread.Actor) []uuid.UUID {
		p, err := r.svc.List(t.Context(), a, r.course, thread.Filter{}, pg(30))
		require.NoError(t, err)
		var ids []uuid.UUID
		for _, it := range p.Items {
			ids = append(ids, it.ID)
		}
		return ids
	}
	require.NotContains(t, list(r.other), id)
	require.Contains(t, list(r.sv), id)
	require.Contains(t, list(r.ta), id)
	require.Equal(t, 404, status(func() error { _, e := r.svc.SimilarFor(t.Context(), r.other, r.course, id); return e }()))
}

// ===== AC3: hàng chờ và xử lý =====

func TestModerationQueue(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	a, b := r.mustCreate(r.sv, "A", clean), r.mustCreate(r.sv, "B", clean)
	require.NoError(t, r.svc.Report(t.Context(), r.other, r.course, thread.KindThread, a, thread.ReportIn{Reason: "SPAM"}))
	require.NoError(t, r.svc.Report(t.Context(), r.ta, r.course, thread.KindThread, a, thread.ReportIn{Reason: "OFFENSIVE"}))
	require.NoError(t, r.report(r.other, thread.KindThread, b))
	pg1, err := r.svc.Queue(t.Context(), r.course, "OPEN", pg(1))
	require.NoError(t, err)
	require.Len(t, pg1.Items, 1)
	require.NotNil(t, pg1.NextCursor)
	all, err := r.svc.Queue(t.Context(), r.course, "OPEN", pg(10))
	require.NoError(t, err)
	require.Len(t, all.Items, 2)
	for _, it := range all.Items {
		if it.ThreadID == a {
			require.Equal(t, 2, it.Reports.Total)
			require.Equal(t, map[string]int{"SPAM": 1, "OFFENSIVE": 1}, it.Reports.ByReason)
		}
	}
	_, err = r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindThread, a, "Lạc đề", 1)
	require.NoError(t, err)
	hid, err := r.svc.Queue(t.Context(), r.course, "HIDDEN", pg(10))
	require.NoError(t, err)
	require.Len(t, hid.Items, 1)
	require.False(t, hid.Items[0].HiddenByAuto)
	require.Equal(t, "VALIDATION_FAILED", code(func() error { _, e := r.svc.Queue(t.Context(), r.course, "X", pg(10)); return e }()))
}

func TestHideRequiresReason(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "A", clean)
	_, err := r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindThread, id, "  ", 1)
	require.Equal(t, "VALIDATION_FAILED", code(err))
	long := ""
	for range 201 {
		long += "x"
	}
	_, err = r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindThread, id, long, 1)
	require.Equal(t, "VALIDATION_FAILED", code(err))
	require.NoError(t, r.report(r.other, thread.KindThread, id))
	res, err := r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindThread, id, "Lạc đề", 1)
	require.NoError(t, err)
	require.True(t, res.Hidden)
	require.Equal(t, 1, r.count(`select count(*) from post_reports where target_id=$1 and status='ACTIONED'`, id))
	require.Equal(t, 1, r.count(`select count(*) from forum_threads where id=$1 and hidden_by=$2`, id, r.teacher.UserID))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity='forum_thread' and entity_id=$1 and action='thread.hide'`, id.String()))
	require.Zero(t, r.count(`select count(*) from audit_log where entity_id=$1 and (before::text like '%Câu hỏi%' or after::text like '%Điều 5%')`, id.String()), "audit không chứa nội dung bài")
}

func TestUnhide(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "A", clean)
	res, err := r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindThread, id, "Lạc đề", 1)
	require.NoError(t, err)
	_, err = r.svc.Unhide(t.Context(), r.teacher, r.course, thread.KindThread, id, 1)
	require.Equal(t, "VERSION_CONFLICT", code(err))
	un, err := r.svc.Unhide(t.Context(), r.teacher, r.course, thread.KindThread, id, res.Version)
	require.NoError(t, err)
	require.False(t, un.Hidden)
	require.False(t, r.hidden(id))
	_, err = r.svc.Unhide(t.Context(), r.teacher, r.course, thread.KindThread, id, un.Version)
	require.Equal(t, "POST_STATE_CONFLICT", code(err), "bài không ẩn")
}

func TestDismissReportsUnhides(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "A", clean)
	for _, a := range r.reporters(3) {
		require.NoError(t, r.report(a, thread.KindThread, id))
	}
	require.True(t, r.hidden(id))
	v := r.count(`select version from forum_threads where id=$1`, id)
	res, err := r.svc.DismissReports(t.Context(), r.teacher, r.course, thread.KindThread, id, int32(v))
	require.NoError(t, err)
	require.False(t, res.Hidden, "bài tự ẩn hiện lại")
	require.Equal(t, 3, r.count(`select count(*) from post_reports where target_id=$1 and status='DISMISSED'`, id))
	// bài ẩn thủ công vẫn ẩn sau "Giữ bài"
	id2 := r.mustCreate(r.sv, "B", clean)
	h, err := r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindThread, id2, "Vi phạm", 1)
	require.NoError(t, err)
	res, err = r.svc.DismissReports(t.Context(), r.teacher, r.course, thread.KindThread, id2, h.Version)
	require.NoError(t, err)
	require.True(t, res.Hidden)
}

func TestModerationRBAC(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	id := h.mustCreate(h.sv, "Hỏi", clean)
	admin := h.user("Quản trị", "ADMIN", "")
	base := "/courses/" + h.course.String()
	for _, tc := range []struct {
		name string
		id   uuid.UUID
		role auth.Role
		want int
	}{ // thứ tự cố định: giảng viên ẩn trước (200), trợ giảng đến sau với version cũ (409)
		{"giảng viên", h.teacher.UserID, auth.RoleTeacher, 200}, {"trợ giảng", h.ta.UserID, auth.RoleTA, 409}, {"sinh viên", h.other.UserID, auth.RoleStudent, 403},
		{"admin", admin, auth.RoleAdmin, 403}, {"không JWT", uuid.Nil, "", 401},
	} {
		res := h.do(tc.id, tc.role, http.MethodPost, base+"/threads/"+id.String()+"/hide", `{"reason":"x","version":1}`)
		require.Equal(t, tc.want, res.StatusCode, "hide "+tc.name)
		res = h.do(tc.id, tc.role, http.MethodGet, base+"/moderation/queue", "")
		want := tc.want
		if want == 409 {
			want = 200
		}
		require.Equal(t, want, res.StatusCode, "queue "+tc.name)
	}
}

func TestModerationVersionConflict(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "A", clean)
	_, err := r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindThread, id, "Lạc đề", 1)
	require.NoError(t, err)
	_, err = r.svc.Hide(t.Context(), r.ta, r.course, thread.KindThread, id, "Lạc đề", 1)
	require.Equal(t, "VERSION_CONFLICT", code(err), "người sau nhận 409")
}

// ===== AC4: chủ bài được báo =====

func TestAuthorNotifiedOnHide(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "A", clean)
	_, err := r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindThread, id, "Lạc đề", 1)
	require.NoError(t, err)
	m := r.lastMsg(thread.TopicHidden)
	for range 2 { // giao lại: một dòng
		require.NoError(t, r.notifier().HandleHidden(t.Context(), m))
	}
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='POST_HIDDEN' and title='Bài của bạn đã bị ẩn: Lạc đề' and link=$2`, r.sv.UserID, "/threads/"+id.String()))
	// tự ẩn: lời khác
	id2 := r.mustCreate(r.sv, "B", clean)
	for _, a := range r.reporters(3) {
		require.NoError(t, r.report(a, thread.KindThread, id2))
	}
	require.NoError(t, r.notifier().HandleHidden(t.Context(), r.lastMsg(thread.TopicHidden)))
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='POST_HIDDEN' and title='Bài của bạn đang được giảng viên xem xét'`, r.sv.UserID))
	require.Zero(t, r.count(`select count(*) from mail_outbox`), "không gửi mail cho việc này")
}

func TestAuthorSeesReasonNotReporters(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "A", clean)
	for _, a := range r.reporters(3) {
		require.NoError(t, r.svc.Report(t.Context(), a, r.course, thread.KindThread, id, thread.ReportIn{Reason: "OFFENSIVE"}))
	}
	v, err := r.svc.Get(t.Context(), r.sv, r.course, id, pg(30))
	require.NoError(t, err)
	raw, _ := json.Marshal(v)
	require.Contains(t, string(raw), "Nhiều báo cáo, chờ duyệt")
	for _, k := range []string{"OFFENSIVE", "reporter", r.other.UserID.String()} {
		require.NotContains(t, string(raw), k)
	}
}

func TestAuthorNotifiedOnUnhide(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "A", clean)
	h, err := r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindThread, id, "Lạc đề", 1)
	require.NoError(t, err)
	_, err = r.svc.Unhide(t.Context(), r.teacher, r.course, thread.KindThread, id, h.Version)
	require.NoError(t, err)
	require.NoError(t, r.notifier().HandleUnhidden(t.Context(), r.lastMsg(thread.TopicUnhidden)))
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='POST_UNHIDDEN' and title='Bài của bạn đã hiện lại'`, r.sv.UserID))
}

// ===== AC5: chủ bài sửa =====

func (r *rig) edit(a thread.Actor, id uuid.UUID, title, body string, v int32, redact bool) (thread.ThreadView, error) {
	return r.svc.EditThread(r.t.Context(), a, r.course, id, thread.EditThreadIn{Title: title, Body: body, Version: v, Redact: redact})
}

func TestEditRunsFirewall(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	_, err := r.edit(r.sv, id, "Hỏi", "MSSV 20229001 của em hỏi Điều 5", 1, false)
	require.Equal(t, "PII_DETECTED", code(err))
	v, err := r.edit(r.sv, id, "Hỏi lại", "Điều 6 quy chế nói gì?", 1, false)
	require.NoError(t, err)
	require.True(t, v.Thread.Edited)
	require.Equal(t, int32(2), v.Thread.Version)
	// "Ẩn thông tin rồi lưu"
	v, err = r.edit(r.sv, id, "Hỏi lại", "Em tên Ngô Ngọc Cẩm hỏi Điều 7 quy chế", 2, true)
	require.NoError(t, err)
	require.NotContains(t, v.Thread.Body, "Ngô Ngọc Cẩm")
}

func TestRawPIINeverStoredOnEdit(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	_, err := r.edit(r.sv, id, "Hỏi", "MSSV 20229002 và email cam.nn@sv.edu.vn hỏi Điều 5", 1, false)
	require.Error(t, err)
	var n int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from forum_threads t where t::text like '%20229002%' or t::text like '%cam.nn@sv.edu.vn%'`).Scan(&n))
	require.Zero(t, n)
	require.Zero(t, r.count(`select count(*) from audit_log t where t::text like '%20229002%'`))
}

func TestEditLockedWhenVerified(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, r.aiPost(id), thread.Verify, "", 0)
	require.NoError(t, err)
	_, err = r.edit(r.sv, id, "Mới", "Nội dung mới", 1, false)
	require.Equal(t, "EDIT_LOCKED_VERIFIED", code(err))
}

func TestEditOnlyByOwner(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	for _, a := range []thread.Actor{r.other, r.ta, r.teacher} {
		_, err := r.edit(a, id, "Mới", "Nội dung mới", 1, false)
		require.Equal(t, 403, status(err))
	}
	pid := r.comment(r.other, id)
	_, err := r.svc.EditPost(t.Context(), r.sv, r.course, pid, thread.EditPostIn{Body: "Sửa hộ", Version: 1})
	require.Equal(t, 403, status(err))
	got, err := r.svc.EditPost(t.Context(), r.other, r.course, pid, thread.EditPostIn{Body: "Bình luận đã sửa", Version: 1})
	require.NoError(t, err)
	require.True(t, got.Edited)
	_, err = r.svc.EditPost(t.Context(), r.other, r.course, pid, thread.EditPostIn{Body: "Lại sửa", Version: 1})
	require.Equal(t, "VERSION_CONFLICT", code(err))
}

func TestEditHiddenKeepsHidden(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	h, err := r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindThread, id, "Lạc đề", 1)
	require.NoError(t, err)
	v, err := r.edit(r.sv, id, "Hỏi lại", "Điều 6 quy chế nói gì?", h.Version, false)
	require.NoError(t, err)
	require.NotNil(t, v.Thread.Hidden)
	require.True(t, r.hidden(id), "sửa không tự bỏ ẩn")
}

func TestEditWhileAIPendingRejectsOldAnswer(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	old := r.aiPost(id)
	before := r.count(`select count(*) from notifications where user_id=$1`, r.sv.UserID)
	_, err := r.edit(r.sv, id, "Hỏi lại", "Điều 6 quy chế nói gì?", 1, false)
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from forum_posts where id=$1 and verification_state='REJECTED'`, old))
	require.Equal(t, 1, r.count(`select count(*) from forum_threads where id=$1 and ai_state='PENDING'`, id))
	require.Equal(t, before, r.count(`select count(*) from notifications where user_id=$1`, r.sv.UserID), "không báo chủ bài")
	require.Equal(t, 2, r.count(`select count(*) from jobs where kind='thread.answer'`), "việc AI được xếp lại")
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='thread.edited'`))
}

// ===== AC6: chủ bài xoá =====

func TestDeleteSoft(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	pid := r.comment(r.other, id)
	require.NoError(t, r.report(r.ta, thread.KindPost, pid)) // báo cáo OPEN: Staff vẫn thấy trong hàng chờ
	require.NoError(t, r.svc.DeletePost(t.Context(), r.other, r.course, pid))
	require.Equal(t, 1, r.count(`select count(*) from forum_posts where id=$1 and deleted_at is not null`, pid))
	q, err := r.svc.Queue(t.Context(), r.course, "OPEN", pg(10))
	require.NoError(t, err)
	require.Len(t, q.Items, 1)
	require.NoError(t, r.svc.DeleteThread(t.Context(), r.sv, r.course, id))
	require.Equal(t, 1, r.count(`select count(*) from forum_threads where id=$1 and deleted_at is not null`, id))
	for _, a := range []thread.Actor{r.sv, r.teacher} {
		_, err := r.svc.Get(t.Context(), a, r.course, id, pg(30))
		require.Equal(t, 404, status(err), "kể cả giảng viên")
	}
	require.Equal(t, 404, status(r.svc.DeleteThread(t.Context(), r.sv, r.course, id)))
}

func TestDeleteOnlyByOwner(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	for _, a := range []thread.Actor{r.other, r.ta, r.teacher} {
		require.Equal(t, 403, status(r.svc.DeleteThread(t.Context(), a, r.course, id)))
	}
	pid := r.comment(r.other, id)
	require.Equal(t, 403, status(r.svc.DeletePost(t.Context(), r.sv, r.course, pid)))
}

func TestDeleteRemovesKnowledge(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, r.aiPost(id), thread.Verify, "", 0)
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from knowledge_items where source_type='POST'`))
	require.NoError(t, r.svc.DeleteThread(t.Context(), r.sv, r.course, id))
	require.Zero(t, r.count(`select count(*) from knowledge_items where source_type='POST'`))
	require.Zero(t, r.count(`select count(*) from content_chunks c join documents d on d.id=c.document_id where d.category='THREAD_KNOWLEDGE'`))
}

// ===== AC7: ghim =====

func TestPinLimit(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	var ids []uuid.UUID
	for i := range 4 {
		ids = append(ids, r.mustCreate(r.sv, fmt.Sprintf("Hỏi %d", i), clean))
	}
	for _, id := range ids[:3] {
		require.NoError(t, r.svc.Pin(t.Context(), r.teacher, r.course, id, true))
	}
	require.Equal(t, "PIN_LIMIT", code(r.svc.Pin(t.Context(), r.teacher, r.course, ids[3], true)))
	require.NoError(t, r.svc.Pin(t.Context(), r.teacher, r.course, ids[0], true), "ghim lại thread đã ghim không tính thêm")
	require.NoError(t, r.svc.Pin(t.Context(), r.ta, r.course, ids[0], false))
	require.NoError(t, r.svc.Pin(t.Context(), r.ta, r.course, ids[3], true))
}

func TestPinnedSeparateFromItems(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	var ids []uuid.UUID
	for i := range 4 {
		ids = append(ids, r.mustCreate(r.sv, fmt.Sprintf("Hỏi %d", i), clean))
	}
	require.NoError(t, r.svc.Pin(t.Context(), r.teacher, r.course, ids[0], true))
	p1, err := r.svc.List(t.Context(), r.other, r.course, thread.Filter{}, pg(2))
	require.NoError(t, err)
	require.Len(t, p1.Pinned, 1)
	require.Equal(t, ids[0], p1.Pinned[0].ID)
	require.True(t, p1.Pinned[0].Pinned)
	for _, it := range p1.Items {
		require.NotEqual(t, ids[0], it.ID, "không lặp ghim trong items")
	}
	require.NotNil(t, p1.NextCursor)
	cur, err := httpx.ParseCursor(*p1.NextCursor)
	require.NoError(t, err)
	p2, err := r.svc.List(t.Context(), r.other, r.course, thread.Filter{}, httpx.PageParams{Limit: 2, Cursor: &cur})
	require.NoError(t, err)
	require.Empty(t, p2.Pinned, "ghim chỉ ở trang đầu")
}

func TestPinRBAC(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	id := h.mustCreate(h.sv, "Hỏi", clean)
	p := "/courses/" + h.course.String() + "/threads/" + id.String() + "/pin"
	require.Equal(t, 403, h.do(h.other.UserID, auth.RoleStudent, http.MethodPost, p, "").StatusCode)
	require.Equal(t, 403, h.do(h.user("Quản trị", "ADMIN", ""), auth.RoleAdmin, http.MethodPost, p, "").StatusCode)
	require.Equal(t, 204, h.do(h.ta.UserID, auth.RoleTA, http.MethodPost, p, "").StatusCode)
	require.Equal(t, 204, h.do(h.teacher.UserID, auth.RoleTeacher, http.MethodPost, p[:len(p)-3]+"unpin", "").StatusCode)
}

func TestFiltersStillWork(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	pin := r.mustCreate(r.sv, "Ghim về tuần", clean)
	r.mustCreate(r.sv, "Khác", clean)
	require.NoError(t, r.svc.Pin(t.Context(), r.teacher, r.course, pin, true))
	p, err := r.svc.List(t.Context(), r.other, r.course, thread.Filter{Q: "tuần"}, pg(10))
	require.NoError(t, err)
	require.Len(t, p.Pinned, 1, "bộ lọc áp lên cả ghim")
	require.Empty(t, p.Items)
	p, err = r.svc.List(t.Context(), r.other, r.course, thread.Filter{Q: "không-khớp"}, pg(10))
	require.NoError(t, err)
	require.Empty(t, p.Pinned)
}

// ===== AC8: chủ thread được báo =====

func TestThreadReplyNotifiesOwnerOnce(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	r.comment(r.other, id)
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='THREAD_REPLY'`, r.sv.UserID))
}

func TestVerifiedNotifiesOwner(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, r.aiPost(id), thread.Verify, "", 0)
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='THREAD_VERIFIED'`, r.sv.UserID))
}

func TestRejectDoesNotNotifyOwner(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, r.aiPost(id), thread.Reject, "", 0)
	require.NoError(t, err)
	require.Zero(t, r.count(`select count(*) from notifications where user_id=$1 and type='THREAD_VERIFIED'`, r.sv.UserID))
}

func TestOwnCommentNoSelfNotify(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	r.comment(r.sv, id)
	require.Zero(t, r.count(`select count(*) from notifications where user_id=$1 and type='THREAD_REPLY'`, r.sv.UserID))
}

// ===== AC9: AI_PENDING gộp =====

func TestAIPendingCoalesced(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	n := r.notifier()
	r.answered()
	r.answered()
	require.NoError(t, n.HandleAIPending(t.Context(), r.lastMsg(thread.TopicAIAnswerd)))
	for _, u := range []uuid.UUID{r.teacher.UserID, r.ta.UserID} {
		require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='AI_PENDING' and title='2 câu trả lời AI chờ duyệt' and read_at is null`, u))
	}
	require.Zero(t, r.count(`select count(*) from notifications where user_id=$1 and type='AI_PENDING'`, r.sv.UserID))
	require.NoError(t, n.HandleAIPending(t.Context(), r.lastMsg(thread.TopicAIAnswerd))) // giao lại: không thêm dòng
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='AI_PENDING'`, r.teacher.UserID))
}

func TestAIPendingZeroMarksRead(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	n := r.notifier()
	id := r.answered()
	require.NoError(t, n.HandleAIPending(t.Context(), r.lastMsg(thread.TopicAIAnswerd)))
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, r.aiPost(id), thread.Verify, "", 0)
	require.NoError(t, err)
	require.NoError(t, n.HandleAIPending(t.Context(), r.lastMsg(thread.TopicPostDecided)))
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='AI_PENDING' and read_at is not null`, r.teacher.UserID))
}

// ===== AC11 / AC12: tri thức =====

func (r *rig) syncKnowledge() {
	r.t.Helper()
	s := &knowledge.Syncer{Pool: r.pool, Embed: func(context.Context, string) ([]float32, error) { return make([]float32, 1536), nil }}
	rows, err := r.pool.Query(r.t.Context(), `select id, payload from outbox where topic = 'knowledge.sync' order by created_at`)
	require.NoError(r.t, err)
	var msgs []outbox.Message
	for rows.Next() {
		m := outbox.Message{Topic: knowledge.TopicSync}
		require.NoError(r.t, rows.Scan(&m.ID, &m.Payload))
		msgs = append(msgs, m)
	}
	rows.Close()
	for _, m := range msgs {
		require.NoError(r.t, s.Handle(r.t.Context(), m))
	}
}

func TestVerifyLoadsKnowledge(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, r.aiPost(id), thread.Verify, "", 0)
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from knowledge_items where source_type='POST'`))
	require.Equal(t, 1, r.count(`select count(*) from content_chunks c join knowledge_items k on k.chunk_id=c.id where c.embedding is null`), "chưa nhúng: không lọt RAG")
	r.syncKnowledge()
	require.Equal(t, 1, r.count(`select count(*) from content_chunks c join knowledge_items k on k.chunk_id=c.id where c.embedding is not null and c.text like '%Hỏi%' or c.text like '%Câu hỏi:%'`))
}

func TestCorrectReplacesKnowledge(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	pid := r.aiPost(id)
	v, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Verify, "", 0)
	require.NoError(t, err)
	_, err = r.svc.Decide(t.Context(), r.teacher, r.course, pid, thread.Correct, "Bản đã sửa duy nhất", v.Version)
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from knowledge_items where source_type='POST'`), "thay đoạn cũ, không thêm dòng")
	require.Equal(t, 1, r.count(`select count(*) from content_chunks c join knowledge_items k on k.chunk_id=c.id where c.text like '%Bản đã sửa duy nhất%'`))
}

func TestKnowledgeEmbedFailureKeepsVerify(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	pid := r.aiPost(id)
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Verify, "", 0)
	require.NoError(t, err)
	s := &knowledge.Syncer{Pool: r.pool, Embed: func(context.Context, string) ([]float32, error) { return nil, fmt.Errorf("nhà cung cấp chết") }}
	require.Error(t, s.Handle(t.Context(), r.lastMsg(knowledge.TopicSync)))
	require.Equal(t, 1, r.count(`select count(*) from forum_posts where id=$1 and verification_state='VERIFIED'`, pid), "xác nhận vẫn giữ")
	require.Equal(t, 1, r.count(`select count(*) from knowledge_items where source_type='POST'`), "mục tri thức còn, chờ nhúng lại")
}

func TestEmbedDownStillCountsAIPending(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	n := r.notifier()
	id := r.answered()
	require.NoError(t, n.HandleAIPending(t.Context(), r.lastMsg(thread.TopicAIAnswerd)))
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, r.aiPost(id), thread.Verify, "", 0)
	require.NoError(t, err)
	// knowledge.sync (topic riêng) lỗi nhúng KHÔNG nằm trong chuỗi `thread.post_decided`: đếm lại vẫn chạy trọn
	s := &knowledge.Syncer{Pool: r.pool, Embed: func(context.Context, string) ([]float32, error) { return nil, fmt.Errorf("chết") }}
	require.Error(t, s.Handle(t.Context(), r.lastMsg(knowledge.TopicSync)))
	require.NoError(t, n.HandleAIPending(t.Context(), r.lastMsg(thread.TopicPostDecided)))
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='AI_PENDING' and read_at is not null`, r.teacher.UserID))
}

func TestKnowledgeIsolatedByCourse(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, r.aiPost(id), thread.Verify, "", 0)
	require.NoError(t, err)
	r.syncKnowledge()
	other := r.newCourse(r.teacher.UserID)
	require.Zero(t, r.count(`select count(*) from content_chunks c where $1 = any(c.course_ids) and c.text like '%Câu hỏi:%'`, other))
	require.Equal(t, 1, r.count(`select count(*) from content_chunks c where $1 = any(c.course_ids) and c.text like '%Câu hỏi:%'`, r.course))
}

func TestRejectedNotRetrieved(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	pid := r.aiPost(id)
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Verify, "", 0)
	require.NoError(t, err)
	_, err = r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Reject, "", 2)
	require.NoError(t, err)
	require.Zero(t, r.count(`select count(*) from knowledge_items where source_type='POST'`), "gỡ khỏi kho tri thức cùng giao dịch")
	require.GreaterOrEqual(t, r.count(`select count(*) from outbox where topic='document.changed'`), 1, "vô hiệu cache RAG")
}

func TestHiddenPostNotRetrieved(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	pid := r.aiPost(id)
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Verify, "", 0)
	require.NoError(t, err)
	h, err := r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindPost, pid, "Sai", int32(r.count(`select version from forum_posts where id=$1`, pid)))
	require.NoError(t, err)
	require.Zero(t, r.count(`select count(*) from knowledge_items where source_type='POST'`))
	_, err = r.svc.Unhide(t.Context(), r.teacher, r.course, thread.KindPost, pid, h.Version)
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from knowledge_items where source_type='POST'`), "bỏ ẩn nạp lại")
}

func TestRejectBumpsRagVersion(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	pid := r.aiPost(id)
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Verify, "", 0)
	require.NoError(t, err)
	before := r.count(`select count(*) from outbox where topic='document.changed'`)
	_, err = r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Reject, "", 2)
	require.NoError(t, err)
	require.Greater(t, r.count(`select count(*) from outbox where topic='document.changed'`), before)
}

// ===== AC13 / AC15 / AC16 =====

func TestModerationRBACMatrix(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	id := h.mustCreate(h.sv, "Hỏi", clean)
	outsider := h.user("Người lạ", "STUDENT", "")
	base := "/courses/" + h.course.String()
	rep := func(uid uuid.UUID, role auth.Role) int {
		return h.do(uid, role, http.MethodPost, base+"/threads/"+id.String()+"/report", `{"reason":"SPAM"}`).StatusCode
	}
	require.Equal(t, 201, rep(h.other.UserID, auth.RoleStudent))
	require.Equal(t, 201, rep(h.ta.UserID, auth.RoleTA))
	require.Equal(t, 201, rep(h.teacher.UserID, auth.RoleTeacher))
	require.Equal(t, 403, rep(h.user("Quản trị", "ADMIN", ""), auth.RoleAdmin))
	require.Equal(t, 403, rep(outsider, auth.RoleStudent), "ngoài lớp")
	for _, route := range []string{"/hide", "/unhide", "/dismiss-reports", "/pin"} {
		require.Equal(t, 403, h.do(h.other.UserID, auth.RoleStudent, http.MethodPost, base+"/threads/"+id.String()+route, `{"reason":"x","version":1}`).StatusCode, route)
	}
	res := h.do(h.sv.UserID, auth.RoleStudent, http.MethodPut, base+"/threads/"+id.String(), `{"title":"Sửa","body":"Nội dung mới","version":2}`)
	require.Equal(t, 200, res.StatusCode)
	res = h.do(h.teacher.UserID, auth.RoleTeacher, http.MethodPut, base+"/threads/"+id.String(), `{"title":"Sửa","body":"Nội dung mới","version":3}`)
	require.Equal(t, 403, res.StatusCode, "chỉ chủ bài")
}

func TestAutoHideRaceOnce(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	rs := r.reporters(4)
	require.NoError(t, r.report(rs[0], thread.KindThread, id))
	require.NoError(t, r.report(rs[1], thread.KindThread, id))
	var wg sync.WaitGroup
	for _, a := range rs[2:] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// báo cáo thứ 3 tự ẩn; người đến sau thấy bài đã ẩn nên nhận 404 (đúng 4.10.4)
			if err := r.report(a, thread.KindThread, id); err != nil {
				require.Equal(t, 404, status(err))
			}
		}()
	}
	wg.Wait()
	require.True(t, r.hidden(id))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='thread.hidden'`), "đúng một lần tự ẩn")
	m := r.lastMsg(thread.TopicHidden)
	for range 2 {
		require.NoError(t, r.notifier().HandleHidden(t.Context(), m))
	}
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='POST_HIDDEN'`, r.sv.UserID), "đúng một thông báo cho chủ bài")
}

func TestReportRetryAfterOffline(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	id := h.mustCreate(h.sv, "Hỏi", clean)
	p := "/courses/" + h.course.String() + "/threads/" + id.String() + "/report"
	require.Equal(t, 201, h.do(h.other.UserID, auth.RoleStudent, http.MethodPost, p, `{"reason":"SPAM"}`).StatusCode)
	res := h.do(h.other.UserID, auth.RoleStudent, http.MethodPost, p, `{"reason":"SPAM"}`) // gửi lại sau khi mất mạng
	require.Equal(t, 409, res.StatusCode)
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	require.Equal(t, "ALREADY_REPORTED", body.Code)
	require.Equal(t, 1, h.count(`select count(*) from post_reports`))
}

func TestHideDeletedPost(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	pid := r.comment(r.other, id)
	require.NoError(t, r.svc.DeletePost(t.Context(), r.other, r.course, pid))
	_, err := r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindPost, pid, "Lạc đề", 2)
	require.Equal(t, 404, status(err))
}

func TestHiddenThreadNotInSimilar(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	a, b := r.mustCreate(r.sv, "Thi lại", clean), r.mustCreate(r.other, "Thi lại nữa", clean)
	v := make([]float32, 1536)
	v[0] = 1
	for _, id := range []uuid.UUID{a, b} {
		_, err := r.pool.Exec(t.Context(), `update forum_threads set embedding = $2::vector where id = $1`, id, vecLit(v))
		require.NoError(t, err)
	}
	got, err := r.svc.Similar(t.Context(), r.course, a)
	require.NoError(t, err)
	require.Len(t, got, 1)
	_, err = r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindThread, b, "Lạc đề", 1)
	require.NoError(t, err)
	got, err = r.svc.Similar(t.Context(), r.course, a)
	require.NoError(t, err)
	require.Empty(t, got, "thread ẩn không lọt vào 'tương tự'")
}

func vecLit(v []float32) string {
	s := "["
	for i, x := range v {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprint(x)
	}
	return s + "]"
}

var _ = time.Second
