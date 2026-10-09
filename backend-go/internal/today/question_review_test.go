package today_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/store"
)

func (r *rig) pendingQuestions(c cls, n int, archived bool) {
	r.t.Helper()
	var by string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select created_by::text from courses where id=$1`, c.id).Scan(&by))
	for range n {
		_, err := r.pool.Exec(r.t.Context(), `insert into question_bank (course_id, type, title, topic, stem, answer_key, origin, review_status, created_by, archived_at)
			values ($1, 'TRUE_FALSE', 'câu chờ', 't', 's', '{"value":true}', 'AI_DRAFT', 'PENDING', $2, case when $3 then now() end)`, c.id, by, archived)
		require.NoError(r.t, err)
	}
}

func questionItems(res resp) []map[string]any {
	var out []map[string]any
	for _, a := range actionsOf(res) {
		if m := a.(map[string]any); m["kind"] == "QUESTION_REVIEW" {
			out = append(out, m)
		}
	}
	return out
}

// TestQuestionReviewProvider — US-PE-03 AC14: có câu PENDING → việc `QUESTION_REVIEW` ("{N} câu hỏi chờ duyệt · lớp {mã lớp}", liên kết lọc PENDING) cho Giảng viên và TA của lớp;
// câu đã lưu trữ không tính; sinh viên và Staff lớp khác không thấy; hết câu thì việc biến mất.
func TestQuestionReviewProvider(t *testing.T) {
	r := newRig(t)
	c1, c2 := r.klass("761981"), r.klass("761982")
	_, gv := r.teaching(c1)
	ta, taSess := r.person(store.UserRoleTA, true)
	r.enrollAt(c1, ta, "TA", "ACTIVE", "", 24*time.Hour)
	_, other := r.teaching(c2)
	_, sv := r.student("B20DC000001")
	r.pendingQuestions(c1, 3, false)
	r.pendingQuestions(c1, 2, true) // lưu trữ: không tính
	for name, s := range map[string]session{"Giảng viên": gv, "TA": taSess} {
		items := questionItems(r.tget(s, "/me/today"))
		require.Len(t, items, 1, name)
		require.Equal(t, "3 câu hỏi chờ duyệt · lớp 761981", items[0]["title"], name)
		require.Equal(t, "/questions?review_status=PENDING&course="+c1.sid(), items[0]["href"], name)
		require.Equal(t, c1.sid(), items[0]["course"].(map[string]any)["id"], name)
	}
	require.Empty(t, questionItems(r.tget(other, "/me/today")), "Staff lớp khác không thấy")
	res := r.tget(sv, "/me/today")
	require.Equal(t, http.StatusOK, res.code)
	require.Empty(t, questionItems(res), "sinh viên không bao giờ thấy")
	require.NotContains(t, string(res.body), "QUESTION_REVIEW")
	res = r.tget(gv, "/courses/"+c1.sid()+"/today")
	require.Len(t, questionItems(res), 1)
	_, err := r.pool.Exec(t.Context(), `update question_bank set review_status='APPROVED', reviewed_by=created_by, reviewed_at=now() where course_id=$1`, c1.id)
	require.NoError(t, err)
	require.Empty(t, questionItems(r.tget(gv, "/me/today")))
}

// TestQuestionReviewInvalidates — AC14: duyệt hết câu thì việc biến khỏi "Hôm nay" ≤ 2 s nhờ sự kiện `question.reviewed`, không đợi TTL 60 s.
func TestQuestionReviewInvalidates(t *testing.T) {
	r := newRig(t)
	r.worker()
	c := r.klass("761983")
	g, gv := r.teaching(c)
	r.pendingQuestions(c, 1, false)
	require.Len(t, questionItems(r.tgetC(gv, "/me/today")), 1)
	var qid, ver string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select id::text, version::text from question_bank where course_id=$1`, c.id).Scan(&qid, &ver))
	svc := &exam.Service{Pool: r.pool}
	id := mustUUID(t, qid)
	_, err := svc.Review(t.Context(), g.ID, c.id, id, exam.DecisionApprove, 1)
	require.NoError(t, err)
	took := waitFor(t, 2*time.Second, "QUESTION_REVIEW biến khỏi Hôm nay", func() bool {
		return !strings.Contains(string(r.tgetC(gv, "/me/today").body), "QUESTION_REVIEW")
	})
	require.Less(t, took, 2*time.Second)
}

func mustUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}
