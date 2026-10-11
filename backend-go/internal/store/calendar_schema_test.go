package store_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestSchemaCalendar — US-P8-03 AC1: cột / kiểu / ràng buộc của calendar_events và reminder_log (SRS 5.1, 5.2).
func TestSchemaCalendar(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	cols := func(table string) map[string]string {
		rows, err := conn.Query(ctx, `select column_name, data_type || ':' || is_nullable from information_schema.columns where table_name=$1`, table)
		require.NoError(t, err)
		defer rows.Close()
		out := map[string]string{}
		for rows.Next() {
			var c, d string
			require.NoError(t, rows.Scan(&c, &d))
			out[c] = d
		}
		return out
	}
	require.Equal(t, map[string]string{"id": "uuid:NO", "course_id": "uuid:NO", "type": "USER-DEFINED:NO", "title": "text:NO", "starts_at": "timestamp with time zone:NO", "ends_at": "timestamp with time zone:YES",
		"location": "text:YES", "description": "text:YES", "ref_type": "text:YES", "ref_id": "uuid:YES", "created_by": "uuid:NO", "version": "integer:NO", "created_at": "timestamp with time zone:NO", "updated_at": "timestamp with time zone:NO"}, cols("calendar_events"))
	require.Equal(t, map[string]string{"id": "uuid:NO", "user_id": "uuid:NO", "course_id": "uuid:NO", "source_type": "USER-DEFINED:NO", "source_id": "uuid:NO", "starts_at": "timestamp with time zone:NO", "kind": "text:NO", "created_at": "timestamp with time zone:NO"}, cols("reminder_log"))
	ins := func(typ, title string, start, end any, loc, desc any, ref any) error {
		_, err := conn.Exec(ctx, `insert into calendar_events (course_id, type, title, starts_at, ends_at, location, description, ref_type, ref_id, created_by) values ($1, $2::calendar_event_type, $3, $4, $5, $6, $7, $8, $9, $10)`,
			f.c1, typ, title, start, end, loc, desc, ref, nil, f.teacher)
		return err
	}
	now := time.Now()
	require.NoError(t, ins("EXAM", "Thi giữa kỳ", now, now.Add(time.Hour), "P.301", "mô tả", nil))
	require.NoError(t, ins("OTHER", "Nộp bài", now, nil, nil, nil, nil))
	for name, err := range map[string]error{
		"loại lạ":            ins("LECTURE", "x", now, nil, nil, nil, nil),
		"tiêu đề rỗng":       ins("EXAM", "", now, nil, nil, nil, nil),
		"tiêu đề > 120":      ins("EXAM", strings.Repeat("a", 121), now, nil, nil, nil, nil),
		"kết thúc ≤ bắt đầu": ins("EXAM", "x", now, now, nil, nil, nil),
		"địa điểm > 80":      ins("EXAM", "x", now, nil, strings.Repeat("a", 81), nil, nil),
		"mô tả > 1000":       ins("EXAM", "x", now, nil, nil, strings.Repeat("a", 1001), nil),
		"ref_type lẻ":        ins("EXAM", "x", now, nil, nil, nil, "ASSIGNMENT"),
	} {
		require.Contains(t, []string{"23514", "22P02"}, sqlState(err), name)
	}
	_, err := conn.Exec(ctx, `insert into calendar_events (course_id, type, title, starts_at, created_by) values ($1, 'OTHER', 'x', now(), $2)`, uuid.New(), f.teacher)
	require.Equal(t, "23503", sqlState(err), "FK courses")
	_, err = conn.Exec(ctx, `update calendar_events set version = 0 where course_id=$1`, f.c1)
	require.Equal(t, "23514", sqlState(err))
	// trigger set_updated_at
	var before, after time.Time
	require.NoError(t, conn.QueryRow(ctx, `select min(updated_at) from calendar_events where course_id=$1`, f.c1).Scan(&before))
	time.Sleep(10 * time.Millisecond)
	_, err = conn.Exec(ctx, `update calendar_events set title='Đổi' where course_id=$1 and type='OTHER'`, f.c1)
	require.NoError(t, err)
	require.NoError(t, conn.QueryRow(ctx, `select max(updated_at) from calendar_events where course_id=$1`, f.c1).Scan(&after))
	require.True(t, after.After(before))
	var idx int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_indexes where indexname='calendar_events_course_starts_idx' and indexdef like '%(course_id, starts_at, id)%'`).Scan(&idx))
	require.Equal(t, 1, idx)
}

// TestReminderLogUnique — AC1: khoá (user, nguồn, nguồn_id, starts_at, kind) duy nhất; kind chỉ T24H; đổi starts_at là khoá mới.
func TestReminderLogUnique(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	src := uuid.New()
	at := time.Now().Add(time.Hour).Truncate(time.Microsecond)
	ins := func(starts time.Time, kind string) error {
		_, err := conn.Exec(ctx, `insert into reminder_log (user_id, course_id, source_type, source_id, starts_at, kind) values ($1, $2, 'WEEKLY_EXAM', $3, $4, $5)`, f.s1, f.c1, src, starts, kind)
		return err
	}
	require.NoError(t, ins(at, "T24H"))
	require.Equal(t, "23505", sqlState(ins(at, "T24H")))
	require.NoError(t, ins(at.Add(time.Hour), "T24H"), "đổi giờ là khoá mới")
	require.Equal(t, "23514", sqlState(ins(at, "T1H")))
	var n int
	_, err := conn.Exec(ctx, `insert into reminder_log (user_id, course_id, source_type, source_id, starts_at) values ($1, $2, 'WEEKLY_EXAM', $3, $4) on conflict do nothing`, f.s1, f.c1, src, at)
	require.NoError(t, err)
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from reminder_log where source_id=$1`, src).Scan(&n))
	require.Equal(t, 2, n)
}

// TestICSTokenHashCheck — AC1, AC7: users.ics_token chỉ nhận NULL hoặc 64 chữ hex thường (SHA-256); token thô bị CHECK từ chối (23514).
func TestICSTokenHashCheck(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	set := func(v any) error {
		_, err := conn.Exec(ctx, `update users set ics_token=$2 where id=$1`, f.s1, v)
		return err
	}
	require.NoError(t, set(strings.Repeat("a", 64)))
	require.NoError(t, set(nil))
	for name, bad := range map[string]string{"token thô base64url": "Zm9vYmFyYmF6cXV4MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM", "63 ký tự": strings.Repeat("a", 63), "65 ký tự": strings.Repeat("a", 65), "chữ hoa": strings.Repeat("A", 64), "không hex": strings.Repeat("g", 64)} {
		require.Equal(t, "23514", sqlState(set(bad)), name)
	}
}
