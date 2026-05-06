# observer

Common observability library for Go services using OpenTelemetry (logs, metrics, traces).

This package provides:
- OTLP HTTP logs via `slog` (through the OTEL log SDK bridge)
- OTLP HTTP metrics exporter
- OTLP HTTP trace exporter
- gRPC server and client instrumentation (via OpenTelemetry stats handlers)
- Performance-optimized `slog` handlers for trace context and automatic error stack traces

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
			ExporterType: "otlp", // Default for all signals
			Endpoint:     "localhost:4318",
			Insecure:     true,
			Resource: map[string]string{
				"deployment.environment": "dev",
			},
			// Granular control (optional)
			DisableMetrics: true, // Silence metrics in dev
		},
	}

	ctx := context.Background()

	// 1. Setup Logging (OTEL or Standard)
	logProvider, err := log.SetupOTELLogger(ctx, cfg.Log, cfg.OTEL)
	if err != nil || logProvider == nil {
		log.SetupLogger(cfg.Log) // Fallback to standard stdout logger
	} else {
		defer logProvider.Shutdown(ctx)
	}

	// 2. Setup Traces
	traceProvider, err := trace.SetupTrace(ctx, cfg.OTEL)
	if err == nil && traceProvider != nil {
		defer traceProvider.Shutdown(ctx)
	}

	// 3. Setup Metrics
	meterProvider, err := metrics.SetupMetrics(ctx, cfg.OTEL)
	if err == nil && meterProvider != nil {
		defer meterProvider.Shutdown(ctx)
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
	ExporterType string // otlp|stdout|none (default for all signals)
	Endpoint     string // OTLP endpoint
	Insecure     bool   // Allow insecure connection
	Resource     map[string]string

	// Optional: granular control over signals
	TracesExporter  string // overrides ExporterType for traces
	MetricsExporter string // overrides ExporterType for metrics
	LogsExporter    string // overrides ExporterType for logs

	DisableTraces  bool
	DisableMetrics bool
	DisableLogs    bool
}
```

## Performance & Best Practices

### Context-Aware Logging
The `log` package is optimized for zero-allocations in the hot path. It uses `slog` handlers to automatically:
1. Extract `trace_id` and `span_id` from `context.Context`.
2. Detect `error` attributes and append stack traces automatically (if available via `github.com/pkg/errors` or similar).

```go
// Trace context and error stack traces are handled automatically
log.Error(ctx, "database connection failed", "error", err)
```

### Granular Signal Control
In local development, you might want to see logs but silence noisy traces or metrics. You can do this by setting `ExporterType: "none"` or using the `DisableX` flags in `OTELConfig`.

### Provider Shutdown
**Crucial:** Always shutdown the providers returned by the setup functions. Failing to do so can lead to memory leaks, lost telemetry data, or hanging background goroutines.

## Signals and exporters

- Logs: `log.SetupOTELLogger`
- Traces: `trace.SetupTrace`
- Metrics: `metrics.SetupMetrics`

All exporters support `otlp` (HTTP), `stdout` (for local debugging), and `none`.

## gRPC instrumentation

The `grpc/server` and `grpc/client` packages provide OpenTelemetry stats handlers that automatically instrument all RPCs.

### Server
```go
srv := grpc.NewServer(grpcserver.StatsHandler()...)
```

### Client
```go
conn, err := grpc.NewClient("localhost:50051", grpcclient.StatsHandler()...)
```

## Examples

See `examples/main.go` for a complete, runnable example of the optimized setup.
