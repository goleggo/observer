package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/goleggo/observer/config"
)

func SetupTrace(ctx context.Context, cfg config.OTELConfig) (*trace.TracerProvider, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
		resource.WithFromEnv(),
		resource.WithAttributes(resourceAttrs(cfg.Resource)...),
	)
	if err != nil {
		return nil, err
	}

	var exporter trace.SpanExporter
	switch cfg.ExporterType {
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

func resourceAttrs(attrs map[string]string) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}

	kv := make([]attribute.KeyValue, 0, len(attrs))
	for key, value := range attrs {
		if key == "" {
			continue
		}
		kv = append(kv, attribute.String(key, value))
	}
	return kv
}
