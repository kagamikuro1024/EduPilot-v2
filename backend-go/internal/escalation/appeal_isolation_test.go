package escalation_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/assignment"
	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/store"
)

// appeal gửi một phúc khảo bài tập thật (qua assignment.Service) của sinh viên A ở `course`; trả id phúc khảo.
func (r *rig) appeal(course uuid.UUID) uuid.UUID {
	r.t.Helper()
	var asg, sub uuid.UUID
	r.one(&asg, `insert into assignments (course_id, title, max_score, due_at, submission_code, created_by, published_at, rubric)
		values ($1, 'BT03', 10, now() + interval '2 days', 'BT03', $2, now(), '[{"id":"c1","name":"Khái niệm","max_score":"10","guidance":""}]'::jsonb) returning id`, course, r.teacher)
	r.one(&sub, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at)
		values ($1, $2, $3, 'IN_APP', 1, 'k', 'a.txt', 1, 'text/plain', 'x', 'MATCHED', 'IN_APP', now()) returning id`, course, asg, r.a.UserID)
	r.exec(`insert into grading_results (course_id, submission_id, run_state, status, criteria, total, ai_total_before_late, published_at)
		values ($1, $2, 'DONE', 'PUBLISHED', '[{"id":"c1","score":"7.00"}]'::jsonb, 7, 7, now())`, course, sub)
	svc := &assignment.Service{Pool: r.pool}
	out, err := svc.FileAppeal(r.t.Context(), assignment.Actor{UserID: r.a.UserID, Student: true}, course, asg, assignment.AppealIn{CriteriaIDs: []string{"c1"}, Reason: "Em nghĩ phần này đã đủ ý rồi ạ."})
	require.NoError(r.t, err)
	return out.Appeal.ID
}

func (r *rig) one(dst any, q string, args ...any) {
	r.t.Helper()
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), q, args...).Scan(dst), q)
}

// TestAppealNotInInboxOrMeTickets — phúc khảo không vào hàng chờ của nhân viên (mọi trạng thái).
func TestAppealNotInInboxOrMeTickets(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.appeal(r.course)
	for _, st := range []string{"OPEN", "ALL"} {
		page, err := r.svc.ListStaff(t.Context(), []uuid.UUID{r.course}, st, httpx.PageParams{Limit: 50})
		require.NoError(t, err)
		require.Empty(t, page.Items, st)
	}
}

// TestAppealNotCountedInTicketCap — sinh viên có 4 câu hỏi chờ + 1 phúc khảo vẫn nhờ được câu thứ 5; phúc khảo không tạo vé.
func TestAppealNotCountedInTicketCap(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.fill(r.course, 4)
	r.appeal(r.course)
	_, m := r.pair("Câu thứ năm?", "Trả lời")
	_, err := r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)
	require.Equal(t, 5, r.count(`select count(*) from escalation_tickets where student_id = $1`, r.a.UserID))
}

// TestAdminStalledIgnoresAppeal — phúc khảo chờ rất lâu không làm lớp thành "ticket quá hạn" (ADMIN không biết lớp có phúc khảo — D39).
func TestAdminStalledIgnoresAppeal(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.appeal(r.course)
	r.exec(`update grade_appeals set created_at = now() - interval '30 days' where course_id = $1`, r.course)
	rows, err := store.New(r.pool).TicketStalledCourses(t.Context())
	require.NoError(t, err)
	for _, x := range rows {
		require.NotEqual(t, r.course, x.CourseID)
	}
}

// TestAppealNotInKnowledge — phúc khảo (gửi + quyết) không sinh đoạn kiến thức / tài liệu nào của lớp.
func TestAppealNotInKnowledge(t *testing.T) {
	r := newRig(t, escalation.Config{})
	before := r.count(`select count(*) from content_chunks where $1 = any(course_ids)`, r.course)
	id := r.appeal(r.course)
	svc := &assignment.Service{Pool: r.pool}
	_, err := svc.DecideAppeal(t.Context(), assignment.Actor{UserID: r.teacher}, r.course, id, assignment.DecideIn{Decision: "UPHELD", Response: "Thầy giữ nguyên điểm.", Version: 1})
	require.NoError(t, err)
	require.Equal(t, before, r.count(`select count(*) from content_chunks where $1 = any(course_ids)`, r.course))
	require.Zero(t, r.count(`select count(*) from escalation_tickets where course_id = $1`, r.course))
}
