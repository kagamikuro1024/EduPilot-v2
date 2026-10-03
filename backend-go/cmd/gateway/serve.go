package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/db"
	applog "github.com/edupilot/backend-go/internal/platform/log"
	appotel "github.com/edupilot/backend-go/internal/platform/otel"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

// probeTimeout: hạn của `gateway -healthcheck` (Docker healthcheck).
const probeTimeout = 3 * time.Second

// runServe chạy HTTP gateway: đọc cấu hình → chờ DB/Redis → mở cổng → tắt êm khi SIGTERM.
func runServe(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	healthcheck := fs.Bool("healthcheck", false, "gọi /healthz cục bộ rồi thoát (Docker healthcheck)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Load(getenv, config.Gateway)
	log := applog.NewTo(stdout, cfg, "gateway")
	if err != nil {
		logConfigError(log, err)
		return 1
	}
	if err := refuseTestBuild(cfg); err != nil {
		log.Error(err.Error(), "app_env", cfg.AppEnv)
		return 1
	}
	if *healthcheck {
		return probe(httpURL(cfg.HTTPAddr) + "/healthz")
	}

	appredis.SetLogger(log) // go-redis cũng phải ghi JSON (AC4)

	start := time.Now()
	log.Info("config loaded", cfg.LogAttrs()...)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	shutdownOtel, err := appotel.Setup(ctx, cfg)
	if err != nil {
		log.Error("không dựng được trace", "error", err.Error())
		return 1
	}
	defer func() {
		sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), probeTimeout)
		defer cancel()
		_ = shutdownOtel(sctx)
	}()

	startCtx, startSpan := appotel.StartRoot(ctx, "gateway.startup")

	pool, err := db.NewPool(startCtx, cfg, log)
	if err != nil {
		log.ErrorContext(startCtx, "không dựng được pool Postgres", "error", err.Error())
		startSpan.End()
		return 1
	}
	defer pool.Close()

	rdb, err := appredis.New(startCtx, cfg.RedisURL)
	if err != nil {
		log.ErrorContext(startCtx, "không dựng được client Redis", "error", err.Error())
		startSpan.End()
		return 1
	}
	defer func() { _ = rdb.Close() }()

	deps := httpapi.Deps{Cfg: cfg, Log: log, DB: pool, Redis: rdb, Clock: clock.Real{}, State: httpapi.NewState()}
	if err := httpapi.WaitForDeps(startCtx, log, cfg.StartupTimeout,
		httpapi.DBDep(deps), httpapi.RedisDep(deps)); err != nil {
		startSpan.End()
		return 1
	}

	llmRT, err := llm.NewRuntime(startCtx, cfg, pool, rdb.Client, log)
	if err != nil {
		log.ErrorContext(startCtx, "không dựng được cổng LLM", "error", err.Error())
		startSpan.End()
		return 1
	}
	defer func() {
		cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), probeTimeout)
		defer cancel()
		llmRT.Close(cctx) // đẩy hết llm_audit còn đệm trước khi đóng pool
	}()
	deps.LLM = llmRT

	srv := httpapi.NewServer(deps)
	var lc net.ListenConfig
	ln, err := lc.Listen(startCtx, "tcp", cfg.HTTPAddr)
	if err != nil {
		log.ErrorContext(startCtx, "không mở được cổng", "addr", cfg.HTTPAddr, "error", err.Error())
		startSpan.End()
		return 1
	}
	log.InfoContext(startCtx, "gateway ready", "addr", cfg.HTTPAddr, "startup_ms", time.Since(start).Milliseconds())
	startSpan.End()

	serveErr := httpapi.Serve(ctx, srv, ln, deps)

	stopCtx, stopSpan := appotel.StartRoot(context.WithoutCancel(ctx), "gateway.shutdown")
	defer stopSpan.End()
	if serveErr != nil {
		log.ErrorContext(stopCtx, "gateway dừng bất thường", "error", serveErr.Error())
		return 1
	}
	log.InfoContext(stopCtx, "gateway stopped")
	return 0
}

// logConfigError ghi ĐÚNG MỘT dòng mức error: thiếu biến → mảng `missing` (chỉ tên);
// giá trị sai → tên biến + lý do, không bao giờ in giá trị (US-PG-01 AC1, AC2).
func logConfigError(log *slog.Logger, err error) {
	var missing *config.ErrMissingEnv
	if errors.As(err, &missing) {
		log.Error("thiếu biến môi trường bắt buộc", "missing", missing.Names)
		return
	}
	var invalid *config.ErrInvalidEnv
	if errors.As(err, &invalid) {
		log.Error(invalid.Error(), "invalid", invalid.Names)
		return
	}
	log.Error("cấu hình không hợp lệ", "error", err.Error())
}

// httpURL đổi địa chỉ lắng nghe (`:8080`, `0.0.0.0:8080`) thành URL gọi được từ chính container.
func httpURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://127.0.0.1" + addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

// probe gọi một endpoint health cục bộ; 200 → 0, ngược lại → 1.
func probe(url string) int {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 1
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 1
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
