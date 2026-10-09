package today_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

func kindsOf(res resp) map[string]map[string]any {
	out := map[string]map[string]any{}
	if rec, ok := res.json()["recommended"].(map[string]any); ok { // việc đầu bảng nằm ở `recommended`, phần còn lại ở `actions`
		out[rec["kind"].(string)] = rec
	}
	for _, a := range actionsOf(res) {
		m := a.(map[string]any)
		out[m["kind"].(string)] = m
	}
	return out
}

// TestExamResultTodayItems — US-PE-08 AC4/AC16/AC17: sau công bố sinh viên có EXAM_RESULT (bậc 44, 7 ngày) và, khi được trả lời phúc khảo, EXAM_APPEAL_REPLY (bậc 46);
// Staff có EXAM_GRADE_ERROR (bậc 15, cả TA), còn EXAM_APPEAL (22) và EXAM_PUBLISH_HOLD (32) CHỈ Giảng viên.
func TestExamResultTodayItems(t *testing.T) {
	r := newRig(t)
	c := r.klass("761986")
	gvUser, gv := r.teaching(c)
	ta, taSess := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", 24*time.Hour)
	svUser, sv := r.student("B20DC000006")
	r.enrollAt(c, svUser, "STUDENT", "ACTIVE", "", 24*time.Hour)
	ctx := t.Context()

	pub := r.examAt(c, gvUser.ID, "Giữa kỳ", "PUBLISHED", -5*time.Hour, -time.Hour, 45)
	var att uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `insert into exam_attempts (course_id, exam_id, student_id, started_at, deadline_at, status, submitted_at, submit_reason, auto_score, graded_at)
		values ($1, $2, $3, now() - interval '3 hours', now() - interval '2 hours', 'GRADED', now() - interval '2 hours', 'MANUAL', 7, now()) returning id`, c.id, pub, svUser.ID).Scan(&att))
	_, err := r.pool.Exec(ctx, `update exams set published_at = $2 where id=$1`, pub, r.clk.Now().Add(-time.Hour))
	require.NoError(t, err)
	_, err = r.pool.Exec(ctx, `insert into exam_appeals (course_id, exam_id, attempt_id, student_id, reason) values ($1, $2, $3, $4, 'xin xem lại')`, c.id, pub, att, svUser.ID)
	require.NoError(t, err)

	k := kindsOf(r.tget(sv, "/me/today"))
	require.Contains(t, k, "EXAM_RESULT")
	require.Equal(t, "Điểm bài thi Giữa kỳ đã công bố", k["EXAM_RESULT"]["title"])
	require.NotContains(t, k, "EXAM_APPEAL_REPLY", "phúc khảo còn OPEN")
	g := kindsOf(r.tget(gv, "/me/today"))
	require.Contains(t, g, "EXAM_APPEAL")
	require.Equal(t, "1 yêu cầu xem lại điểm · Giữa kỳ", g["EXAM_APPEAL"]["title"])
	require.NotContains(t, kindsOf(r.tget(taSess, "/me/today")), "EXAM_APPEAL", "TA không có việc phúc khảo")

	// đã trả lời → sinh viên có EXAM_APPEAL_REPLY; việc của Giảng viên biến mất
	_, err = r.pool.Exec(ctx, `update exam_appeals set status='UPHELD', response='giữ nguyên', responded_by=$2, responded_at=$3 where attempt_id=$1`, att, gvUser.ID, r.clk.Now().Add(-time.Minute))
	require.NoError(t, err)
	d2 := r.tget(sv, "/me/today")
	require.Contains(t, kindsOf(d2), "EXAM_RESULT", "sinh viên chỉ thấy MỘT việc đầu bảng: kết quả (44) đứng trước phản hồi (46)")

	// quá 7 ngày kể từ công bố: EXAM_RESULT hết hạn, còn phản hồi phúc khảo mới nhận vẫn hiện
	_, err = r.pool.Exec(ctx, `update exams set published_at = $2 where id=$1`, pub, r.clk.Now().Add(-8*24*time.Hour))
	require.NoError(t, err)
	k = kindsOf(r.tget(sv, "/me/today"))
	require.NotContains(t, k, "EXAM_RESULT")
	require.Contains(t, k, "EXAM_APPEAL_REPLY")

	// Staff: bài CLOSED đang hoãn công bố + một bản nộp lỗi
	held := r.examAt(c, gvUser.ID, "Cuối kỳ", "CLOSED", -5*time.Hour, -time.Hour, 45)
	_, err = r.pool.Exec(ctx, `update exams set publish_hold = true where id=$1`, held)
	require.NoError(t, err)
	var qid, item, sub uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `insert into question_bank (course_id, type, title, topic, stem, created_by) values ($1, 'CODE', 'c', 't', 's', $2) returning id`, c.id, gvUser.ID).Scan(&qid))
	_, err = r.pool.Exec(ctx, `insert into code_problems (question_id, course_id) values ($1, $2)`, qid, c.id)
	require.NoError(t, err)
	require.NoError(t, r.pool.QueryRow(ctx, `insert into exam_items (course_id, exam_id, question_id, position) values ($1, $2, $3, 1) returning id`, c.id, held, qid).Scan(&item))
	var att2 uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `insert into exam_attempts (course_id, exam_id, student_id, started_at, deadline_at, status, submitted_at, submit_reason) values ($1, $2, $3, now() - interval '3 hours', now() - interval '2 hours', 'GRADING', now() - interval '2 hours', 'MANUAL') returning id`, c.id, held, svUser.ID).Scan(&att2))
	require.NoError(t, r.pool.QueryRow(ctx, `insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256, status, verdict, judged_at)
		values ($1, $2, $3, $4, $5, $6, 'SUBMIT', 'cpp17', 'x', repeat('a', 64), 'ERROR', 'IE', now()) returning id`, c.id, held, att2, item, qid, svUser.ID).Scan(&sub))
	g = kindsOf(r.tget(gv, "/me/today"))
	require.Contains(t, g, "EXAM_GRADE_ERROR")
	require.Equal(t, "1 bài code chấm lỗi hệ thống · Cuối kỳ", g["EXAM_GRADE_ERROR"]["title"])
	require.NotContains(t, g, "EXAM_PUBLISH_HOLD", "còn lượt chưa chấm: chưa phải 'đã chấm xong'")
	tk := kindsOf(r.tget(taSess, "/me/today"))
	require.Contains(t, tk, "EXAM_GRADE_ERROR", "TA cũng thấy lỗi chấm")
	_, err = r.pool.Exec(ctx, `update code_submissions set status='DONE', verdict='AC' where id=$1`, sub)
	require.NoError(t, err)
	_, err = r.pool.Exec(ctx, `update exam_attempts set status='GRADED', auto_score=10, graded_at=now() where id=$1`, att2)
	require.NoError(t, err)
	g = kindsOf(r.tget(gv, "/me/today"))
	require.Contains(t, g, "EXAM_PUBLISH_HOLD")
	require.NotContains(t, kindsOf(r.tget(taSess, "/me/today")), "EXAM_PUBLISH_HOLD", "TA không thấy")
}
