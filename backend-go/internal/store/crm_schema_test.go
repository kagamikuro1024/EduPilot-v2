package store_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// crmFx: hai lớp; lớp 1 có giảng viên, hai sinh viên ACTIVE (s1, s2) và hai buổi; lớp 2 có sinh viên s1 và một buổi; s3 chưa vào lớp nào.
type crmFx struct {
	chatFx
	s3                 uuid.UUID
	sess1, sess2, sesC uuid.UUID // buổi của lớp 1 (hai buổi) và lớp 2
}

func newCrmFx(t *testing.T, ctx context.Context, conn *pgx.Conn) crmFx {
	t.Helper()
	f := crmFx{chatFx: newChatFx(t, ctx, conn)}
	one := func(dst *uuid.UUID, sql string, args ...any) {
		require.NoError(t, conn.QueryRow(ctx, sql, args...).Scan(dst), sql)
	}
	one(&f.s3, `insert into users (email, full_name, role) values ($1, 'SV3', 'STUDENT') returning id`, "s3."+strings.ToLower(uuid.NewString()[:6])+"@example.test")
	enr := `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, $3, 'ACTIVE', 'ADMIN')`
	for _, e := range []struct {
		c, u uuid.UUID
		r    string
	}{{f.c1, f.teacher, "TEACHER"}, {f.c1, f.s1, "STUDENT"}, {f.c1, f.s2, "STUDENT"}, {f.c2, f.s1, "STUDENT"}} {
		_, err := conn.Exec(ctx, enr, e.c, e.u, e.r)
		require.NoError(t, err)
	}
	ses := `insert into class_sessions (course_id, session_no, starts_at, ends_at) values ($1, $2::int, now() + $2::int * interval '1 day', now() + $2::int * interval '1 day' + interval '2 hours') returning id`
	one(&f.sess1, ses, f.c1, 1)
	one(&f.sess2, ses, f.c1, 2)
	one(&f.sesC, ses, f.c2, 1)
	return f
}

func (f crmFx) att(extra string) string {
	return fmt.Sprintf(`insert into attendance_records (course_id, session_id, student_id, status, marked_by %s`, extra)
}

// TestSchemaAttendance — US-P5-01 AC2: đúng cột / enum / chỉ mục của SRS 5.2; UNIQUE (course_id, session_id, student_id); CHECK → 23514.
func TestSchemaAttendance(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	require.Equal(t, []string{"id", "course_id", "session_id", "student_id", "status", "note", "marked_by", "version", "created_at", "updated_at"}, tableCols(t, ctx, conn, "attendance_records"))
	require.Equal(t, []string{"PRESENT", "LATE", "EXCUSED", "ABSENT"}, enumLabels(t, ctx, conn, "attendance_status"))
	require.Equal(t, []string{"id", "course_id", "session_id", "student_id", "status", "note", "marked_by", "version", "created_at", "updated_at"}, tableCols(t, ctx, conn, "attendance_records"))

	f := newCrmFx(t, ctx, conn)
	ok := f.att(fmt.Sprintf(`) values ('%s', '%s', '%s', 'PRESENT', '%s')`, f.c1, f.sess1, f.s1, f.teacher))
	_, err := conn.Exec(ctx, ok)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, ok)
	require.Equal(t, "23505", sqlState(err), "mỗi (buổi, sinh viên) một dòng")
	const check = "23514"
	base := fmt.Sprintf(`'%s', '%s', '%s'`, f.c1, f.sess2, f.s2)
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"status ngoài enum":    {"22P02", f.att(fmt.Sprintf(`) values (%s, 'LATE_ISH', '%s')`, base, f.teacher))},
		"note > 200":           {check, f.att(fmt.Sprintf(`, note) values (%s, 'ABSENT', '%s', repeat('x', 201))`, base, f.teacher))},
		"version ≥ 1":          {check, f.att(fmt.Sprintf(`, version) values (%s, 'ABSENT', '%s', 0)`, base, f.teacher))},
		"thiếu người điểm":     {"23502", f.att(fmt.Sprintf(`) values (%s, 'ABSENT', NULL)`, base))},
		"người điểm lạ":        {"23503", f.att(fmt.Sprintf(`) values (%s, 'ABSENT', '%s')`, base, uuid.New()))},
		"note đúng 200 hợp lệ": {"", f.att(fmt.Sprintf(`, note) values (%s, 'EXCUSED', '%s', repeat('x', 200))`, base, f.teacher))},
	})
	var idx int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_indexes where tablename = 'attendance_records'
		and indexname in ('attendance_records_pkey', 'attendance_records_session_student_key', 'attendance_course_student_idx')`).Scan(&idx))
	require.Equal(t, 3, idx, "PK + UNIQUE(course_id, session_id, student_id) + (course_id, student_id, session_id); không có chỉ mục (course_id, session_id) riêng (TLR-3)")
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_indexes where tablename = 'attendance_records' and indexname = 'attendance_course_session_idx'`).Scan(&idx))
	require.Zero(t, idx)
}

func enumLabels(t *testing.T, ctx context.Context, conn *pgx.Conn, enum string) []string {
	t.Helper()
	rows, err := conn.Query(ctx, `select e.enumlabel from pg_enum e join pg_type t on t.oid = e.enumtypid where t.typname = $1 order by e.enumsortorder`, enum)
	require.NoError(t, err)
	defer rows.Close()
	var got []string
	for rows.Next() {
		var l string
		require.NoError(t, rows.Scan(&l))
		got = append(got, l)
	}
	require.NoError(t, rows.Err())
	return got
}

// TestAttendanceCrossCourseFK — AC2: DB chặn trộn lớp (buổi của lớp khác, sinh viên ngoài lớp, sinh viên của lớp khác) → 23503.
func TestAttendanceCrossCourseFK(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newCrmFx(t, ctx, conn)
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"buổi của lớp khác":                    {"23503", f.att(fmt.Sprintf(`) values ('%s', '%s', '%s', 'PRESENT', '%s')`, f.c1, f.sesC, f.s1, f.teacher))},
		"sinh viên ngoài lớp (chưa ghi danh)":  {"23503", f.att(fmt.Sprintf(`) values ('%s', '%s', '%s', 'PRESENT', '%s')`, f.c1, f.sess1, f.s3, f.teacher))},
		"sinh viên chỉ ở lớp 1, ghi vào lớp 2": {"23503", f.att(fmt.Sprintf(`) values ('%s', '%s', '%s', 'PRESENT', '%s')`, f.c2, f.sesC, f.s2, f.teacher))},
		"đúng lớp": {"", f.att(fmt.Sprintf(`) values ('%s', '%s', '%s', 'PRESENT', '%s')`, f.c2, f.sesC, f.s1, f.teacher))},
	})
	// buổi đã có điểm danh không xoá được (khoá ngoại là lưới an toàn của DELETE …/sessions/{sid})
	_, err := conn.Exec(ctx, fmt.Sprintf(`delete from class_sessions where id = '%s'`, f.sesC))
	require.Equal(t, "23503", sqlState(err))
}

func (f crmFx) part(cols, vals string) string {
	return fmt.Sprintf(`insert into participation_events (course_id, student_id, created_by, client_event_id, %s) values ('%s', '%s', '%s', '%s', %s)`, cols, f.c1, f.s1, f.teacher, uuid.New(), vals)
}

// TestSchemaParticipation — AC2: cột của SRS 5.3, enum, client_event_id UNIQUE toàn cục, chỉ mục.
func TestSchemaParticipation(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	require.Equal(t, []string{"id", "course_id", "student_id", "session_id", "type", "points", "note", "client_event_id", "created_by", "created_at", "updated_at"}, tableCols(t, ctx, conn, "participation_events"))
	require.Equal(t, []string{"SPEAK", "BONUS", "PENALTY"}, enumLabels(t, ctx, conn, "participation_type"))
	f := newCrmFx(t, ctx, conn)
	ev := uuid.New()
	q := fmt.Sprintf(`insert into participation_events (course_id, student_id, created_by, client_event_id, type, points) values ('%s', '%s', '%s', '%s', 'SPEAK', 0.25)`, f.c1, f.s1, f.teacher, ev)
	_, err := conn.Exec(ctx, q)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, strings.Replace(q, f.c1.String(), f.c2.String(), 1))
	require.Equal(t, "23505", sqlState(err), "client_event_id duy nhất toàn cục")
	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_indexes where tablename = 'participation_events' and indexname in ('participation_course_student_idx', 'participation_course_session_idx', 'participation_events_client_event_key')`).Scan(&n))
	require.Equal(t, 3, n)
}

// TestParticipationChecks — AC2: dấu của points theo type, miền [-5, 5], lệch lớp → 23503.
func TestParticipationChecks(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newCrmFx(t, ctx, conn)
	const check = "23514"
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"SPEAK âm":             {check, f.part("type, points", "'SPEAK', -0.25")},
		"BONUS = 0":            {check, f.part("type, points", "'BONUS', 0")},
		"BONUS âm":             {check, f.part("type, points", "'BONUS', -1")},
		"PENALTY dương":        {check, f.part("type, points", "'PENALTY', 1")},
		"PENALTY = 0":          {check, f.part("type, points", "'PENALTY', 0")},
		"trên 5":               {check, f.part("type, points", "'BONUS', 5.01")},
		"dưới -5":              {check, f.part("type, points", "'PENALTY', -5.01")},
		"note > 200":           {check, f.part("type, points, note", "'SPEAK', 0.25, repeat('x', 201)")},
		"SPEAK = 0 hợp lệ":     {"", f.part("type, points", "'SPEAK', 0")},
		"BONUS = 5 hợp lệ":     {"", f.part("type, points", "'BONUS', 5")},
		"PENALTY = -5 hợp lệ":  {"", f.part("type, points", "'PENALTY', -5")},
		"buổi của lớp khác":    {"23503", f.part("type, points, session_id", fmt.Sprintf("'SPEAK', 0.25, '%s'", f.sesC))},
		"buổi đúng lớp hợp lệ": {"", f.part("type, points, session_id", fmt.Sprintf("'SPEAK', 0.25, '%s'", f.sess1))},
		"sinh viên ngoài lớp":  {"23503", strings.Replace(f.part("type, points", "'SPEAK', 0.25"), f.s1.String(), f.s3.String(), 1)},
	})
}

// TestSchemaObservations — AC3: cột của SRS 5.4, ràng buộc tags / body / kind ⇔ ai_edited, chỉ mục bắt đầu bằng course_id.
func TestSchemaObservations(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	require.Equal(t, []string{"id", "course_id", "student_id", "author_id", "kind", "tags", "body", "ai_edited", "version", "created_at", "updated_at"}, tableCols(t, ctx, conn, "student_observations"))
	require.Equal(t, []string{"NOTE", "AI_SUMMARY"}, enumLabels(t, ctx, conn, "observation_kind"))
	f := newCrmFx(t, ctx, conn)
	obs := func(cols, vals string) string {
		return fmt.Sprintf(`insert into student_observations (course_id, student_id, author_id, %s) values ('%s', '%s', '%s', %s)`, cols, f.c1, f.s1, f.teacher, vals)
	}
	const check = "23514"
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"body rỗng":           {check, obs("body", "''")},
		"body > 2000":         {check, obs("body", "repeat('x', 2001)")},
		"body = 2000 hợp lệ":  {"", obs("body", "repeat('x', 2000)")},
		"6 thẻ":               {check, obs("body, tags", "'a', '{a,b,c,d,e,f}'")},
		"thẻ > 30 ký tự":      {check, obs("body, tags", "'a', ARRAY[repeat('x', 31)]")},
		"NOTE mà ai_edited":   {check, obs("body, ai_edited", "'a', true")},
		"AI_SUMMARY đã sửa":   {"", obs("body, kind, ai_edited", "'a', 'AI_SUMMARY', true")},
		"version 0":           {check, obs("body, version", "'a', 0")},
		"sinh viên ngoài lớp": {"23503", strings.Replace(obs("body", "'a'"), f.s1.String(), f.s3.String(), 1)},
		"tác giả lạ":          {"23503", strings.Replace(obs("body", "'a'"), f.teacher.String(), uuid.NewString(), 1)},
	})
}

// TestSchemaStudentRisk — AC3: cột của SRS 5.5, PK (course_id, student_id), level AT_RISK ⇔ entered_at, chỉ mục.
func TestSchemaStudentRisk(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	require.Equal(t, []string{"course_id", "student_id", "level", "reasons", "weeks", "last_week_asks", "last_week_posts", "entered_at", "computed_at", "created_at", "updated_at"}, tableCols(t, ctx, conn, "student_risk"))
	require.Equal(t, []string{"NONE", "WATCH", "AT_RISK"}, enumLabels(t, ctx, conn, "risk_level"))
	f := newCrmFx(t, ctx, conn)
	risk := func(cols, vals string) string {
		return fmt.Sprintf(`insert into student_risk (course_id, student_id, computed_at, %s) values ('%s', '%s', now(), %s)`, cols, f.c1, f.s1, vals)
	}
	const check = "23514"
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"AT_RISK thiếu entered_at": {check, risk("level", "'AT_RISK'")},
		"WATCH có entered_at":      {check, risk("level, entered_at", "'WATCH', now()")},
		"reasons không phải mảng":  {check, risk("reasons", "'{}'")},
		"weeks không phải mảng":    {check, risk("weeks", "'{}'")},
		"đếm âm":                   {check, risk("last_week_asks", "-1")},
		"sinh viên ngoài lớp":      {"23503", strings.Replace(risk("level", "'NONE'"), f.s1.String(), f.s3.String(), 1)},
		"AT_RISK hợp lệ":           {"", risk("level, entered_at", "'AT_RISK', now()")},
	})
	_, err := conn.Exec(ctx, risk("level", "'NONE'"))
	require.Equal(t, "23505", sqlState(err), "một dòng cho mỗi (lớp, sinh viên)")
	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_indexes where tablename = 'student_risk' and indexname in ('student_risk_level_idx', 'student_risk_entered_idx')`).Scan(&n))
	require.Equal(t, 2, n)
}

// TestSchemaClassSessionsCrm — AC1, AC3: cột và khoá thêm ở class_sessions (UNIQUE (course_id, id), attendance_done_at/by đi đôi).
func TestSchemaClassSessionsCrm(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	require.Equal(t, []string{"id", "course_id", "session_no", "starts_at", "ends_at", "room", "topic", "version", "created_at", "updated_at", "attendance_done_at", "attendance_done_by"}, tableCols(t, ctx, conn, "class_sessions"))
	f := newCrmFx(t, ctx, conn)
	_, err := conn.Exec(ctx, fmt.Sprintf(`update class_sessions set attendance_done_at = now() where id = '%s'`, f.sess1))
	require.Equal(t, "23514", sqlState(err), "done_at và done_by đi đôi")
	_, err = conn.Exec(ctx, fmt.Sprintf(`update class_sessions set attendance_done_at = now(), attendance_done_by = '%s' where id = '%s'`, f.teacher, f.sess1))
	require.NoError(t, err)
	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_constraint where conname = 'class_sessions_course_id_key' and contype = 'u'`).Scan(&n))
	require.Equal(t, 1, n)
}

// TestViewsColumns — AC3: hai view có đúng danh sách cột ghim ở SRS 5.6; chỉ sinh viên ACTIVE; số liệu đúng.
func TestViewsColumns(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	require.Equal(t, []string{"course_id", "student_id", "recorded", "present", "late", "excused", "absent", "last_marked_at"}, tableCols(t, ctx, conn, "v_attendance_summary"))
	require.Equal(t, []string{"course_id", "student_id", "full_name", "student_code", "joined_at", "recorded", "present", "late", "excused", "absent", "speak_count", "points_total",
		"study_seconds_7d", "last_week_asks", "last_week_posts", "risk_level", "risk_entered_at", "risk_computed_at"}, tableCols(t, ctx, conn, "v_student_360"))

	f := newCrmFx(t, ctx, conn)
	for _, r := range []struct {
		sess uuid.UUID
		st   string
	}{{f.sess1, "PRESENT"}, {f.sess2, "ABSENT"}} {
		_, err := conn.Exec(ctx, f.att(fmt.Sprintf(`) values ('%s', '%s', '%s', '%s', '%s')`, f.c1, r.sess, f.s1, r.st, f.teacher)))
		require.NoError(t, err)
	}
	for _, q := range []string{f.part("type, points", "'SPEAK', 0.25"), f.part("type, points", "'SPEAK', 0.25"), f.part("type, points", "'PENALTY', -1")} {
		_, err := conn.Exec(ctx, q)
		require.NoError(t, err)
	}
	_, err := conn.Exec(ctx, fmt.Sprintf(`insert into student_risk (course_id, student_id, level, entered_at, computed_at, last_week_asks, last_week_posts) values ('%s', '%s', 'AT_RISK', now(), now(), 3, 1)`, f.c1, f.s1))
	require.NoError(t, err)

	var recorded, present, absent, speak, asks, posts int
	var points, risk string
	require.NoError(t, conn.QueryRow(ctx, `select recorded, present, absent, speak_count, points_total::text, last_week_asks, last_week_posts, risk_level::text from v_student_360 where course_id = $1 and student_id = $2`, f.c1, f.s1).
		Scan(&recorded, &present, &absent, &speak, &points, &asks, &posts, &risk))
	require.Equal(t, []any{2, 1, 1, 2, "-0.50", 3, 1, "AT_RISK"}, []any{recorded, present, absent, speak, points, asks, posts, risk})
	// sinh viên chưa có dữ liệu: 0 và NONE; giảng viên không nằm trong view; sinh viên lớp khác không lẫn vào
	require.NoError(t, conn.QueryRow(ctx, `select recorded, speak_count, points_total::text, risk_level::text from v_student_360 where course_id = $1 and student_id = $2`, f.c1, f.s2).Scan(&recorded, &speak, &points, &risk))
	require.Equal(t, []any{0, 0, "0.00", "NONE"}, []any{recorded, speak, points, risk})
	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from v_student_360 where course_id = $1`, f.c1).Scan(&n))
	require.Equal(t, 2, n)
	// sinh viên bị mời ra không còn trong v_student_360 nhưng dòng điểm danh vẫn còn trong v_attendance_summary (F2: giữ nguyên dòng)
	_, err = conn.Exec(ctx, `update enrollments set status = 'REMOVED', removed_at = now() where course_id = $1 and user_id = $2`, f.c1, f.s1)
	require.NoError(t, err)
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from v_student_360 where course_id = $1 and student_id = $2`, f.c1, f.s1).Scan(&n))
	require.Zero(t, n)
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from v_attendance_summary where course_id = $1 and student_id = $2`, f.c1, f.s1).Scan(&n))
	require.Equal(t, 1, n)
}

// TestIndexesStartWithCourseID — AC3, luật 13: mọi chỉ mục PHỨC HỢP trên bảng mới có course_id đứng đầu (UNIQUE client_event_id một cột không thuộc phạm vi).
func TestIndexesStartWithCourseID(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	rows, err := conn.Query(ctx, `
		select c.relname, i.relname, a.attname
		  from pg_index x
		  join pg_class c on c.oid = x.indrelid
		  join pg_class i on i.oid = x.indexrelid
		  join pg_attribute a on a.attrelid = c.oid and a.attnum = x.indkey[0]
		 where c.relnamespace = 'public'::regnamespace
		   and c.relname in ('attendance_records', 'participation_events', 'student_observations', 'student_risk')
		   and x.indnatts > 1`)
	require.NoError(t, err)
	defer rows.Close()
	seen := 0
	for rows.Next() {
		var table, index, first string
		require.NoError(t, rows.Scan(&table, &index, &first))
		require.Equal(t, "course_id", first, "%s.%s", table, index)
		seen++
	}
	require.NoError(t, rows.Err())
	require.GreaterOrEqual(t, seen, 8, "đã kiểm đủ các chỉ mục phức hợp của bốn bảng")
}

// TestMigration00013SingleAlter — AC1: tệp chỉ ALTER đúng một bảng cũ (class_sessions); mọi thứ khác là CREATE.
func TestMigration00013SingleAlter(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("../../db/migrations/00013_crm.sql")
	require.NoError(t, err)
	var alters []string
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(ln, "ALTER TABLE") {
			alters = append(alters, ln)
		}
	}
	require.Equal(t, []string{"ALTER TABLE class_sessions"}, alters)
}
