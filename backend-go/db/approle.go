package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// AppRole là vai Postgres của gateway / worker (migration 00030): không superuser, không chủ bảng.
const AppRole = "edupilot_app"

// SetAppRolePassword bật LOGIN và đặt mật khẩu cho AppRole qua kết nối của vai CHỦ (ownerURL). Gọi sau `up`, mỗi lần migrate chạy (idempotent).
// createInSchema = true (chỉ APP_ENV=test) cấp thêm CREATE trên schema public cho bản dựng `testroutes` tự tạo bảng thử lúc khởi động.
func SetAppRolePassword(ctx context.Context, ownerURL, password string, createInSchema bool) error {
	conn, err := pgx.Connect(ctx, ownerURL)
	if err != nil {
		return fmt.Errorf("kết nối Postgres: %w", err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	// ALTER ROLE không nhận tham số ($1): dựng câu lệnh bằng format(%I, %L) ở phía server để mật khẩu luôn được trích dẫn đúng.
	var stmt string
	if err := conn.QueryRow(ctx, `select format('ALTER ROLE %I WITH LOGIN PASSWORD %L', $1::text, $2::text)`, AppRole, password).Scan(&stmt); err != nil {
		return fmt.Errorf("dựng ALTER ROLE: %w", err)
	}
	if _, err := conn.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("đặt mật khẩu vai %s (đã chạy migrate 00030?): %w", AppRole, err)
	}
	if createInSchema {
		if _, err := conn.Exec(ctx, "GRANT CREATE ON SCHEMA public TO "+AppRole); err != nil {
			return fmt.Errorf("cấp CREATE cho vai %s: %w", AppRole, err)
		}
	}
	return nil
}
