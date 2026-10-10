package testutil

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/db"
)

// MigratedPostgresURL trả URL tới một database mới đã chạy `migrate up` (schema 00001 đầy đủ).
func MigratedPostgresURL(t testing.TB) string {
	t.Helper()
	url := PostgresURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := db.Migrate(ctx, url, "up", io.Discard); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	return url
}

// RuntimePool trả pool tới một database mới đã migrate, cấu hình ĐÚNG như runtime (`QueryExecModeExec` để chạy qua PgBouncer transaction mode):
// test dùng pool mặc định của pgx che mất lỗi mã hoá tham số (vd. `[]uuid.UUID` rỗng → "unable to encode … OID 0"; QC BUG-1 của US-P3-05).
func RuntimePool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(MigratedPostgresURL(t))
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
