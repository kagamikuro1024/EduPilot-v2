// Worker EduPilot: vòng lặp nền (relay + consumer outbox cắm vào sau), `/healthz` nội bộ, `-healthcheck`.
package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/db"
	applog "github.com/edupilot/backend-go/internal/platform/log"
	appotel "github.com/edupilot/backend-go/internal/platform/otel"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// shutdownTimeout của worker là 10 s cố định (SRS 8.1).
	shutdownTimeout = 10 * time.Second
	// probeTimeout: hạn của `-healthcheck` và của ping trong `/healthz`.
	probeTimeout = 3 * time.Second
	// heartbeatInterval: nhịp vòng lặp nền; mỗi nhịp mở một span gốc.
	heartbeatInterval = time.Second
	// staleAfter: `/healthz` báo hỏng nếu vòng lặp lỡ nhịp quá lâu.
	staleAfter = 15 * time.Second
)

func main() { os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr)) }

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("worker", flag.ContinueOnError)
	fs.SetOutput(stderr)
	healthcheck := fs.Bool("healthcheck", false, "gọi /healthz cục bộ rồi thoát (Docker healthcheck)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Load(getenv, config.Worker)
	log := applog.NewTo(stdout, cfg, "worker")
	if err != nil {
		logConfigError(log, err)
		return 1
	}
	if *healthcheck {
		return probe(httpURL(cfg.WorkerHealthAddr) + "/healthz")
	}
	appredis.SetLogger(log) // go-redis cũng phải ghi JSON (AC4)
	log.Info("config loaded", cfg.LogAttrs()...)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	return serve(ctx, cfg, log)
}

// serve dựng phụ thuộc rồi chạy vòng lặp nền tới khi ctx huỷ.
func serve(ctx context.Context, cfg config.Config, log *slog.Logger) int {
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

	startCtx, startSpan := appotel.StartRoot(ctx, "worker.startup")
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

	deps := Deps{Cfg: cfg, Log: log, DB: pool, Redis: rdb}
	if err := httpapi.WaitForDeps(startCtx, log, cfg.StartupTimeout, dbDep(deps), redisDep(deps)); err != nil {
		startSpan.End()
		return 1
	}
	startSpan.End()

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", cfg.WorkerHealthAddr)
	if err != nil {
		log.ErrorContext(ctx, "không mở được cổng health", "addr", cfg.WorkerHealthAddr, "error", err.Error())
		return 1
	}
	return runLoop(ctx, deps, ln, newTasks(deps))
}

// runLoop chạy vòng lặp nhịp + các việc nền + `/healthz` trên ln, rồi tắt trong ≤ 10 s.
func runLoop(ctx context.Context, d Deps, ln net.Listener, tasks []Task) int {
	hb := newHeartbeat()
	health := &http.Server{
		Handler:           healthHandler(d, hb),
		ReadHeaderTimeout: httpapi.ReadHeaderTimeout,
		ReadTimeout:       httpapi.ReadTimeout,
		WriteTimeout:      httpapi.WriteTimeout,
		IdleTimeout:       httpapi.IdleTimeout,
		MaxHeaderBytes:    httpapi.MaxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(d.Log.Handler(), slog.LevelWarn),
	}
	go func() {
		if err := health.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			d.Log.ErrorContext(ctx, "health server dừng", "error", err.Error())
		}
	}()

	var wg sync.WaitGroup
	for _, t := range tasks {
		wg.Add(1)
		go func(t Task) {
			defer wg.Done()
			if err := t.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				d.Log.ErrorContext(ctx, "việc nền dừng bất thường", "task", t.Name(), "error", err.Error())
			}
		}(t)
	}

	d.Log.InfoContext(ctx, "worker ready", "health_addr", d.Cfg.WorkerHealthAddr, "tasks", len(tasks))
	tick := time.NewTicker(heartbeatInterval)
	defer tick.Stop()
	for done := false; !done; {
		select {
		case <-ctx.Done():
			done = true
		case <-tick.C:
			// Mỗi nhịp là một span GỐC: dòng log của nhịp có trace_id riêng, khác 0 (AC4).
			tickCtx, span := appotel.StartRoot(ctx, "worker.tick")
			hb.beat()
			d.Log.DebugContext(tickCtx, "worker tick")
			span.End()
		}
	}

	stopCtx, stopSpan := appotel.StartRoot(context.WithoutCancel(ctx), "worker.shutdown")
	defer stopSpan.End()
	sctx, cancel := context.WithTimeout(stopCtx, shutdownTimeout)
	defer cancel()
	_ = health.Shutdown(sctx)

	stopped := make(chan struct{})
	go func() { wg.Wait(); close(stopped) }()
	select {
	case <-stopped:
	case <-sctx.Done():
		d.Log.WarnContext(stopCtx, "forced shutdown", "tasks", len(tasks))
	}
	d.Log.InfoContext(stopCtx, "worker stopped")
	return 0
}

// heartbeat ghi thời điểm nhịp gần nhất của vòng lặp (dùng cho `/healthz`).
type heartbeat struct {
	mu sync.Mutex
	at time.Time
}

func newHeartbeat() *heartbeat { return &heartbeat{at: time.Now()} }

func (h *heartbeat) beat() {
	h.mu.Lock()
	h.at = time.Now()
	h.mu.Unlock()
}

func (h *heartbeat) fresh() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return time.Since(h.at) < staleAfter
}

// healthHandler: 200 khi vòng lặp còn nhịp và DB + Redis ping được (US-PG-01 AC12).
func healthHandler(d Deps, hb *heartbeat) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), probeTimeout)
		defer cancel()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if !hb.fresh() || d.DB.Ping(ctx) != nil || d.Redis.Ping(ctx).Err() != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"not_ready"}` + "\n"))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
	})
	return mux
}

func dbDep(d Deps) httpapi.Dep {
	return httpapi.Dep{Name: "db", Ping: func(ctx context.Context) error { return d.DB.Ping(ctx) }}
}

func redisDep(d Deps) httpapi.Dep {
	return httpapi.Dep{Name: "redis", Ping: func(ctx context.Context) error { return d.Redis.Ping(ctx).Err() }}
}

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

// Deps là phụ thuộc của việc nền trong worker.
type Deps struct {
	Cfg   config.Config
	Log   *slog.Logger
	DB    *pgxpool.Pool
	Redis *appredis.Client
}
