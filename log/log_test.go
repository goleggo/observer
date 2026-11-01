package log

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/goleggo/observer/config"
)

type tracedErr struct{ msg, stack string }

func (e tracedErr) Error() string      { return e.msg }
func (e tracedErr) StackTrace() string { return e.stack }

type formatterErr struct{}

func (formatterErr) Error() string { return "boom" }
func (formatterErr) Format(s fmt.State, verb rune) {
	if verb == 'v' && s.Flag('+') {
		_, _ = io.WriteString(s, "boom\nmain.f\n\tmain.go:1")
		return
	}
	_, _ = io.WriteString(s, "boom")
}

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(withStackTrace(NewConsoleHandler(config.LogConfig{Level: "debug", Format: "json"}, buf)))
}

func TestStackTrace(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
		err   error
		want  string
	}{
		{name: "StackTracer", level: slog.LevelError, err: tracedErr{"x", "frame-1"}, want: "frame-1"},
		{name: "wrapped StackTracer", level: slog.LevelError, err: fmt.Errorf("ctx: %w", tracedErr{"x", "frame-2"}), want: "frame-2"},
		{name: "fmt.Formatter error", level: slog.LevelError, err: formatterErr{}, want: "main.go:1"},
		{name: "plain error has none", level: slog.LevelError, err: errors.New("plain")},
		{name: "below error level has none", level: slog.LevelWarn, err: tracedErr{"x", "frame-3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			newTestLogger(&buf).Log(context.Background(), tt.level, "failed", "count", 3, "error", tt.err)
			got := buf.String()
			if tt.want == "" {
				if strings.Contains(got, "stacktrace") {
					t.Fatalf("unexpected stacktrace: %s", got)
				}
				return
			}
			if !strings.Contains(got, `"stacktrace":`) || !strings.Contains(got, tt.want) {
				t.Fatalf("want stacktrace with %q, got %s", tt.want, got)
			}
		})
	}
}

func TestTraceIDs(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{TraceID: trace.TraceID{0xab}, SpanID: trace.SpanID{0xcd}})
	var buf bytes.Buffer
	newTestLogger(&buf).With("k", "v").InfoContext(trace.ContextWithSpanContext(context.Background(), sc), "hi")
	for _, want := range []string{sc.TraceID().String(), sc.SpanID().String(), `"k":"v"`} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("missing %s in %s", want, buf.String())
		}
	}
}

func TestLevel(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(NewConsoleHandler(config.LogConfig{Level: "warn"}, &buf))
	l.Info("hidden")
	l.Warn("shown")
	if strings.Contains(buf.String(), "hidden") || !strings.Contains(buf.String(), "shown") {
		t.Fatalf("level filter broken: %s", buf.String())
	}
}

func TestSetupOTELLoggerKeepsConsoleWhenCollectorIsDown(t *testing.T) {
	old := slog.Default()
	t.Cleanup(func() { slog.SetDefault(old) })

	out := captureStdout(t, func() {
		ctx := context.Background()
		lp, err := SetupOTELLogger(ctx, config.LogConfig{Level: "info"}, config.OTELConfig{ServiceName: "svc", ExporterType: "otlp", Endpoint: "127.0.0.1:1", Insecure: true})
		if err != nil {
			t.Fatal(err)
		}
		Info(ctx, "hello console")
		Debug(ctx, "below level")
		_ = lp.Shutdown(ctx) // fails: the collector is down on purpose
	})
	if !strings.Contains(out, "level=INFO msg=\"hello console\"") {
		t.Fatalf("console line missing:\n%s", out)
	}
	if strings.Contains(out, "below level") {
		t.Fatalf("debug record leaked:\n%s", out)
	}
}

func TestSetupOTELLoggerDisabled(t *testing.T) {
	lp, err := SetupOTELLogger(context.Background(), config.LogConfig{}, config.OTELConfig{DisableLogs: true})
	if lp != nil || err != nil {
		t.Fatalf("got %v, %v", lp, err)
	}
	if _, err := SetupOTELLogger(context.Background(), config.LogConfig{}, config.OTELConfig{LogsExporter: "kafka"}); err == nil {
		t.Fatal("unknown exporter must fail")
	}
}

func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() { b, _ := io.ReadAll(r); done <- string(b) }()
	defer func() { os.Stdout = old }()
	f()
	_ = w.Close()
	return <-done
}
