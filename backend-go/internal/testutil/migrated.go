package testutil

import (
	"context"
	"io"
	"testing"
	"time"

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
