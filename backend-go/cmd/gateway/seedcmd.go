package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/crm"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/seedtool"
)

// seedEnvs là DANH SÁCH CHO PHÉP của ba lệnh seed (FEAT-crm-360 SRS 4.10, TLR-13, proposals #35): cấu hình thật chỉ có dev | test | production; mọi giá trị khác — production, lạ, rỗng — bị từ chối.
var seedEnvs = []string{"dev", "test"} //nolint:gochecknoglobals // danh sách cho phép, bất biến

// isSeedCmd: tên lệnh con thuộc nhóm seed.
func isSeedCmd(name string) bool {
	return name == "seed-study-time" || name == "seed-age" || name == "crm-risk-run" || name == "seed-chat"
}

// runSeedCmd chạy `gateway admin seed-study-time|seed-age|crm-risk-run|seed-chat`. Mã thoát: 0 xong (in số dòng đã đổi), 1 lỗi hoặc môi trường không được phép, 2 dùng sai lệnh.
func runSeedCmd(args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) int {
	name := args[0]
	fail := func(code int, msg string) int {
		_, _ = fmt.Fprintf(stderr, "admin %s: %s\n", name, msg)
		return code
	}
	fs := flag.NewFlagSet("admin "+name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	course := fs.String("course", "", "mã lớp (class_code)")
	weeks := fs.Int("weeks", 10, "số tuần")
	profile := fs.String("profile", "", "tệp hồ sơ thời gian học (JSON); `-` đọc từ stdin (gateway chạy trong container không thấy thư mục seed/)")
	chatPlan := fs.String("plan", "", "kế hoạch lịch sử chat (JSON); `-` đọc từ stdin (với seed-chat)")
	tickets := fs.Bool("tickets", false, "lùi mốc thời gian của ticket")
	ticket := fs.String("ticket", "", "id ticket cần lùi (với --tickets)")
	hours := fs.Int("hours", 192, "số giờ lùi (với --tickets)")
	activity := fs.Bool("activity", false, "rải tin chat / thread qua các tuần trước")
	now := fs.String("now", "", "đồng hồ của lần chạy (RFC 3339)")
	if err := fs.Parse(args[1:]); err != nil || fs.NArg() > 0 {
		return fail(2, "cờ không hợp lệ.")
	}
	if !slices.Contains(seedEnvs, getenv("APP_ENV")) { // kiểm TRƯỚC mọi việc khác, kể cả đọc DATABASE_URL
		return fail(1, fmt.Sprintf("chỉ chạy khi APP_ENV thuộc %s (hiện: %q)", strings.Join(seedEnvs, ", "), getenv("APP_ENV")))
	}
	if strings.TrimSpace(*course) == "" {
		return fail(2, "thiếu --course.")
	}
	clk := time.Now()
	if *now != "" {
		t, err := time.Parse(time.RFC3339, *now)
		if err != nil {
			return fail(2, "--now phải theo RFC 3339.")
		}
		clk = t
	}
	var body []byte
	switch name {
	case "seed-study-time":
		if *profile == "" {
			return fail(2, "thiếu --profile.")
		}
		var raw []byte
		var err error
		if *profile == "-" {
			raw, err = io.ReadAll(io.LimitReader(stdin, 1<<20))
		} else {
			raw, err = os.ReadFile(*profile)
		}
		if err != nil {
			return fail(1, "không đọc được hồ sơ: "+err.Error())
		}
		body = raw
	case "seed-chat":
		if *chatPlan != "-" {
			return fail(2, "seed-chat chỉ nhận --plan - (JSON qua stdin).")
		}
		raw, err := io.ReadAll(io.LimitReader(stdin, 4<<20))
		if err != nil {
			return fail(1, "không đọc được kế hoạch: "+err.Error())
		}
		body = raw
	case "seed-age":
		if *tickets == *activity {
			return fail(2, "chọn đúng một trong --tickets hoặc --activity.")
		}
		if *tickets && *ticket == "" {
			return fail(2, "--tickets cần --ticket <id>.")
		}
	}
	url := strings.TrimSpace(getenv("DATABASE_URL"))
	if url == "" {
		return fail(1, "thiếu biến môi trường DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return fail(1, "DATABASE_URL không hợp lệ")
	}
	defer pool.Close()
	cid, err := seedtool.CourseID(ctx, pool, strings.TrimSpace(*course))
	if err != nil {
		return fail(1, err.Error())
	}
	switch name {
	case "seed-study-time":
		p, err := seedtool.ParseProfile(body)
		if err != nil {
			return fail(1, err.Error())
		}
		n, err := seedtool.SeedStudyTime(ctx, pool, cid, *weeks, p, clk)
		if err != nil {
			return fail(1, err.Error())
		}
		_, _ = fmt.Fprintf(stdout, "seed-study-time: %d dòng\n", n)
	case "seed-chat":
		p, err := seedtool.ParseChatPlan(body)
		if err != nil {
			return fail(1, err.Error())
		}
		n, err := seedtool.SeedChat(ctx, pool, cid, *weeks, p, clk)
		if err != nil {
			return fail(1, err.Error())
		}
		_, _ = fmt.Fprintf(stdout, "seed-chat: %d tin\n", n)
	case "seed-age":
		var n int64
		if *tickets {
			tid, perr := uuid.Parse(*ticket)
			if perr != nil {
				return fail(2, "--ticket không phải UUID.")
			}
			n, err = seedtool.AgeTickets(ctx, pool, cid, tid, *hours)
		} else {
			n, err = seedtool.AgeActivity(ctx, pool, cid, *weeks, clk)
		}
		if err != nil {
			return fail(1, err.Error())
		}
		_, _ = fmt.Fprintf(stdout, "seed-age: %d dòng\n", n)
	case "crm-risk-run":
		svc := &crm.Service{Pool: pool, Clock: clock.Real{}, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
		if err := svc.RunCourse(ctx, cid, clk); err != nil {
			return fail(1, err.Error())
		}
		var atRisk, watch int
		_ = pool.QueryRow(ctx, `select count(*) filter (where level = 'AT_RISK'), count(*) filter (where level = 'WATCH') from student_risk where course_id = $1`, cid).Scan(&atRisk, &watch)
		_, _ = fmt.Fprintf(stdout, "crm-risk-run: AT_RISK %d, WATCH %d\n", atRisk, watch)
	}
	return 0
}
