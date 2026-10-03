package store_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

// TestLLMSchema — US-P1-01 AC1: cột, kiểu, nullable, mặc định của 5 bảng khớp SRS FEAT-llm-gateway 5.2.
func TestLLMSchema(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)

	type col struct{ typ, null, def string }
	const (
		nn = "NOT NULL"
		nl = "NULL"
		ts = "timestamp with time zone"
	)
	idCol := col{"uuid", nn, "uuidv7()"}
	created := col{ts, nn, "now()"}
	want := map[string]map[string]col{
		"llm_providers": {
			"id": idCol, "type": {"text", nn, ""}, "name": {"text", nn, ""}, "base_url": {"text", nl, ""},
			"api_key_enc": {"bytea", nl, ""}, "enabled": {"boolean", nn, "true"},
			"rpm_limit": {"integer", nl, ""}, "tpm_limit": {"integer", nl, ""},
			"last_test_ok": {"boolean", nl, ""}, "last_test_at": {ts, nl, ""}, "last_test_error": {"text", nl, ""},
			"version": {"integer", nn, "1"}, "created_at": created, "updated_at": created,
		},
		"llm_models": {
			"id": idCol, "provider_id": {"uuid", nn, ""}, "model": {"text", nn, ""}, "kind": {"text", nn, ""},
			"dims": {"integer", nl, ""}, "price_in": {"numeric(14,4)", nn, "0"}, "price_out": {"numeric(14,4)", nn, "0"},
			"enabled": {"boolean", nn, "true"}, "created_at": created, "updated_at": created,
		},
		"llm_task_routes": {
			"id": idCol, "task": {"text", nn, ""}, "model_id": {"uuid", nn, ""}, "fallback_order": {"integer", nn, ""},
			"params": {"jsonb", nn, "'{}'::jsonb"}, "version": {"integer", nn, "1"}, "created_at": created, "updated_at": created,
		},
		"llm_audit": {
			"id": idCol, "task": {"text", nn, ""}, "lane": {"text", nn, ""}, "provider": {"text", nl, ""}, "model": {"text", nl, ""},
			"tokens_in": {"integer", nn, "0"}, "tokens_out": {"integer", nn, "0"}, "latency_ms": {"integer", nn, "0"},
			"queue_wait_ms": {"integer", nn, "0"}, "attempts": {"integer", nn, "1"}, "fallback_index": {"integer", nn, "0"},
			"cost_est": {"numeric(14,4)", nn, "0"}, "status": {"text", nn, ""}, "error_kind": {"text", nl, ""},
			"degraded": {"boolean", nn, "false"}, "pii_masked_count": {"integer", nn, "0"},
			"user_id": {"uuid", nl, ""}, "course_id": {"uuid", nl, ""}, "trace_id": {"text", nn, ""}, "created_at": created,
		},
		"llm_budgets": {
			"id": idCol, "scope": {"text", nn, ""}, "course_id": {"uuid", nl, ""},
			"daily_limit": {"numeric(14,2)", nl, ""}, "monthly_limit": {"numeric(14,2)", nl, ""},
			"version": {"integer", nn, "1"}, "created_at": created, "updated_at": created,
		},
	}

	var tables int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from information_schema.tables
		where table_schema='public' and table_name like 'llm\_%'`).Scan(&tables))
	require.Equal(t, 5, tables)

	for table, cols := range want {
		rows, err := conn.Query(ctx, `
			select a.attname, format_type(a.atttypid, a.atttypmod), a.attnotnull, coalesce(pg_get_expr(d.adbin, d.adrelid), '')
			  from pg_attribute a
			  left join pg_attrdef d on d.adrelid = a.attrelid and d.adnum = a.attnum
			 where a.attrelid = ('public.' || $1)::regclass and a.attnum > 0 and not a.attisdropped`, table)
		require.NoError(t, err)
		got := map[string]col{}
		for rows.Next() {
			var name, typ, def string
			var notNull bool
			require.NoError(t, rows.Scan(&name, &typ, &notNull, &def))
			null := nl
			if notNull {
				null = nn
			}
			got[name] = col{typ, null, def}
		}
		require.NoError(t, rows.Err())
		require.Equal(t, cols, got, "bảng %s", table)
	}

	// Không FK tới users / courses; chỉ llm_models → llm_providers và llm_task_routes → llm_models.
	rows, err := conn.Query(ctx, `select conrelid::regclass::text, confrelid::regclass::text
		from pg_constraint where contype='f' and conrelid::regclass::text like 'llm\_%' order by 1`)
	require.NoError(t, err)
	var fks []string
	for rows.Next() {
		var a, b string
		require.NoError(t, rows.Scan(&a, &b))
		fks = append(fks, a+"→"+b)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"llm_models→llm_providers", "llm_task_routes→llm_models"}, fks)
}

func sqlState(err error) string {
	if pg, ok := errorsAsPg(err); ok {
		return pg.Code
	}
	return ""
}

func errorsAsPg(err error) (*pgconn.PgError, bool) {
	for err != nil {
		if pg, ok := err.(*pgconn.PgError); ok { //nolint:errorlint // đi tay để không kéo thêm import
			return pg, true
		}
		u, ok := err.(interface{ Unwrap() error }) //nolint:errorlint
		if !ok {
			return nil, false
		}
		err = u.Unwrap()
	}
	return nil, false
}

// TestLLMConstraints — US-P1-01 AC2: dữ liệu sai bị DB từ chối đúng SQLSTATE.
func TestLLMConstraints(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)

	const (
		okProv  = "00000000-0000-7000-8000-0000000000a1"
		okModel = "00000000-0000-7000-8000-0000000000b1"
		course  = "00000000-0000-7000-8000-0000000000c1"
	)
	mustExec := func(sql string, args ...any) {
		t.Helper()
		_, err := conn.Exec(ctx, sql, args...)
		require.NoError(t, err, sql)
	}
	mustExec(`insert into llm_providers (id, type, name) values ($1, 'fake', 'P-ok')`, okProv)
	mustExec(`insert into llm_models (id, provider_id, model, kind) values ($1, $2, 'm-ok', 'chat')`, okModel, okProv)
	mustExec(`insert into llm_task_routes (task, model_id, fallback_order) values ('CHAT', $1, 0)`, okModel)
	mustExec(`insert into llm_budgets (scope) values ('system')`)
	mustExec(`insert into llm_budgets (scope, course_id) values ('course', $1)`, course)

	cases := []struct{ name, sql, state string }{
		{"provider.type lạ", `insert into llm_providers (type, name) values ('mistral', 'x1')`, "23514"},
		{"openai_compatible thiếu base_url", `insert into llm_providers (type, name) values ('openai_compatible', 'x2')`, "23514"},
		{"provider.name rỗng", `insert into llm_providers (type, name) values ('fake', '')`, "23514"},
		{"provider.name trùng", `insert into llm_providers (type, name) values ('fake', 'P-ok')`, "23505"},
		{"provider thiếu type", `insert into llm_providers (name) values ('x3')`, "23502"},
		{"rpm_limit = 0", `insert into llm_providers (type, name, rpm_limit) values ('fake', 'x4', 0)`, "23514"},
		{"model.kind lạ", `insert into llm_models (provider_id, model, kind) values ('` + okProv + `', 'm2', 'image')`, "23514"},
		{"embedding thiếu dims", `insert into llm_models (provider_id, model, kind) values ('` + okProv + `', 'm3', 'embedding')`, "23514"},
		{"model trùng trong nhà", `insert into llm_models (provider_id, model, kind) values ('` + okProv + `', 'm-ok', 'chat')`, "23505"},
		{"giá vào âm", `insert into llm_models (provider_id, model, kind, price_in) values ('` + okProv + `', 'm4', 'chat', -1)`, "23514"},
		{"giá ra âm", `insert into llm_models (provider_id, model, kind, price_out) values ('` + okProv + `', 'm5', 'chat', -0.0001)`, "23514"},
		{"route.task lạ", `insert into llm_task_routes (task, model_id, fallback_order) values ('SING', '` + okModel + `', 1)`, "23514"},
		{"route trùng (task, order)", `insert into llm_task_routes (task, model_id, fallback_order) values ('CHAT', '` + okModel + `', 0)`, "23505"},
		{"fallback_order âm", `insert into llm_task_routes (task, model_id, fallback_order) values ('CLASSIFY', '` + okModel + `', -1)`, "23514"},
		{"route trỏ mô hình không tồn tại", `insert into llm_task_routes (task, model_id, fallback_order) values ('CLASSIFY', gen_random_uuid(), 0)`, "23503"},
		{"budget course thiếu course_id", `insert into llm_budgets (scope) values ('course')`, "23514"},
		{"budget system có course_id", `insert into llm_budgets (scope, course_id) values ('system', '` + course + `')`, "23514"},
		{"hai budget system", `insert into llm_budgets (scope) values ('system')`, "23505"},
		{"hai budget cùng course", `insert into llm_budgets (scope, course_id) values ('course', '` + course + `')`, "23505"},
		{"budget ngày > tháng", `insert into llm_budgets (scope, course_id, daily_limit, monthly_limit) values ('course', gen_random_uuid(), 10, 5)`, "23514"},
		{"audit.status lạ", `insert into llm_audit (task, lane, status, trace_id) values ('CHAT', 'INTERACTIVE', 'weird', 't')`, "23514"},
		{"audit.lane lạ", `insert into llm_audit (task, lane, status, trace_id) values ('CHAT', 'FAST', 'ok', 't')`, "23514"},
		{"audit thiếu trace_id", `insert into llm_audit (task, lane, status) values ('CHAT', 'INTERACTIVE', 'ok')`, "23502"},
		{"xoá mô hình đang được tuyến dùng", `delete from llm_models where id = '` + okModel + `'`, "23503"},
	}
	require.GreaterOrEqual(t, len(cases), 14)
	for _, tc := range cases {
		_, err := conn.Exec(ctx, tc.sql)
		require.Error(t, err, tc.name)
		require.Equal(t, tc.state, sqlState(err), "%s: %v", tc.name, err)
	}

	// Xoá nhà cung cấp cuốn theo mô hình của nó, nhưng bị chặn khi tuyến còn dùng (FK route → model không cascade).
	_, err := conn.Exec(ctx, `delete from llm_providers where id = $1`, okProv)
	require.Equal(t, "23503", sqlState(err))
	mustExec(`delete from llm_task_routes`)
	mustExec(`delete from llm_providers where id = $1`, okProv)
	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from llm_models`).Scan(&n))
	require.Zero(t, n)
}

// TestLLMIndexes — US-P1-01 AC3: truy vấn của GET usage dùng chỉ mục trên 20.000 dòng; danh sách chỉ mục khớp SRS 5.3.
func TestLLMIndexes(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)

	_, err := conn.Exec(ctx, `
		insert into llm_audit (task, lane, status, trace_id, course_id, created_at, latency_ms, tokens_in, cost_est)
		select (array['CHAT','CLASSIFY','UTILITY','GRADING'])[1 + i % 4], 'BATCH', 'ok', md5(i::text),
		       case when i % 50 = 0 then '00000000-0000-7000-8000-0000000000c1'::uuid else null end,
		       now() - (i || ' seconds')::interval, i % 3000, i % 500, 1.5
		  from generate_series(1, 20000) i`)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `analyze llm_audit`)
	require.NoError(t, err)

	plan := func(sql string, args ...any) string {
		t.Helper()
		var raw []byte
		require.NoError(t, conn.QueryRow(ctx, "explain (format json) "+sql, args...).Scan(&raw))
		var doc []map[string]any
		require.NoError(t, json.Unmarshal(raw, &doc))
		return string(raw)
	}
	cases := []struct {
		name, sql string
		index     []string
	}{
		{"usage theo tác vụ", `select task, count(*), percentile_cont(0.5) within group (order by latency_ms) from llm_audit
			where created_at >= now() - interval '1 minute' and task = 'CHAT' group by task`, []string{"llm_audit_task_created_idx", "llm_audit_created_idx"}},
		{"usage theo ngày", `select date_trunc('day', created_at), sum(cost_est) from llm_audit
			where created_at >= now() - interval '1 minute' group by 1`, []string{"llm_audit_created_idx"}},
		{"usage theo lớp", `select task, count(*) from llm_audit where course_id = '00000000-0000-7000-8000-0000000000c1'
			and created_at >= now() - interval '7 days' group by task`, []string{"llm_audit_course_created_idx"}},
		{"theo trace", `select * from llm_audit where trace_id = md5('7')`, []string{"llm_audit_trace_idx"}},
	}
	for _, tc := range cases {
		p := plan(tc.sql)
		hit := false
		for _, ix := range tc.index {
			hit = hit || strings.Contains(p, ix)
		}
		require.True(t, hit, "%s: plan không dùng %v: %s", tc.name, tc.index, p)
		require.NotContains(t, p, `"Seq Scan"`, tc.name)
	}

	rows, err := conn.Query(ctx, `select indexname from pg_indexes where tablename='llm_audit' order by 1`)
	require.NoError(t, err)
	var got []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		got = append(got, s)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"llm_audit_course_created_idx", "llm_audit_created_idx", "llm_audit_pkey", "llm_audit_task_created_idx", "llm_audit_trace_idx"}, got)

	// Chỉ mục duy nhất từng phần của llm_budgets và (task, fallback_order), (provider_id, model).
	rows2, err := conn.Query(ctx, `select tablename || '.' || indexname from pg_indexes
		where tablename in ('llm_budgets','llm_task_routes','llm_models') and indexdef like 'CREATE UNIQUE%' and indexname not like '%pkey' order by 1`)
	require.NoError(t, err)
	var uq []string
	for rows2.Next() {
		var s string
		require.NoError(t, rows2.Scan(&s))
		uq = append(uq, s)
	}
	require.NoError(t, rows2.Err())
	require.Equal(t, []string{"llm_budgets.llm_budgets_course_uq", "llm_budgets.llm_budgets_system_uq", "llm_models.llm_models_uq", "llm_task_routes.llm_task_routes_uq"}, uq)
}
