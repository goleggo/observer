package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/goleggo/observer/config"
	"github.com/goleggo/observer/internal/otelhelper"
)

func SetupTrace(ctx context.Context, cfg config.OTELConfig) (*trace.TracerProvider, error) {
	if cfg.DisableTraces || otelhelper.GetExporterType(cfg.TracesExporter, cfg.ExporterType) == "none" {
		return nil, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
		resource.WithFromEnv(),
		resource.WithAttributes(otelhelper.ResourceAttrs(cfg.Resource)...),
	)
	if err != nil {
		return nil, err
	}

	var exporter trace.SpanExporter
	exporterType := otelhelper.GetExporterType(cfg.TracesExporter, cfg.ExporterType)

	switch exporterType {
	case "stdout":
		exporter, err = stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
		if err != nil {
			return nil, err
		}
	case "otlp":
		fallthrough
	default:
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		exporter, err = otlptracehttp.New(ctx, opts...)
		if err != nil {
			return nil, err
		}
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}
