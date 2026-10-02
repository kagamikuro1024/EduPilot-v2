package db_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
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

	// `up` lần hai là no-op: không thêm dòng goose_db_version, version vẫn là bản mới nhất (2).
	var gooseRows, version, tables int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from goose_db_version`).Scan(&gooseRows))
	require.NoError(t, db.Migrate(ctx, url, "up", io.Discard))
	var gooseRows2 int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from goose_db_version`).Scan(&gooseRows2))
	require.Equal(t, gooseRows, gooseRows2, "up lần hai phải là no-op")
	require.NoError(t, conn.QueryRow(ctx,
		`select version_id from goose_db_version where is_applied order by id desc limit 1`).Scan(&version))
	require.Equal(t, 2, version)

	// down lùi về version 0: 5 bảng nền + 5 bảng llm_* biến mất, extension vector GIỮ LẠI.
	require.NoError(t, db.Migrate(ctx, url, "down", io.Discard))
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_tables where schemaname='public'
		and tablename in ('users','audit_log','outbox','jobs','idempotency_keys')`).Scan(&tables))
	require.Equal(t, 0, tables)
	var llmTables int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_tables where schemaname='public' and tablename like 'llm\_%'`).Scan(&llmTables))
	require.Equal(t, 0, llmTables)
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
}
