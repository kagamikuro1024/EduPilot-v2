package today_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

// similarityPair dựng một bài code đã đóng có MỘT cặp bản nộp nghi giống nhau (gắn cờ, NEW) và trả id bài thi + id cặp.
func (r *rig) similarityPair(c cls, by uuid.UUID, title string, flagged bool) (uuid.UUID, uuid.UUID) {
	r.t.Helper()
	ctx := r.t.Context()
	eid := r.examAt(c, by, title, "CLOSED", -5*time.Hour, -time.Hour, 45)
	var qid, item uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(ctx, `insert into question_bank (course_id, type, title, topic, stem, created_by) values ($1, 'CODE', 'a+b', 't', 's', $2) returning id`, c.id, by).Scan(&qid))
	_, err := r.pool.Exec(ctx, `insert into code_problems (question_id, course_id) values ($1, $2)`, qid, c.id)
	require.NoError(r.t, err)
	require.NoError(r.t, r.pool.QueryRow(ctx, `insert into exam_items (course_id, exam_id, question_id, position) values ($1, $2, $3, 1) returning id`, c.id, eid, qid).Scan(&item))
	var att, sub [2]uuid.UUID
	for i := range 2 {
		var uid uuid.UUID
		require.NoError(r.t, r.pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, $2, 'STUDENT') returning id`, fmt.Sprintf("sim.%s@example.test", uuid.NewString()[:8]), fmt.Sprintf("SV %d", i)).Scan(&uid))
		require.NoError(r.t, r.pool.QueryRow(ctx, `insert into exam_attempts (course_id, exam_id, student_id, started_at, deadline_at, status, submitted_at, submit_reason) values ($1, $2, $3, now() - interval '3 hours', now() - interval '2 hours', 'GRADING', now() - interval '2 hours', 'MANUAL') returning id`, c.id, eid, uid).Scan(&att[i]))
		require.NoError(r.t, r.pool.QueryRow(ctx, `insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256) values ($1, $2, $3, $4, $5, $6, 'SUBMIT', 'cpp17', 'x', repeat('a', 64)) returning id`, c.id, eid, att[i], item, qid, uid).Scan(&sub[i]))
	}
	if att[0].String() > att[1].String() {
		att[0], att[1], sub[0], sub[1] = att[1], att[0], sub[1], sub[0]
	}
	var pair uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(ctx, `insert into similarity_reports (course_id, exam_id, problem_id, run_id, submission_a, submission_b, attempt_a, attempt_b, score, shared_fingerprints, flagged)
		values ($1, $2, $3, $4, $5, $6, $7, $8, 0.9, 30, $9) returning id`, c.id, eid, qid, uuid.New(), sub[0], sub[1], att[0], att[1], flagged).Scan(&pair))
	return eid, pair
}

func similarityItems(res resp) []map[string]any {
	var out []map[string]any
	for _, a := range actionsOf(res) {
		if m := a.(map[string]any); m["kind"] == "EXAM_SIMILARITY" {
			out = append(out, m)
		}
	}
	return out
}

// TestSimilarityTodayProvider — US-PE-07 AC9: việc `EXAM_SIMILARITY` (bậc 55, CHỈ Giảng viên): "{N} cặp bài code nghi giống nhau · {tên bài thi}"; cặp không gắn cờ không tính;
// TA và sinh viên không thấy; biến mất khi không còn cặp gắn cờ ở trạng thái NEW (đã xem).
func TestSimilarityTodayProvider(t *testing.T) {
	r := newRig(t)
	c := r.klass("761985")
	gvUser, gv := r.teaching(c)
	ta, taSess := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", 24*time.Hour)
	_, sv := r.student("B20DC000005")
	eid, pair := r.similarityPair(c, gvUser.ID, "Giữa kỳ", true)
	r.similarityPair(c, gvUser.ID, "Không cờ", false)
	items := similarityItems(r.tget(gv, "/me/today"))
	require.Len(t, items, 1)
	require.Equal(t, "1 cặp bài code nghi giống nhau · Giữa kỳ", items[0]["title"])
	require.Equal(t, fmt.Sprintf("/exams/%s/similarity?course=%s", eid, c.sid()), items[0]["href"])
	require.Empty(t, similarityItems(r.tget(taSess, "/me/today")), "TA không thấy")
	require.Empty(t, similarityItems(r.tget(sv, "/me/today")), "sinh viên không thấy")
	_, err := r.pool.Exec(t.Context(), `update similarity_reports set review_state='CLEARED', reviewed_by=(select created_by from exams where id=$2), reviewed_at=now() where id=$1`, pair, eid)
	require.NoError(t, err)
	require.Empty(t, similarityItems(r.tget(gv, "/me/today")), "đã xem hết → việc biến mất")
}
