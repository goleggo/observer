package log

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/goleggo/observer/config"
)

func SetupOTELLogger(ctx context.Context, logCfg config.LogConfig, otelCfg config.OTELConfig) (*sdklog.LoggerProvider, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(otelCfg.ServiceName),
		),
		resource.WithFromEnv(),
		resource.WithAttributes(resourceAttrs(otelCfg.Resource)...),
	)
	if err != nil {
		return nil, err
	}

	opts := []otlploghttp.Option{}
	if otelCfg.Endpoint != "" {
		opts = append(opts, otlploghttp.WithEndpoint(otelCfg.Endpoint))
	}
	if otelCfg.Insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}

	exporter, err := otlploghttp.New(ctx, opts...)
	if err != nil {
		return nil, err
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
