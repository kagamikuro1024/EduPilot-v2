package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/user"
)

const adminTimeout = 30 * time.Second

// runAdmin chạy `gateway admin create --email E --name N` (SRS FEAT-account-security 4.2.6): tạo ADMIN đầu tiên.
// Mật khẩu CHỈ từ biến ADMIN_PASSWORD hoặc stdin — không có cờ --password (mọi cờ lạ ⇒ thoát 2). Mã thoát: 0 tạo xong hoặc đã có,
// 1 lỗi (mật khẩu yếu, DB…), 2 dùng sai lệnh.
func runAdmin(args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) int {
	usage := func(msg string) int {
		_, _ = fmt.Fprintln(stderr, "admin: "+msg)
		_, _ = fmt.Fprintln(stderr, "Dùng: gateway admin create --email E --name N   (mật khẩu: biến ADMIN_PASSWORD hoặc stdin)")
		return 2
	}
	if len(args) == 0 || args[0] != "create" {
		return usage("lệnh con phải là `create`.")
	}
	fs := flag.NewFlagSet("admin create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	email := fs.String("email", "", "email của quản trị viên")
	name := fs.String("name", "", "họ tên")
	if err := fs.Parse(args[1:]); err != nil || fs.NArg() > 0 {
		return usage("Dùng biến ADMIN_PASSWORD hoặc stdin.")
	}
	if strings.TrimSpace(*email) == "" || strings.TrimSpace(*name) == "" {
		return usage("cần cả --email và --name.")
	}
	url := strings.TrimSpace(getenv("DATABASE_URL"))
	if url == "" {
		_, _ = fmt.Fprintln(stderr, "admin: thiếu biến môi trường DATABASE_URL")
		return 1
	}
	cost := 12
	if v := strings.TrimSpace(getenv("BCRYPT_COST")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 4 || n > 14 {
			_, _ = fmt.Fprintln(stderr, "admin: BCRYPT_COST cần số nguyên trong 4–14")
			return 1
		}
		cost = n
	}
	password := getenv("ADMIN_PASSWORD")
	if password == "" {
		line, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			_, _ = fmt.Fprintln(stderr, "admin: không đọc được mật khẩu từ stdin")
			return 1
		}
		password = strings.TrimRight(line, "\r\n")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminTimeout)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "admin: DATABASE_URL không hợp lệ")
		return 1
	}
	defer pool.Close()

	created, err := user.CreateAdmin(ctx, pool, clock.Real{}, *email, *name, password, cost)
	var ve *auth.ValidationError
	switch {
	case errors.As(err, &ve):
		for _, p := range ve.Problems {
			_, _ = fmt.Fprintf(stderr, "admin: %s: %s\n", p.Field, p.Message) // không in mật khẩu
		}
		return 1
	case err != nil:
		_, _ = fmt.Fprintln(stderr, "admin: không tạo được quản trị viên:", err)
		return 1
	case !created:
		_, _ = fmt.Fprintln(stdout, "Quản trị viên này đã tồn tại; không đổi gì.")
	default:
		_, _ = fmt.Fprintln(stdout, "Đã tạo quản trị viên.")
	}
	return 0
}
