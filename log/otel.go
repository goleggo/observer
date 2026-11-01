package log

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/goleggo/observer/config"
	"github.com/goleggo/observer/internal/otelhelper"
)

// SetupOTELLogger installs a slog default that sends records to the OTEL log exporter and, unless
// logCfg.DisableConsole is set, to stdout as well, so logs survive a collector outage. It returns a nil provider when
// OTEL logs are disabled; call SetupLogger then.
func SetupOTELLogger(ctx context.Context, logCfg config.LogConfig, otelCfg config.OTELConfig) (*sdklog.LoggerProvider, error) {
	kind, err := otelhelper.ExporterType(otelCfg.LogsExporter, otelCfg.ExporterType)
	if err != nil || otelCfg.DisableLogs || kind == otelhelper.ExporterNone {
		return nil, err
	}
	res, err := otelhelper.Resource(ctx, otelCfg)
	if err != nil {
		return nil, err
	}
	exporter, err := newExporter(ctx, kind, otelCfg)
	if err != nil {
		return nil, err
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)
	global.SetLoggerProvider(provider)

	var h slog.Handler = levelFilterHandler{
		handler: otelslog.NewHandler(otelCfg.ServiceName, otelslog.WithLoggerProvider(provider)),
		min:     parseLevel(logCfg.Level),
	}
	if !logCfg.DisableConsole {
		h = slog.NewMultiHandler(NewConsoleHandler(logCfg, os.Stdout), h)
	}
	slog.SetDefault(slog.New(withStackTrace(h)))

	return provider, nil
}

func newExporter(ctx context.Context, kind string, cfg config.OTELConfig) (sdklog.Exporter, error) {
	if kind == otelhelper.ExporterStdout {
		return stdoutlog.New()
	}
	ep, err := otelhelper.ParseEndpoint(cfg.Endpoint, cfg.Insecure, "/v1/logs")
	if err != nil {
		return nil, err
	}
	var opts []otlploghttp.Option
	if ep.IsSet() {
		opts = append(opts, otlploghttp.WithEndpoint(ep.Host))
	}
	if ep.Path != "" {
		opts = append(opts, otlploghttp.WithURLPath(ep.Path))
	}
	if ep.Insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}
	return otlploghttp.New(ctx, opts...)
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
