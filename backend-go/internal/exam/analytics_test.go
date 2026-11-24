package exam_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

// aRig: lớp có bài thi theo trạng thái, sinh viên ACTIVE và lượt GRADED với điểm cho trước.
type aRig struct {
	*rig
	students []uuid.UUID
	items    map[uuid.UUID][]uuid.UUID // exam → exam_items.id
	base     time.Time
}

func newARig(t *testing.T, students int) *aRig {
	r := &aRig{rig: newRig(t), items: map[uuid.UUID][]uuid.UUID{}, base: time.Date(2026, 11, 2, 9, 0, 0, 0, time.FixedZone("ICT", 7*3600))} // thứ Hai
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TEACHER', 'ACTIVE', 'ADMIN')`, r.course, r.teacher)
	for i := range students {
		var id uuid.UUID
		require.NoError(t, r.pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ($1, $2, 'STUDENT') returning id`, fmt.Sprintf("sv%d.%s@example.test", i, uuid.NewString()[:6]), fmt.Sprintf("Sinh Viên %02d", i)).Scan(&id))
		r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, r.course, id)
		r.students = append(r.students, id)
	}
	return r
}

// exam tạo bài thi ở trạng thái `st`, mở lúc base + day ngày, với một câu trắc nghiệm và một câu code.
func (r *aRig) exam(title, st string, day int) uuid.UUID {
	r.t.Helper()
	var eid uuid.UUID
	opens := r.base.AddDate(0, 0, day)
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into exams (course_id, title, status, opens_at, closes_at, duration_minutes, max_score, published_at, created_by)
		values ($1, $2, $3::exam_status, $4, $4::timestamptz + interval '2 hours', 30, 10, case when $3 = 'PUBLISHED' then $4 end, $5) returning id`, r.course, title, st, opens, r.teacher).Scan(&eid))
	for pos, q := range []uuid.UUID{r.mcq(title + " MCQ").ID, r.code(title + " CODE").ID} {
		var item uuid.UUID
		require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into exam_items (course_id, exam_id, question_id, position) values ($1, $2, $3, $4) returning id`, r.course, eid, q, pos+1).Scan(&item))
		r.items[eid] = append(r.items[eid], item)
	}
	return eid
}

// attempt ghi lượt GRADED của sinh viên `s` với điểm `score`; mcqRight: câu trắc nghiệm đúng; codeRatio: điểm đạt của câu code (tối đa 5); ce: biên dịch lỗi.
func (r *aRig) attempt(e uuid.UUID, s int, score string, mcqRight bool, codeRatio string, ce bool) {
	r.t.Helper()
	mcq := "0"
	if mcqRight {
		mcq = "5"
	}
	rows := []map[string]any{
		{"item_id": r.items[e][0], "points": "5", "max": "5", "earned": mcq},
		{"item_id": r.items[e][1], "points": "5", "max": "5", "earned": codeRatio, "code": map[string]any{"compile_ok": !ce}},
	}
	b, _ := json.Marshal(rows)
	r.exec(`insert into exam_attempts (course_id, exam_id, student_id, status, deadline_at, submitted_at, submit_reason, auto_score, breakdown, graded_at)
		values ($1, $2, $3, 'GRADED', now() + interval '1 hour', now(), 'MANUAL', $4, $5::jsonb, now())`, r.course, e, r.students[s], score, string(b))
}

// TestExamStatsPublishedOnly — US-P10-06 AC2: chỉ bài PUBLISHED vào overview / trend / hard-items / declining / compare; điểm đã sửa thắng điểm tự chấm; số tính lại từ SQL.
func TestExamStatsPublishedOnly(t *testing.T) {
	r := newARig(t, 6)
	ctx := t.Context()
	e1, e2 := r.exam("Tuần 1", "PUBLISHED", 0), r.exam("Tuần 2", "PUBLISHED", 7)
	for i, sc := range []string{"2.00", "4.00", "6.00", "8.00", "10.00", "5.00"} {
		r.attempt(e1, i, sc, i%2 == 0, "5", false)
	}
	for i, sc := range []string{"3.00", "5.00", "7.00", "9.00"} {
		r.attempt(e2, i, sc, i%2 == 0, "0.5", i == 0)
	}
	// ba bài chưa công bố có điểm — không được lọt vào đâu
	for _, st := range []string{"CLOSED", "OPEN", "DRAFT"} {
		e := r.exam("Chưa công bố "+st, st, 14)
		for i := range 6 {
			r.attempt(e, i, "10.00", true, "1", false)
		}
	}

	ov, err := r.svc.AnalyticsOverview(ctx, r.course, nil, "", true)
	require.NoError(t, err)
	require.Equal(t, "exam", ov.Scope)
	require.Len(t, ov.Exams, 1)
	require.Equal(t, e2, ov.Exams[0].ID, "mặc định: bài PUBLISHED mới nhất")
	require.Equal(t, 4, ov.Submissions)
	require.Equal(t, "6.00", *ov.Mean)
	require.Equal(t, "6.00", *ov.Median)
	require.Equal(t, "0.75", *ov.PassRate, "ngưỡng đạt = 50 % điểm tối đa: 5, 7, 9 đạt, 3 không")
	require.Len(t, ov.Distribution, 10)
	sum := 0
	for _, b := range ov.Distribution {
		sum += b.Count
	}
	require.Equal(t, 4, sum)

	// điểm đã sửa thắng điểm tự chấm, số liệu đổi ngay (không cache phía máy chủ)
	r.exec(`update exam_attempts set adjusted_score = 9.00, adjusted_reason = 'phúc khảo' where exam_id = $1 and student_id = $2`, e2, r.students[0])
	ov, err = r.svc.AnalyticsOverview(ctx, r.course, &e2, "", true)
	require.NoError(t, err)
	require.Equal(t, "7.50", *ov.Mean)

	tr, err := r.svc.AnalyticsTrend(ctx, r.course, true)
	require.NoError(t, err)
	require.Len(t, tr, 2, "trend chỉ gồm bài PUBLISHED")
	require.Equal(t, []uuid.UUID{e1, e2}, []uuid.UUID{tr[0].ExamID, tr[1].ExamID}, "cũ → mới")
	require.Equal(t, "5.83", *tr[0].Mean)

	hard, next, err := r.svc.AnalyticsHardItems(ctx, r.course, &e1, "", 10)
	require.NoError(t, err)
	require.Nil(t, next)
	require.Len(t, hard, 2)
	require.Equal(t, "MCQ", hard[0].Type, "trắc nghiệm sai 3/6 = 0,5 đứng trước code sai 0")
	require.Equal(t, "0.5000", hard[0].WrongRate)
	require.Equal(t, "CODE", hard[1].Type)
	require.NotNil(t, hard[1].CERate)

	cmp, err := r.svc.AnalyticsCompare(ctx, r.teacher)
	require.NoError(t, err)
	require.Len(t, cmp, 1)
	require.Equal(t, 10, cmp[0].Submissions, "6 + 4 lượt của hai bài PUBLISHED, không tính bài chưa công bố")
	require.True(t, cmp[0].Enough)

	// bài không PUBLISHED / không tồn tại → 404
	closed := r.exam("Đã đóng", "CLOSED", 21)
	for _, id := range []uuid.UUID{closed, uuid.New()} {
		_, err = r.svc.AnalyticsOverview(ctx, r.course, &id, "", true)
		var ae *apierr.Error
		require.ErrorAs(t, err, &ae)
		require.Equal(t, http.StatusNotFound, ae.Status)
	}
}

// TestExamStatsAdminNoIdentity — AC7: ADMIN chỉ số tổng (số bài nộp, tỉ lệ đạt), không trung bình / trung vị / phân bố.
func TestExamStatsAdminNoIdentity(t *testing.T) {
	r := newARig(t, 3)
	e := r.exam("Tuần 1", "PUBLISHED", 0)
	for i, sc := range []string{"4.00", "6.00", "8.00"} {
		r.attempt(e, i, sc, true, "1", false)
	}
	ov, err := r.svc.AnalyticsOverview(t.Context(), r.course, nil, "", false)
	require.NoError(t, err)
	require.Equal(t, 3, ov.Submissions)
	require.Equal(t, "0.67", *ov.PassRate)
	require.Nil(t, ov.Mean)
	require.Nil(t, ov.Median)
	require.Nil(t, ov.Distribution)
	raw, _ := json.Marshal(ov)
	require.NotContains(t, string(raw), "student", "không danh tính")
	tr, err := r.svc.AnalyticsTrend(t.Context(), r.course, false)
	require.NoError(t, err)
	require.Nil(t, tr[0].Mean)
	require.Equal(t, 3, tr[0].Submissions)
}

// TestExamStatsWeek — AC3: gộp các bài trong tuần (thứ Hai → Chủ nhật, giờ +07); tuần trống → cấu trúc rỗng; exam_id + week → 422.
func TestExamStatsWeek(t *testing.T) {
	r := newARig(t, 2)
	ctx := t.Context()
	a, b := r.exam("Thứ Hai", "PUBLISHED", 0), r.exam("Thứ Sáu", "PUBLISHED", 4)
	r.exam("Tuần sau", "PUBLISHED", 7)
	r.attempt(a, 0, "4.00", true, "1", false)
	r.attempt(a, 1, "6.00", true, "1", false)
	r.attempt(b, 0, "8.00", true, "1", false)

	for _, week := range []string{"2026-W45", "2026-11-04"} {
		ov, err := r.svc.AnalyticsOverview(ctx, r.course, nil, week, true)
		require.NoError(t, err, week)
		require.Equal(t, "week", ov.Scope)
		require.Len(t, ov.Exams, 2, week)
		require.Equal(t, 3, ov.Submissions)
		require.Equal(t, "6.00", *ov.Mean, week)
	}
	ov, err := r.svc.AnalyticsOverview(ctx, r.course, nil, "2026-W20", true)
	require.NoError(t, err)
	require.Empty(t, ov.Exams)
	require.Zero(t, ov.Submissions)
	require.Nil(t, ov.Mean)

	for name, tc := range map[string]struct {
		eid  *uuid.UUID
		week string
	}{"cả hai": {&a, "2026-W45"}, "week sai": {nil, "tuần trước"}, "week 99": {nil, "2026-W99"}} {
		_, err = r.svc.AnalyticsOverview(ctx, r.course, tc.eid, tc.week, true)
		var ae *apierr.Error
		require.ErrorAs(t, err, &ae, name)
		require.Equal(t, http.StatusUnprocessableEntity, ae.Status, name)
	}

	// lớp chưa có bài công bố: cấu trúc rỗng, không lỗi
	empty := newARig(t, 0)
	ov, err = empty.svc.AnalyticsOverview(ctx, empty.course, nil, "", true)
	require.NoError(t, err)
	require.Empty(t, ov.Exams)
	hs, _, err := empty.svc.AnalyticsHardItems(ctx, empty.course, nil, "", 10)
	require.NoError(t, err)
	require.Empty(t, hs)
}

// TestExamStatsDeclining — AC5: chỉ sinh viên giảm NGHIÊM NGẶT ở 3 bài LIÊN TIẾP; thiếu bài ở giữa làm đứt chuỗi; phân trang con trỏ.
func TestExamStatsDeclining(t *testing.T) {
	r := newARig(t, 6)
	ctx := t.Context()
	e := []uuid.UUID{r.exam("B1", "PUBLISHED", 0), r.exam("B2", "PUBLISHED", 7), r.exam("B3", "PUBLISHED", 14), r.exam("B4", "PUBLISHED", 21)}
	set := func(s int, scores ...string) {
		for i, sc := range scores {
			if sc != "" {
				r.attempt(e[i], s, sc, true, "1", false)
			}
		}
	}
	set(0, "9.00", "7.00", "5.00", "")     // giảm 3 liên tiếp → có
	set(1, "9.00", "7.00", "", "")         // chỉ 2 bài → không
	set(2, "9.00", "", "5.00", "3.00")     // thiếu bài giữa → B3, B4 chỉ 2 bài liền kề → không
	set(3, "9.00", "8.00", "8.00", "")     // không giảm nghiêm ngặt → không
	set(4, "8.00", "9.00", "7.00", "5.00") // B2 > B3 > B4 liền kề → có (bộ ba mới nhất)
	set(5, "6.00", "5.00", "4.00", "3.00") // chuỗi dài → bộ ba mới nhất B2-B4

	rows, next, err := r.svc.AnalyticsDeclining(ctx, r.course, "", 10)
	require.NoError(t, err)
	require.Nil(t, next)
	var names []string
	for _, x := range rows {
		names = append(names, x.Name)
	}
	require.Equal(t, []string{"Sinh Viên 00", "Sinh Viên 04", "Sinh Viên 05"}, names)
	require.Equal(t, []string{"9.00", "7.00", "5.00"}, []string{rows[0].Scores[0].Score, rows[0].Scores[1].Score, rows[0].Scores[2].Score})
	require.Equal(t, []string{"B2", "B3", "B4"}, []string{rows[2].Scores[0].Title, rows[2].Scores[1].Title, rows[2].Scores[2].Title})

	// con trỏ: từng dòng một, không trùng, không sót
	var got []string
	cursor := ""
	for range 5 {
		page, nx, err := r.svc.AnalyticsDeclining(ctx, r.course, cursor, 1)
		require.NoError(t, err)
		for _, x := range page {
			got = append(got, x.Name)
		}
		if nx == nil {
			break
		}
		cursor = *nx
	}
	require.Equal(t, names, got)

	// không đổi điểm rủi ro: không có bảng nào bị ghi bởi thao tác đọc
	_, _, err = r.svc.AnalyticsDeclining(ctx, r.course, "rac", 1)
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apierr.InvalidCursor, ae.Code)
}

// TestExamStatsHardItemsCursor — AC1, AC9: câu sai nhiều có phân trang con trỏ ổn định; câu code gắn nhãn khác trắc nghiệm.
func TestExamStatsHardItemsCursor(t *testing.T) {
	r := newARig(t, 4)
	ctx := t.Context()
	e := r.exam("B1", "PUBLISHED", 0)
	for i := range 4 {
		r.attempt(e, i, "5.00", i < 2, "1.25", i < 2)
	}
	var got []exam.HardItem
	cursor := ""
	for range 4 {
		page, nx, err := r.svc.AnalyticsHardItems(ctx, r.course, &e, cursor, 1)
		require.NoError(t, err)
		got = append(got, page...)
		if nx == nil {
			break
		}
		cursor = *nx
	}
	require.Len(t, got, 2)
	require.Equal(t, "CODE", got[0].Type)
	require.Equal(t, "0.7500", got[0].WrongRate, "code: 1 − tỉ lệ điểm trung bình")
	require.Equal(t, "0.5000", *got[0].CERate)
	require.Equal(t, "MCQ", got[1].Type)
	require.Equal(t, "0.5000", got[1].WrongRate, "trắc nghiệm: 2/4 sai")
	require.Nil(t, got[1].CERate)
	require.True(t, strings.Compare(got[0].ItemID.String(), got[1].ItemID.String()) != 0)
}

// TestExamStatsCompareThreshold — AC6: lớp dưới 5 bài nộp «Chưa đủ» (không số); đúng 5 thì hiện; chỉ lớp mình dạy.
func TestExamStatsCompareThreshold(t *testing.T) {
	r := newARig(t, 5)
	ctx := t.Context()
	e := r.exam("B1", "PUBLISHED", 0)
	for i := range 4 {
		r.attempt(e, i, "6.00", true, "1", false)
	}
	rows, err := r.svc.AnalyticsCompare(ctx, r.teacher)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.False(t, rows[0].Enough)
	require.Nil(t, rows[0].Mean)
	require.Equal(t, 4, rows[0].Submissions)

	r.attempt(e, 4, "6.00", true, "1", false)
	rows, err = r.svc.AnalyticsCompare(ctx, r.teacher)
	require.NoError(t, err)
	require.True(t, rows[0].Enough)
	require.Equal(t, "6.00", *rows[0].Mean)

	other := uuid.New()
	rows, err = r.svc.AnalyticsCompare(ctx, other)
	require.NoError(t, err)
	require.Empty(t, rows, "không dạy lớp nào → không thấy lớp nào")
}
