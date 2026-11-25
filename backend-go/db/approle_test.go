package db_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/db"
	"github.com/edupilot/backend-go/internal/testutil"
)

// TC-P3-01-49 (DEF-R6-01): gateway / worker nối bằng vai ứng dụng — không superuser, không vượt RLS; migrate vẫn bằng vai chủ.
func TestAppRoleIsNotSuperuser(t *testing.T) {
	t.Parallel()
	ownerURL := testutil.PostgresURL(t)
	ctx := t.Context()
	require.NoError(t, db.Migrate(ctx, ownerURL, "up", io.Discard))
	require.NoError(t, db.SetAppRolePassword(ctx, ownerURL, "app-pw-test", false))

	owner := connect(t, ctx, ownerURL, "", "")
	app := connect(t, ctx, ownerURL, db.AppRole, "app-pw-test")

	var user string
	var super, bypass bool
	require.NoError(t, app.QueryRow(ctx, `select current_user, rolsuper, rolbypassrls from pg_roles where rolname = current_user`).Scan(&user, &super, &bypass))
	require.Equal(t, db.AppRole, user)
	require.False(t, super)
	require.False(t, bypass)

	// Một dòng thật (bỏ kiểm FK bằng vai chủ): UPDATE theo từng dòng mới kích hoạt trigger.
	_, err := owner.Exec(ctx, `set session_replication_role = replica`)
	require.NoError(t, err)
	_, err = owner.Exec(ctx, `insert into pii_events (course_id, user_id, channel, pii_type, count, action)
		select uuidv7(), uuidv7(), (enum_range(null::chat_channel))[1], (enum_range(null::pii_kind))[1], 1, (enum_range(null::pii_action))[1]`)
	require.NoError(t, err)

	// Vai ứng dụng không phải chủ: trigger append-only không gỡ được, và nó vẫn chặn UPDATE.
	_, err = app.Exec(ctx, `update pii_events set count = 9`)
	var pe *pgconn.PgError
	require.True(t, errors.As(err, &pe), "err=%v", err)
	require.Equal(t, "42501", pe.Code)
	_, err = app.Exec(ctx, `alter table pii_events disable trigger pii_events_no_update`)
	require.True(t, errors.As(err, &pe), "err=%v", err)
	require.Equal(t, "42501", pe.Code)
	_, err = app.Exec(ctx, `create table app_must_not_ddl (id int)`)
	require.True(t, errors.As(err, &pe), "err=%v", err)
	require.Equal(t, "42501", pe.Code)

	// Mọi bảng nghiệp vụ dùng được; bảng goose thì không; bảng tạo SAU (migration sau này) tự có quyền.
	_, err = owner.Exec(ctx, `create table zz_later_table (id int)`)
	require.NoError(t, err)
	rows, err := owner.Query(ctx, `select c.relname from pg_class c join pg_namespace n on n.oid = c.relnamespace
		where n.nspname = 'public' and c.relkind in ('r', 'p') and c.relname <> 'goose_db_version'
		  and not (has_table_privilege($1, c.oid, 'SELECT') and has_table_privilege($1, c.oid, 'INSERT')
		       and has_table_privilege($1, c.oid, 'UPDATE') and has_table_privilege($1, c.oid, 'DELETE'))`, db.AppRole)
	require.NoError(t, err)
	missing, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	require.Empty(t, missing)
	var goose bool
	require.NoError(t, owner.QueryRow(ctx, `select has_table_privilege($1, 'goose_db_version', 'SELECT')`, db.AppRole).Scan(&goose))
	require.False(t, goose)
}

// Sau `migrate down` rồi `up` vai vẫn dùng được (vai là đối tượng cụm, không bị migration down xoá) và không còn quyền thừa.
func TestAppRoleSurvivesDownUp(t *testing.T) {
	t.Parallel()
	ownerURL := testutil.PostgresURL(t)
	ctx := t.Context()
	require.NoError(t, db.Migrate(ctx, ownerURL, "up", io.Discard))
	require.NoError(t, db.Migrate(ctx, ownerURL, "down", io.Discard))
	require.NoError(t, db.Migrate(ctx, ownerURL, "up", io.Discard))
	require.NoError(t, db.SetAppRolePassword(ctx, ownerURL, "app-pw-test", true))
	app := connect(t, ctx, ownerURL, db.AppRole, "app-pw-test")
	var n int
	require.NoError(t, app.QueryRow(ctx, `select count(*) from users`).Scan(&n))
	// APP_ENV=test: testroutes tự tạo bảng thử bằng vai ứng dụng.
	_, err := app.Exec(ctx, `create table _probe_items (id int)`)
	require.NoError(t, err)
}

func connect(t *testing.T, ctx context.Context, url, user, password string) *pgx.Conn {
	t.Helper()
	cfg, err := pgx.ParseConfig(url)
	require.NoError(t, err)
	if user != "" {
		cfg.User, cfg.Password = user, password
	}
	c, err := pgx.ConnectConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close(context.WithoutCancel(ctx)) })
	return c
}
