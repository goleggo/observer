package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/goleggo/observer/config"
)

func SetupTrace(ctx context.Context, cfg config.OTELConfig) error {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
		resource.WithFromEnv(),
		resource.WithAttributes(resourceAttrs(cfg.Resource)...),
	)
	if err != nil {
		return err
	}

	var exporter trace.SpanExporter
	switch cfg.ExporterType {
	case "stdout":
		exporter, err = stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
		if err != nil {
			return err
		}
	case "otlp":
		fallthrough
	default:
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()))
		}
		exporter, err = otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return err
		}
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return nil
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
