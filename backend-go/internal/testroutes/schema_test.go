//go:build testroutes

package testroutes

import (
	"context"
	"io"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/db"
	"github.com/edupilot/backend-go/internal/testutil"
)

// QC L2.8(b): volume cũ đã có `_test_items` do vai CHỦ tạo; gateway chạy bằng vai ứng dụng (không chủ bảng, không CREATE) thì khởi động không được lỗi 42501.
func TestEnsureSchemaWhenTableOwnedByAnotherRole(t *testing.T) {
	t.Parallel()
	ownerURL := testutil.PostgresURL(t)
	ctx := t.Context()
	require.NoError(t, db.Migrate(ctx, ownerURL, "up", io.Discard))

	owner, err := pgxpool.New(ctx, ownerURL)
	require.NoError(t, err)
	t.Cleanup(owner.Close)
	require.NoError(t, ensureSchema(ctx, Deps{DB: owner})) // lần đầu bằng vai chủ (bản dựng cũ)

	require.NoError(t, db.SetAppRolePassword(ctx, ownerURL, "app-pw-test", false)) // bản thường: không CREATE trên schema
	cfg := owner.Config()
	cfg.ConnConfig.User, cfg.ConnConfig.Password = db.AppRole, "app-pw-test"
	app, err := pgxpool.NewWithConfig(context.WithoutCancel(ctx), cfg)
	require.NoError(t, err)
	t.Cleanup(app.Close)

	require.NoError(t, ensureSchema(ctx, Deps{DB: app}))
}
