# observer

Common observability library for Go services using OpenTelemetry (logs, metrics, traces).

This package provides:
- OTLP HTTP logs via `slog` (through the OTEL log SDK bridge)
- OTLP HTTP metrics exporter
- OTLP HTTP trace exporter
- Convenience helpers for context-aware logging

## Install

```bash
go get github.com/goleggo/observer
```

## Quick start

```go
package main

import (
	"context"

	"github.com/goleggo/observer/config"
	"github.com/goleggo/observer/log"
	"github.com/goleggo/observer/metrics"
	"github.com/goleggo/observer/trace"
)

func main() {
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
		OTEL: config.OTELConfig{
			ServiceName:  "orders-api",
			ExporterType: "otlp",
			Endpoint:     "localhost:4318",
			Insecure:     true,
			Resource: map[string]string{
				"deployment.environment": "dev",
				"service.version":        "0.1.0",
			},
		},
	}

	ctx := context.Background()

	logShutdown, err := log.SetupOTELLogger(ctx, cfg.Log, cfg.OTEL)
	if err != nil {
		log.SetupLogger(cfg.Log)
		log.Error(ctx, "failed to setup OTLP logger", "error", err)
		return
	}
	defer logShutdown(ctx)

	if err := trace.SetupTrace(ctx, cfg.OTEL); err != nil {
		log.Error(ctx, "failed to setup tracer", "error", err)
		return
	}

	if err := metrics.SetupMetrics(ctx, cfg.OTEL); err != nil {
		log.Error(ctx, "failed to setup metrics", "error", err)
		return
	}

	log.Info(ctx, "service started")
}
```

## Configuration

### LogConfig

```go
type LogConfig struct {
	Level  string // info|debug|warn|error
	Format string // text|json
}
```

### OTELConfig

```go
type OTELConfig struct {
	ServiceName  string
	ExporterType string // otlp|stdout
	Endpoint     string // OTLP endpoint
	Insecure     bool   // Allow insecure connection
	Resource     map[string]string
}
```

## Signals and exporters

- Logs: OTLP HTTP via `log.SetupOTELLogger`
- Traces: OTLP HTTP via `trace.SetupTrace`
- Metrics: OTLP HTTP via `metrics.SetupMetrics`

Default OTEL Collector ports:
- OTLP HTTP: `4318`
- OTLP gRPC: `4317`

If you want different endpoints per signal, create separate configs per call.

## Resource attributes

You can set resource attributes two ways:

1) In code with `cfg.OTEL.Resource`:

```go
cfg.OTEL.Resource = map[string]string{
	"deployment.environment": "prod",
	"service.version":        "1.2.3",
	"region":                 "ap-southeast-1",
}
```

2) Through the environment (merged automatically):

```bash
OTEL_RESOURCE_ATTRIBUTES=deployment.environment=prod,service.version=1.2.3,region=ap-southeast-1
```

## Logging helpers

The `log` package wraps `slog` and adds trace/span IDs when present in the context.

```go
log.Info(ctx, "order created", "order_id", id)
log.Error(ctx, "payment failed", "error", err)
```

## Examples

See `examples/main.go` for a runnable example.

## Notes

- `SetupLogger` remains available for stdout-only logging.
- All signals (logs, traces, metrics) use OTLP/HTTP.
