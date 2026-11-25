package exam_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
)

// qc3 (PE01-36, AC12): lớp ARCHIVED — lượt đang chạy vẫn nộp được; bắt đầu mới / lưu → 409; đọc vẫn được.
func TestQC3ArchivedSubmitStillWorks(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, sv2, tab := r.student("ACTIVE"), r.student("ACTIVE"), uuid.New()
	e := r.openExam("lưu trữ nộp", true, r.mcqSet(3)...)
	v, _ := r.start(e, sv, tab)
	_, err := r.save(v, sv, tab, answerCorrect(v.Items[0]))
	require.NoError(t, err)
	r.exec(`update courses set status='ARCHIVED', archived_at=now(), join_enabled=false where id=$1`, r.course)
	// bắt đầu lượt mới → 409 COURSE_ARCHIVED
	_, _, err = r.svc.StartAttempt(t.Context(), sv2, r.course, e.ID, uuid.New())
	st, code := apiStatus(t, err)
	t.Logf("start sau lưu trữ: %d %s", st, code)
	require.Equal(t, 409, st)
	require.Equal(t, "COURSE_ARCHIVED", code)
	// đọc: danh sách bài thi + kết quả staff vẫn được
	_, err = r.svc.ListExams(t.Context(), r.course, "", nil, 30)
	require.NoError(t, err)
	// lượt đang chạy vẫn nộp được
	sub, err := r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
	require.NoError(t, err)
	t.Logf("nộp sau lưu trữ: status=%s answered=%d/%d", sub.Status, sub.Answered, sub.Total)
	require.Equal(t, "GRADED", sub.Status)
	// sau khi nộp, lưu thêm → 409 (đã nộp)
	_, err = r.save(v, sv, tab, pick(v.Items[1], 0))
	st, code = apiStatus(t, err)
	t.Logf("lưu sau nộp: %d %s", st, code)
	require.Equal(t, 409, st)
	_ = exam.QuestionIn{}
}
