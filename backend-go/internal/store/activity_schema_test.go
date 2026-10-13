package store_test

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/db"
)

// US-P5-03 AC1 (SRS FEAT-crm-360 5.7–5.8): 00015_observation.

var view360Cols = []string{"course_id", "student_id", "full_name", "student_code", "joined_at", "recorded", "present", "late", "excused", "absent", "speak_count", "points_total",
	"study_seconds_7d", "last_week_asks", "last_week_posts", "risk_level", "risk_entered_at", "risk_computed_at"}

func TestSchemaStudyTimeDaily(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	require.Equal(t, []string{"id", "course_id", "student_id", "date", "area", "active_seconds", "created_at", "updated_at"}, tableCols(t, ctx, conn, "study_time_daily"))
	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_indexes where tablename = 'study_time_daily' and indexname in ('study_time_daily_pkey', 'study_time_daily_key', 'study_time_course_date_idx', 'study_time_student_idx')`).Scan(&n))
	require.Equal(t, 4, n)
	f := newCrmFx(t, ctx, conn)
	ins := func(course, student any, area string, secs int) error {
		_, err := conn.Exec(ctx, `insert into study_time_daily (course_id, student_id, date, area, active_seconds) values ($1, $2, current_date, $3, $4)`, course, student, area, secs)
		return err
	}
	require.NoError(t, ins(f.c1, f.s1, "CHAT", 30))
	require.Equal(t, "23505", sqlState(ins(f.c1, f.s1, "CHAT", 60)), "UNIQUE (course_id, student_id, date, area)")
	require.NoError(t, ins(f.c2, f.s1, "CHAT", 30), "cùng sinh viên, lớp khác: không xung đột")
	require.Equal(t, "23503", sqlState(ins(f.c1, f.s3, "CHAT", 30)), "sinh viên ngoài lớp: khoá ngoại phức hợp")
}

func TestStudyTimeChecks(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newCrmFx(t, ctx, conn)
	ins := func(area string, secs int) error {
		_, err := conn.Exec(ctx, `insert into study_time_daily (course_id, student_id, date, area, active_seconds) values ($1, $2, current_date - $3::int, $4, $5)`, f.c1, f.s2, secs, area, secs)
		return err
	}
	require.Equal(t, "23514", sqlState(ins("EXAM", 10)), "area ngoài danh sách")
	require.Equal(t, "23514", sqlState(ins("chat", 10)))
	require.Equal(t, "23514", sqlState(ins("CHAT", -1)))
	require.Equal(t, "23514", sqlState(ins("CHAT", 86401)))
	require.NoError(t, ins("CHAT", 0))
	require.NoError(t, ins("THREADS", 86400))
	for _, a := range []string{"PRACTICE", "LIBRARY", "CALENDAR"} {
		require.NoError(t, ins(a, 5), a)
	}
}

func TestSchemaInsightReports(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	require.Equal(t, []string{"id", "course_id", "kind", "period_from", "period_to", "payload", "data_watermark", "generated_by", "version", "created_at", "updated_at"}, tableCols(t, ctx, conn, "insight_reports"))
	f := newCrmFx(t, ctx, conn)
	ins := func(kind, payload string) error {
		_, err := conn.Exec(ctx, `insert into insight_reports (course_id, kind, payload, data_watermark, generated_by) values ($1, $2, $3::jsonb, now(), $4)`, f.c1, kind, payload, f.teacher)
		return err
	}
	require.NoError(t, ins("KNOWLEDGE_GAP", `{}`))
	require.Equal(t, "23514", sqlState(ins("OTHER", `{}`)))
	require.Equal(t, "23514", sqlState(ins("KNOWLEDGE_GAP", `[]`)), "payload phải là object")
}

// TestMigration00015TwoAlters — đúng hai ALTER bảng cũ (proposals #5); view thay bằng CREATE OR REPLACE, không phải ALTER TABLE.
func TestMigration00015TwoAlters(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("../../db/migrations/00015_observation.sql")
	require.NoError(t, err)
	var alters []string
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(ln, "ALTER TABLE") {
			alters = append(alters, ln)
		}
	}
	require.Equal(t, []string{"ALTER TABLE user_settings ADD COLUMN track_study_time boolean NOT NULL DEFAULT true;", "ALTER TABLE courses ADD COLUMN allow_study_time_opt_out boolean NOT NULL DEFAULT true;"}, alters)
	ctx, conn := connect(t)
	for _, c := range []struct{ table, col string }{{"user_settings", "track_study_time"}, {"courses", "allow_study_time_opt_out"}} {
		var def string
		require.NoError(t, conn.QueryRow(ctx, `select column_default from information_schema.columns where table_name = $1 and column_name = $2`, c.table, c.col).Scan(&def))
		require.Equal(t, "true", def, c.col)
	}
}

// TestStudent360StudySecondsFrom00015 — sau 00013 mọi dòng study_seconds_7d = 0; sau 00015 có 300 s trong 7 ngày → 300 (300 s cách 8 ngày không tính); down → 0; cột giống hệt.
func TestStudent360StudySecondsFrom00015(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newCrmFx(t, ctx, conn)
	for _, d := range []struct{ days, secs int }{{0, 120}, {6, 180}, {8, 300}} {
		_, err := conn.Exec(ctx, `insert into study_time_daily (course_id, student_id, date, area, active_seconds)
			values ($1, $2, (now() at time zone 'Asia/Ho_Chi_Minh')::date - $3::int, 'CHAT', $4)`, f.c1, f.s1, d.days, d.secs)
		require.NoError(t, err)
	}
	read := func() (secs int, cols []string) {
		require.NoError(t, conn.QueryRow(ctx, `select study_seconds_7d from v_student_360 where course_id = $1 and student_id = $2`, f.c1, f.s1).Scan(&secs))
		return secs, tableCols(t, ctx, conn, "v_student_360")
	}
	secs, cols := read()
	require.Equal(t, 300, secs, "120 + 180 trong 7 ngày; 300 s cách 8 ngày không tính")
	require.Equal(t, view360Cols, cols)

	sqlDB, err := sql.Open("pgx", conn.Config().ConnString())
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.FS())
	require.NoError(t, err)
	_, err = p.DownTo(ctx, 14)
	require.NoError(t, err)
	secs, cols = read()
	require.Zero(t, secs, "sau down: lại bản 00013 (study_seconds_7d = 0)")
	require.Equal(t, view360Cols, cols, "cột trước / sau giống hệt")
	_, err = p.UpTo(ctx, 15)
	require.NoError(t, err)
	secs, cols = read()
	require.Zero(t, secs, "bảng được tạo lại rỗng")
	require.Equal(t, view360Cols, cols)
}
