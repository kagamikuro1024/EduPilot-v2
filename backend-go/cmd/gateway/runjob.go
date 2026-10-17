package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
)

// runJob chạy MỘT lần job nền: `gateway admin run-job ticket.sla|ticket.autoclose [--now RFC3339]` (US-P4-05, QC chạy ca "quá hạn" không chờ nhịp).
// Chỉ `APP_ENV` = dev | test (production, staging, rỗng → thoát 1, không ghi gì). `--now` thay đồng hồ của lần chạy này, không đổi đồng hồ của gateway / worker đang chạy.
// SLATick ghi giai đoạn có điều kiện trong cùng giao dịch với outbox nên chạy song song với worker không gửi trùng.
func runJob(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	usage := func(msg string) int {
		_, _ = fmt.Fprintln(stderr, "admin run-job: "+msg)
		_, _ = fmt.Fprintln(stderr, "Dùng: gateway admin run-job ticket.sla|ticket.autoclose [--now 2026-10-13T09:30:00+07:00]")
		return 2
	}
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return usage("thiếu tên job.")
	}
	name := args[0]
	if name != "ticket.sla" && name != "ticket.autoclose" {
		return usage("job không có: " + name)
	}
	fs := flag.NewFlagSet("admin run-job", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	now := fs.String("now", "", "đồng hồ của lần chạy (RFC 3339)")
	if err := fs.Parse(args[1:]); err != nil || fs.NArg() > 0 {
		return usage("cờ không hợp lệ.")
	}
	var clk clock.Clock = clock.Real{}
	if *now != "" {
		t, err := time.Parse(time.RFC3339, *now)
		if err != nil {
			return usage("--now phải theo RFC 3339.")
		}
		clk = clock.NewFake(t)
	}
	if env := getenv("APP_ENV"); env != "dev" && env != "test" {
		_, _ = fmt.Fprintf(stderr, "admin run-job: chỉ chạy khi APP_ENV là dev hoặc test (hiện: %q)\n", env)
		return 1
	}
	cfg, err := config.Load(getenv, config.Gateway)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "admin run-job: cấu hình không hợp lệ:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), adminTimeout)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "admin run-job: DATABASE_URL không hợp lệ")
		return 1
	}
	defer pool.Close()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	// Notify bắt buộc: thiếu nó `remindClaimed` bỏ qua im lặng (TC-21). Không đẩy SSE (không Redis): chuông nhận qua đọc bù.
	svc := &escalation.Service{Pool: pool, Members: course.Resolver{Pool: pool}, Log: log, Cfg: escalation.ConfigFrom(cfg), Clock: clk, Notify: notify.New(pool, nil, log)}
	switch name {
	case "ticket.sla":
		if err := svc.SLATick(ctx); err != nil {
			_, _ = fmt.Fprintln(stderr, "admin run-job: ticket.sla lỗi:", err)
			return 1
		}
		_, _ = fmt.Fprintln(stdout, "ticket.sla xong")
	case "ticket.autoclose":
		n, err := svc.AutoClose(ctx)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "admin run-job: ticket.autoclose lỗi:", err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "ticket.autoclose xong: %d ticket đã đóng\n", n)
	}
	return 0
}
