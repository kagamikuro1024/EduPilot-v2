// Gateway EduPilot: HTTP API (P0 chỉ có /healthz).
package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/platform"
)

const addr = ":8080"

func main() { os.Exit(run()) }

func run() int {
	healthcheck := flag.Bool("healthcheck", false, "gọi /healthz cục bộ rồi thoát (dùng cho Docker healthcheck)")
	flag.Parse()
	if *healthcheck {
		return probe()
	}

	log := platform.NewLogger()
	if _, err := platform.LoadConfig(os.Getenv); err != nil {
		log.Error("cấu hình không hợp lệ", "error", err.Error())
		return 1
	}

	srv := &http.Server{Addr: addr, Handler: httpapi.NewRouter(), ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("gateway đang chạy", "addr", addr)

	select {
	case err := <-errc:
		log.Error("server dừng bất thường", "error", err.Error())
		return 1
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("dừng êm thất bại", "error", err.Error())
		return 1
	}
	log.Info("gateway đã dừng")
	return 0
}

func probe() int {
	c := http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://127.0.0.1" + addr + "/healthz")
	if err != nil {
		return 1
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
