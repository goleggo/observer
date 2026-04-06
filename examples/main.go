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
	// How to use:
	// - Set OTEL_RESOURCE_ATTRIBUTES or cfg.OTEL.Resource for common attributes.
	// - Logs use OTLP/HTTP (default collector port 4318).
	// - Traces/metrics use OTLP/gRPC (default collector port 4317).
	// - This example shares a single endpoint; adjust per your collector setup.
	// Config for logging, tracing, and metrics
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "text",
		},
		OTEL: config.OTELConfig{
			ServiceName:  "observer-example",
			ExporterType: "otlp", // or "stdout"
			Endpoint:     "localhost:4318",
			Insecure:     true,
			Resource: map[string]string{
				"deployment.environment": "dev",
				"service.version":        "0.1.0",
			},
		},
	}

	ctx := context.Background()

	// Setup OTLP logger (OTLP HTTP)
	provider, err := log.SetupOTELLogger(ctx, cfg.Log, cfg.OTEL)
	if err != nil {
		log.SetupLogger(cfg.Log)
		log.Error(ctx, "failed to setup OTLP logger", "error", err)
		return
	}
	defer func() {
		if err := provider.Shutdown(ctx); err != nil {
			log.Error(ctx, "failed to shutdown OTLP logger", "error", err)
		}
	}()

	// Setup tracer
	if _, err := trace.SetupTrace(ctx, cfg.OTEL); err != nil {
		log.Error(ctx, "failed to setup tracer", "error", err)
		return
	}

	// Setup metrics
	if _, err := metrics.SetupMetrics(ctx, cfg.OTEL); err != nil {
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
