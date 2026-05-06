package config

// LogConfig holds configuration for the logger.
type LogConfig struct {
	Level  string // e.g., "info", "debug", "error"
	Format string // e.g., "text", "json"
}

// OTELConfig holds configuration for OpenTelemetry exporters and resources.
type OTELConfig struct {
	ServiceName  string
	ExporterType string // Default exporter type: "otlp", "stdout", "none"
	Endpoint     string // OTLP endpoint
	Insecure     bool   // Allow insecure connection (for local testing)
	Resource     map[string]string

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
