// Lệnh mock-graph chạy máy chủ Microsoft Graph giả lập (cổng 8090) cho US-P7-11. Chỉ dùng ở môi trường local / test.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/edupilot/backend-go/cmd/mock-graph/server"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" { // distroless không có curl
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8090/__mock/requests", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		_ = resp.Body.Close()
		return
	}
	addr, dir := os.Getenv("MOCK_GRAPH_ADDR"), os.Getenv("MOCK_GRAPH_DATA")
	if addr == "" {
		addr = ":8090"
	}
	if dir == "" {
		dir = "testdata/mock-graph"
	}
	srv, err := server.New(server.Options{DataDir: dir})
	if err != nil {
		slog.Error("mock-graph: không nạp được dữ liệu", "error", err.Error())
		os.Exit(1)
	}
	slog.Info("mock-graph sẵn sàng", "addr", addr, "data", dir)
	hs := &http.Server{Addr: addr, Handler: srv, ReadHeaderTimeout: 10 * time.Second}
	if err := hs.ListenAndServe(); err != nil {
		slog.Error("mock-graph dừng", "error", err.Error())
		os.Exit(1)
	}
}
