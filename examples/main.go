package main

import (
	"context"
	"time"

	"github.com/goleggo/observer/config"
	"github.com/goleggo/observer/log"
	"github.com/goleggo/observer/metrics"
	"github.com/goleggo/observer/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func main() {
	// Config for logging, tracing, and metrics
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "text",
		},
		OTEL: config.OTELConfig{
			ServiceName:  "observer-example",
			ExporterType: "stdout", // or "otlp"
			Endpoint:     "localhost:4317",
			Insecure:     true,
		},
	}

	// Setup logger
	log.SetupLogger(cfg.Log)
	ctx := context.Background()

	// Setup tracer
	if err := trace.SetupTrace(ctx, cfg.OTEL); err != nil {
		log.Error(ctx, "failed to setup tracer", "error", err)
		return
	}

	// Setup metrics
	if err := metrics.SetupMetrics(ctx, cfg.OTEL); err != nil {
		log.Error(ctx, "failed to setup metrics", "error", err)
		return
	}

	// Example span usage
	tracer := otel.Tracer("observer-example")
	ctx, span := tracer.Start(ctx, "example-operation")
	defer span.End()

	// Example metric usage
	meter := otel.GetMeterProvider().Meter("observer-example")
	counter, err := meter.Int64Counter("example_counter")
	if err != nil {
		log.Error(ctx, "failed to create counter", "error", err)
	} else {
		counter.Add(ctx, 1, metric.WithAttributes(attribute.String("key", "value")))
		log.Info(ctx, "Incremented example_counter metric")
	}

	log.Info(ctx, "This is an info log with trace context")
	log.Debug(ctx, "This is a debug log with trace context")
	log.Error(ctx, "This is an error log with trace context")

	time.Sleep(1 * time.Second) // Give time for exporter to flush
}
