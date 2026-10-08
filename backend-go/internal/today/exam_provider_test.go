package today_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/today"
)

// examAt chèn một bài với giờ mở / đóng tương đối so với đồng hồ của rig.
func (r *rig) examAt(c cls, by uuid.UUID, title, status string, opens, closes time.Duration, minutes int) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	now := r.clk.Now()
	var o, cl, d any
	if status != "DRAFT" {
		o, cl, d = now.Add(opens), now.Add(closes), minutes
	}
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into exams (course_id, title, status, opens_at, closes_at, duration_minutes, published_at, created_by)
		values ($1, $2, $3::exam_status, $4, $5, $6, case when $3 = 'PUBLISHED' then now() end, $7) returning id`, c.id, title, status, o, cl, d, by).Scan(&id))
	return id
}

func (r *rig) attempt(c cls, eid uuid.UUID, who store.User, status string, deadline time.Duration) {
	r.t.Helper()
	now := r.clk.Now()
	if status == "IN_PROGRESS" {
		_, err := r.pool.Exec(r.t.Context(), `insert into exam_attempts (course_id, exam_id, student_id, started_at, deadline_at) values ($1, $2, $3, $4, $5)`, c.id, eid, who.ID, now.Add(-10*time.Minute), now.Add(deadline))
		require.NoError(r.t, err)
		return
	}
	_, err := r.pool.Exec(r.t.Context(), `insert into exam_attempts (course_id, exam_id, student_id, status, started_at, deadline_at, submitted_at, submit_reason) values ($1, $2, $3, $4::attempt_status, $5::timestamptz, $6, $5::timestamptz + interval '5 minutes', 'MANUAL')`,
		c.id, eid, who.ID, status, now.Add(-30*time.Minute), now.Add(deadline))
	require.NoError(r.t, err)
}

// examItems: việc EXAM_* trong phản hồi của Staff (`actions`).
func examItems(res resp) []map[string]any {
	var out []map[string]any
	for _, a := range actionsOf(res) {
		if m := a.(map[string]any); strings.HasPrefix(m["kind"].(string), "EXAM_") {
			out = append(out, m)
		}
	}
	return out
}

// recommendedExam trả `recommended` của sinh viên nếu là việc EXAM_*, ngược lại nil.
func recommendedExam(res resp) map[string]any {
	m, _ := res.json()["recommended"].(map[string]any)
	if m == nil || !strings.HasPrefix(m["kind"].(string), "EXAM_") {
		return nil
	}
	return m
}

var weekdayVI = []string{"Chủ nhật", "Thứ Hai", "Thứ Ba", "Thứ Tư", "Thứ Năm", "Thứ Sáu", "Thứ Bảy"}

// TestExamProvidersStudent — US-PE-04 AC11: EXAM_IN_PROGRESS (5) · EXAM_OPEN (8) · EXAM_UPCOMING (42, ≤ 48 giờ); chuỗi đúng SRS 4.10; việc biến mất khi nộp / bài đóng;
// DRAFT, bài xa hơn 48 giờ, bài đã đóng và bài lớp khác không hiện; Staff không thấy; `recommended` là việc đầu.
func TestExamProvidersStudent(t *testing.T) {
	r := newRig(t)
	c, other := r.klass("761971"), r.klass("761972")
	gv, gvS := r.teaching(c, other)
	sv, svS := r.student("")
	r.enrollAt(c, sv, "STUDENT", "ACTIVE", "", time.Hour)
	_, svOtherS := r.student("") // sinh viên không ở lớp nào

	prog := r.examAt(c, gv.ID, "Giữa kỳ", "OPEN", -10*time.Minute, 2*time.Hour, 60)
	r.attempt(c, prog, sv, "IN_PROGRESS", 20*time.Minute)
	open := r.examAt(c, gv.ID, "Tuần 9", "OPEN", -10*time.Minute, 2*time.Hour, 30)
	soon := r.examAt(c, gv.ID, "Tuần 10", "SCHEDULED", 3*time.Hour, 5*time.Hour, 45)
	_ = r.examAt(c, gv.ID, "Xa", "SCHEDULED", 72*time.Hour, 74*time.Hour, 45)
	_ = r.examAt(c, gv.ID, "Nháp", "DRAFT", 0, 0, 0)
	_ = r.examAt(c, gv.ID, "Đã đóng", "CLOSED", -5*time.Hour, -time.Hour, 45)
	done := r.examAt(c, gv.ID, "Đã nộp", "OPEN", -10*time.Minute, 2*time.Hour, 30)
	r.attempt(c, done, sv, "GRADING", 20*time.Minute)
	_ = r.examAt(other, gv.ID, "Lớp khác", "OPEN", -10*time.Minute, 2*time.Hour, 30)

	// Provider: mọi việc của sinh viên, đã xếp theo bậc (5 < 8 < 42).
	v := today.Viewer{UserID: sv.ID, Role: today.RoleStudent, EmailVerified: true, Now: r.clk.Now(),
		Courses: []today.CourseRef{{ID: c.id, ClassCode: "761971", RoleInCourse: "STUDENT"}}}
	got, err := today.StudentProvider{Pool: r.pool}.Items(t.Context(), v, today.Scope{})
	require.NoError(t, err)
	today.Rank(got)
	require.Len(t, got, 3, "đang làm dở, đang mở, sắp mở")
	require.Equal(t, []today.Kind{today.KindExamInProgress, today.KindExamOpen, today.KindExamUpcoming}, []today.Kind{got[0].Kind, got[1].Kind, got[2].Kind}, "bậc 5 < 8 < 42")
	require.Equal(t, []int{5, 8, 42}, []int{got[0].Tier, got[1].Tier, got[2].Tier})
	require.Equal(t, "Bài thi Giữa kỳ đang làm dở", got[0].Title)
	require.Equal(t, "Còn 20 phút. Làm tiếp.", got[0].Reason)
	require.Equal(t, "/exams/"+prog.String()+"/take", got[0].Href)
	zone := time.FixedZone("ICT", 7*3600)
	closes := r.clk.Now().Add(2 * time.Hour).In(zone)
	require.Equal(t, "Bài thi Tuần 9 đang mở", got[1].Title)
	require.Equal(t, fmt.Sprintf("Mở đến %s. Bạn có 30 phút để làm.", closes.Format("15:04 02/01")), got[1].Reason)
	require.Equal(t, "/exams/"+open.String()+"/take", got[1].Href)
	opens := r.clk.Now().Add(3 * time.Hour).In(zone)
	require.Equal(t, fmt.Sprintf("Tuần 10 · %s, %s", weekdayVI[opens.Weekday()], opens.Format("15:04")), got[2].Title)
	require.Equal(t, "Làm trong 45 phút. Chuẩn bị máy tính nếu có bài lập trình.", got[2].Reason)
	require.Equal(t, "/exams/"+soon.String()+"/take", got[2].Href)
	require.Equal(t, c.id, got[0].Course.ID)
	one := v
	one.Courses = []today.CourseRef{{ID: other.id, ClassCode: "761972", RoleInCourse: "STUDENT"}}
	none, err := today.StudentProvider{Pool: r.pool}.Items(t.Context(), one, today.Scope{})
	require.NoError(t, err)
	require.Len(t, none, 1, "lớp khác chỉ có bài của lớp đó")
	require.Equal(t, "Bài thi Lớp khác đang mở", none[0].Title)

	// HTTP: `recommended` là việc đầu theo khoá xếp; nộp bài → việc kế; bài đóng → việc kế; hết giờ → không còn.
	rec := recommendedExam(r.tget(svS, "/me/today"))
	require.NotNil(t, rec)
	require.Equal(t, "EXAM_IN_PROGRESS", rec["kind"])
	require.Equal(t, "Bài thi Giữa kỳ đang làm dở", rec["title"])
	require.Equal(t, "EXAM_IN_PROGRESS", recommendedExam(r.tget(svS, "/courses/"+c.sid()+"/today"))["kind"])
	require.Nil(t, recommendedExam(r.tget(svS, "/courses/"+other.sid()+"/today")), "không phải lớp của sinh viên")
	// Staff và người ngoài lớp không thấy
	require.Empty(t, examItems(r.tget(gvS, "/me/today")))
	require.Nil(t, recommendedExam(r.tget(svOtherS, "/me/today")))
	require.NotContains(t, string(r.tget(gvS, "/me/today").body), "EXAM_OPEN")
	// nộp bài → hết việc dở dang
	_, err = r.pool.Exec(t.Context(), `update exam_attempts set status='GRADED', submitted_at=now(), submit_reason='MANUAL', auto_score=1, graded_at=now() where exam_id=$1`, prog)
	require.NoError(t, err)
	require.Equal(t, "EXAM_OPEN", recommendedExam(r.tget(svS, "/me/today"))["kind"])
	// bài đóng → hết việc đang mở; chỉ còn bài sắp mở
	_, err = r.pool.Exec(t.Context(), `update exams set status='CLOSED' where id=$1`, open)
	require.NoError(t, err)
	require.Equal(t, "EXAM_UPCOMING", recommendedExam(r.tget(svS, "/me/today"))["kind"])
	_, err = r.pool.Exec(t.Context(), `update exams set status='DRAFT', opens_at=null, closes_at=null, duration_minutes=null where id=$1`, soon)
	require.NoError(t, err)
	require.Nil(t, recommendedExam(r.tget(svS, "/me/today")))
}

// TestExamTodayInvalidation — AC11: lên lịch / bỏ lịch xoá cache "Hôm nay" của sinh viên ≤ 2 s qua sự kiện outbox (không đợi TTL 60 s).
func TestExamTodayInvalidation(t *testing.T) {
	r := newRig(t)
	r.worker()
	c := r.klass("761973")
	gv, _ := r.teaching(c)
	sv, svS := r.student("")
	r.enrollAt(c, sv, "STUDENT", "ACTIVE", "", time.Hour)
	require.Nil(t, recommendedExam(r.tgetC(svS, "/me/today")), "cache đã mồi (rỗng)")

	svc := &exam.Service{Pool: r.pool, Clock: r.clk}
	q, err := svc.Create(t.Context(), gv.ID, c.id, exam.QuestionIn{Type: "TRUE_FALSE", Title: "Đúng sai", Topic: "t", Stem: "1<2", Value: new(true)})
	require.NoError(t, err)
	_, err = svc.Review(t.Context(), gv.ID, c.id, q.ID, exam.DecisionApprove, q.Version)
	require.NoError(t, err)
	opens, closes := r.clk.Now().Add(3*time.Hour), r.clk.Now().Add(5*time.Hour)
	e, err := svc.CreateExam(t.Context(), gv.ID, c.id, exam.ExamIn{Title: new("Tuần 9"), OpensAt: &opens, ClosesAt: &closes, DurationMinutes: new(45)})
	require.NoError(t, err)
	e, err = svc.PutItems(t.Context(), gv.ID, c.id, e.ID, exam.ItemsIn{Items: []exam.ItemIn{{QuestionID: q.ID, Points: decimal.NewFromInt(1)}}}, e.Version)
	require.NoError(t, err)
	require.Nil(t, recommendedExam(r.tgetC(svS, "/me/today")), "nháp: chưa có việc")

	_, err = svc.ScheduleExam(t.Context(), gv.ID, c.id, e.ID)
	require.NoError(t, err)
	took := waitFor(t, 2*time.Second, "EXAM_UPCOMING hiện sau khi lên lịch", func() bool { return recommendedExam(r.tgetC(svS, "/me/today")) != nil })
	require.Less(t, took, 2*time.Second)

	_, err = svc.UnscheduleExam(t.Context(), gv.ID, c.id, e.ID)
	require.NoError(t, err)
	waitFor(t, 2*time.Second, "EXAM_UPCOMING biến sau khi bỏ lịch", func() bool { return recommendedExam(r.tgetC(svS, "/me/today")) == nil })
}
