package store_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/testutil"
)

// connect mở kết nối tới một database mới đã chạy migration 00001.
func connect(t *testing.T) (context.Context, *pgx.Conn) {
	t.Helper()
	ctx := t.Context()
	conn, err := pgx.Connect(ctx, testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.WithoutCancel(ctx)) })
	return ctx, conn
}

// TestSchema_UsersColumns — US-PG-02 AC2: đúng 16 cột theo SRS 5.1 (tên, kiểu, nullable, mặc định).
func TestSchema_UsersColumns(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)

	want := map[string][3]string{
		"id":                     {"uuid", "NOT NULL", "uuidv7()"},
		"email":                  {"text", "NOT NULL", ""},
		"password_hash":          {"text", "NULL", ""},
		"full_name":              {"text", "NOT NULL", ""},
		"role":                   {"user_role", "NOT NULL", ""},
		"student_code":           {"text", "NULL", ""},
		"email_verified_at":      {"timestamp with time zone", "NULL", ""},
		"failed_logins":          {"integer", "NOT NULL", "0"},
		"locked_until":           {"timestamp with time zone", "NULL", ""},
		"status":                 {"user_status", "NOT NULL", "'INVITED'::user_status"},
		"ics_token":              {"text", "NULL", ""},
		"tracking_notice_ack_at": {"timestamp with time zone", "NULL", ""},
		"last_login_at":          {"timestamp with time zone", "NULL", ""},
		"version":                {"integer", "NOT NULL", "1"},
		"created_at":             {"timestamp with time zone", "NOT NULL", "now()"},
		"updated_at":             {"timestamp with time zone", "NOT NULL", "now()"},
	}

	rows, err := conn.Query(ctx, `
		select a.attname, format_type(a.atttypid, a.atttypmod), a.attnotnull,
		       coalesce(pg_get_expr(d.adbin, d.adrelid), '')
		  from pg_attribute a
		  left join pg_attrdef d on d.adrelid = a.attrelid and d.adnum = a.attnum
		 where a.attrelid = 'public.users'::regclass and a.attnum > 0 and not a.attisdropped`)
	require.NoError(t, err)
	defer rows.Close()

	got := map[string][3]string{}
	for rows.Next() {
		var name, typ, def string
		var notNull bool
		require.NoError(t, rows.Scan(&name, &typ, &notNull, &def))
		nullable := "NULL"
		if notNull {
			nullable = "NOT NULL"
		}
		got[name] = [3]string{typ, nullable, def}
	}
	require.NoError(t, rows.Err())

	require.Len(t, got, 16, "users phải có đúng 16 cột")
	require.Equal(t, want, got)

	// 6 cột dành sẵn cho P2/P5/P8 (AC2).
	for _, c := range []string{"email_verified_at", "failed_logins", "locked_until", "status", "ics_token", "tracking_notice_ack_at"} {
		require.Contains(t, got, c)
	}
}

// TestSchema_UsersConstraints — US-PG-02 AC3: dữ liệu sai bị từ chối với đúng SQLSTATE.
func TestSchema_UsersConstraints(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)

	cases := []struct {
		name       string
		sql        string
		sqlState   string // "" = phải chèn được
		constraint string
	}{
		{"email chữ hoa", `insert into users(email,full_name,role) values('A@X.com','T','STUDENT')`, "23514", "users_email_lower_chk"},
		{"email trùng", `insert into users(email,full_name,role) values('dup@x.com','T','STUDENT');
			insert into users(email,full_name,role) values('dup@x.com','T2','STUDENT')`, "23505", "users_email_key"},
		{"role ngoài enum", `insert into users(email,full_name,role) values('r@x.com','T','SUPERUSER')`, "22P02", ""},
		{"failed_logins âm", `insert into users(email,full_name,role,failed_logins) values('f@x.com','T','STUDENT',-1)`, "23514", "users_failed_logins_chk"},
		{"ACTIVE mật khẩu rỗng", `insert into users(email,full_name,role,status,password_hash) values('a1@x.com','T','STUDENT','ACTIVE','')`, "23514", "users_active_password_chk"},
		{"ACTIVE không mật khẩu", `insert into users(email,full_name,role,status) values('a2@x.com','T','STUDENT','ACTIVE')`, "23514", "users_active_password_chk"},
		{"student_code cho TEACHER", `insert into users(email,full_name,role,student_code) values('s@x.com','T','TEACHER','22010001')`, "23514", "users_student_code_role_chk"},
		{"version = 0", `insert into users(email,full_name,role,version) values('v@x.com','T','STUDENT',0)`, "23514", "users_version_chk"},
		{"INVITED không mật khẩu là hợp lệ", `insert into users(email,full_name,role,status) values('inv@x.com','T','STUDENT','INVITED')`, "", ""},
		{"ACTIVE có bcrypt", `insert into users(email,full_name,role,status,password_hash) values('act@x.com','T','TEACHER','ACTIVE','$2b$12$abcdefghijklmnopqrstuv')`, "", ""},
		{"student_code cho STUDENT", `insert into users(email,full_name,role,student_code) values('stu@x.com','T','STUDENT','22010002')`, "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := conn.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = tx.Rollback(ctx) }()

			_, err = tx.Exec(ctx, tc.sql)
			if tc.sqlState == "" {
				require.NoError(t, err)
				return
			}
			var pgErr *pgconn.PgError
			require.ErrorAs(t, err, &pgErr)
			require.Equal(t, tc.sqlState, pgErr.Code)
			if tc.constraint != "" {
				require.Equal(t, tc.constraint, pgErr.ConstraintName)
			}
		})
	}

	// Mặc định của SRS 5.1.
	var status string
	var failedLogins, version int
	require.NoError(t, conn.QueryRow(ctx, `insert into users(email,full_name,role) values('def@x.com','T','STUDENT')
		returning status::text, failed_logins, version`).Scan(&status, &failedLogins, &version))
	require.Equal(t, "INVITED", status)
	require.Equal(t, 0, failedLogins)
	require.Equal(t, 1, version)

	// Trigger users_set_updated_at: UPDATE làm updated_at > created_at.
	var bumped bool
	require.NoError(t, conn.QueryRow(ctx, `update users set full_name='T2' where email='def@x.com'
		returning updated_at > created_at`).Scan(&bumped))
	require.True(t, bumped)
}

// TestSchema_AuditAppendOnly — US-PG-02 AC5: UPDATE/DELETE/TRUNCATE bị chặn bằng SQLSTATE 42501.
func TestSchema_AuditAppendOnly(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)

	_, err := conn.Exec(ctx, `insert into audit_log(entity,entity_id,action) values('user','1','create')`)
	require.NoError(t, err, "INSERT vào audit_log vẫn phải được")

	for _, tc := range []struct {
		name string
		sql  string
	}{
		{"update", `update audit_log set action='edit'`},
		{"delete", `delete from audit_log`},
		{"truncate", `truncate audit_log`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := conn.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = tx.Rollback(ctx) }()

			_, err = tx.Exec(ctx, tc.sql)
			var pgErr *pgconn.PgError
			require.ErrorAs(t, err, &pgErr)
			require.Equal(t, "42501", pgErr.Code)
			require.Equal(t, "audit_log is append-only", pgErr.Message)
		})
	}

	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from audit_log`).Scan(&n))
	require.Equal(t, 1, n, "dòng audit_log phải còn nguyên")
}
