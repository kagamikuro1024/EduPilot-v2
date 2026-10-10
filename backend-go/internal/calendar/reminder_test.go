package calendar_test

import (
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/calendar"
	"github.com/edupilot/backend-go/internal/mail"
)

func (f *fx) reminder() *calendar.Reminder {
	return &calendar.Reminder{Svc: f.svc, Name: "t", Every: time.Minute, Lead: 24 * time.Hour}
}

func (f *fx) tick() int {
	f.t.Helper()
	n, err := f.reminder().Tick(f.t.Context())
	require.NoError(f.t, err)
	return n
}

func (f *fx) bells(u uuid.UUID) int {
	return count(f, `select count(*) from notifications where user_id = $1 and type = 'REMINDER'`, u)
}

// TestReminderOncePerEvent — AC11: mỗi (người, nguồn, starts_at) đúng một reminder_log + một chuông + một thư; link đúng loại nguồn.
func TestReminderOncePerEvent(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.event(f.course, "OTHER", "Nộp báo cáo", 5*time.Hour)
	e := f.exam(f.course, "Thi giữa kỳ", "SCHEDULED", 10*time.Hour)
	require.Equal(t, 4, f.tick()) // 2 sinh viên × 2 nguồn
	for _, u := range []uuid.UUID{f.sv, f.sv2} {
		require.Equal(t, 2, f.bells(u))
		require.Equal(t, 1, count(f, `select count(*) from notifications where user_id = $1 and link = '/exams/' || $2::text || '/take'`, u, e))
		require.Equal(t, 1, count(f, `select count(*) from notifications where user_id = $1 and link = '/calendar'`, u))
	}
	require.Equal(t, 4, count(f, `select count(*) from reminder_log`))
	require.Equal(t, 4, count(f, `select count(*) from mail_outbox where template = 'reminder'`))
	require.Equal(t, 4, count(f, `select count(*) from outbox where topic = 'mail.send'`))
	// ngoài cửa sổ: sau 24 giờ và đã qua không nhắc
	f.event(f.course, "OTHER", "Xa", 30*time.Hour)
	f.event(f.course, "OTHER", "Đã qua", -time.Hour)
	require.Zero(t, f.tick())
}

// TestReminderRerunNoDup — AC11: chạy lại tick nhiều lần không thêm gì.
func TestReminderRerunNoDup(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.event(f.course, "OTHER", "A", 5*time.Hour)
	require.Equal(t, 2, f.tick())
	for range 3 {
		require.Zero(t, f.tick())
	}
	require.Equal(t, 2, count(f, `select count(*) from reminder_log`))
	require.Equal(t, 2, count(f, `select count(*) from notifications where type = 'REMINDER'`))
	require.Equal(t, 2, count(f, `select count(*) from mail_outbox`))
}

// TestReminderParallelWorkers — AC11: 20 tick song song → số reminder_log = số người × nguồn, không lỗi, không trùng.
func TestReminderParallelWorkers(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	for i := range 30 {
		u := f.user("S", "STUDENT", "ACTIVE")
		_ = i
		f.enroll(f.course, u, "STUDENT", "ACTIVE")
	}
	f.event(f.course, "OTHER", "A", 5*time.Hour)
	f.exam(f.course, "Thi", "SCHEDULED", 6*time.Hour)
	var wg sync.WaitGroup
	var total, mu = 0, sync.Mutex{}
	for range 20 {
		wg.Go(func() {
			n, err := f.reminder().Tick(t.Context())
			require.NoError(t, err)
			mu.Lock()
			total += n
			mu.Unlock()
		})
	}
	wg.Wait()
	const want = 32 * 2
	require.Equal(t, want, total, "mỗi nhắc được đúng một worker tạo")
	require.Equal(t, want, count(f, `select count(*) from reminder_log`))
	require.Equal(t, want, count(f, `select count(*) from notifications where type = 'REMINDER'`))
	require.Equal(t, want, count(f, `select count(*) from mail_outbox where template = 'reminder'`))
}

// TestReminderRescheduledNotifiesAgain — AC11: đổi starts_at sau khi đã nhắc → nhắc lại đúng một lần với giờ mới.
func TestReminderRescheduledNotifiesAgain(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ev := f.event(f.course, "OTHER", "A", 5*time.Hour)
	require.Equal(t, 2, f.tick())
	_, err := f.pool.Exec(t.Context(), `update calendar_events set starts_at = starts_at + interval '2 hours' where id = $1`, ev)
	require.NoError(t, err)
	require.Equal(t, 2, f.tick())
	require.Zero(t, f.tick())
	require.Equal(t, 2, f.bells(f.sv))
	require.Equal(t, 4, count(f, `select count(*) from reminder_log`))
}

// TestReminderSkipsDeleted — AC11: nguồn bị xoá / bài thi bỏ lịch trước giờ nhắc → không nhắc.
func TestReminderSkipsDeleted(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ev := f.event(f.course, "OTHER", "A", 5*time.Hour)
	e := f.exam(f.course, "Thi", "SCHEDULED", 6*time.Hour)
	_, err := f.pool.Exec(t.Context(), `delete from calendar_events where id = $1`, ev)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `update exams set status = 'DRAFT', opens_at = null, closes_at = null, duration_minutes = null where id = $1`, e)
	require.NoError(t, err)
	require.Zero(t, f.tick())
	require.Zero(t, count(f, `select count(*) from reminder_log`))
}

// TestReminderRecipientsStudentsOnly — AC12: chỉ sinh viên ACTIVE của lớp; Staff, PENDING, REMOVED không nhận.
func TestReminderRecipientsStudentsOnly(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	pending, removed := f.user("P", "STUDENT", "ACTIVE"), f.user("R", "STUDENT", "ACTIVE")
	f.enroll(f.course, pending, "STUDENT", "PENDING")
	f.enroll(f.course, removed, "STUDENT", "REMOVED")
	f.event(f.course, "OTHER", "A", 5*time.Hour)
	require.Equal(t, 2, f.tick())
	for _, u := range []uuid.UUID{f.teacher, f.ta, pending, removed} {
		require.Zero(t, f.bells(u))
	}
	require.Equal(t, 1, f.bells(f.sv))
}

func (f *fx) prefs(u uuid.UUID, js string) {
	f.t.Helper()
	_, err := f.pool.Exec(f.t.Context(), `insert into user_settings (user_id, preferences) values ($1, $2::jsonb) on conflict (user_id) do update set preferences = excluded.preferences`, u, js)
	require.NoError(f.t, err)
}

// TestReminderPerTypeToggle — AC12: mặc định bài thi + sự kiện khác bật, buổi học tắt; tắt một loại → không chuông lẫn mail loại đó.
func TestReminderPerTypeToggle(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.session(f.course, 1, 3*time.Hour, "Buổi")
	f.exam(f.course, "Thi", "SCHEDULED", 4*time.Hour)
	f.event(f.course, "OTHER", "Khác", 5*time.Hour)
	f.event(f.course, "EXAM", "Thi thực hành", 6*time.Hour)
	f.prefs(f.sv2, `{"reminders":{"exam":false,"other":false,"class_session":true}}`)
	require.Equal(t, 3+1, f.tick()) // sv: exam + EXAM event + OTHER (3); sv2: buổi học (1)
	require.Equal(t, 3, f.bells(f.sv))
	require.Equal(t, 1, f.bells(f.sv2))
	require.Zero(t, count(f, `select count(*) from notifications where user_id = $1 and title like '%Thi%'`, f.sv2))
	require.Equal(t, 1, count(f, `select count(*) from mail_outbox where to_addr = (select email from users where id = $1)`, f.sv2))
	require.Zero(t, count(f, `select count(*) from notifications where user_id = $1 and title like '%Buổi 1%'`, f.sv))
}

// TestReminderMailFlag — AC12: mail chỉ khi remind_deadline_by_mail và email đã xác minh; chuông vẫn có khi tắt mail.
func TestReminderMailFlag(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	_, err := f.pool.Exec(t.Context(), `insert into user_settings (user_id, remind_deadline_by_mail) values ($1, false)`, f.sv)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `update users set email_verified_at = null where id = $1`, f.sv2)
	require.NoError(t, err)
	v := f.user("V", "STUDENT", "ACTIVE")
	f.enroll(f.course, v, "STUDENT", "ACTIVE")
	f.event(f.course, "OTHER", "A", 5*time.Hour)
	require.Equal(t, 3, f.tick())
	for _, u := range []uuid.UUID{f.sv, f.sv2, v} {
		require.Equal(t, 1, f.bells(u))
	}
	require.Equal(t, 1, count(f, `select count(*) from mail_outbox where template = 'reminder'`))
	require.Equal(t, 1, count(f, `select count(*) from mail_outbox where to_addr = (select email from users where id = $1)`, v))
}

// TestReminderArchivedCourse — AC12: lớp ARCHIVED → không nhắc.
func TestReminderArchivedCourse(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	arch := f.newCourse("ARCHIVED")
	f.enroll(arch, f.sv, "STUDENT", "ACTIVE")
	f.event(arch, "OTHER", "A", 5*time.Hour)
	require.Zero(t, f.tick())
	require.Zero(t, f.bells(f.sv))
}

// TestReminderDisabledUser — AC12: tài khoản DISABLED → không nhắc.
func TestReminderDisabledUser(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	_, err := f.pool.Exec(t.Context(), `update users set status = 'DISABLED' where id = $1`, f.sv)
	require.NoError(t, err)
	f.event(f.course, "OTHER", "A", 5*time.Hour)
	require.Equal(t, 1, f.tick())
	require.Zero(t, f.bells(f.sv))
	require.Equal(t, 1, f.bells(f.sv2))
}

// TestReminderMailDownKeepsBell — AC13: không có consumer mail (hệ thống mail hỏng) → chuông vẫn có, thư nằm hàng đợi để thử lại, chạy lại không tạo thêm.
func TestReminderMailDownKeepsBell(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.event(f.course, "OTHER", "A", 5*time.Hour)
	require.Equal(t, 2, f.tick())
	require.Equal(t, 1, f.bells(f.sv))
	require.Equal(t, 2, count(f, `select count(*) from mail_outbox where status = 'QUEUED'`))
	require.Zero(t, f.tick())
	require.Equal(t, 2, count(f, `select count(*) from mail_outbox`))
	require.Equal(t, 2, count(f, `select count(*) from reminder_log`))
}

// TestReminderBatches200 — AC13: 1.000 sinh viên × 1 sự kiện: đủ 1.000 reminder_log / chuông / thư, một lần; ≤ 60 s.
func TestReminderBatches200(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	_, err := f.pool.Exec(t.Context(), `with u as (insert into users (email, full_name, role, status, email_verified_at, password_hash) select 'bulk' || g || '-' || gen_random_uuid() || '@example.test', 'S' || g, 'STUDENT', 'ACTIVE', now(), 'x' from generate_series(1, 998) g returning id)
		insert into enrollments (course_id, user_id, role_in_course, status, joined_via) select $1, id, 'STUDENT', 'ACTIVE', 'ADMIN' from u`, f.course)
	require.NoError(t, err)
	f.event(f.course, "OTHER", "A", 5*time.Hour)
	start := time.Now()
	require.Equal(t, 1000, f.tick())
	if os.Getenv("EP_SKIP_TIMING") == "" {
		require.Less(t, time.Since(start), 60*time.Second)
	}
	require.Equal(t, 1000, count(f, `select count(*) from reminder_log`))
	require.Equal(t, 1000, count(f, `select count(*) from notifications where type = 'REMINDER'`))
	require.Equal(t, 1000, count(f, `select count(*) from mail_outbox where template = 'reminder'`))
	require.Zero(t, f.tick())
}

// TestReminderMailContentMinimal — AC13: payload chỉ có tên sự kiện và giờ; thư có đúng một liên kết APP_PUBLIC_URL/calendar, không điểm, không tên người.
func TestReminderMailContentMinimal(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.event(f.course, "OTHER", "Nộp báo cáo", 5*time.Hour)
	f.tick()
	var payload []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `select payload from mail_outbox where template = 'reminder' limit 1`).Scan(&payload))
	var m map[string]any
	require.NoError(t, json.Unmarshal(payload, &m))
	require.Len(t, m, 2)
	require.Equal(t, "Nộp báo cáo", m["event_title"])
	r, err := mail.Render("reminder", map[string]string{"EventTitle": "Nộp báo cáo", "At": "14:00 21/09/2026", "CalendarURL": "http://app.test/calendar"})
	require.NoError(t, err)
	require.Contains(t, r.Text, "Nộp báo cáo bắt đầu lúc 14:00 21/09/2026")
	require.Equal(t, 1, countOf(r.Text, "http://app.test/calendar"))
	require.Equal(t, 1, countOf(r.Text, "http"))
	require.NotContains(t, r.Subject, "Nộp báo cáo")
}

func countOf(s, sub string) int {
	n := 0
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			n++
		}
	}
	return n
}
