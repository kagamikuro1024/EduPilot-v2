// Package log: slog JSON ra stdout; mọi dòng có time, level, msg, service, instance, trace_id (SRS 4.1 FR-4).
package log

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"os"

	"github.com/edupilot/backend-go/internal/platform/config"
	"go.opentelemetry.io/otel/trace"
)

// New trả logger ghi ra stdout cho một dịch vụ (`gateway` | `worker`).
func New(cfg config.Config, service string) *slog.Logger { return NewTo(os.Stdout, cfg, service) }

// NewTo như New nhưng ghi ra w (test, hoặc stdout do lệnh truyền vào).
func NewTo(w io.Writer, cfg config.Config, service string) *slog.Logger {
	instance := cfg.InstanceID
	if instance == "" {
		instance, _ = os.Hostname()
	}
	if instance == "" {
		instance = service
	}
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level(cfg.LogLevel)})
	return slog.New(&traceHandler{Handler: h.WithAttrs([]slog.Attr{
		slog.String("service", service), slog.String("instance", instance),
	}), fallback: randomTraceID()})
}

func level(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// traceHandler gắn `trace_id` vào mọi dòng: id của span trong ctx, hoặc id dự phòng của tiến trình
// (dòng ghi trước khi có span vẫn phải là 32 hex khác toàn 0 — US-PG-01 AC4).
type traceHandler struct {
	slog.Handler
	fallback string
}

func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	id := h.fallback
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		id = sc.TraceID().String()
	}
	r.AddAttrs(slog.String("trace_id", id))
	return h.Handler.Handle(ctx, r)
}

func (h *traceHandler) WithAttrs(as []slog.Attr) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithAttrs(as), fallback: h.fallback}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithGroup(name), fallback: h.fallback}
}

func randomTraceID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[0] |= 1 // không bao giờ toàn số 0
	return hex.EncodeToString(b[:])
}
