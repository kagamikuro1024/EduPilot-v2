// Package otel: trace OpenTelemetry. Tôn trọng `traceparent` đến; xuất OTLP/HTTP khi có
// OTEL_EXPORTER_OTLP_ENDPOINT, không thì chỉ sinh id (SRS 4.1 FR-5).
package otel

import (
	"context"

	"github.com/edupilot/backend-go/internal/platform/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// scope là tên instrumentation của mã EduPilot.
const scope = "github.com/edupilot/backend-go"

// Setup cài tracer provider + propagator W3C toàn tiến trình và trả hàm tắt.
func Setup(ctx context.Context, cfg config.Config) (func(context.Context) error, error) {
	opts := []sdktrace.TracerProviderOption{sdktrace.WithResource(resource(cfg))}
	if cfg.OTLPEndpoint != "" {
		exp, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithBatcher(exp))
	}
	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{}))
	return tp.Shutdown, nil
}

// Tracer trả tracer dùng chung của mã EduPilot.
func Tracer() trace.Tracer { return otel.GetTracerProvider().Tracer(scope) }

// StartRoot mở một span GỐC (không nối vào span cha): khởi động, dừng, mỗi nhịp worker.
func StartRoot(ctx context.Context, name string) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name, trace.WithNewRoot())
}

// resource mô tả dịch vụ trong mọi span (service.name = gateway | worker).
func resource(cfg config.Config) *sdkresource.Resource {
	return sdkresource.NewSchemaless(
		attribute.String("service.name", "edupilot-"+cfg.Service()),
		attribute.String("service.instance.id", cfg.InstanceID),
		attribute.String("deployment.environment", cfg.AppEnv),
	)
}
