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
	Logger *slog.Logger
)

func SetupLogger(cfg config.LogConfig) *slog.Logger {
	level := parseLevel(cfg.Level)

	var handler slog.Handler
	if strings.ToLower(cfg.Format) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
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
	Logger.Info(msg, append(args, traceAttrs(ctx)...)...)
}

func Error(ctx context.Context, msg string, args ...any) {
	var newArgs []any
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key, val := args[i], args[i+1]
			if err, ok := val.(error); ok {
				newArgs = append(newArgs, key, err)
				if st := getStackTrace(err); st != "" {
					newArgs = append(newArgs, "stacktrace", st)
				}
			} else {
				newArgs = append(newArgs, key, val)
			}
		}
	}
	Logger.Error(msg, append(newArgs, traceAttrs(ctx)...)...)
}

func Debug(ctx context.Context, msg string, args ...any) {
	Logger.Debug(msg, append(args, traceAttrs(ctx)...)...)
}

func traceAttrs(ctx context.Context) []any {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return nil
	}
	return []any{
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("span_id", span.SpanContext().SpanID().String()),
	}
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
