package log

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/goleggo/observer/config"
)

var (
	Logger = slog.Default()
)

func SetupLogger(cfg config.LogConfig) *slog.Logger {
	level := parseLevel(cfg.Level)

	var handler slog.Handler
	if strings.ToLower(cfg.Format) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}

	// Wrap handler with performance-optimized middlewares
	handler = &otelContextHandler{handler}
	handler = &errorHandler{handler}

	Logger = slog.New(handler)
	slog.SetDefault(Logger)
	return Logger
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "warn":
		return slog.LevelWarn
	case "debug":
		return slog.LevelDebug
	case "error":
		return slog.LevelError
	case "info":
		fallthrough
	default:
		return slog.LevelInfo
	}
}

func SetLogger(l *slog.Logger) {
	Logger = l
}

func Info(ctx context.Context, msg string, args ...any) {
	Logger.InfoContext(ctx, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	Logger.ErrorContext(ctx, msg, args...)
}

func Debug(ctx context.Context, msg string, args ...any) {
	Logger.DebugContext(ctx, msg, args...)
}

// otelContextHandler extracts trace/span IDs from context and adds them to the log record.
type otelContextHandler struct {
	slog.Handler
}

func (h *otelContextHandler) Handle(ctx context.Context, r slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		r.AddAttrs(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *otelContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otelContextHandler{h.Handler.WithAttrs(attrs)}
}

func (h *otelContextHandler) WithGroup(name string) slog.Handler {
	return &otelContextHandler{h.Handler.WithGroup(name)}
}

// errorHandler automatically extracts stack traces from error attributes.
type errorHandler struct {
	slog.Handler
}

func (h *errorHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		r.Attrs(func(a slog.Attr) bool {
			if err, ok := a.Value.Any().(error); ok {
				if st := getStackTrace(err); st != "" {
					r.AddAttrs(slog.String("stacktrace", st))
					return false // stop after first error with stacktrace for performance
				}
			}
			return true
		})
	}
	return h.Handler.Handle(ctx, r)
}

func (h *errorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &errorHandler{h.Handler.WithAttrs(attrs)}
}

func (h *errorHandler) WithGroup(name string) slog.Handler {
	return &errorHandler{h.Handler.WithGroup(name)}
}

func getStackTrace(err error) string {
	type stackTracer interface {
		StackTrace() string
	}
	if stErr, ok := err.(stackTracer); ok {
		return stErr.StackTrace()
	}
	return ""
}
