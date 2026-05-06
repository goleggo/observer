package log

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/goleggo/observer/config"
	"github.com/goleggo/observer/internal/otelhelper"
)

func SetupOTELLogger(ctx context.Context, logCfg config.LogConfig, otelCfg config.OTELConfig) (*sdklog.LoggerProvider, error) {
	if otelCfg.DisableLogs || otelhelper.GetExporterType(otelCfg.LogsExporter, otelCfg.ExporterType) == "none" {
		return nil, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(otelCfg.ServiceName),
		),
		resource.WithFromEnv(),
		resource.WithAttributes(otelhelper.ResourceAttrs(otelCfg.Resource)...),
	)
	if err != nil {
		return nil, err
	}

	var exporter sdklog.Exporter
	exporterType := otelhelper.GetExporterType(otelCfg.LogsExporter, otelCfg.ExporterType)

	switch exporterType {
	case "stdout":
		exporter, err = stdoutlog.New()
		if err != nil {
			return nil, err
		}
	case "otlp":
		fallthrough
	default:
		opts := []otlploghttp.Option{}
		if otelCfg.Endpoint != "" {
			opts = append(opts, otlploghttp.WithEndpoint(otelCfg.Endpoint))
		}
		if otelCfg.Insecure {
			opts = append(opts, otlploghttp.WithInsecure())
		}
		exporter, err = otlploghttp.New(ctx, opts...)
		if err != nil {
			return nil, err
		}
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)
	global.SetLoggerProvider(provider)

	handler := otelslog.NewHandler(otelCfg.ServiceName, otelslog.WithLoggerProvider(provider))
	Logger = slog.New(levelFilterHandler{
		handler: handler,
		min:     parseLevel(logCfg.Level),
	})
	slog.SetDefault(Logger)

	return provider, nil
}

type levelFilterHandler struct {
	handler slog.Handler
	min     slog.Level
}

func (h levelFilterHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if level < h.min {
		return false
	}
	return h.handler.Enabled(ctx, level)
}

func (h levelFilterHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.handler.Handle(ctx, record)
}

func (h levelFilterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return levelFilterHandler{handler: h.handler.WithAttrs(attrs), min: h.min}
}

func (h levelFilterHandler) WithGroup(name string) slog.Handler {
	return levelFilterHandler{handler: h.handler.WithGroup(name), min: h.min}
}
