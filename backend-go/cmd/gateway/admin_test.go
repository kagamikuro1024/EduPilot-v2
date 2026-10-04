package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/testutil"
)

func adminEnv(url string, extra map[string]string) func(string) string {
	env := map[string]string{"DATABASE_URL": url, "BCRYPT_COST": "4"}
	for k, v := range extra {
		env[k] = v
	}
	return func(k string) string { return env[k] }
}

func runAdminCmd(t *testing.T, args []string, env func(string) string, stdin string) (code int, out, errOut string) {
	t.Helper()
	var o, e bytes.Buffer
	code = runAdmin(args, env, strings.NewReader(stdin), &o, &e)
	return code, o.String(), e.String()
}

// US-P2-06 AC9.
func TestAdminCreate(t *testing.T) {
	url := testutil.MigratedPostgresURL(t)
	pool, err := pgxpool.New(t.Context(), url)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	count := func(where string) int {
		var n int
		require.NoError(t, pool.QueryRow(t.Context(), `select count(*) from users where `+where).Scan(&n))
		return n
	}
	const pw = "Mat-khau-quan-tri-2026"
	args := []string{"create", "--email", "Quan.Tri@edupilot.test", "--name", "Quản trị"}

	code, out, errOut := runAdminCmd(t, args, adminEnv(url, map[string]string{"ADMIN_PASSWORD": pw}), "")
	require.Equal(t, 0, code, errOut)
	require.Contains(t, out, "Đã tạo")
	require.Equal(t, 1, count(`role = 'ADMIN' and status = 'ACTIVE' and email = 'quan.tri@edupilot.test' and email_verified_at is not null and password_hash like '$2a$04$%'`))
	var audit int
	require.NoError(t, pool.QueryRow(t.Context(), `select count(*) from audit_log where action = 'admin_bootstrap' and actor_id is null`).Scan(&audit))
	require.Equal(t, 1, audit)

	// lần hai: thoát 0, không đổi mật khẩu (kể cả khi mật khẩu khác)
	var before string
	require.NoError(t, pool.QueryRow(t.Context(), `select password_hash from users where email = 'quan.tri@edupilot.test'`).Scan(&before))
	code, out, _ = runAdminCmd(t, args, adminEnv(url, map[string]string{"ADMIN_PASSWORD": "Mat-khau-khac-2026-xyz"}), "")
	require.Equal(t, 0, code)
	require.Contains(t, out, "đã tồn tại")
	var after string
	require.NoError(t, pool.QueryRow(t.Context(), `select password_hash from users where email = 'quan.tri@edupilot.test'`).Scan(&after))
	require.Equal(t, before, after)
	require.Equal(t, 1, count(`role = 'ADMIN'`))

	// mật khẩu yếu ⇒ 1 và không tạo gì; thông báo không lặp lại mật khẩu
	code, _, errOut = runAdminCmd(t, []string{"create", "--email", "yeu@edupilot.test", "--name", "Yếu"}, adminEnv(url, map[string]string{"ADMIN_PASSWORD": "123"}), "")
	require.Equal(t, 1, code)
	require.NotContains(t, errOut, "123 ")
	require.Equal(t, 0, count(`email = 'yeu@edupilot.test'`))

	// mật khẩu từ stdin
	code, _, errOut = runAdminCmd(t, []string{"create", "--email", "stdin@edupilot.test", "--name", "Từ stdin"}, adminEnv(url, nil), pw+"\n")
	require.Equal(t, 0, code, errOut)
	require.Equal(t, 1, count(`email = 'stdin@edupilot.test' and role = 'ADMIN'`))
}

func TestAdminCreateRejectsPasswordFlag(t *testing.T) {
	t.Parallel()
	env := adminEnv("postgres://x:y@127.0.0.1:1/z", map[string]string{"ADMIN_PASSWORD": "Mat-khau-quan-tri-2026"})
	for _, args := range [][]string{
		{"create", "--email", "a@b.cc", "--name", "A", "--password", "Mat-khau-quan-tri-2026"},
		{"create", "--email", "a@b.cc", "--name", "A", "-password=x"},
		{"create", "--email", "a@b.cc", "--name", "A", "--co-la"},
		{"create", "--email", "a@b.cc", "--name", "A", "thua"},
		{"tao"},
		{},
	} {
		code, _, errOut := runAdminCmd(t, args, env, "")
		require.Equal(t, 2, code, "%v", args)
		require.Contains(t, errOut, "Dùng")
	}
	code, _, errOut := runAdminCmd(t, []string{"create", "--email", "a@b.cc", "--name", "A", "--password", "x"}, env, "")
	require.Equal(t, 2, code)
	require.Contains(t, errOut, "Dùng biến ADMIN_PASSWORD hoặc stdin.")
	require.NotContains(t, errOut, "Mat-khau-quan-tri-2026")
}
