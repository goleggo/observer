package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/goleggo/observer/config"
)

func TestSetupMetrics(t *testing.T) {
	ctx := context.Background()
	mp, err := SetupMetrics(ctx, config.OTELConfig{MetricsExporter: "none", ExporterType: "otlp"})
	if mp != nil || err != nil {
		t.Fatalf("none: got %v, %v", mp, err)
	}
	if _, err := SetupMetrics(ctx, config.OTELConfig{ExporterType: "prometheus"}); err == nil {
		t.Fatal("unknown exporter must fail")
	}

	mp, err = SetupMetrics(ctx, config.OTELConfig{ExporterType: "otlp", Endpoint: "http://127.0.0.1:1", MetricInterval: time.Hour})
	if err != nil || mp == nil {
		t.Fatalf("otlp: got %v, %v", mp, err)
	}
	_ = mp.Shutdown(ctx) // the final export fails: nothing listens on :1
}
