package httpapi

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// traceMiddleware (M1): otelhttp đọc `traceparent` đến hoặc sinh trace mới; mọi span sau nằm dưới nó.
func traceMiddleware(d Deps) func(http.Handler) http.Handler {
	return otelhttp.NewMiddleware("edupilot-"+d.Cfg.Service(),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}))
}

// requestIDMiddleware (M1): đặt `X-Request-Id` (giữ của client nếu hợp lệ, không thì = trace_id)
// và `X-Instance-Id` cho mọi phản hồi (SRS 6.5).
func requestIDMiddleware(d Deps) func(http.Handler) http.Handler {
	valid := regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
	instance := d.Cfg.InstanceID
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-Id")
			if !valid.MatchString(id) {
				id = trace.SpanContextFromContext(r.Context()).TraceID().String()
			}
			w.Header().Set("X-Request-Id", id)
			if instance != "" {
				w.Header().Set("X-Instance-Id", instance)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// recoverMiddleware (M2): panic → log stack + 500 INTERNAL, tiến trình không chết.
func recoverMiddleware(d Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w}
			defer recoverPanic(r.Context(), d, sw, r)
			next.ServeHTTP(sw, r)
		})
	}
}

// recoverPanic chạy trong defer của recoverMiddleware: log stack rồi trả 500 INTERNAL.
// `http.ErrAbortHandler` KHÔNG bị nuốt — panic tiếp để net/http đóng kết nối như quy ước (FR-26).
func recoverPanic(ctx context.Context, d Deps, sw *statusWriter, r *http.Request) {
	rec := recover()
	if rec == nil {
		return
	}
	if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
		panic(rec)
	}
	if d.Log != nil {
		d.Log.ErrorContext(ctx, "panic trong handler",
			"method", r.Method, "path", r.URL.Path, "stack", string(debug.Stack()))
	}
	if !sw.wrote {
		apierr.Write(sw, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
	}
}

// drainingMiddleware (M3b): sau SIGTERM, request MỚI nhận 503 NOT_READY (request đang chạy vẫn
// chạy tiếp tới SHUTDOWN_TIMEOUT). `/api/v1/readyz` tự trả 503 kèm details nên được đi tiếp.
func drainingMiddleware(d Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if d.State == nil || !d.State.Draining() || r.URL.Path == "/api/v1/readyz" {
				next.ServeHTTP(w, r)
				return
			}
			apierr.Write(w, r, apierr.New(http.StatusServiceUnavailable, apierr.NotReady).
				WithDetails(map[string]any{"draining": true}).WithRetryAfter(5))
		})
	}
}

// accessLogMiddleware (M3): một dòng mỗi request; không ghi thân, không ghi truy vấn (tránh PII).
func accessLogMiddleware(d Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r)
			if d.Log == nil {
				return
			}
			d.Log.InfoContext(r.Context(), "request",
				"method", r.Method, "path", r.URL.Path, "status", sw.status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", w.Header().Get("X-Request-Id"))
		})
	}
}

// timeoutMiddleware (M5): gắn deadline REQUEST_TIMEOUT vào ctx của request để DB/Redis bị huỷ theo;
// handler trả về mà chưa ghi gì và ctx đã hết hạn / bị huỷ → 504 DEADLINE_EXCEEDED (SRS 3.4).
func timeoutMiddleware(d Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d.Cfg.RequestTimeout)
			defer cancel()
			sw := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r.WithContext(ctx))
			if !sw.wrote && ctx.Err() != nil {
				apierr.Write(sw, r, apierr.New(http.StatusGatewayTimeout, apierr.DeadlineExceeded))
			}
		})
	}
}

// bodyLimitMiddleware (M5): thân > MAX_BODY_BYTES → 413 PAYLOAD_TOO_LARGE, chạy TRƯỚC kiểm
// `Idempotency-Key` và trước handler (SRS 3.1, #Q-QC-01-7).
func bodyLimitMiddleware(d Deps) func(http.Handler) http.Handler {
	max := d.Cfg.MaxBodyBytes
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > max {
				apierr.Write(w, r, apierr.New(http.StatusRequestEntityTooLarge, apierr.PayloadTooLarge).
					WithDetails(map[string]any{"max_bytes": max}))
				return
			}
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, max)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// statusWriter nhớ mã trạng thái và việc đã ghi hay chưa (middleware cần biết để không ghi đè).
type statusWriter struct {
	http.ResponseWriter
	code  int
	wrote bool
}

func (w *statusWriter) WriteHeader(code int) {
	if w.wrote {
		return
	}
	w.code, w.wrote = code, true
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.code, w.wrote = http.StatusOK, true
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap cho http.ResponseController (SSE cần Flush và gia hạn write deadline).
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *statusWriter) status() int {
	if w.code == 0 {
		return http.StatusOK
	}
	return w.code
}
