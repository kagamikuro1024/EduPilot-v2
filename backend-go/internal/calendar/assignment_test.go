package calendar_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
)

// asg chèn một bài tập (đã công bố nếu published) có hạn sau `in`.
func (f *fx) asg(course uuid.UUID, code string, in time.Duration, published bool) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	pub := "null"
	if published {
		pub = "now()"
	}
	require.NoError(f.t, f.pool.QueryRow(f.t.Context(), `insert into assignments (course_id, title, due_at, submission_code, created_by, published_at) values ($1, $2, $3, $4, $5, `+pub+`) returning id`,
		course, "Bài tập "+code, f.clk.Now().Add(in), code, f.teacher).Scan(&id))
	return id
}

func (f *fx) submit(course, a, student uuid.UUID) {
	f.t.Helper()
	_, err := f.pool.Exec(f.t.Context(), `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method)
		values ($1, $2, $3, 'IN_APP', 1, 'k', 'a.pdf', 1, 'application/pdf', 'x', 'MATCHED', 'IN_APP')`, course, a, student)
	require.NoError(f.t, err)
}

func (f *fx) assignmentItem(u uuid.UUID, role auth.Role, id uuid.UUID) map[string]any {
	for _, it := range f.items(u, role, -time.Hour, 10*24*time.Hour) {
		if it["id"] == "assignment:"+id.String() {
			return it
		}
	}
	return nil
}

// TestDueChangeCalendarAndToday — nguồn lịch thứ tư: bài đã công bố hiện ở lịch (DEADLINE, href theo vai, personal_state theo bài nộp), bài nháp thì không; đổi hạn → lịch, ICS và nhắc theo giờ mới.
func TestDueChangeCalendarAndToday(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	a := f.asg(f.course, "BT03", 48*time.Hour, true)
	draft := f.asg(f.course, "NHAP", 48*time.Hour, false)
	require.Nil(t, f.assignmentItem(f.sv, auth.RoleStudent, draft))

	sv := f.assignmentItem(f.sv, auth.RoleStudent, a)
	require.NotNil(t, sv)
	require.Equal(t, "DEADLINE", sv["type"])
	require.Equal(t, "assignment", sv["source"])
	require.Equal(t, "/assignments/"+a.String(), sv["href"])
	require.Equal(t, "NOT_STARTED", sv["personal_state"])
	require.Nil(t, sv["ends_at"])
	f.submit(f.course, a, f.sv)
	require.Equal(t, "SUBMITTED", f.assignmentItem(f.sv, auth.RoleStudent, a)["personal_state"])
	require.Equal(t, "/grading?tab=assignments", f.assignmentItem(f.ta, auth.RoleTA, a)["href"])

	// ICS: UID riêng.
	tok, _ := f.icsToken(f.sv2, auth.RoleStudent)
	_, _, body := f.feed(tok)
	require.Contains(t, body, "UID:assignment-"+a.String()+"@edupilot")
	require.NotContains(t, body, draft.String())

	// đổi hạn (sửa trực tiếp như PUT làm) → lịch đổi ngay, không cache cũ.
	newDue := f.clk.Now().Add(96 * time.Hour)
	_, err := f.pool.Exec(t.Context(), `update assignments set due_at = $2 where id = $1`, a, newDue)
	require.NoError(t, err)
	got := f.assignmentItem(f.sv2, auth.RoleStudent, a)
	at, err := time.Parse(time.RFC3339Nano, got["starts_at"].(string)) //nolint:forcetypeassert // JSON string
	require.NoError(t, err)
	require.WithinDuration(t, newDue, at, time.Millisecond)
}

// TestReminderOnlyNotSubmitted — nhắc hạn bài tập (nguồn ASSIGNMENT): chỉ sinh viên chưa nộp, một lần, chuông ghi giờ hạn, link tới bài; đổi hạn → nhắc lại theo giờ mới.
func TestReminderOnlyNotSubmitted(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	a := f.asg(f.course, "BT03", 5*time.Hour, true)
	f.asg(f.course, "NHAP", 5*time.Hour, false) // bài nháp không nhắc
	f.submit(f.course, a, f.sv)
	require.Equal(t, 1, f.tick(), "chỉ sv2 chưa nộp")
	require.Equal(t, 1, f.bells(f.sv2))
	require.Zero(t, f.bells(f.sv))
	require.Equal(t, 1, count(f, `select count(*) from notifications where user_id = $1 and link = '/assignments/' || $2::text and title like '%hạn%'`, f.sv2, a))
	require.Equal(t, 1, count(f, `select count(*) from reminder_log where source_type = 'ASSIGNMENT' and source_id = $1`, a))
	require.Zero(t, f.tick())
	// đổi hạn: nhắc lại theo giờ mới (reminder_log khoá theo starts_at)
	_, err := f.pool.Exec(t.Context(), `update assignments set due_at = due_at + interval '1 hour' where id = $1`, a)
	require.NoError(t, err)
	require.Equal(t, 1, f.tick())
}

// TestCalendarDeadlineUnion — bài đã công bố gộp vào lịch như nguồn thứ tư (không nhân bản); bài nháp và bài của lớp khác thì không.
func TestCalendarDeadlineUnion(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	a := f.asg(f.course, "BT03", 48*time.Hour, true)
	f.asg(f.course, "NHAP", 48*time.Hour, false)
	other := f.newCourse("ACTIVE")
	f.asg(other, "KHAC", 48*time.Hour, true)
	var ids []string
	for _, it := range f.items(f.sv, auth.RoleStudent, -time.Hour, 10*24*time.Hour) {
		if it["source"] == "assignment" {
			ids = append(ids, it["id"].(string)) //nolint:forcetypeassert // JSON string
		}
	}
	require.Equal(t, []string{"assignment:" + a.String()}, ids)
}

// TestReminderDedupe — nhắc hạn bài tập chạy lại nhiều lần không thêm chuông / reminder_log.
func TestReminderDedupe(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	a := f.asg(f.course, "BT03", 5*time.Hour, true)
	require.Equal(t, 2, f.tick())
	for range 3 {
		require.Zero(t, f.tick())
	}
	require.Equal(t, 2, count(f, `select count(*) from reminder_log where source_type = 'ASSIGNMENT' and source_id = $1`, a))
	require.Equal(t, 2, count(f, `select count(*) from notifications where type = 'REMINDER'`))
}
