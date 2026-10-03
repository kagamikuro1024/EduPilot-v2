package store_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAuthSchema — US-P2-01 AC1: 4 bảng của 00004 có đủ cột/kiểu/null/mặc định, 3 enum, đủ chỉ mục theo SRS 5.2–5.6.
func TestAuthSchema(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)

	type col struct{ typ, null, def string }
	const (
		nn = "NOT NULL"
		nl = "NULL"
		ts = "timestamp with time zone"
	)
	idCol := col{"uuid", nn, "uuidv7()"}
	now := col{ts, nn, "now()"}
	want := map[string]map[string]col{
		"auth_sessions": {
			"id": idCol, "user_id": {"uuid", nn, ""}, "refresh_hash": {"character(64)", nn, ""}, "prev_refresh_hash": {"character(64)", nl, ""},
			"user_agent": {"text", nl, ""}, "device_label": {"text", nl, ""}, "ip": {"inet", nl, ""}, "created_at": now,
			"rotated_at": {ts, nl, ""}, "last_used_at": now, "expires_at": {ts, nn, ""}, "absolute_expires_at": {ts, nn, ""},
			"revoked_at": {ts, nl, ""}, "revoked_reason": {"text", nl, ""}, "updated_at": now,
		},
		"auth_tokens": {
			"id": idCol, "user_id": {"uuid", nn, ""}, "kind": {"auth_token_kind", nn, ""}, "token_hash": {"character(64)", nn, ""},
			"expires_at": {ts, nn, ""}, "used_at": {ts, nl, ""}, "revoked_at": {ts, nl, ""}, "created_by": {"uuid", nl, ""}, "created_at": now,
		},
		"login_attempts": {
			"id": idCol, "email_hash": {"character(64)", nn, ""}, "user_id": {"uuid", nl, ""}, "ip": {"inet", nl, ""},
			"user_agent": {"text", nl, ""}, "outcome": {"login_outcome", nn, ""}, "created_at": now,
		},
		"mail_outbox": {
			"id": idCol, "to_addr": {"text", nn, ""}, "template": {"text", nn, ""}, "payload": {"jsonb", nn, "'{}'::jsonb"},
			"status": {"mail_status", nn, "'QUEUED'::mail_status"}, "attempts": {"integer", nn, "0"}, "last_error": {"text", nl, ""},
			"dedupe_key": {"text", nl, ""}, "sent_at": {ts, nl, ""}, "created_at": now, "updated_at": now,
		},
	}
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

	enums := map[string][]string{
		"auth_token_kind": {"VERIFY_EMAIL", "RESET_PASSWORD", "INVITE"},
		"mail_status":     {"QUEUED", "SENT", "DEAD"},
		"login_outcome":   {"SUCCESS", "BAD_PASSWORD", "UNKNOWN_EMAIL", "THROTTLED", "LOCKED", "DISABLED"},
	}
	for name, labels := range enums {
		var got []string
		rows, err := conn.Query(ctx, `select e.enumlabel from pg_enum e join pg_type t on t.oid = e.enumtypid where t.typname = $1 order by e.enumsortorder`, name)
		require.NoError(t, err)
		for rows.Next() {
			var l string
			require.NoError(t, rows.Scan(&l))
			got = append(got, l)
		}
		require.NoError(t, rows.Err())
		require.Equal(t, labels, got, "enum %s", name)
	}

	indexes := []string{
		"auth_sessions_pkey", "auth_sessions_refresh_hash_key", "auth_sessions_prev_hash_idx", "auth_sessions_user_active_idx", "auth_sessions_expires_idx",
		"auth_tokens_pkey", "auth_tokens_token_hash_key", "auth_tokens_user_kind_idx",
		"login_attempts_pkey", "login_attempts_email_idx", "login_attempts_ip_idx", "login_attempts_user_idx",
		"mail_outbox_pkey", "mail_outbox_dedupe_key_key", "mail_outbox_queued_idx",
	}
	for _, ix := range indexes {
		var n int
		require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_indexes where schemaname='public' and indexname=$1`, ix).Scan(&n))
		require.Equal(t, 1, n, "thiếu chỉ mục %s", ix)
	}
	// Chỉ mục riêng phần (partial) đúng điều kiện SRS.
	var def string
	require.NoError(t, conn.QueryRow(ctx, `select indexdef from pg_indexes where indexname='mail_outbox_dedupe_key_key'`).Scan(&def))
	require.Contains(t, def, "UNIQUE")
	require.Contains(t, def, "dedupe_key IS NOT NULL")
	require.NoError(t, conn.QueryRow(ctx, `select indexdef from pg_indexes where indexname='auth_sessions_user_active_idx'`).Scan(&def))
	require.Contains(t, def, "revoked_at IS NULL")

	// Chỉ FK users(id) ON DELETE CASCADE ở auth_sessions/auth_tokens; login_attempts.user_id không FK (sống lâu hơn tài khoản).
	var fks int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_constraint where contype='f' and conrelid::regclass::text in ('auth_sessions','auth_tokens') and confdeltype='c' and confrelid='users'::regclass`).Scan(&fks))
	require.Equal(t, 2, fks)
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_constraint where contype='f' and conrelid::regclass::text in ('login_attempts','mail_outbox')`).Scan(&fks))
	require.Zero(t, fks)
}

// TestAuthConstraints — US-P2-01 AC2: dữ liệu sai bị DB từ chối đúng SQLSTATE.
func TestAuthConstraints(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)

	const uid = "00000000-0000-7000-8000-0000000000d1"
	hex := func(c byte) string {
		b := make([]byte, 64)
		for i := range b {
			b[i] = c
		}
		return string(b)
	}
	mustExec := func(sql string, args ...any) {
		t.Helper()
		_, err := conn.Exec(ctx, sql, args...)
		require.NoError(t, err, sql)
	}
	mustExec(`insert into users (id, email, full_name, role) values ($1, 'ac@example.test', 'AC', 'STUDENT')`, uid)
	mustExec(`insert into auth_tokens (user_id, kind, token_hash, expires_at) values ($1, 'INVITE', $2, now() + interval '1 hour')`, uid, hex('a'))
	mustExec(`insert into auth_sessions (user_id, refresh_hash, expires_at, absolute_expires_at) values ($1, $2, now() + interval '1 day', now() + interval '30 days')`, uid, hex('a'))
	mustExec(`insert into mail_outbox (to_addr, template, dedupe_key) values ('x@example.test', 'verify_email', 'dk-1')`)

	const (
		check  = "23514"
		unique = "23505"
		enum   = "22P02" // giá trị enum ngoài tập
	)
	tokIns := `insert into auth_tokens (user_id, kind, token_hash, expires_at%s) values ($1, %s, $2, now() + interval '1 hour'%s)`
	sessIns := `insert into auth_sessions (user_id, refresh_hash, expires_at, absolute_expires_at%s) values ($1, $2, %s%s)`
	okExp := `now() + interval '1 day', now() + interval '30 days'`
	cases := []struct {
		name  string
		sql   string
		args  []any
		state string
	}{
		{"token_hash không phải hex", fmt.Sprintf(tokIns, "", "'INVITE'", ""), []any{uid, "zz" + hex('b')[2:]}, check},
		{"token_hash quá ngắn", fmt.Sprintf(tokIns, "", "'INVITE'", ""), []any{uid, "abc"}, check},
		{"token_hash chữ hoa", fmt.Sprintf(tokIns, "", "'INVITE'", ""), []any{uid, hex('A')}, check},
		{"token_hash trùng", fmt.Sprintf(tokIns, "", "'INVITE'", ""), []any{uid, hex('a')}, unique},
		{"kind ngoài tập", fmt.Sprintf(tokIns, "", "'SESSION'", ""), []any{uid, hex('c')}, enum},
		{"vừa used_at vừa revoked_at", fmt.Sprintf(tokIns, ", used_at, revoked_at", "'INVITE'", ", now(), now()"), []any{uid, hex('d')}, check},
		{"refresh_hash sai độ dài", fmt.Sprintf(sessIns, "", okExp, ""), []any{uid, "abc"}, check},
		{"refresh_hash không hex", fmt.Sprintf(sessIns, "", okExp, ""), []any{uid, "Z" + hex('e')[1:]}, check},
		{"refresh_hash trùng", fmt.Sprintf(sessIns, "", okExp, ""), []any{uid, hex('a')}, unique},
		{"revoked_at thiếu revoked_reason", fmt.Sprintf(sessIns, ", revoked_at", okExp, ", now()"), []any{uid, hex('f')}, check},
		{"revoked_reason thiếu revoked_at", fmt.Sprintf(sessIns, ", revoked_reason", okExp, ", 'LOGOUT'"), []any{uid, hex('f')}, check},
		{"revoked_reason lạ", fmt.Sprintf(sessIns, ", revoked_at, revoked_reason", okExp, ", now(), 'BORED'"), []any{uid, hex('f')}, check},
		{"absolute < expires", fmt.Sprintf(sessIns, "", `now() + interval '2 days', now() + interval '1 day'`, ""), []any{uid, hex('f')}, check},
		{"to_addr chữ hoa", `insert into mail_outbox (to_addr, template) values ('Ab@example.test', 'verify_email')`, nil, check},
		{"to_addr không có @", `insert into mail_outbox (to_addr, template) values ('abc', 'verify_email')`, nil, check},
		{"status ngoài tập", `insert into mail_outbox (to_addr, template, status) values ('a@example.test', 'verify_email', 'SENDING')`, nil, enum},
		{"dedupe_key trùng", `insert into mail_outbox (to_addr, template, dedupe_key) values ('y@example.test', 'verify_email', 'dk-1')`, nil, unique},
		{"SENT thiếu sent_at", `insert into mail_outbox (to_addr, template, status) values ('a@example.test', 'verify_email', 'SENT')`, nil, check},
		{"template sai dạng", `insert into mail_outbox (to_addr, template) values ('a@example.test', 'Verify-Email')`, nil, check},
		{"attempts > 4", `insert into mail_outbox (to_addr, template, attempts) values ('a@example.test', 'verify_email', 5)`, nil, check},
		{"email_hash không hex", `insert into login_attempts (email_hash, outcome) values ('nothex', 'SUCCESS')`, nil, check},
		{"email_hash 64 ký tự nhưng không hex", `insert into login_attempts (email_hash, outcome) values (repeat('g', 64), 'SUCCESS')`, nil, check},
		{"outcome ngoài tập", `insert into login_attempts (email_hash, outcome) values (repeat('a', 64), 'WEIRD')`, nil, enum},
	}
	require.GreaterOrEqual(t, len(cases), 12)
	for _, tc := range cases {
		_, err := conn.Exec(ctx, tc.sql, tc.args...)
		require.Error(t, err, tc.name)
		require.Equal(t, tc.state, sqlState(err), "%s: %v", tc.name, err)
	}
	// Dòng hợp lệ vẫn được nhận (bảng ca kiểm không tự chặn mọi thứ).
	mustExec(`insert into login_attempts (email_hash, outcome) values (repeat('a', 64), 'BAD_PASSWORD')`)
	mustExec(`insert into mail_outbox (to_addr, template, status, sent_at) values ('ok@example.test', 'verify_email', 'SENT', now())`)
}
