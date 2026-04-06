package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/goleggo/observer/config"
)

func SetupMetrics(ctx context.Context, cfg config.OTELConfig) (*metric.MeterProvider, error) {
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

	var exporter metric.Exporter
	switch cfg.ExporterType {
	case "stdout":
		exporter, err = stdoutmetric.New(stdoutmetric.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
	case "otlp":
		fallthrough
	default:
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		exporter, err = otlpmetrichttp.New(ctx, opts...)
		if err != nil {
			return nil, err
		}
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(provider)
	return provider, nil
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
