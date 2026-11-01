package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

func seedEnv(url, appEnv string) func(string) string {
	return adminEnv(url, map[string]string{"APP_ENV": appEnv})
}

type seedWorld struct {
	pool          *pgxpool.Pool
	url           string
	course        uuid.UUID
	teacher, stu1 uuid.UUID
}

func newSeedWorld(t *testing.T) seedWorld {
	t.Helper()
	url := testutil.MigratedPostgresURL(t)
	pool, err := pgxpool.New(t.Context(), url)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	hash, err := auth.HashPassword("Mat-khau-thu-2026", 4)
	require.NoError(t, err)
	q := store.New(pool)
	mk := func(email string, role store.UserRole) uuid.UUID {
		u, err := q.InsertUser(t.Context(), store.InsertUserParams{Email: email, FullName: email, Role: role, Status: store.UserStatusACTIVE, PasswordHash: &hash})
		require.NoError(t, err)
		return u.ID
	}
	w := seedWorld{pool: pool, url: url, teacher: mk("teacher@edupilot.local", store.UserRoleTEACHER), stu1: mk("sv.gioi@edupilot.local", store.UserRoleSTUDENT)}
	require.NoError(t, pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', '761987', 'An ninh mạng', '2026-2027-HK1', 'AN7K2MQ', $1) returning id`, w.teacher).Scan(&w.course))
	for u, role := range map[uuid.UUID]string{w.teacher: "TEACHER", w.stu1: "STUDENT"} {
		_, err := pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, $3::enrollment_role, 'ACTIVE', 'ADMIN')`, w.course, u, role)
		require.NoError(t, err)
	}
	return w
}

func runSeed(t *testing.T, env func(string) string, args ...string) (int, string, string) {
	t.Helper()
	return runSeedIn(t, env, "", args...)
}

func runSeedIn(t *testing.T, env func(string) string, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var o, e bytes.Buffer
	code := runAdmin(args, env, strings.NewReader(stdin), &o, &e)
	return code, o.String(), e.String()
}

// AC2: lệnh chỉ chạy với APP_ENV thuộc danh sách cho phép; production, staging, rỗng bị từ chối (mã ≠ 0) và KHÔNG chạm DB.
func TestSeedCmdEnvAllowlist(t *testing.T) {
	w := newSeedWorld(t)
	prof := filepath.Join(t.TempDir(), "p.json")
	require.NoError(t, os.WriteFile(prof, []byte(`{"default":{"minutes_per_week":60}}`), 0o600))
	cmds := [][]string{
		{"seed-study-time", "--course", "761987", "--weeks", "2", "--profile", prof},
		{"seed-age", "--course", "761987", "--activity"},
		{"crm-risk-run", "--course", "761987"},
	}
	for _, env := range []string{"production", "staging", "", "Production", "prod", "local", "xyz"} {
		for _, c := range cmds {
			code, out, errOut := runSeed(t, seedEnv(w.url, env), c...)
			require.NotZero(t, code, "%s với APP_ENV=%q", c[0], env)
			require.Empty(t, out)
			require.Contains(t, errOut, "APP_ENV")
		}
	}
	var n int
	require.NoError(t, w.pool.QueryRow(t.Context(), `select count(*) from study_time_daily`).Scan(&n))
	require.Zero(t, n, "môi trường bị từ chối không được ghi gì")
	for _, env := range []string{"test", "dev"} {
		code, out, errOut := runSeed(t, seedEnv(w.url, env), cmds[0]...)
		require.Zero(t, code, errOut)
		require.Regexp(t, `seed-study-time: \d+ dòng`, out)
	}
}

// AC2: số dòng in ra; chạy lại in 0 và không nhân đôi.
func TestSeedStudyTimeCmdIdempotent(t *testing.T) {
	w := newSeedWorld(t)
	prof := filepath.Join(t.TempDir(), "p.json")
	require.NoError(t, os.WriteFile(prof, []byte(`{"default":{"minutes_per_week":90},"students":{"sv.gioi@edupilot.local":{"minutes_per_week":300,"weeks":{"1":30}}}}`), 0o600))
	args := []string{"seed-study-time", "--course", "761987", "--weeks", "3", "--profile", prof, "--now", "2026-10-29T10:00:00+07:00"}
	code, out, errOut := runSeed(t, seedEnv(w.url, "test"), args...)
	require.Zero(t, code, errOut)
	require.NotContains(t, out, "seed-study-time: 0 dòng")
	var rows, secs int
	require.NoError(t, w.pool.QueryRow(t.Context(), `select count(*), sum(active_seconds) from study_time_daily where course_id = $1`, w.course).Scan(&rows, &secs))
	require.Positive(t, rows)
	code, out, errOut = runSeed(t, seedEnv(w.url, "test"), args...)
	require.Zero(t, code, errOut)
	require.Contains(t, out, "seed-study-time: 0 dòng")
	var rows2, secs2 int
	require.NoError(t, w.pool.QueryRow(t.Context(), `select count(*), sum(active_seconds) from study_time_daily where course_id = $1`, w.course).Scan(&rows2, &secs2))
	require.Equal(t, [2]int{rows, secs}, [2]int{rows2, secs2})
	code, _, _ = runSeed(t, seedEnv(w.url, "test"), "seed-study-time", "--course", "000000", "--profile", prof)
	require.NotZero(t, code, "lớp không có")
	code, _, _ = runSeed(t, seedEnv(w.url, "test"), "seed-study-time", "--course", "761987", "--profile", prof, "--bogus")
	require.Equal(t, 2, code, "cờ lạ")
}

// AC4: seed-age lùi created_at VÀ waiting_since (và các mốc có giá trị) của đúng một ticket; không đổi trạng thái / nội dung.
func TestSeedAgeTicketsMovesWaitingSince(t *testing.T) {
	w := newSeedWorld(t)
	var other, tid uuid.UUID
	for _, p := range []*uuid.UUID{&other, &tid} {
		require.NoError(t, w.pool.QueryRow(t.Context(), `insert into escalation_tickets (course_id, kind, origin, source_type, source_id, student_id, status, answer)
			values ($1, 'TEACHER_OUTREACH', 'STAFF_OUTREACH', 'OUTREACH', gen_random_uuid(), $2, 'OPEN', 'nội dung giữ nguyên') returning id`, w.course, w.stu1).Scan(p))
	}
	before := func(id uuid.UUID) (created, waiting time.Time, status, answer string) {
		require.NoError(t, w.pool.QueryRow(t.Context(), `select created_at, waiting_since, status::text, answer from escalation_tickets where id = $1`, id).Scan(&created, &waiting, &status, &answer))
		return
	}
	c0, w0, _, _ := before(tid)
	oc0, ow0, _, _ := before(other)
	code, out, errOut := runSeed(t, seedEnv(w.url, "dev"), "seed-age", "--course", "761987", "--tickets", "--ticket", tid.String(), "--hours", "100")
	require.Zero(t, code, errOut)
	require.Contains(t, out, "seed-age: 1 dòng")
	c1, w1, st, ans := before(tid)
	require.WithinDuration(t, c0.Add(-100*time.Hour), c1, time.Second)
	require.WithinDuration(t, w0.Add(-100*time.Hour), w1, time.Second, "waiting_since cũng phải lùi")
	require.Equal(t, "OPEN", st)
	require.Equal(t, "nội dung giữ nguyên", ans)
	oc1, ow1, _, _ := before(other)
	require.True(t, oc0.Equal(oc1) && ow0.Equal(ow1), "ticket khác không bị chạm")
	var overdue int
	require.NoError(t, w.pool.QueryRow(t.Context(), `select count(*) from escalation_tickets where status = 'OPEN' and waiting_since < now() - interval '80 hours'`).Scan(&overdue))
	require.Equal(t, 1, overdue)
	code, _, _ = runSeed(t, seedEnv(w.url, "dev"), "seed-age", "--course", "761987", "--tickets")
	require.Equal(t, 2, code, "thiếu --ticket")
	code, _, _ = runSeed(t, seedEnv(w.url, "dev"), "seed-age", "--course", "761987")
	require.Equal(t, 2, code, "thiếu --tickets / --activity")
}

// AC4: --activity chuyển hoạt động của tuần hiện tại về các tuần trước; chạy lại không dịch thêm; không đổi nội dung.
func TestSeedAgeActivityIdempotent(t *testing.T) {
	w := newSeedWorld(t)
	var sess uuid.UUID
	require.NoError(t, w.pool.QueryRow(t.Context(), `insert into chat_sessions (course_id, user_id) values ($1, $2) returning id`, w.course, w.stu1).Scan(&sess))
	for i := range 4 {
		_, err := w.pool.Exec(t.Context(), `insert into chat_messages (course_id, session_id, user_id, role, content) values ($1, $2, $3, 'USER', $4)`, w.course, sess, w.stu1, "câu hỏi "+string(rune('a'+i)))
		require.NoError(t, err)
	}
	now := time.Now()
	args := []string{"seed-age", "--course", "761987", "--activity", "--weeks", "10", "--now", now.Format(time.RFC3339)}
	code, out, errOut := runSeed(t, seedEnv(w.url, "test"), args...)
	require.Zero(t, code, errOut)
	require.Contains(t, out, "seed-age: 4 dòng")
	var inCurrentWeek int
	require.NoError(t, w.pool.QueryRow(t.Context(), `select count(*) from chat_messages where course_id = $1 and created_at >= date_trunc('week', now() at time zone 'Asia/Ho_Chi_Minh')`, w.course).Scan(&inCurrentWeek))
	require.Zero(t, inCurrentWeek, "mọi tin đã rời tuần hiện tại")
	var distinctWeeks int
	require.NoError(t, w.pool.QueryRow(t.Context(), `select count(distinct date_trunc('week', created_at)) from chat_messages where course_id = $1`, w.course).Scan(&distinctWeeks))
	require.GreaterOrEqual(t, distinctWeeks, 2, "rải qua nhiều tuần")
	code, out, errOut = runSeed(t, seedEnv(w.url, "test"), args...)
	require.Zero(t, code, errOut)
	require.Contains(t, out, "seed-age: 0 dòng")
	var content string
	require.NoError(t, w.pool.QueryRow(t.Context(), `select string_agg(content, ',' order by content) from chat_messages where course_id = $1`, w.course).Scan(&content))
	require.Equal(t, "câu hỏi a,câu hỏi b,câu hỏi c,câu hỏi d", content)
}

// Hồ sơ đọc từ stdin (--profile -): gateway chạy trong container không thấy thư mục seed/.
func TestSeedStudyTimeProfileFromStdin(t *testing.T) {
	w := newSeedWorld(t)
	code, out, errOut := runSeedIn(t, seedEnv(w.url, "test"), "{\"default\":{\"minutes_per_week\":60}}", "seed-study-time", "--course", "761987", "--weeks", "2", "--profile", "-")
	require.Zero(t, code, errOut)
	require.Regexp(t, `seed-study-time: [1-9]\d* dòng`, out)
	code, _, errOut = runSeedIn(t, seedEnv(w.url, "test"), "{\"default\":{\"minutes\":60}}", "seed-study-time", "--course", "761987", "--profile", "-")
	require.NotZero(t, code, "khoá lạ trong hồ sơ bị từ chối")
	require.Contains(t, errOut, "study_profile")
}

// TestSeedAgeAssignment — US-S9-SEED: `seed-age --assignment BT03 --hours 96` lùi hạn + bài nộp + thư NGUYÊN khoảng cách tương đối; chạy lại không lùi thêm; bài khác không bị chạm; APP_ENV production bị từ chối.
func TestSeedAgeAssignment(t *testing.T) {
	testutil.RequireContainers(t)
	w := newSeedWorld(t)
	ctx := t.Context()
	asg := func(code string) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, w.pool.QueryRow(ctx, `insert into assignments (course_id, title, max_score, due_at, submission_code, created_by, published_at, rubric)
			values ($1, $2, 10, now() + interval '1 minute', $2, $3, now(), '[{"id":"c1","name":"x","max_score":"10","guidance":""}]'::jsonb) returning id`, w.course, code, w.teacher).Scan(&id))
		return id
	}
	a3, a4 := asg("BT03"), asg("BT04")
	sub := func(a uuid.UUID, after string) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, w.pool.QueryRow(ctx, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at, received_at)
			values ($1, $2, $3, 'IN_APP', 1, 'k', 'a.txt', 1, 'text/plain', 'x', 'MATCHED', 'IN_APP', now() + $4::interval, now() + $4::interval) returning id`, w.course, a, w.stu1, after).Scan(&id))
		return id
	}
	sub(a3, "10 seconds")
	s4 := sub(a4, "0 seconds")
	_, err := w.pool.Exec(ctx, `insert into mail_inbound (course_id, uidvalidity, message_uid, from_email, subject, received_at, status) values ($1, 1, 1, 'a@x.vn', 's', now(), 'IGNORED')`, w.course)
	require.NoError(t, err)
	gap := func() (time.Duration, time.Duration) { // (due → bài nộp BT03, hạn BT03 → bây giờ)
		var due, rcv time.Time
		require.NoError(t, w.pool.QueryRow(ctx, `select a.due_at, s.received_at from assignments a join submissions s on s.assignment_id = a.id where a.id = $1`, a3).Scan(&due, &rcv))
		return rcv.Sub(due), time.Since(due)
	}
	before, _ := gap()
	var other time.Time
	require.NoError(t, w.pool.QueryRow(ctx, `select received_at from submissions where id = $1`, s4).Scan(&other))
	code, out, _ := runSeed(t, seedEnv(w.url, "dev"), "seed-age", "--course", "761987", "--assignment", "BT03", "--hours", "96")
	require.Equal(t, 0, code)
	require.Contains(t, out, "seed-age: 3 dòng") // bài tập + 1 bài nộp BT03 + 1 thư; bài nộp BT04 không bị chạm
	after, age := gap()
	require.Equal(t, before, after, "khoảng cách tương đối giữ nguyên")
	require.Greater(t, age, 95*time.Hour)
	var other2 time.Time
	require.NoError(t, w.pool.QueryRow(ctx, `select received_at from submissions where id = $1`, s4).Scan(&other2))
	require.True(t, other.Equal(other2), "bài nộp của bài khác không bị chạm")
	code, out, _ = runSeed(t, seedEnv(w.url, "dev"), "seed-age", "--course", "761987", "--assignment", "BT03", "--hours", "96")
	require.Equal(t, 0, code)
	require.Contains(t, out, "seed-age: 0 dòng", "chạy lại không lùi thêm")
	code, _, _ = runSeed(t, seedEnv(w.url, "production"), "seed-age", "--course", "761987", "--assignment", "BT03")
	require.Equal(t, 1, code)
	code, _, _ = runSeed(t, seedEnv(w.url, "dev"), "seed-age", "--course", "761987", "--assignment", "BT03", "--activity")
	require.Equal(t, 2, code, "hai chế độ một lúc")
}
