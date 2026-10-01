// Package db nhúng migration goose vào binary và chạy chúng qua kết nối Postgres TRỰC TIẾP
// (không qua PgBouncer — US-PG-02 AC7): `gateway migrate up|down|status`.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"io/fs"

	_ "github.com/jackc/pgx/v5/stdlib" // driver "pgx" cho database/sql (goose cần *sql.DB)
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// FS trả cây migration đã nhúng (gốc là thư mục migrations) — dùng cho test và sqlc.
func FS() fs.FS {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil { // không xảy ra: thư mục cố định ở trên
		panic(err)
	}
	return sub
}

// Migrate chạy một lệnh migration. cmd: "up" | "down" | "status".
// "down" lùi về version 0 (xoá mọi object của 00001, giữ extension vector).
func Migrate(ctx context.Context, databaseURL string, cmd string, out io.Writer) error {
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("mở kết nối Postgres: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("kết nối Postgres: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, FS())
	if err != nil {
		return fmt.Errorf("khởi tạo goose: %w", err)
	}

	switch cmd {
	case "up":
		results, err := provider.Up(ctx)
		if err != nil {
			return fmt.Errorf("goose up: %w", err)
		}
		for _, r := range results {
			fmt.Fprintf(out, "goose: applied %s (%s)\n", r.Source.Path, r.Duration)
		}
		version, err := provider.GetDBVersion(ctx)
		if err != nil {
			return fmt.Errorf("đọc version: %w", err)
		}
		fmt.Fprintf(out, "goose: up to date, version=%d (%d migration mới)\n", version, len(results))
	case "down":
		results, err := provider.DownTo(ctx, 0)
		if err != nil {
			return fmt.Errorf("goose down: %w", err)
		}
		for _, r := range results {
			fmt.Fprintf(out, "goose: rolled back %s (%s)\n", r.Source.Path, r.Duration)
		}
		fmt.Fprintf(out, "goose: version=0 (%d migration đã lùi)\n", len(results))
	case "status":
		statuses, err := provider.Status(ctx)
		if err != nil {
			return fmt.Errorf("goose status: %w", err)
		}
		for _, s := range statuses {
			applied := "pending"
			if s.State == goose.StateApplied {
				applied = s.AppliedAt.UTC().Format("2006-01-02 15:04:05 MST")
			}
			fmt.Fprintf(out, "%-8d %-28s %s\n", s.Source.Version, s.Source.Path, applied)
		}
	default:
		return fmt.Errorf("lệnh migrate không hợp lệ: %q (dùng up|down|status)", cmd)
	}
	return nil
}
