package otelhelper

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"

	"github.com/goleggo/observer/config"
)

const (
	ExporterOTLP   = "otlp"
	ExporterStdout = "stdout"
	ExporterNone   = "none"
)

// ExporterType picks the signal's exporter over the default, with otlp when both are empty.
func ExporterType(specific, global string) (string, error) {
	t := specific
	if t == "" {
		t = global
	}
	switch t = strings.ToLower(t); t {
	case "":
		return ExporterOTLP, nil
	case ExporterOTLP, ExporterStdout, ExporterNone:
		return t, nil
	}
	return "", fmt.Errorf("observer: unknown exporter %q (want otlp, stdout or none)", t)
}

// Resource builds the resource shared by all signals. OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES override the
// service name, and cfg.Resource overrides both.
func Resource(ctx context.Context, cfg config.OTELConfig) (*resource.Resource, error) {
	opts := []resource.Option{resource.WithSchemaURL(semconv.SchemaURL), resource.WithTelemetrySDK()}
	if cfg.ServiceName != "" {
		opts = append(opts, resource.WithAttributes(semconv.ServiceName(cfg.ServiceName)))
	}
	opts = append(opts, resource.WithFromEnv(), resource.WithAttributes(resourceAttrs(cfg.Resource)...))
	return resource.New(ctx, opts...)
}

func resourceAttrs(attrs map[string]string) []attribute.KeyValue {
	kv := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		if k != "" {
			kv = append(kv, attribute.String(k, v))
		}
	}
	return kv
}

// Endpoint is an OTLP/HTTP target for one signal. An empty Host means "use the OTEL_EXPORTER_OTLP_* env".
type Endpoint struct {
	Host     string
	Path     string
	Insecure bool
}

func (e Endpoint) IsSet() bool { return e.Host != "" }

// ParseEndpoint accepts "host:port" or a base URL ("http://collector:4318/prefix"). A URL works like
// OTEL_EXPORTER_OTLP_ENDPOINT: signalPath ("/v1/traces") is appended to its path, and the scheme decides Insecure.
func ParseEndpoint(raw string, insecure bool, signalPath string) (Endpoint, error) {
	if raw == "" || !strings.Contains(raw, "://") {
		return Endpoint{Host: raw, Insecure: insecure}, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return Endpoint{}, fmt.Errorf("observer: endpoint %q: %w", raw, err)
	}
	if u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return Endpoint{}, fmt.Errorf("observer: endpoint %q: want host:port or an http(s) URL", raw)
	}
	e := Endpoint{Host: u.Host, Insecure: u.Scheme == "http"}
	if p := strings.TrimSuffix(u.Path, "/"); p != "" {
		e.Path = p + signalPath
	}
	return e, nil
}
