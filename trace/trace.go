package trace

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/goleggo/observer/config"
	"github.com/goleggo/observer/internal/otelhelper"
)

// SetupTrace installs the W3C trace-context and baggage propagator and the global tracer provider. The propagator is
// set even when traces are disabled, so a service still forwards its callers' traceparent. It returns a nil provider
// when traces are disabled.
func SetupTrace(ctx context.Context, cfg config.OTELConfig) (*trace.TracerProvider, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	if cfg.SampleRatio < 0 || cfg.SampleRatio > 1 {
		return nil, fmt.Errorf("observer: sample ratio %v is outside [0, 1]", cfg.SampleRatio)
	}
	kind, err := otelhelper.ExporterType(cfg.TracesExporter, cfg.ExporterType)
	if err != nil || cfg.DisableTraces || kind == otelhelper.ExporterNone {
		return nil, err
	}
	res, err := otelhelper.Resource(ctx, cfg)
	if err != nil {
		return nil, err
	}
	exporter, err := newExporter(ctx, kind, cfg)
	if err != nil {
		return nil, err
	}

	opts := []trace.TracerProviderOption{trace.WithBatcher(exporter), trace.WithResource(res)}
	if cfg.SampleRatio > 0 {
		opts = append(opts, trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(cfg.SampleRatio))))
	}
	tp := trace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	return tp, nil
}

func newExporter(ctx context.Context, kind string, cfg config.OTELConfig) (trace.SpanExporter, error) {
	if kind == otelhelper.ExporterStdout {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
	ep, err := otelhelper.ParseEndpoint(cfg.Endpoint, cfg.Insecure, "/v1/traces")
	if err != nil {
		return nil, err
	}
	var opts []otlptracehttp.Option
	if ep.IsSet() {
		opts = append(opts, otlptracehttp.WithEndpoint(ep.Host))
	}
	if ep.Path != "" {
		opts = append(opts, otlptracehttp.WithURLPath(ep.Path))
	}
	if ep.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	return otlptracehttp.New(ctx, opts...)
}
