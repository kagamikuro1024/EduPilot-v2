// Lệnh mock-imap chạy máy chủ IMAP thử (cổng 1143) + cổng điều khiển HTTP (8091) cho seed / demo. Chỉ dùng ở compose local / test-seed.
package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/edupilot/backend-go/cmd/mock-imap/server"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	ctl := env("MOCK_IMAP_CTL", ":8091")
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" { // distroless không có curl
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8091/__mock/health", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		_ = resp.Body.Close()
		return
	}
	srv, err := server.New(server.Options{Username: env("MOCK_IMAP_USER", "nop@edupilot.test"), Password: env("MOCK_IMAP_PASSWORD", "mock-imap-password"), AuthServID: env("MOCK_IMAP_AUTHSERV_ID", "mx.edupilot.test")})
	if err != nil {
		slog.Error("mock-imap: không dựng được", "error", err.Error())
		os.Exit(1)
	}
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", env("MOCK_IMAP_ADDR", ":1143"))
	if err != nil {
		slog.Error("mock-imap: không mở được cổng IMAP", "error", err.Error())
		os.Exit(1)
	}
	go func() {
		if err := srv.ServeIMAP(ln); err != nil {
			slog.Error("mock-imap: IMAP dừng", "error", err.Error())
			os.Exit(1)
		}
	}()
	slog.Info("mock-imap sẵn sàng", "imap", ln.Addr().String(), "ctl", ctl)
	hs := &http.Server{Addr: ctl, Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	if err := hs.ListenAndServe(); err != nil {
		slog.Error("mock-imap: cổng điều khiển dừng", "error", err.Error())
		os.Exit(1)
	}
}
