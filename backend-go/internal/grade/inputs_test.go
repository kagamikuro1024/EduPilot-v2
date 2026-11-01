package grade_test

import (
	"encoding/csv"
	"os"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
)

func atoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	require.NoError(t, err)
	return n
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// examRig: lớp có một thành phần `weekly` với một item EXAM nối bài thi `status` (thang bài thi examMax, thang item itemMax) và hai sinh viên.
type examRig struct {
	*tdb
	course, teacher, scheme, item, exam, s1, s2 uuid.UUID
}

func newExamRig(t *testing.T, status, examMax, itemMax string) examRig {
	t.Helper()
	d := newTDB(t)
	r := examRig{tdb: d}
	r.teacher = d.user("TEACHER", "GV")
	r.course = d.course(r.teacher, "L1")
	r.s1 = d.student(r.course, "SV Một", "20220001")
	r.s2 = d.student(r.course, "SV Hai", "20220002")
	var comp uuid.UUID
	r.scheme, comp = d.scheme(r.course, r.teacher, 1)
	r.exam = d.exam(r.course, r.teacher, status, examMax)
	r.item = d.examItem(r.course, r.scheme, comp, r.exam, "weekly", itemMax)
	return r
}

func (r examRig) inputs(ids ...uuid.UUID) map[uuid.UUID]grade.Input {
	r.t.Helper()
	in, err := grade.LoadInputs(r.ctx, r.q, r.course, r.scheme, ids)
	require.NoError(r.t, err)
	return in
}

// TestExamScorePublishedOnly — AC4: bài đã PUBLISHED mới có điểm; bài chưa công bố → thiếu (không phải 0).
func TestExamScorePublishedOnly(t *testing.T) {
	t.Parallel()
	r := newExamRig(t, "CLOSED", "10", "10")
	r.attempt(r.course, r.exam, r.s1, "7.50", "")
	_, ok := r.inputs(r.s1)[r.s1].Scores[r.item]
	require.False(t, ok, "bài chưa công bố: thiếu")
	r.exec(`update exams set status = 'PUBLISHED', published_at = now() where id = $1`, r.exam)
	got := r.inputs(r.s1)[r.s1].Scores[r.item]
	require.True(t, got.Equal(dec("7.5")), got.String())
}

// TestExamScoreFollowsAdjust — sửa điểm / chấm lại ở bài thi → lần đọc sau thấy giá trị mới (không sao chép).
func TestExamScoreFollowsAdjust(t *testing.T) {
	t.Parallel()
	r := newExamRig(t, "PUBLISHED", "10", "10")
	r.attempt(r.course, r.exam, r.s1, "6.00", "")
	require.True(t, r.inputs(r.s1)[r.s1].Scores[r.item].Equal(dec("6")))
	r.exec(`update exam_attempts set adjusted_score = 8.25, adjusted_reason = 'Phúc khảo' where exam_id = $1 and student_id = $2`, r.exam, r.s1)
	require.True(t, r.inputs(r.s1)[r.s1].Scores[r.item].Equal(dec("8.25")), "COALESCE(adjusted, auto)")
	r.exec(`update exam_attempts set adjusted_score = null, adjusted_reason = null, auto_score = 9 where exam_id = $1 and student_id = $2`, r.exam, r.s1)
	require.True(t, r.inputs(r.s1)[r.s1].Scores[r.item].Equal(dec("9")), "chấm lại")
}

// TestExamScoreMissingIsNotZero — không có lượt / lượt đang làm → thiếu, không phải 0.
func TestExamScoreMissingIsNotZero(t *testing.T) {
	t.Parallel()
	r := newExamRig(t, "PUBLISHED", "10", "10")
	r.attempt(r.course, r.exam, r.s1, "0.00", "")
	r.exec(`insert into exam_attempts (course_id, exam_id, student_id, deadline_at) values ($1, $2, $3, now() + interval '1 hour')`, r.course, r.exam, r.s2)
	in := r.inputs(r.s1, r.s2)
	z, ok := in[r.s1].Scores[r.item]
	require.True(t, ok && z.IsZero(), "điểm 0 thật là 0")
	_, ok = in[r.s2].Scores[r.item]
	require.False(t, ok, "lượt đang làm: thiếu")
	require.Contains(t, in, r.s2, "sinh viên không có điểm vẫn có đầu vào (Scores rỗng)")
}

// TestExamScoreScale — quy thang: score ÷ exams.max_score × item.max_score (decimal, chia 20 chữ số).
func TestExamScoreScale(t *testing.T) {
	t.Parallel()
	r := newExamRig(t, "PUBLISHED", "10", "30")
	r.attempt(r.course, r.exam, r.s1, "10.00", "")
	r.attempt(r.course, r.exam, r.s2, "7.00", "")
	in := r.inputs(r.s1, r.s2)
	require.True(t, in[r.s1].Scores[r.item].Equal(dec("30")))
	require.True(t, in[r.s2].Scores[r.item].Equal(dec("21")))

	r2 := newExamRig(t, "PUBLISHED", "30", "10")
	r2.attempt(r2.course, r2.exam, r2.s1, "10.00", "")
	got := r2.inputs(r2.s1)[r2.s1].Scores[r2.item]
	require.Equal(t, "3.33333333333333333333", got.String(), "10/30 × 10: không chặt ở 16 chữ số")
}

// TestExamScoreMatchesSeedCSV — 24 dòng MCQ của seed/expected_exam_scores.csv đi qua bộ đọc nguyên giá trị.
func TestExamScoreMatchesSeedCSV(t *testing.T) {
	t.Parallel()
	f, err := os.Open("../../../seed/expected_exam_scores.csv")
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	rows, err := csv.NewReader(f).ReadAll()
	require.NoError(t, err)
	r := newExamRig(t, "PUBLISHED", "10", "10")
	want := map[uuid.UUID]decimal.Decimal{}
	n := 0
	for _, row := range rows[1:] {
		if row[0] != "MCQ" || row[4] == "" {
			continue
		}
		s := r.student(r.course, row[1], "")
		r.attempt(r.course, r.exam, s, row[4], "")
		want[s] = dec(row[4])
		n++
	}
	require.Equal(t, 24, n)
	ids := make([]uuid.UUID, 0, n)
	for id := range want {
		ids = append(ids, id)
	}
	in := r.inputs(ids...)
	for id, w := range want {
		require.True(t, in[id].Scores[r.item].Equal(w), "%s: %s ≠ %s", id, in[id].Scores[r.item], w)
	}
}

// TestInputsFromP5 — bonus = Σ SPEAK + BONUS; penalty_event = |Σ PENALTY|; absent = số dòng ABSENT.
func TestInputsFromP5(t *testing.T) {
	t.Parallel()
	r := newExamRig(t, "PUBLISHED", "10", "10")
	s1, s2, t0 := r.s1, r.s2, r.teacher
	r.part(r.course, s1, t0, "SPEAK", "0.25")
	r.part(r.course, s1, t0, "SPEAK", "0.25")
	r.part(r.course, s1, t0, "BONUS", "0.50")
	r.part(r.course, s1, t0, "PENALTY", "-0.50")
	r.part(r.course, s1, t0, "PENALTY", "-0.25")
	for i := 1; i <= 3; i++ {
		r.attend(r.course, r.session(r.course, i), s1, t0, "ABSENT")
	}
	in := r.inputs(s1, s2)
	require.True(t, in[s1].BonusPoints.Equal(dec("1.00")), in[s1].BonusPoints.String())
	require.True(t, in[s1].PenaltyEventPoints.Equal(dec("0.75")), "|Σ PENALTY|")
	require.Equal(t, 3, in[s1].Absent)
	require.True(t, in[s2].BonusPoints.IsZero() && in[s2].PenaltyEventPoints.IsZero())
	require.Zero(t, in[s2].Absent)
}

// TestInputsExcusedLateNotAbsent — EXCUSED và LATE không tính là vắng.
func TestInputsExcusedLateNotAbsent(t *testing.T) {
	t.Parallel()
	r := newExamRig(t, "PUBLISHED", "10", "10")
	for i, st := range []string{"ABSENT", "EXCUSED", "LATE", "PRESENT", "ABSENT"} {
		r.attend(r.course, r.session(r.course, i+1), r.s1, r.teacher, st)
	}
	require.Equal(t, 2, r.inputs(r.s1)[r.s1].Absent)
}

// TestInputsActiveOnly — sinh viên không ACTIVE không có trong đầu vào và danh sách; điểm P5 của người đó không rò sang người khác.
func TestInputsActiveOnly(t *testing.T) {
	t.Parallel()
	r := newExamRig(t, "PUBLISHED", "10", "10")
	r.part(r.course, r.s2, r.teacher, "SPEAK", "0.25")
	r.exec(`update enrollments set status = 'REMOVED', removed_at = now() where course_id = $1 and user_id = $2`, r.course, r.s2)
	in := r.inputs(r.s1, r.s2)
	require.Contains(t, in, r.s1)
	require.NotContains(t, in, r.s2)
	st, err := grade.ActiveStudents(r.ctx, r.q, r.course)
	require.NoError(t, err)
	require.Len(t, st, 1)
	require.Equal(t, r.s1, st[0].ID)
	require.Equal(t, "20220001", st[0].Code)
}

// TestInputsMatchExpectedCRM — A, B, C khớp seed/expected_crm.csv (điểm cộng, số buổi vắng).
func TestInputsMatchExpectedCRM(t *testing.T) {
	t.Parallel()
	f, err := os.Open("../../../seed/expected_crm.csv")
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	rows, err := csv.NewReader(f).ReadAll()
	require.NoError(t, err)
	r := newExamRig(t, "PUBLISHED", "10", "10")
	type want struct {
		bonus  decimal.Decimal
		absent int
	}
	wants := map[uuid.UUID]want{}
	var ids []uuid.UUID
	for _, row := range rows[1:] {
		if row[0] != "sv.gioi" && row[0] != "sv.kha" && row[0] != "sv.nguyco" {
			continue
		}
		s := r.student(r.course, row[0], "")
		ids = append(ids, s)
		var speaks, abs int
		speaks, abs = atoi(t, row[6]), atoi(t, row[5])
		for range speaks {
			r.part(r.course, s, r.teacher, "SPEAK", "0.25")
		}
		for i := range abs {
			r.attend(r.course, r.session(r.course, 100*len(ids)+i), s, r.teacher, "ABSENT")
		}
		wants[s] = want{dec(row[7]), abs}
	}
	require.Len(t, wants, 3)
	in := r.inputs(ids...)
	for id, w := range wants {
		require.True(t, in[id].BonusPoints.Equal(w.bonus), "%s: %s ≠ %s", id, in[id].BonusPoints, w.bonus)
		require.Equal(t, w.absent, in[id].Absent)
	}
}

// TestManualEntriesAndClosed — item MANUAL đọc grade_entries; Closed theo published_at; item EXAM Closed theo trạng thái bài thi.
func TestManualEntriesAndClosed(t *testing.T) {
	t.Parallel()
	r := newExamRig(t, "CLOSED", "10", "10")
	var comp uuid.UUID
	r.one(&comp, `select component_id from grade_items where id = $1`, r.item)
	mid := r.manualItem(r.course, r.scheme, comp, "mid", "10", false)
	pub := r.manualItem(r.course, r.scheme, comp, "pub", "10", true)
	r.entry(r.course, mid, r.s1, r.teacher, "8.50")
	items, err := grade.LoadItems(r.ctx, r.q, r.course, r.scheme)
	require.NoError(t, err)
	byKey := map[string]grade.Item{}
	for _, it := range items {
		byKey[it.Key] = it
	}
	require.False(t, byKey["weekly"].Closed, "bài thi CLOSED chưa công bố")
	require.False(t, byKey["mid"].Closed)
	require.True(t, byKey["pub"].Closed)
	require.Equal(t, pub, byKey["pub"].ID)
	require.True(t, r.inputs(r.s1)[r.s1].Scores[mid].Equal(dec("8.5")))
	r.exec(`update exams set status = 'PUBLISHED', published_at = now() where id = $1`, r.exam)
	items, err = grade.LoadItems(r.ctx, r.q, r.course, r.scheme)
	require.NoError(t, err)
	for _, it := range items {
		if it.Key == "weekly" {
			require.True(t, it.Closed)
			require.True(t, it.ExamMax.Equal(dec("10")))
		}
	}
}

// TestCrossCourseIsolation — điểm / điểm danh của lớp khác không lọt vào đầu vào lớp này.
func TestCrossCourseIsolation(t *testing.T) {
	t.Parallel()
	r := newExamRig(t, "PUBLISHED", "10", "10")
	c2 := r.tdb.course(r.teacher, "L2")
	s := r.student(c2, "SV lớp 2", "20229999")
	r.part(c2, s, r.teacher, "SPEAK", "0.25")
	in, err := grade.LoadInputs(r.ctx, r.q, r.course, r.scheme, []uuid.UUID{s})
	require.NoError(t, err)
	require.Empty(t, in, "sinh viên lớp 2 không có trong đầu vào lớp 1")
	st, err := grade.ActiveStudents(r.ctx, r.q, c2)
	require.NoError(t, err)
	require.Len(t, st, 1)
}

// TestExamFromOtherCourseRejected — FK phức hợp (course_id, exam_id): bài thi của lớp khác không nối được.
func TestExamFromOtherCourseRejected(t *testing.T) {
	t.Parallel()
	r := newExamRig(t, "PUBLISHED", "10", "10")
	c2 := r.tdb.course(r.teacher, "L2")
	other := r.tdb.exam(c2, r.teacher, "PUBLISHED", "10")
	var comp uuid.UUID
	r.one(&comp, `select component_id from grade_items where id = $1`, r.item)
	_, err := r.pool.Exec(r.ctx, `insert into grade_items (course_id, scheme_id, component_id, key, name, source, exam_id) values ($1, $2, $3, 'x', 'x', 'EXAM', $4)`, r.course, r.scheme, comp, other)
	require.Error(t, err)
	require.Contains(t, err.Error(), "23503")
}

// TestLeftStudentExcluded — sinh viên rời lớp không có trong bảng điểm; điểm cũ của họ không mất (grade_entries giữ nguyên).
func TestLeftStudentExcluded(t *testing.T) {
	t.Parallel()
	r := newExamRig(t, "PUBLISHED", "10", "10")
	var comp uuid.UUID
	r.one(&comp, `select component_id from grade_items where id = $1`, r.item)
	mid := r.manualItem(r.course, r.scheme, comp, "mid", "10", true)
	r.entry(r.course, mid, r.s2, r.teacher, "7")
	r.exec(`update enrollments set status = 'REMOVED', removed_at = now() where course_id = $1 and user_id = $2`, r.course, r.s2)
	st, err := grade.ActiveStudents(r.ctx, r.q, r.course)
	require.NoError(t, err)
	require.Len(t, st, 1)
	var n int
	r.one(&n, `select count(*) from grade_entries where student_id = $1`, r.s2)
	require.Equal(t, 1, n)
}
