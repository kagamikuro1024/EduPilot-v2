package today_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/platform/outbox"
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
