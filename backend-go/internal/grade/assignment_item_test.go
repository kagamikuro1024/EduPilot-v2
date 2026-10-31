package grade_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
)

// asgRig: lớp 3 sinh viên, item `weekly` đổi thành ASSIGNMENT (bài 10 điểm), luật MISSING_SCORE 0 cho `weekly`.
type asgRig struct {
	*finRig
	asg, scheme, item uuid.UUID
}

func newAsgRig(t *testing.T, due string, allowLate bool) *asgRig {
	t.Helper()
	return newAsgRigOn(t, newTDB(t), 3, due, allowLate)
}

func newAsgRigOn(t *testing.T, d *tdb, n int, due string, allowLate bool) *asgRig {
	t.Helper()
	r := &asgRig{finRig: newFinRigOn(t, d, n)}
	r.item = r.items["weekly"]
	r.one(&r.scheme, `select scheme_id from grade_items where id = $1`, r.item)
	r.exec(`delete from grade_entries where item_id = $1`, r.item)
	r.one(&r.asg, `insert into assignments (course_id, title, due_at, submission_code, created_by, published_at, allow_late, late_max_days) values ($1, 'BT03', now() + $2::interval, 'BT03', $3, now(), $4, case when $4 then 2 else 0 end) returning id`,
		r.course, due, r.gv, allowLate)
	r.exec(`update grade_items set source = 'ASSIGNMENT', assignment_id = $2, max_score = 10 where id = $1`, r.item, r.asg)
	r.exec(`insert into grade_rules (course_id, scheme_id, kind, applicable, params) values ($1, $2, 'MISSING_SCORE', true, '{"zero_components":["weekly"]}')`, r.course, r.scheme)
	return r
}

// sub chèn bài nộp MATCHED (version v) và, nếu status != "", kết quả chấm với tổng `total`.
func (r *asgRig) sub(student uuid.UUID, version int, status, total string) {
	r.t.Helper()
	var id uuid.UUID
	r.one(&id, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method)
		values ($1, $2, $3, 'IN_APP', $4, 'k', 'a.txt', 1, 'text/plain', 'x', 'MATCHED', 'IN_APP') returning id`, r.course, r.asg, student, version)
	if status != "" {
		pub := "null"
		if status == "PUBLISHED" {
			pub = "now()"
		}
		r.exec(`insert into grading_results (course_id, submission_id, run_state, status, total, published_at) values ($1, $2, 'DONE', $3, $4::numeric, `+pub+`)`, r.course, id, status, total)
	}
}

func (r *asgRig) unmatched(dismissed bool) {
	r.t.Helper()
	r.exec(`insert into submissions (course_id, assignment_id, source, blob_key, file_name, file_size, file_mime, sha256, match_status, unmatched_reason, dismissed_at)
		values ($1, $2, 'UPLOAD', 'k', 'a.txt', 1, 'text/plain', 'x', 'UNMATCHED', 'NO_STUDENT_SIGNAL', case when $3 then now() end)`, r.course, r.asg, dismissed)
}

func (r *asgRig) inputs() map[uuid.UUID]grade.Input {
	r.t.Helper()
	in, err := grade.LoadInputs(r.ctx, r.q, r.course, r.scheme, r.students)
	require.NoError(r.t, err)
	return in
}

// TestAssignmentItemReadsPublishedLatest — điểm item = total PUBLISHED của BẢN NỘP MỚI NHẤT; bản mới chưa công bố → không dùng điểm của bản cũ.
func TestAssignmentItemReadsPublishedLatest(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	r.sub(r.students[0], 1, "PUBLISHED", "8.5")
	r.sub(r.students[1], 1, "PUBLISHED", "9")
	r.sub(r.students[1], 2, "DRAFT", "3") // bản mới nhất chưa công bố
	in := r.inputs()
	require.True(t, in[r.students[0]].Scores[r.item].Equal(decimal.RequireFromString("8.5")))
	_, has := in[r.students[1]].Scores[r.item]
	require.False(t, has, "điểm bản 1 không còn hiện hành")
	require.Equal(t, grade.ReasonAssignmentUngrade, in[r.students[1]].ItemReason[r.item])
	_, has = in[r.students[2]].Scores[r.item]
	require.False(t, has)
}

// TestUnmatchedBlocksZeroScore — còn bài UNMATCHED của bài tập: sinh viên không có bài MATCHED là INCOMPLETE ("Còn bài chưa khớp"), KHÔNG phải 0; có bài MATCHED chưa công bố cũng INCOMPLETE.
func TestUnmatchedBlocksZeroScore(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	r.sub(r.students[0], 1, "PUBLISHED", "8")
	r.sub(r.students[1], 1, "DRAFT", "7")
	r.unmatched(false)
	in := r.inputs()
	require.Equal(t, grade.ReasonUnmatchedPending, in[r.students[2]].ItemReason[r.item])
	require.False(t, in[r.students[2]].ItemClosed[r.item], "không bao giờ 0 khi còn bài chưa khớp")
	require.Equal(t, grade.ReasonAssignmentUngrade, in[r.students[1]].ItemReason[r.item])
	_, err := r.finalize()
	require.Equal(t, 2, incompleteTotal(t, err))
}

// TestNoSubmissionZeroOnlyWhenNoUnmatched — không nộp + cửa đã đóng + không còn UNMATCHED chưa xử lý → "không nộp" (0 theo MISSING_SCORE); bản đã Bỏ qua không tính là chờ.
func TestNoSubmissionZeroOnlyWhenNoUnmatched(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	r.sub(r.students[0], 1, "PUBLISHED", "8")
	r.sub(r.students[1], 1, "PUBLISHED", "6")
	r.unmatched(true) // đã Bỏ qua
	in := r.inputs()
	require.True(t, in[r.students[2]].ItemClosed[r.item])
	_, hasReason := in[r.students[2]].ItemReason[r.item]
	require.False(t, hasReason)
	out, err := r.finalize()
	require.NoError(t, err, "đủ điểm: sinh viên không nộp được 0")
	require.Equal(t, 3, out.Students)
	var qt string
	r.one(&qt, `select (breakdown->>'qt') from final_grade_snapshots where course_id = $1 and student_id = $2`, r.course, r.students[2])
	require.Equal(t, "0", qt)
}

// TestFinalizeIncompleteWhenUnmatched — `Chốt điểm` báo GRADES_INCOMPLETE khi còn bài chưa khớp, rồi qua được sau khi gán / bỏ qua hết.
func TestFinalizeIncompleteWhenUnmatched(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	r.sub(r.students[0], 1, "PUBLISHED", "8")
	r.sub(r.students[1], 1, "PUBLISHED", "6")
	r.unmatched(false)
	_, err := r.finalize()
	require.Equal(t, 1, incompleteTotal(t, err))
	r.exec(`update submissions set dismissed_at = now() where course_id = $1 and match_status = 'UNMATCHED'`, r.course)
	_, err = r.finalize()
	require.NoError(t, err)
}

// TestAssignmentOpenIsIncomplete — cửa còn mở (chưa tới hạn, hoặc còn cửa nộp muộn) → INCOMPLETE, kể cả khi thành phần nằm trong zero_components.
func TestAssignmentOpenIsIncomplete(t *testing.T) {
	r := newAsgRig(t, "2 days", false)
	in := r.inputs()
	require.Equal(t, grade.ReasonAssignmentOpen, in[r.students[0]].ItemReason[r.item])
	late := newAsgRig(t, "-1 day", true) // quá hạn nhưng còn 2 ngày nộp muộn
	require.Equal(t, grade.ReasonAssignmentOpen, late.inputs()[late.students[0]].ItemReason[late.item])
}
