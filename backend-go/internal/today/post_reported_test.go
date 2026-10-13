package today_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/today"
)

// ---- POST_REPORTED (US-P4-06 AC10, bậc 47) ---------------------------------------------------------------------------------

func postReported(res resp) []map[string]any {
	var out []map[string]any
	for _, a := range actionsOf(res) {
		if m := a.(map[string]any); m["kind"] == "POST_REPORTED" {
			out = append(out, m)
		}
	}
	return out
}

func (r *rig) report(c cls, thread thr, reporter store.User) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), `insert into post_reports (course_id, thread_id, reporter_id, reason) values ($1, $2, $3, 'SPAM')`, c.id, thread.id, reporter.ID)
	require.NoError(r.t, err)
}

func TestPostReportedProvider(t *testing.T) {
	r := newRig(t)
	c := r.klass("761981")
	_, gv := r.teaching(c)
	sv, _ := r.enrolledStudent(c)
	rep, _ := r.enrolledStudent(c)
	a := r.thread(c, sv.ID, "Bị báo cáo", "SKIPPED", "", 5*time.Hour)
	b := r.thread(c, sv.ID, "Tự ẩn", "SKIPPED", "", time.Hour)
	r.report(c, a, rep)
	_, err := r.pool.Exec(t.Context(), `update forum_threads set hidden_at = now() - interval '2 hours', hidden_reason = 'Nhiều báo cáo, chờ duyệt' where id = $1`, b.id)
	require.NoError(t, err)
	items := postReported(r.tget(gv, "/me/today"))
	require.Len(t, items, 1)
	require.Equal(t, "2 bài bị báo cáo chờ xem · lớp 761981", items[0]["title"])
	require.Equal(t, "/threads?view=moderation&course="+c.sid(), items[0]["href"])
	require.Equal(t, 47, today.TierPostReported)
}

func TestPostReportedStaffOnly(t *testing.T) {
	r := newRig(t)
	c1, c2 := r.klass("761982"), r.klass("761983")
	_, gv := r.teaching(c1)
	ta, taSess := r.person(store.UserRoleTA, true)
	r.enrollAt(c1, ta, "TA", "ACTIVE", "", 24*time.Hour)
	_, other := r.teaching(c2)
	sv, svSess := r.enrolledStudent(c1)
	rep, _ := r.enrolledStudent(c1)
	r.report(c1, r.thread(c1, sv.ID, "Bị báo cáo", "SKIPPED", "", time.Hour), rep)
	for name, s := range map[string]session{"Giảng viên": gv, "TA": taSess} {
		require.Len(t, postReported(r.tget(s, "/me/today")), 1, name)
	}
	require.Empty(t, postReported(r.tget(other, "/me/today")), "Staff lớp khác không thấy")
	require.NotContains(t, string(r.tget(svSess, "/me/today").body), "POST_REPORTED")
	_, admSess := r.person(store.UserRoleADMIN, true)
	require.NotContains(t, string(r.tget(admSess, "/me/today").body), "POST_REPORTED")
}

func TestPostReportedClears(t *testing.T) {
	r := newRig(t)
	c := r.klass("761984")
	_, gv := r.teaching(c)
	sv, _ := r.enrolledStudent(c)
	rep, _ := r.enrolledStudent(c)
	a := r.thread(c, sv.ID, "Bị báo cáo", "SKIPPED", "", time.Hour)
	r.report(c, a, rep)
	require.Len(t, postReported(r.tget(gv, "/me/today")), 1)
	_, err := r.pool.Exec(t.Context(), `update post_reports set status = 'DISMISSED', handled_by = $2, handled_at = now() where target_id = $1`, a.id, rep.ID)
	require.NoError(t, err)
	require.Empty(t, postReported(r.tget(gv, "/me/today")), "xử lý hết thì biến (≤ 60 s qua vô hiệu cache; DB là nguồn đúng)")
	_ = fmt.Sprint
}

// TestHiddenThreadNotInToday — AC16: thread ẩn không được đếm vào AI_CONFIRM.
func TestHiddenThreadNotInToday(t *testing.T) {
	r := newRig(t)
	c := r.klass("761985")
	_, gv := r.teaching(c)
	sv, _ := r.enrolledStudent(c)
	h := r.thread(c, sv.ID, "CANARY-ẨN", "ANSWERED", "PENDING", time.Hour)
	_, err := r.pool.Exec(t.Context(), `update forum_threads set hidden_at = now(), hidden_reason = 'x' where id = $1`, h.id)
	require.NoError(t, err)
	require.Empty(t, aiConfirm(r.tget(gv, "/me/today")))
	require.NotContains(t, string(r.tget(gv, "/me/today").body), "CANARY-ẨN")
}
