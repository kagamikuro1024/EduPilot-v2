package log_test

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/edupilot/backend-go/internal/platform/config"
	applog "github.com/edupilot/backend-go/internal/platform/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var hex32 = regexp.MustCompile(`^[0-9a-f]{32}$`)

type line struct {
	Time     string `json:"time"`
	Level    string `json:"level"`
	Msg      string `json:"msg"`
	Service  string `json:"service"`
	Instance string `json:"instance"`
	TraceID  string `json:"trace_id"`
}

func parse(t *testing.T, raw string) []line {
	t.Helper()
	out := []line{}
	for _, s := range strings.Split(strings.TrimSpace(raw), "\n") {
		if s == "" {
			continue
		}
		var l line
		if err := json.Unmarshal([]byte(s), &l); err != nil {
			t.Fatalf("dòng log không phải JSON: %s (%v)", s, err)
		}
		out = append(out, l)
	}
	return out
}

// TestHandler_AlwaysTraceID: mọi dòng log — kể cả dòng ghi trước khi có span — đều đủ 6 trường
// và có trace_id 32 hex khác toàn 0 (US-PG-01 AC4).
func TestHandler_AlwaysTraceID(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cfg := config.Config{InstanceID: "gw-1", LogLevel: "debug"}
	log := applog.NewTo(&buf, cfg, "gateway")

	log.Info("không có ctx")
	log.InfoContext(context.Background(), "ctx rỗng")
	log.With("k", "v").WarnContext(context.Background(), "qua WithAttrs")

	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	ctx, span := tp.Tracer("test").Start(context.Background(), "root")
	log.ErrorContext(ctx, "trong span")
	span.End()
	spanTrace := span.SpanContext().TraceID().String()

	lines := parse(t, buf.String())
	if len(lines) != 4 {
		t.Fatalf("số dòng = %d, muốn 4: %s", len(lines), buf.String())
	}
	for i, l := range lines {
		if l.Time == "" || l.Level == "" || l.Msg == "" {
			t.Errorf("dòng %d thiếu time/level/msg: %+v", i, l)
		}
		if l.Service != "gateway" || l.Instance != "gw-1" {
			t.Errorf("dòng %d service/instance = %q/%q", i, l.Service, l.Instance)
		}
		if !hex32.MatchString(l.TraceID) {
			t.Errorf("dòng %d trace_id = %q, muốn 32 hex", i, l.TraceID)
		}
		if l.TraceID == strings.Repeat("0", 32) {
			t.Errorf("dòng %d trace_id toàn 0", i)
		}
	}
	if lines[3].TraceID != spanTrace {
		t.Errorf("dòng trong span có trace_id = %s, muốn %s", lines[3].TraceID, spanTrace)
	}
	for _, l := range lines[:3] {
		if l.TraceID == spanTrace {
			t.Errorf("dòng ngoài span không được mang trace_id của span")
		}
	}
}

func TestNewTo_LevelFromConfig(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := applog.NewTo(&buf, config.Config{LogLevel: "warn", InstanceID: "i"}, "worker")
	log.Info("bỏ qua")
	log.Warn("giữ lại")
	lines := parse(t, buf.String())
	if len(lines) != 1 || lines[0].Msg != "giữ lại" {
		t.Fatalf("log = %s", buf.String())
	}
}
