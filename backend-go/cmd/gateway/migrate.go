package main

import (
	"context"
	"fmt"
	"io"
	"os/signal"
	"syscall"
	"time"

	"github.com/edupilot/backend-go/db"
)

const migrateTimeout = 5 * time.Minute

// runMigrate chạy `gateway migrate [up|down|status]` (mặc định `up`) qua DATABASE_URL — kết nối
// TRỰC TIẾP tới Postgres, không qua PgBouncer (SRS 8.3). Trả 0 khi thành công, 1 khi lỗi.
func runMigrate(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	cmd := "up"
	switch len(args) {
	case 0:
	case 1:
		cmd = args[0]
	default:
		fmt.Fprintln(stderr, "migrate: dùng `gateway migrate [up|down|status]`")
		return 1
	}

	url := getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(stderr, "migrate: thiếu biến môi trường DATABASE_URL")
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, migrateTimeout)
	defer cancel()

	if err := db.Migrate(ctx, url, cmd, stdout); err != nil {
		fmt.Fprintf(stderr, "migrate: %v\n", err)
		return 1
	}
	return 0
}
