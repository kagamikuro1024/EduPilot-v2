package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// Giới hạn cố định của tầng HTTP (SRS 4.1 FR-8, 8.1).
const (
	ReadHeaderTimeout = 5 * time.Second
	ReadTimeout       = 15 * time.Second
	WriteTimeout      = 30 * time.Second // chỉ cho server phụ (worker /healthz); gateway dùng REQUEST_TIMEOUT + WriteMargin
	IdleTimeout       = 60 * time.Second
	MaxHeaderBytes    = 64 << 10

	// WriteMargin: thời gian chừa để ghi lỗi 504 DEADLINE_EXCEEDED SAU khi REQUEST_TIMEOUT hết. Nếu WriteTimeout ≤ REQUEST_TIMEOUT
	// thì máy chủ đóng kết nối đúng lúc đang ghi 504 và client nhận "Empty reply" (BUG-P103-1).
	WriteMargin = 5 * time.Second
)

// depProbeInterval: nhịp thử lại phụ thuộc lúc khởi động (SRS 3.4 — mỗi giây một dòng warn).
const depProbeInterval = time.Second

// depProbeTimeout: mỗi lần ping chỉ chờ ngắn để cả vòng (ping + nghỉ) vẫn đúng một nhịp 1 s — go-redis tự thử dial lại
// nhiều lần, nếu chờ trọn 1 s thì vòng giãn ra 2 s và chỉ còn nửa số dòng warn (BUG-PG-4).
const depProbeTimeout = 400 * time.Millisecond

// maxDrainWindow: giữ cổng mở tối đa ngần này sau SIGTERM trước khi Shutdown (Caddy refresh 2 s, SRS 8.2).
const maxDrainWindow = 3 * time.Second

// NewServer dựng http.Server của gateway với đủ giới hạn và router.
func NewServer(d Deps) *http.Server {
	srv := &http.Server{
		Addr:              d.Cfg.HTTPAddr,
		Handler:           NewRouter(d),
		ReadHeaderTimeout: ReadHeaderTimeout,
		ReadTimeout:       ReadTimeout,
		WriteTimeout:      d.Cfg.RequestTimeout + WriteMargin,
		IdleTimeout:       IdleTimeout,
		MaxHeaderBytes:    MaxHeaderBytes,
	}
	if d.Log != nil {
		// net/http ghi lỗi tầng kết nối bằng log.Logger: ép về slog JSON để MỌI dòng vẫn đúng định dạng.
		srv.ErrorLog = slog.NewLogLogger(d.Log.Handler(), slog.LevelWarn)
	}
	return srv
}

// Serve chạy srv trên ln tới khi ctx bị huỷ (SIGTERM), rồi tắt êm (SRS 4.1 FR-7):
// bật cờ draining (readyz 503 NOT_READY, request mới 503, SSE nhận tín hiệu `shutdown`) → giữ cổng mở
// thêm một khoảng ngắn cho Caddy/LB rút bản này ra → `http.Server.Shutdown` chờ request đang chạy.
// Quá SHUTDOWN_TIMEOUT: cắt, log `warn` `forced shutdown` kèm `cancelled_requests`, trả lỗi (thoát mã 1).
func Serve(ctx context.Context, srv *http.Server, ln net.Listener, d Deps) error {
	if d.State != nil {
		srv.Handler = countInFlight(d.State, srv.Handler)
	}
	errc := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	if d.State != nil {
		d.State.Drain() // readyz 503 NOT_READY ngay, SSE gửi `event: shutdown`
	}
	drain := drainWindow(d.Cfg.ShutdownTimeout)
	time.Sleep(drain)

	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), d.Cfg.ShutdownTimeout-drain)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		var n int64
		if d.State != nil {
			n = d.State.InFlight()
		}
		if d.Log != nil {
			d.Log.WarnContext(ctx, "forced shutdown", "cancelled_requests", n)
		}
		_ = srv.Close()
		return fmt.Errorf("cắt %d request quá SHUTDOWN_TIMEOUT: %w", n, err)
	}
	return nil
}

// drainWindow là khoảng giữ cổng mở sau SIGTERM để Caddy (`dynamic a … refresh 2s`, SRS 8.2) kịp
// thấy readyz 503 và ngừng gửi request mới; luôn nằm trong ngân sách SHUTDOWN_TIMEOUT.
func drainWindow(shutdownTimeout time.Duration) time.Duration {
	if d := shutdownTimeout / 4; d < maxDrainWindow {
		return d
	}
	return maxDrainWindow
}

// countInFlight đếm request đang chạy để báo `cancelled_requests` khi phải cắt.
func countInFlight(st *State, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		st.enter()
		defer st.leave()
		next.ServeHTTP(w, r)
	})
}

// Dep là một phụ thuộc phải sẵn sàng trước khi mở cổng (`db`, `redis`).
type Dep struct {
	Name string
	Ping func(context.Context) error
}

// DBDep và RedisDep là hai phụ thuộc chuẩn của gateway/worker.
func DBDep(d Deps) Dep {
	return Dep{Name: "db", Ping: func(ctx context.Context) error { return pingDB(ctx, d) }}
}

// RedisDep trả phụ thuộc Redis của Deps.
func RedisDep(d Deps) Dep {
	return Dep{Name: "redis", Ping: func(ctx context.Context) error { return pingRedis(ctx, d) }}
}

// WaitForDeps thử lại mỗi giây tới khi mọi phụ thuộc ping được hoặc hết timeout (SRS 3.4):
// mỗi lần lỗi ghi `warn` `dependency not ready` kèm `dependency`; quá hạn ghi `error` cùng msg rồi trả lỗi.
func WaitForDeps(ctx context.Context, log *slog.Logger, timeout time.Duration, deps ...Dep) error {
	deadline := time.Now().Add(timeout)
	for {
		tick := time.Now()
		down := probe(ctx, deps)
		if len(down) == 0 {
			return nil
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			for _, name := range down {
				log.ErrorContext(ctx, "dependency not ready", "dependency", name)
			}
			return fmt.Errorf("phụ thuộc chưa sẵn sàng sau %s: %s", timeout, strings.Join(down, ", "))
		}
		for _, name := range down {
			log.WarnContext(ctx, "dependency not ready", "dependency", name)
		}
		select {
		case <-ctx.Done():
		case <-time.After(depProbeInterval - time.Since(tick)):
		}
	}
}

// probe trả tên các phụ thuộc chưa ping được, theo đúng thứ tự khai báo.
func probe(ctx context.Context, deps []Dep) []string {
	var down []string
	for _, dep := range deps {
		pctx, cancel := context.WithTimeout(ctx, depProbeTimeout)
		err := dep.Ping(pctx)
		cancel()
		if err != nil {
			down = append(down, dep.Name)
		}
	}
	return down
}
