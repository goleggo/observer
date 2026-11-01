package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/goleggo/observer/config"
	"github.com/goleggo/observer/internal/otelhelper"
)

// SetupTrace installs the global tracer provider. It returns nil when traces are disabled.
func SetupTrace(ctx context.Context, cfg config.OTELConfig) (*trace.TracerProvider, error) {
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

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)
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
