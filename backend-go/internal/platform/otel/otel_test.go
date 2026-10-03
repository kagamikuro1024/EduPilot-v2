package otel_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edupilot/backend-go/internal/platform/config"
	applog "github.com/edupilot/backend-go/internal/platform/log"
	appotel "github.com/edupilot/backend-go/internal/platform/otel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// TestStartupHasTrace: khởi động (không có traceparent đến) vẫn có span gốc, nên dòng log khởi động
// mang trace_id 32 hex khác 0 và hai span gốc khác nhau có trace khác nhau (US-PG-01 AC4).
func TestStartupHasTrace(t *testing.T) {
	cfg := config.Config{Role: config.Gateway, InstanceID: "gw-test", AppEnv: "test"}
	shutdown, err := appotel.Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	var buf bytes.Buffer
	log := applog.NewTo(&buf, cfg, "gateway")

	ctx, span := appotel.StartRoot(context.Background(), "gateway.startup")
	log.InfoContext(ctx, "gateway ready", "startup_ms", 12)
	span.End()
	startup := span.SpanContext().TraceID()

	ctx2, span2 := appotel.StartRoot(ctx, "gateway.shutdown")
	log.InfoContext(ctx2, "gateway stopped")
	span2.End()

	if !startup.IsValid() {
		t.Fatalf("span khởi động không có trace id hợp lệ")
	}
	if startup == span2.SpanContext().TraceID() {
		t.Fatalf("StartRoot phải mở trace MỚI, không nối vào span cha")
	}

	var l struct {
		TraceID string `json:"trace_id"`
	}
	first := strings.SplitN(strings.TrimSpace(buf.String()), "\n", 2)[0]
	if err := json.Unmarshal([]byte(first), &l); err != nil {
		t.Fatalf("log không phải JSON: %s", first)
	}
	if l.TraceID != startup.String() {
		t.Fatalf("trace_id của dòng khởi động = %s, muốn %s", l.TraceID, startup)
	}
}

// TestSetup_PropagatesTraceparent: traceparent đến được tôn trọng (SRS 4.1 FR-5).
func TestSetup_PropagatesTraceparent(t *testing.T) {
	cfg := config.Config{Role: config.Gateway, AppEnv: "test"}
	shutdown, err := appotel.Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	r := httptest.NewRequest(http.MethodGet, "/api/v1/khong-co", nil)
	r.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	_, span := appotel.Tracer().Start(ctx, "GET /api/v1/khong-co")
	defer span.End()
	if got := span.SpanContext().TraceID().String(); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("trace_id = %s, muốn kế thừa từ traceparent", got)
	}
}
