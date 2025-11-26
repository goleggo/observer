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

// SetupLogger initializes the slog logger with options from config.
func SetupLogger(cfg config.LogConfig) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "error":
		level = slog.LevelError
	case "info":
		fallthrough
	default:
		level = slog.LevelInfo
	}

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

// SetLogger allows setting a custom slog.Logger instance.
func SetLogger(l *slog.Logger) {
	Logger = l
}

// Info logs an info message with optional trace/span context.
func Info(ctx context.Context, msg string, args ...any) {
	Logger.Info(msg, append(args, traceAttrs(ctx)...)...)
}

// Error logs an error message with optional trace/span context.
func Error(ctx context.Context, msg string, args ...any) {
	// Check if any argument is an error and add stack trace if present
	var newArgs []any
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key, val := args[i], args[i+1]
			if err, ok := val.(error); ok {
				newArgs = append(newArgs, key, err)
				// Add stack trace if available (for wrapped errors)
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

// Debug logs a debug message with optional trace/span context.
func Debug(ctx context.Context, msg string, args ...any) {
	Logger.Debug(msg, append(args, traceAttrs(ctx)...)...)
}

// traceAttrs extracts trace and span IDs from context if present.
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

// getStackTrace tries to extract a stack trace from an error if available.
func getStackTrace(err error) string {
	type stackTracer interface {
		StackTrace() string
	}
	if stErr, ok := err.(stackTracer); ok {
		return stErr.StackTrace()
	}
	return ""
}
