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
	// - All signals (logs, traces, metrics) use OTLP/HTTP (default collector port 4318).
	// - Use grpc/server and grpc/client packages to instrument gRPC handlers.
	// Config for logging, tracing, and metrics
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "debug",
			Format: "text",
		},
		OTEL: config.OTELConfig{
			ServiceName:  "observer-example",
			ExporterType: "stdout", // can be "otlp", "stdout", or "none"
			Endpoint:     "localhost:4318",
			Insecure:     true,
			Resource: map[string]string{
				"deployment.environment": "dev",
				"service.version":        "0.1.0",
			},
			// To handle noisy OTEL logs in dev, you can disable traces or metrics:
			DisableTraces:  false,
			DisableMetrics: false,
		},
	}

	ctx := context.Background()

	// Setup OTLP/Stdout logger
	logProvider, err := log.SetupOTELLogger(ctx, cfg.Log, cfg.OTEL)
	if err != nil {
		log.SetupLogger(cfg.Log)
		log.Error(ctx, "failed to setup OTLP logger", "error", err)
	} else if logProvider == nil {
		// OTEL logs disabled, fallback to standard logger
		log.SetupLogger(cfg.Log)
		log.Info(ctx, "OTEL logs disabled, using standard logger")
	} else {
		defer func() {
			if err := logProvider.Shutdown(ctx); err != nil {
				log.Error(ctx, "failed to shutdown OTLP logger", "error", err)
			}
		}()
	}

	// Setup tracer
	traceProvider, err := trace.SetupTrace(ctx, cfg.OTEL)
	if err != nil {
		log.Error(ctx, "failed to setup tracer", "error", err)
	} else if traceProvider != nil {
		defer func() {
			if err := traceProvider.Shutdown(ctx); err != nil {
				log.Error(ctx, "failed to shutdown tracer", "error", err)
			}
		}()
	}

	// Setup metrics
	meterProvider, err := metrics.SetupMetrics(ctx, cfg.OTEL)
	if err != nil {
		log.Error(ctx, "failed to setup metrics", "error", err)
	} else if meterProvider != nil {
		defer func() {
			if err := meterProvider.Shutdown(ctx); err != nil {
				log.Error(ctx, "failed to shutdown metrics", "error", err)
			}
		}()
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
