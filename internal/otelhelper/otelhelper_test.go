package otelhelper

import (
	"context"
	"testing"

	"github.com/goleggo/observer/config"
)

func TestExporterType(t *testing.T) {
	tests := []struct {
		name, specific, global, want string
		wantErr                      bool
	}{
		{name: "both empty defaults to otlp", want: ExporterOTLP},
		{name: "global applies", global: "stdout", want: ExporterStdout},
		{name: "signal overrides global", specific: "none", global: "otlp", want: ExporterNone},
		{name: "case is ignored", global: "STDOUT", want: ExporterStdout},
		{name: "typo is an error", global: "otlp-grpc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExporterType(tt.specific, tt.global)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Fatalf("ExporterType(%q, %q) = %q, %v", tt.specific, tt.global, got, err)
			}
		})
	}
}

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name, raw string
		insecure  bool
		want      Endpoint
		wantErr   bool
	}{
		{name: "empty leaves the env in charge", raw: ""},
		{name: "empty keeps insecure", insecure: true, want: Endpoint{Insecure: true}},
		{name: "host and port", raw: "collector:4318", insecure: true, want: Endpoint{Host: "collector:4318", Insecure: true}},
		{name: "http url is insecure", raw: "http://collector:4318", want: Endpoint{Host: "collector:4318", Insecure: true}},
		{name: "https url ignores the insecure flag", raw: "https://collector:4318/", insecure: true, want: Endpoint{Host: "collector:4318"}},
		{name: "url path is a prefix", raw: "https://gw.example.com/otlp", want: Endpoint{Host: "gw.example.com", Path: "/otlp/v1/traces"}},
		{name: "grpc scheme is rejected", raw: "grpc://collector:4317", wantErr: true},
		{name: "url without host is rejected", raw: "http://", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEndpoint(tt.raw, tt.insecure, "/v1/traces")
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Fatalf("ParseEndpoint(%q) = %+v, %v; want %+v", tt.raw, got, err, tt.want)
			}
		})
	}
}

func TestResource(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "deployment.environment=env,team=queue")
	res, err := Resource(context.Background(), config.OTELConfig{
		ServiceName: "svc",
		Resource:    map[string]string{"deployment.environment": "cfg", "": "dropped"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, kv := range res.Attributes() {
		got[string(kv.Key)] = kv.Value.Emit()
	}
	for k, want := range map[string]string{"service.name": "svc", "deployment.environment": "cfg", "team": "queue", "telemetry.sdk.language": "go"} {
		if got[k] != want {
			t.Errorf("%s = %q, want %q", k, got[k], want)
		}
	}
	if _, ok := got[""]; ok {
		t.Error("empty key kept")
	}
}
