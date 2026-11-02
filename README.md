# observer

Common observability setup for Go services on OpenTelemetry: logs (slog), traces and metrics over OTLP/HTTP.

- `observer.Setup`: one call for the slog default, the W3C propagator, the tracer and meter providers, and one
  `shutdown`.
- `log`: console logs with `trace_id` / `span_id`, an OTLP log bridge that keeps the console output, and a stack trace
  on error records.
- `trace`: OTLP/HTTP or stdout exporter, `traceparent` + `baggage` propagation, parent-based ratio sampling.
- `metrics`: OTLP/HTTP or stdout exporter, Go runtime metrics, configurable interval.
- `middleware/nethttp`: server spans named after the route, the OTEL HTTP server metrics, one log line per request.
- Separate modules, so the root module does not pull their dependencies:
  - `github.com/goleggo/observer/middleware/gin`: a gin request logger;
  - `github.com/goleggo/observer/grpc`: gRPC client and server stats handlers.

Requires Go 1.26 or later.

## Install

```bash
go get github.com/goleggo/observer
go get github.com/goleggo/observer/middleware/gin # only for gin
go get github.com/goleggo/observer/grpc           # only for gRPC
```

## Quick start

```go
shutdown, err := observer.Setup(ctx, config.Config{
	Log: config.LogConfig{Level: "info", Format: "json"},
	OTEL: config.OTELConfig{
		ServiceName: "orders-api",
		Endpoint:    "http://otel-collector:4318", // or leave empty and set OTEL_EXPORTER_OTLP_ENDPOINT
		SampleRatio: 0.05,
		Resource:    map[string]string{"deployment.environment": "prod"},
	},
})
if err != nil {
	slog.Error("observer setup failed", "error", err)
	os.Exit(1)
}
defer shutdown(context.WithoutCancel(ctx))

srv := &http.Server{Handler: nethttp.Handler(mux, nethttp.WithSkip("/healthz", "/readyz"))}
```

`examples/main.go` is a runnable server (`go run ./examples`, then `curl localhost:8080/hello/world`).

The per-signal functions (`log.SetupOTELLogger`, `log.SetupLogger`, `trace.SetupTrace`, `metrics.SetupMetrics`) are
still exported for services that need them separately. Each returns a nil provider when its signal is off.

## Configuration

| Field | Meaning |
| --- | --- |
| `Log.Level` | `debug`, `info` (default), `warn`, `error` |
| `Log.Format` | `text` (default) or `json` |
| `Log.DisableConsole` | with OTLP logs on, stop writing to stdout as well |
| `OTEL.ServiceName` | `service.name`; `OTEL_SERVICE_NAME` overrides it |
| `OTEL.ExporterType` | `otlp` (default), `stdout` or `none`, for every signal. Anything else is an error. |
| `OTEL.TracesExporter`, `MetricsExporter`, `LogsExporter` | per-signal override of `ExporterType` |
| `OTEL.Endpoint` | `host:port` or a base URL (`https://collector:4318/prefix`). Empty: the `OTEL_EXPORTER_OTLP_*` env. |
| `OTEL.Insecure` | plain HTTP for a `host:port` endpoint; a URL's scheme decides on its own |
| `OTEL.Resource` | extra resource attributes; they override `OTEL_RESOURCE_ATTRIBUTES` |
| `OTEL.SampleRatio` | `0 < r <= 1`: keep that share of new traces, follow the parent otherwise. `0`: `OTEL_TRACES_SAMPLER` decides (default: everything). |
| `OTEL.MetricInterval` | export interval; `0` uses `OTEL_METRIC_EXPORT_INTERVAL` or 60s |
| `OTEL.DisableRuntimeMetrics` | turn off the Go runtime metrics |
| `OTEL.DisableTraces`, `DisableMetrics`, `DisableLogs` | turn a signal off |

Only OTLP over HTTP is supported: point the endpoint at the collector's HTTP port (4318), not the gRPC port (4317).

The propagator is installed even when traces are off, so a service still forwards its callers' `traceparent`.

## Logging

The `log` helpers (`Debug`, `Info`, `Warn`, `Error`) log through `slog.Default()`, so plain `slog` calls get the same
handler.

Error records get a `stacktrace` attribute from the first error attribute that carries one:
- an error that implements `log.StackTracer` (`StackTrace() string`);
- otherwise an error that implements `fmt.Formatter`, rendered with `%+v` (`github.com/pkg/errors`,
  `github.com/cockroachdb/errors`).

The lookup unwraps (`errors.As`).

## HTTP middleware

`nethttp.Handler(mux, opts...)` wraps a `net/http` handler with `otelhttp`, plus a request log line that carries the
trace id. Wrap the mux rather than single routes, so the span name and `http.route` carry the matched pattern.

- `WithSkip(paths...)`: no span, metrics or log for these exact paths.
- `WithoutRequestLog()`: keep spans and metrics, drop the log line (high-traffic services).

The response writer keeps `Flush`, `Hijack` and `Unwrap`, so `http.ResponseController` (write deadlines on a streaming
response) still works.

gin: `router.Use(observergin.Logger())` from `github.com/goleggo/observer/middleware/gin`.

## gRPC

```go
srv := grpc.NewServer(grpcserver.StatsHandler()...)
conn, err := grpc.NewClient("localhost:50051", grpcclient.StatsHandler()...)
```

from `github.com/goleggo/observer/grpc/server` and `.../grpc/client`.

## Upgrading from v0.1

- The `log.Logger` variable is gone: use `slog.Default()` or the returned `*slog.Logger`.
- An unknown exporter type is an error instead of falling back to OTLP.
- An empty `Endpoint` no longer overrides `OTEL_EXPORTER_OTLP_ENDPOINT`.
- `SetupOTELLogger` also writes to stdout; set `Log.DisableConsole` for the old behaviour.
- gin and gRPC support moved to their own modules (see Install).
- The resource now carries `telemetry.sdk.*`, and the semconv schema is v1.43.0.

## Development

`make ci` runs gofmt, vet, golangci-lint, govulncheck, the race tests and a `go mod tidy` check on all three modules.

## Releasing

The nested modules require the root module by version (the `replace` only applies inside this repository), so tag the
root first: `v0.2.0`, then `grpc/v0.2.0` and `middleware/gin/v0.2.0`.
