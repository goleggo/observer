package observer

import (
	"context"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel"

	"github.com/goleggo/observer/config"
)

func TestSetup(t *testing.T) {
	old := slog.Default()
	t.Cleanup(func() { slog.SetDefault(old) })
	ctx := context.Background()

	shutdown, err := Setup(ctx, config.Config{
		Log:  config.LogConfig{Level: "debug"},
		OTEL: config.OTELConfig{ServiceName: "svc", ExporterType: "none"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slog.Default().Enabled(ctx, slog.LevelDebug) {
		t.Error("console logger not installed")
	}
	if len(otel.GetTextMapPropagator().Fields()) == 0 {
		t.Error("propagator not installed")
	}
	if err := shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if err := shutdown(ctx); err != nil {
		t.Fatal("second shutdown must be a no-op:", err)
	}
}

func TestSetupFailsCleanly(t *testing.T) {
	old := slog.Default()
	t.Cleanup(func() { slog.SetDefault(old) })

	shutdown, err := Setup(context.Background(), config.Config{
		OTEL: config.OTELConfig{ExporterType: "none", MetricsExporter: "statsd"},
	})
	if err == nil || shutdown != nil {
		t.Fatalf("shutdown set: %t, err %v; want an error and no shutdown", shutdown != nil, err)
	}
}
