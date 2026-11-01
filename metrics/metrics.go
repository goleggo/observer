package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/goleggo/observer/config"
	"github.com/goleggo/observer/internal/otelhelper"
)

// SetupMetrics installs the global meter provider. It returns nil when metrics are disabled.
func SetupMetrics(ctx context.Context, cfg config.OTELConfig) (*metric.MeterProvider, error) {
	kind, err := otelhelper.ExporterType(cfg.MetricsExporter, cfg.ExporterType)
	if err != nil || cfg.DisableMetrics || kind == otelhelper.ExporterNone {
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

	provider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(provider)
	return provider, nil
}

func newExporter(ctx context.Context, kind string, cfg config.OTELConfig) (metric.Exporter, error) {
	if kind == otelhelper.ExporterStdout {
		return stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	}
	ep, err := otelhelper.ParseEndpoint(cfg.Endpoint, cfg.Insecure, "/v1/metrics")
	if err != nil {
		return nil, err
	}
	var opts []otlpmetrichttp.Option
	if ep.IsSet() {
		opts = append(opts, otlpmetrichttp.WithEndpoint(ep.Host))
	}
	if ep.Path != "" {
		opts = append(opts, otlpmetrichttp.WithURLPath(ep.Path))
	}
	if ep.Insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}
	return otlpmetrichttp.New(ctx, opts...)
}
