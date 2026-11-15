package db_test

import (
	"context"
	"database/sql"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/db"
	"github.com/edupilot/backend-go/internal/testutil"
)

// snapshotSQL mô tả toàn bộ lược đồ public (cột, index, ràng buộc, trigger, hàm, enum) thành một chuỗi
// đã sắp xếp — tương đương `pg_dump -s` của AC6 nhưng không cần binary pg_dump.
const snapshotSQL = `
select coalesce(string_agg(line, E'\n' order by line), '') from (
  select 'col ' || c.relname || '.' || a.attname || ' ' || format_type(a.atttypid, a.atttypmod) || ' ' ||
         a.attnotnull || ' ' || coalesce(pg_get_expr(d.adbin, d.adrelid), '-') as line
    from pg_attribute a
    join pg_class c on c.oid = a.attrelid
    join pg_namespace n on n.oid = c.relnamespace
    left join pg_attrdef d on d.adrelid = a.attrelid and d.adnum = a.attnum
   where n.nspname = 'public' and c.relkind = 'r' and a.attnum > 0 and not a.attisdropped
  union all
  select 'idx ' || indexdef from pg_indexes where schemaname = 'public'
  union all
  select 'con ' || conname || ' ' || pg_get_constraintdef(c.oid)
    from pg_constraint c join pg_namespace n on n.oid = c.connamespace where n.nspname = 'public'
  union all
  select 'trg ' || pg_get_triggerdef(t.oid)
    from pg_trigger t join pg_class c on c.oid = t.tgrelid join pg_namespace n on n.oid = c.relnamespace
   where not t.tgisinternal and n.nspname = 'public'
  union all
  select 'fun ' || pg_get_functiondef(p.oid)
    from pg_proc p join pg_namespace n on n.oid = p.pronamespace
   where n.nspname = 'public' and p.proname in ('set_updated_at', 'audit_log_block_mutation')
  union all
  select 'enum ' || t.typname || ' ' || e.enumlabel || ' ' || e.enumsortorder
    from pg_type t join pg_enum e on e.enumtypid = t.oid join pg_namespace n on n.oid = t.typnamespace
   where n.nspname = 'public'
) s`

func TestMigrations_RoundTrip(t *testing.T) {
	t.Parallel()
	url := testutil.PostgresURL(t)
	ctx := t.Context()

	require.NoError(t, db.Migrate(ctx, url, "up", io.Discard))

	conn, err := pgx.Connect(ctx, url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.WithoutCancel(ctx)) })

	var before string
	require.NoError(t, conn.QueryRow(ctx, snapshotSQL).Scan(&before))
	require.Contains(t, before, "col users.email")

	// `up` lần hai là no-op: không thêm dòng goose_db_version, version vẫn là bản mới nhất (20: 10.5 thêm 00020_decision theo D60 + SRS FEAT-decision-model 5, nhánh sprint/10.5-decision; 19: P9 thêm 00019_practice theo proposals sprint 10 #2 — 00018 của sprint 9.5 nằm ở nhánh khác; 17: P7-01 thêm 00017_assignments theo proposals sprint 9 #2; 16: P6-01 thêm 00016_gradebook theo proposals sprint 8 #3 / #7; 15: P5-03 thêm 00015_observation theo proposals #11 / #5 / #20 / #25; 14: P4-06 thêm 00014_forum_ai_regen theo proposals #11 / #25; 13: P5 thêm 00013_crm theo proposals #11 / #5; 12: P4 thêm 00012_escalation_notify; 11: P3 thêm 00007 / 00008, P8 thêm 00010; US-P8-03 thêm 00011_calendar theo proposals #13 — cập nhật số này khi thêm).
	var gooseRows, version, tables int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from goose_db_version`).Scan(&gooseRows))
	require.NoError(t, db.Migrate(ctx, url, "up", io.Discard))
	var gooseRows2 int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from goose_db_version`).Scan(&gooseRows2))
	require.Equal(t, gooseRows, gooseRows2, "up lần hai phải là no-op")
	require.NoError(t, conn.QueryRow(ctx,
		`select version_id from goose_db_version where is_applied order by id desc limit 1`).Scan(&version))
	require.Equal(t, 27, version) // 27: P-11-44 thêm 00027_llm_audit_rollup (bảng cộng gộp theo giờ của llm_audit, TC-01-13); 26: DEF-R4-01 thêm 00026_courses_class_code_ci (mã lớp không phân biệt hoa/thường); 25: sprint 11 thêm 00025_audit_log_indexes (US-P10-14); 24: 10.8 thêm 00024_attendance_checkin (SRS FEAT-attendance-selfcheck 5); 22: sprint 11 thêm 00022_llm_request_payloads (quan sát AI, P-11-1a); 21: 10.6 thêm 00021_grading_guidance theo SRS FEAT-grading-guidance 5, nhánh sprint/10.6-guidance

	// down lùi về version 0: 5 bảng nền + 5 bảng llm_* biến mất, extension vector GIỮ LẠI.
	require.NoError(t, db.Migrate(ctx, url, "down", io.Discard))
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_tables where schemaname='public'
		and tablename in ('users','audit_log','outbox','jobs','idempotency_keys')`).Scan(&tables))
	require.Equal(t, 0, tables)
	var llmTables int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_tables where schemaname='public' and tablename like 'llm\_%'`).Scan(&llmTables))
	require.Equal(t, 0, llmTables)
	var laterTables int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_tables where schemaname='public' and tablename in
		('courses','enrollments','class_sessions','notifications','user_settings','documents','content_chunks','document_courses',
		 'auth_sessions','auth_tokens','login_attempts','mail_outbox')`).Scan(&laterTables))
	require.Equal(t, 0, laterTables, "down phải xoá cả bảng của 00003 và 00004")
	var fns int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_proc where proname = 'vn_fold'`).Scan(&fns))
	require.Equal(t, 0, fns, "down của 00005 phải xoá hàm vn_fold")
	var hasVector bool
	require.NoError(t, conn.QueryRow(ctx, `select exists(select 1 from pg_extension where extname='vector')`).Scan(&hasVector))
	require.True(t, hasVector, "down không được xoá extension vector")
	var enums int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_type t join pg_namespace n on n.oid=t.typnamespace
		where n.nspname='public' and t.typtype='e'`).Scan(&enums))
	require.Equal(t, 0, enums)

	// up lại: lược đồ giống hệt lúc đầu.
	require.NoError(t, db.Migrate(ctx, url, "up", io.Discard))
	var after string
	require.NoError(t, conn.QueryRow(ctx, snapshotSQL).Scan(&after))
	require.Equal(t, before, after, "lược đồ sau down+up phải giống hệt")

	// status chạy được và nêu cả hai migration.
	var out strings.Builder
	require.NoError(t, db.Migrate(ctx, url, "status", &out))
	require.Contains(t, out.String(), "00001_pg_platform.sql")
	require.Contains(t, out.String(), "00002_llm.sql")
	require.Contains(t, out.String(), "00003_course_foundation.sql")
	require.Contains(t, out.String(), "00004_auth_hardening.sql")
}

// P-11-48: DB đã áp 00024 / 00025… mà THIẾU 00022 (nhánh sprint đánh số 22 merge sau) → `up` áp 22 không lỗi, không đổi tên tệp.
func TestMigrations_UpAppliesMissingOlder(t *testing.T) {
	t.Parallel()
	url := testutil.PostgresURL(t)
	ctx := t.Context()

	// dựng DB như thiếu 00022: áp mọi bản trừ 00022 bằng một cây tệp đã lọc
	filtered := fstest.MapFS{}
	require.NoError(t, fs.WalkDir(db.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), "00022_") {
			return err
		}
		b, rerr := fs.ReadFile(db.FS(), p)
		filtered[p] = &fstest.MapFile{Data: b}
		return rerr
	}))
	sqlDB, err := sql.Open("pgx", url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	prov, err := goose.NewProvider(goose.DialectPostgres, sqlDB, filtered)
	require.NoError(t, err)
	_, err = prov.Up(ctx)
	require.NoError(t, err)

	conn, err := pgx.Connect(ctx, url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.WithoutCancel(ctx)) })
	var has bool
	require.NoError(t, conn.QueryRow(ctx, `select to_regclass('public.llm_request_payloads') is not null`).Scan(&has))
	require.False(t, has, "tiền điều kiện: 00022 chưa áp")

	require.NoError(t, db.Migrate(ctx, url, "up", io.Discard), "up phải áp 00022 dù đã có phiên bản cao hơn")
	require.NoError(t, conn.QueryRow(ctx, `select to_regclass('public.llm_request_payloads') is not null`).Scan(&has))
	require.True(t, has, "00022 đã áp")
	var applied int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from goose_db_version where version_id = 22 and is_applied`).Scan(&applied))
	require.Equal(t, 1, applied)
	require.NoError(t, db.Migrate(ctx, url, "up", io.Discard), "up lần hai là no-op")
}
