package today_test

import (
	"encoding/json"
	"fmt"
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
