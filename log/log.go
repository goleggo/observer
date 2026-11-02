// Package log sets slog up for a service. The helpers log through slog.Default, so a logger set by SetupLogger,
// SetupOTELLogger or SetLogger is safe to swap while other goroutines log.
package log

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/goleggo/observer/config"
)

// SetupLogger installs a stdout logger (text, or json when cfg.Format is "json") as slog's default.
func SetupLogger(cfg config.LogConfig) *slog.Logger {
	l := slog.New(withStackTrace(NewConsoleHandler(cfg, os.Stdout)))
	slog.SetDefault(l)
	return l
}

// NewConsoleHandler returns a text or json handler that adds trace_id and span_id from the context.
func NewConsoleHandler(cfg config.LogConfig, w io.Writer) slog.Handler {
	opts := &slog.HandlerOptions{Level: parseLevel(cfg.Level)}
	var h slog.Handler = slog.NewTextHandler(w, opts)
	if strings.EqualFold(cfg.Format, "json") {
		h = slog.NewJSONHandler(w, opts)
	}
	return otelContextHandler{h}
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	}
	return slog.LevelInfo
}

// SetLogger installs l as slog's default.
func SetLogger(l *slog.Logger) { slog.SetDefault(l) }

func Debug(ctx context.Context, msg string, args ...any) {
	slog.Default().DebugContext(ctx, msg, args...)
}
func Info(ctx context.Context, msg string, args ...any) {
	slog.Default().InfoContext(ctx, msg, args...)
}
func Warn(ctx context.Context, msg string, args ...any) {
	slog.Default().WarnContext(ctx, msg, args...)
}
func Error(ctx context.Context, msg string, args ...any) {
	slog.Default().ErrorContext(ctx, msg, args...)
}

type otelContextHandler struct {
	slog.Handler
}

func (h otelContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h otelContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return otelContextHandler{h.Handler.WithAttrs(attrs)}
}

func (h otelContextHandler) WithGroup(name string) slog.Handler {
	return otelContextHandler{h.Handler.WithGroup(name)}
}

// StackTracer is the interface an error implements to put its stack trace in an error record.
type StackTracer interface {
	error
	StackTrace() string
}

type formattedError interface {
	error
	fmt.Formatter
}

func withStackTrace(h slog.Handler) slog.Handler { return stackTraceHandler{h} }

// stackTraceHandler adds a "stacktrace" attribute to error records from the first error attribute that carries one:
// a StackTracer, or else an error that implements fmt.Formatter (github.com/pkg/errors,
// github.com/cockroachdb/errors), rendered with %+v.
type stackTraceHandler struct {
	slog.Handler
}

func (h stackTraceHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < slog.LevelError {
		return h.Handler.Handle(ctx, r)
	}
	var st string
	r.Attrs(func(a slog.Attr) bool {
		if a.Value.Kind() != slog.KindAny {
			return true
		}
		if err, ok := a.Value.Any().(error); ok {
			st = stackTrace(err)
		}
		return st == ""
	})
	if st != "" {
		r = r.Clone()
		r.AddAttrs(slog.String("stacktrace", st))
	}
	return h.Handler.Handle(ctx, r)
}

func (h stackTraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return stackTraceHandler{h.Handler.WithAttrs(attrs)}
}

func (h stackTraceHandler) WithGroup(name string) slog.Handler {
	return stackTraceHandler{h.Handler.WithGroup(name)}
}

func stackTrace(err error) string {
	if st, ok := errors.AsType[StackTracer](err); ok {
		return st.StackTrace()
	}
	if f, ok := errors.AsType[formattedError](err); ok {
		return fmt.Sprintf("%+v", f)
	}
	return ""
}
