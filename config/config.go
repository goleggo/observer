package config

// LogConfig holds configuration for the logger.
type LogConfig struct {
	Level  string // e.g., "info", "debug", "error"
	Format string // e.g., "text", "json"
}

// OTELConfig holds configuration for OpenTelemetry exporters and resources.
type OTELConfig struct {
	ServiceName  string
	ExporterType string // "otlp" (default), "stdout" or "none"; anything else is an error
	// Endpoint is the OTLP/HTTP collector: "host:port" or a base URL ("https://collector:4318"). Empty uses the
	// OTEL_EXPORTER_OTLP_ENDPOINT env. There is no gRPC exporter: point it at the collector's HTTP port (4318).
	Endpoint string
	Insecure bool // plain HTTP for a host:port endpoint; a URL's scheme decides on its own
	Resource map[string]string

	// Optional: granular control over signals
	TracesExporter  string // overrides ExporterType for traces
	MetricsExporter string // overrides ExporterType for metrics
	LogsExporter    string // overrides ExporterType for logs

	DisableTraces  bool
	DisableMetrics bool
	DisableLogs    bool
}

// Config holds configuration for all observability components.
type Config struct {
	Log  LogConfig
	OTEL OTELConfig
}
