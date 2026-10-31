package today_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/today"
)

func dueItems(res resp) []map[string]any {
	var out []map[string]any
	if rec, ok := res.json()["recommended"].(map[string]any); ok && rec["kind"] == "ASSIGNMENT_DUE" {
		out = append(out, rec)
	}
	for _, a := range actionsOf(res) {
		if m := a.(map[string]any); m["kind"] == "ASSIGNMENT_DUE" {
			out = append(out, m)
		}
	}
	return out
}

func (r *rig) asg(c cls, code string, due time.Duration, published bool, extra string) string {
	r.t.Helper()
	pub := "null"
	if published {
		pub = "now()"
	}
	return r.scalar(`insert into assignments (course_id, title, due_at, submission_code, created_by, published_at`+extraCols(extra)+`) select $1, $2, $3, $4, u.id, `+pub+extraVals(extra)+` from users u where u.role = 'ADMIN' limit 1 returning id`,
		c.id, "Bài "+code, r.clk.Now().Add(due), code)
}

func extraCols(e string) string {
	if e == "late" {
		return ", allow_late, late_max_days"
	}
	return ""
}

func extraVals(e string) string {
	if e == "late" {
		return ", true, 2"
	}
	return ""
}

// TestTodayAssignmentDue — ASSIGNMENT_DUE (bậc 41): bài đã công bố, chưa nộp, hạn ≤ 24 giờ hoặc đang nộp muộn; nháp / hạn xa / đã nộp không có mục; chỉ sinh viên thấy;
// một mục mỗi lớp (bài gần hạn nhất); nộp → outbox xoá cache → mục biến mất.
func TestTodayAssignmentDue(t *testing.T) {
	r := newRig(t)
	c := r.klass("764001")
	su, sv := r.student("")
	r.enrollAt(c, su, "STUDENT", "ACTIVE", "", time.Hour)
	gv, gvSess := r.teaching(c)
	_ = gv
	r.asg(c, "NHAP", 5*time.Hour, false, "") // chưa công bố
	r.asg(c, "XA", 72*time.Hour, true, "")   // hạn xa
	r.asg(c, "QUA", -3*time.Hour, true, "")  // quá hạn, không cho nộp muộn
	require.Empty(t, dueItems(r.tget(sv, "/me/today")))

	near := r.asg(c, "GAN", 18*time.Hour, true, "")
	r.asg(c, "GAN2", 20*time.Hour, true, "") // cùng lớp: chỉ một mục
	items := dueItems(r.tgetC(sv, "/me/today"))
	require.Empty(t, items, "cache còn bản cũ tới khi có sự kiện")
	inv := today.Invalidator{Pool: r.pool, Redis: r.rdb}
	payload, _ := json.Marshal(map[string]string{"course_id": c.sid()})
	require.NoError(t, inv.Handle(t.Context(), outbox.Message{Topic: today.TopicAssignmentPublished, Payload: payload}))
	items = dueItems(r.tgetC(sv, "/me/today"))
	require.Len(t, items, 1)
	require.Equal(t, "/assignments/"+near, items[0]["href"])
	require.Equal(t, "Bài GAN", items[0]["title"])
	require.Equal(t, "Còn 18 giờ — Nộp bài", items[0]["reason"])
	require.EqualValues(t, today.TierAssignmentDue, 41)
	require.Empty(t, dueItems(r.tget(gvSess, "/me/today")), "nhân viên không có mục này")

	// đang nộp muộn: quá hạn nhưng còn cửa muộn
	late := r.asg(c, "MUON", -3*time.Hour, true, "late")
	require.NoError(t, inv.Handle(t.Context(), outbox.Message{Topic: today.TopicAssignmentPublished, Payload: payload}))
	items = dueItems(r.tgetC(sv, "/me/today"))
	require.Len(t, items, 1, "một mục mỗi lớp: bài có hạn gần nhất (đang muộn)")
	require.Equal(t, "/assignments/"+late, items[0]["href"])
	require.Equal(t, "Đang nộp muộn — Nộp bài", items[0]["reason"])

	// nộp bài → hết mục (xoá cache theo sự kiện submission.created)
	for _, id := range []string{late, near} {
		r.exec(`insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method)
			values ($1, $2, $3, 'IN_APP', 1, 'k', 'a.txt', 1, 'text/plain', 'x', 'MATCHED', 'IN_APP')`, c.id, id, su.ID)
	}
	sp, _ := json.Marshal(map[string]string{"course_id": c.sid(), "user_id": su.ID.String()})
	require.NoError(t, inv.Handle(t.Context(), outbox.Message{Topic: today.TopicSubmissionCreated, Payload: sp}))
	items = dueItems(r.tgetC(sv, "/me/today"))
	require.Len(t, items, 1, "còn bài GAN2 chưa nộp")
	require.Equal(t, "Bài GAN2", items[0]["title"])
}

func unmatchedItems(res resp) []map[string]any {
	var out []map[string]any
	if rec, ok := res.json()["recommended"].(map[string]any); ok && rec["kind"] == "UNMATCHED_SUBMISSION" {
		out = append(out, rec)
	}
	for _, a := range actionsOf(res) {
		if m := a.(map[string]any); m["kind"] == "UNMATCHED_SUBMISSION" {
			out = append(out, m)
		}
	}
	return out
}

func (r *rig) unmatched(c cls, n int) {
	r.t.Helper()
	for i := range n {
		r.exec(`insert into submissions (course_id, source, blob_key, file_name, file_size, file_mime, sha256, match_status, unmatched_reason, external_ref, external_ts)
			values ($1, 'UPLOAD', 'k', 'a.txt', 1, 'text/plain', 'x', 'UNMATCHED', 'NO_STUDENT_SIGNAL', $2, '1970-01-01Z')`, c.id, fmt.Sprintf("ref-%s-%d", c.code, i))
	}
}

// TestProviderUnmatchedSubmission — UNMATCHED_SUBMISSION (bậc 60): "{n} bài nộp chưa khớp sinh viên · lớp {mã}", một mục mỗi lớp; TEACHER và TA thấy, STUDENT và ADMIN không; không lẫn lớp khác.
func TestProviderUnmatchedSubmission(t *testing.T) {
	r := newRig(t)
	c, other := r.klass("764101"), r.klass("764102")
	_, gvSess := r.teaching(c)
	ta, taSess := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", 24*time.Hour)
	su, svSess := r.student("")
	r.enrollAt(c, su, "STUDENT", "ACTIVE", "", time.Hour)
	_, adSess := r.person(store.UserRoleADMIN, true)
	r.unmatched(c, 3)
	r.unmatched(other, 5)
	for name, s := range map[string]session{"giảng viên": gvSess, "trợ giảng": taSess} {
		items := unmatchedItems(r.tget(s, "/me/today"))
		require.Len(t, items, 1, name)
		require.Equal(t, "3 bài nộp chưa khớp sinh viên · lớp 764101", items[0]["title"], name)
		require.Equal(t, "/grading?tab=unmatched&course="+c.sid(), items[0]["href"], name)
	}
	for name, s := range map[string]session{"sinh viên": svSess, "admin": adSess} {
		require.Empty(t, unmatchedItems(r.tget(s, "/me/today")), name)
	}
	require.EqualValues(t, today.TierUnmatchedSubmission, 60)
}

// TestProviderUnmatchedCleared — gán / bỏ qua hết → outbox `submission.resolved` xoá cache → mục biến mất.
func TestProviderUnmatchedCleared(t *testing.T) {
	r := newRig(t)
	c := r.klass("764103")
	_, gvSess := r.teaching(c)
	r.unmatched(c, 2)
	require.Len(t, unmatchedItems(r.tget(gvSess, "/me/today")), 1)
	r.exec(`update submissions set dismissed_at = now() where course_id = $1`, c.id)
	require.Len(t, unmatchedItems(r.tgetC(gvSess, "/me/today")), 1, "cache còn bản cũ tới khi có sự kiện")
	payload, _ := json.Marshal(map[string]string{"course_id": c.sid()})
	inv := today.Invalidator{Pool: r.pool, Redis: r.rdb}
	require.NoError(t, inv.Handle(t.Context(), outbox.Message{Topic: today.TopicSubmissionResolved, Payload: payload}))
	require.Empty(t, unmatchedItems(r.tgetC(gvSess, "/me/today")))
}

func kindItems(res resp, kind string) []map[string]any {
	var out []map[string]any
	if rec, ok := res.json()["recommended"].(map[string]any); ok && rec["kind"] == kind {
		out = append(out, rec)
	}
	for _, a := range actionsOf(res) {
		if m := a.(map[string]any); m["kind"] == kind {
			out = append(out, m)
		}
	}
	return out
}

// gres chèn một bài nộp MATCHED (phiên bản v của sinh viên mới) + kết quả hiện hành.
func (r *rig) gres(c cls, asg string, v int, runState, status string, flags string) {
	r.t.Helper()
	su, _ := r.student("")
	r.enrollAt(c, su, "STUDENT", "ACTIVE", "", time.Hour)
	r.insertRes(c, asg, su.ID.String(), v, runState, status, flags)
}

func (r *rig) insertRes(c cls, asg, stu string, v int, runState, status, flags string) {
	r.t.Helper()
	sub := r.scalar(`insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method)
		values ($1, $2, $3, 'IN_APP', $4, 'k', 'a.txt', 1, 'text/plain', 'x', 'MATCHED', 'IN_APP') returning id`, c.id, asg, stu, v)
	r.exec(`insert into grading_results (course_id, submission_id, run_state, status, flags, total, published_at) values ($1, $2, $3, $4, $5::text[], case when $3 = 'DONE' then 5 end, case when $4 = 'PUBLISHED' then now() end)`, c.id, sub, runState, status, flags)
}

// TestProviderGradingError — GRADING_ERROR (bậc 28): "{n} bài chấm lỗi · {bài}", một mục mỗi lớp; bản cũ bị thay không tính; TEACHER + TA thấy, STUDENT / ADMIN không; hết việc (retry) + sự kiện → mục biến mất.
func TestProviderGradingError(t *testing.T) {
	r := newRig(t)
	c := r.klass("764201")
	_, gvSess := r.teaching(c)
	su, svSess := r.student("")
	r.enrollAt(c, su, "STUDENT", "ACTIVE", "", time.Hour)
	_, adSess := r.person(store.UserRoleADMIN, true)
	a := r.asg(c, "BT03", 72*time.Hour, true, "")
	r.gres(c, a, 1, "FAILED", "DRAFT", "{}")
	r.gres(c, a, 1, "FAILED", "DRAFT", "{}")
	old, _ := r.student("")
	r.enrollAt(c, old, "STUDENT", "ACTIVE", "", time.Hour)
	r.insertRes(c, a, old.ID.String(), 1, "FAILED", "DRAFT", "{}") // bản 1 bị thay bởi bản 2
	r.scalar(`insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method)
		values ($1, $2, $3, 'IN_APP', 2, 'k', 'a.txt', 1, 'text/plain', 'x', 'MATCHED', 'IN_APP') returning id`, c.id, a, old.ID.String())
	items := kindItems(r.tget(gvSess, "/me/today"), "GRADING_ERROR")
	require.Len(t, items, 1)
	require.Equal(t, "2 bài chấm lỗi · Bài BT03", items[0]["title"])
	require.Equal(t, "/grading?status=FAILED", items[0]["href"])
	require.EqualValues(t, today.TierGradingError, 28)
	for name, s := range map[string]session{"sinh viên": svSess, "admin": adSess} {
		require.Empty(t, kindItems(r.tget(s, "/me/today"), "GRADING_ERROR"), name)
	}
	r.exec(`update grading_results set run_state = 'QUEUED' where course_id = $1`, c.id)
	payload, _ := json.Marshal(map[string]string{"course_id": c.sid()})
	require.NoError(t, today.Invalidator{Pool: r.pool, Redis: r.rdb}.Handle(t.Context(), outbox.Message{Topic: today.TopicSubmissionResolved, Payload: payload}))
	require.Empty(t, kindItems(r.tgetC(gvSess, "/me/today"), "GRADING_ERROR"))
}

// TestProviderGradingReview — GRADING_REVIEW (bậc 30): "{n} bài cần xem kỹ · {bài}" chỉ cho DRAFT xong có cờ; đã duyệt thì biến mất.
func TestProviderGradingReview(t *testing.T) {
	r := newRig(t)
	c := r.klass("764202")
	_, gvSess := r.teaching(c)
	a := r.asg(c, "BT03", 72*time.Hour, true, "")
	r.gres(c, a, 1, "DONE", "DRAFT", "{NEEDS_REVIEW}")
	r.gres(c, a, 1, "DONE", "DRAFT", "{INJECTION_SUSPECTED,OFF_TOPIC}")
	r.gres(c, a, 1, "DONE", "DRAFT", "{}")                // không cờ
	r.gres(c, a, 1, "DONE", "REVIEWED", "{NEEDS_REVIEW}") // đã duyệt
	items := kindItems(r.tget(gvSess, "/me/today"), "GRADING_REVIEW")
	require.Len(t, items, 1)
	require.Equal(t, "2 bài cần xem kỹ · Bài BT03", items[0]["title"])
	require.Equal(t, "/grading", items[0]["href"])
	require.EqualValues(t, today.TierGradingReview, 30)
	r.exec(`update grading_results set status = 'REVIEWED' where course_id = $1 and needs_review`, c.id)
	payload, _ := json.Marshal(map[string]string{"course_id": c.sid()})
	require.NoError(t, today.Invalidator{Pool: r.pool, Redis: r.rdb}.Handle(t.Context(), outbox.Message{Topic: today.TopicSubmissionResolved, Payload: payload}))
	require.Empty(t, kindItems(r.tgetC(gvSess, "/me/today"), "GRADING_REVIEW"))
}

func (r *rig) queries(s session, path string) int64 {
	r.t.Helper()
	before := r.qc.n.Load()
	require.Equal(r.t, http.StatusOK, r.tget(s, path).code)
	return r.qc.n.Load() - before
}

// TestProviderStaffAggregatedOneQuery — SRS 4.12: các kind của nhân viên (UNMATCHED_SUBMISSION, GRADING_ERROR, GRADING_REVIEW) do MỘT truy vấn tổng hợp cho mọi lớp: số truy vấn của giảng viên có đủ dữ liệu ở 4 lớp bằng của giảng viên không có dữ liệu.
func TestProviderStaffAggregatedOneQuery(t *testing.T) {
	r := newRig(t)
	var with, without []cls
	for i := range 4 {
		with = append(with, r.klass(fmt.Sprintf("76430%d", i)))
		without = append(without, r.klass(fmt.Sprintf("76431%d", i)))
	}
	_, gData := r.teaching(with...)
	_, gNone := r.teaching(without...)
	for _, c := range with {
		a := r.asg(c, "BT"+c.code, 72*time.Hour, true, "")
		r.gres(c, a, 1, "FAILED", "DRAFT", "{}")
		r.gres(c, a, 1, "DONE", "DRAFT", "{NEEDS_REVIEW}")
		r.unmatched(c, 2)
	}
	res := r.tget(gData, "/me/today")
	require.Equal(t, http.StatusOK, res.code)
	for _, k := range []string{"UNMATCHED_SUBMISSION", "GRADING_ERROR", "GRADING_REVIEW"} {
		require.Len(t, kindItems(res, k), 4, k)
	}
	data := r.queries(gData, "/me/today")
	none := r.queries(gNone, "/me/today")
	require.LessOrEqual(t, data-none, int64(1), "kind của nhân viên thêm tối đa MỘT truy vấn, không tăng theo số lớp / số kind")
}

// TestProviderStudentAggregatedOneQuery — ba kind của sinh viên cùng một truy vấn khác: sinh viên của 4 lớp có bài tập đến hạn không tốn thêm truy vấn theo số lớp.
func TestProviderStudentAggregatedOneQuery(t *testing.T) {
	r := newRig(t)
	su, sSess := r.student("")
	nu, nSess := r.student("")
	for i := range 4 {
		c := r.klass(fmt.Sprintf("76440%d", i))
		r.enrollAt(c, su, "STUDENT", "ACTIVE", "", time.Hour)
		r.asg(c, "BT"+c.code, 12*time.Hour, true, "")
		d := r.klass(fmt.Sprintf("76441%d", i))
		r.enrollAt(d, nu, "STUDENT", "ACTIVE", "", time.Hour)
	}
	require.Len(t, dueItems(r.tget(sSess, "/me/today")), 1, "một mục ASSIGNMENT_DUE cho cả 4 lớp (bài gần hạn nhất)")
	data := r.queries(sSess, "/me/today")
	none := r.queries(nSess, "/me/today")
	require.LessOrEqual(t, data-none, int64(1), "kind của sinh viên thêm tối đa MỘT truy vấn, không tăng theo số lớp")
}

// TestProviderPublishPendingTier33 — PUBLISH_PENDING (bậc 33): "{n} bài đã duyệt chờ công bố · {bài}" cho TEACHER; công bố hết + sự kiện → biến mất.
func TestProviderPublishPendingTier33(t *testing.T) {
	r := newRig(t)
	c := r.klass("764301")
	_, gvSess := r.teaching(c)
	a := r.asg(c, "BT03", 72*time.Hour, true, "")
	r.gres(c, a, 1, "DONE", "REVIEWED", "{}")
	r.gres(c, a, 1, "DONE", "REVIEWED", "{NEEDS_REVIEW}")
	r.gres(c, a, 1, "DONE", "DRAFT", "{}") // chưa duyệt: không tính
	items := kindItems(r.tget(gvSess, "/me/today"), "PUBLISH_PENDING")
	require.Len(t, items, 1)
	require.Equal(t, "2 bài đã duyệt chờ công bố · Bài BT03", items[0]["title"])
	require.Equal(t, "/grading", items[0]["href"])
	require.EqualValues(t, today.TierPublishPending, 33)
	r.exec(`update grading_results set status = 'PUBLISHED', published_at = now() where course_id = $1 and status = 'REVIEWED'`, c.id)
	payload, _ := json.Marshal(map[string]string{"course_id": c.sid()})
	require.NoError(t, today.Invalidator{Pool: r.pool, Redis: r.rdb}.Handle(t.Context(), outbox.Message{Topic: today.TopicSubmissionResolved, Payload: payload}))
	require.Empty(t, kindItems(r.tgetC(gvSess, "/me/today"), "PUBLISH_PENDING"))
}

// TestPublishPendingNotSameTierAsExamHold — bậc 33 khác bậc 32 của EXAM_PUBLISH_HOLD cùng vai TEACHER và không trùng bậc nào khác của Giảng viên.
func TestPublishPendingNotSameTierAsExamHold(t *testing.T) {
	require.NotEqual(t, today.TierExamPublishHold, today.TierPublishPending)
	for _, other := range []int{today.TierEmailMismatch, today.TierExamSimilarity, today.TierExamAppeal, today.TierExamGradeError, today.TierGradingError, today.TierGradingReview, today.TierUnmatchedSubmission,
		today.TierJoinRequest, today.TierTicket, today.TierAIConfirm, today.TierGradeSchemeUnconfirmed, today.TierCourseSetup, today.TierQuestionReview, today.TierPostReported} {
		require.NotEqual(t, other, today.TierPublishPending)
	}
}

// TestProviderPublishPendingTeacherOnly — TA, sinh viên và ADMIN không có PUBLISH_PENDING (công bố chỉ TEACHER).
func TestProviderPublishPendingTeacherOnly(t *testing.T) {
	r := newRig(t)
	c := r.klass("764302")
	_, gvSess := r.teaching(c)
	ta, taSess := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", 24*time.Hour)
	su, svSess := r.student("")
	r.enrollAt(c, su, "STUDENT", "ACTIVE", "", time.Hour)
	_, adSess := r.person(store.UserRoleADMIN, true)
	a := r.asg(c, "BT03", 72*time.Hour, true, "")
	r.gres(c, a, 1, "DONE", "REVIEWED", "{}")
	require.Len(t, kindItems(r.tget(gvSess, "/me/today"), "PUBLISH_PENDING"), 1)
	for name, s := range map[string]session{"trợ giảng": taSess, "sinh viên": svSess, "admin": adSess} {
		require.Empty(t, kindItems(r.tget(s, "/me/today"), "PUBLISH_PENDING"), name)
	}
}

// TestProviderAssignmentGradedPublishedOnly — ASSIGNMENT_GRADED (bậc 43): chỉ kết quả ĐÃ PUBLISHED của CHÍNH mình trong 7 ngày; nháp / duyệt chưa công bố / của người khác / quá 7 ngày không có.
func TestProviderAssignmentGradedPublishedOnly(t *testing.T) {
	r := newRig(t)
	c := r.klass("764303")
	su, sv := r.student("")
	r.enrollAt(c, su, "STUDENT", "ACTIVE", "", time.Hour)
	other, otherSess := r.student("")
	r.enrollAt(c, other, "STUDENT", "ACTIVE", "", time.Hour)
	a := r.asg(c, "BT03", 72*time.Hour, true, "")
	r.insertRes(c, a, su.ID.String(), 1, "DONE", "DRAFT", "{}")
	require.Empty(t, kindItems(r.tget(sv, "/me/today"), "ASSIGNMENT_GRADED"), "nháp không bao giờ hiện")
	r.exec(`update grading_results set status = 'REVIEWED' where course_id = $1`, c.id)
	inv := today.Invalidator{Pool: r.pool, Redis: r.rdb}
	payload, _ := json.Marshal(map[string]string{"course_id": c.sid(), "user_id": su.ID.String()})
	require.NoError(t, inv.Handle(t.Context(), outbox.Message{Topic: today.TopicSubmissionCreated, Payload: payload}))
	require.Empty(t, kindItems(r.tgetC(sv, "/me/today"), "ASSIGNMENT_GRADED"), "đã duyệt chưa công bố: không hiện")
	r.exec(`update grading_results set status = 'PUBLISHED', published_at = now() where course_id = $1`, c.id)
	require.NoError(t, inv.Handle(t.Context(), outbox.Message{Topic: today.TopicSubmissionCreated, Payload: payload}))
	items := kindItems(r.tgetC(sv, "/me/today"), "ASSIGNMENT_GRADED")
	require.Len(t, items, 1)
	require.Equal(t, "/assignments/"+a, items[0]["href"])
	require.Equal(t, "Điểm Bài BT03 đã có — Xem", items[0]["reason"])
	require.EqualValues(t, today.TierAssignmentGraded, 43)
	require.Empty(t, kindItems(r.tget(otherSess, "/me/today"), "ASSIGNMENT_GRADED"), "kết quả của người khác")
	r.exec(`update grading_results set published_at = now() - interval '8 days' where course_id = $1`, c.id)
	require.NoError(t, inv.Handle(t.Context(), outbox.Message{Topic: today.TopicSubmissionCreated, Payload: payload}))
	require.Empty(t, kindItems(r.tgetC(sv, "/me/today"), "ASSIGNMENT_GRADED"), "quá 7 ngày")
}
