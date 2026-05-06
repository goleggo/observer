package otelhelper

import (
	"go.opentelemetry.io/otel/attribute"
)

// ResourceAttrs converts a map of attributes to OpenTelemetry attribute KeyValues.
func ResourceAttrs(attrs map[string]string) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}

	kv := make([]attribute.KeyValue, 0, len(attrs))
	for key, value := range attrs {
		if key == "" {
			continue
		}
		kv = append(kv, attribute.String(key, value))
	}
	return kv
}

// GetExporterType returns the specific exporter type if provided, otherwise the global default.
func GetExporterType(specific, global string) string {
	if specific != "" {
		return specific
	}
	if global != "" {
		return global
	}
	return "otlp"
}
