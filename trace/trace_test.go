package trace

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/goleggo/observer/config"
)

func TestSetupTraceDisabledStillPropagates(t *testing.T) {
	tp, err := SetupTrace(context.Background(), config.OTELConfig{ExporterType: "none"})
	if err != nil || tp != nil {
		t.Fatalf("SetupTrace = %v, %v; want nil, nil", tp, err)
	}

	const parent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	in := http.Header{"Traceparent": {parent}}
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), propagation.HeaderCarrier(in))
	if !oteltrace.SpanContextFromContext(ctx).IsValid() {
		t.Fatal("traceparent was not extracted")
	}
	out := http.Header{}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(out))
	if out.Get("Traceparent") != parent {
		t.Fatalf("injected %q, want %q", out.Get("Traceparent"), parent)
	}
}

func TestSetupTraceRejectsBadConfig(t *testing.T) {
	for name, cfg := range map[string]config.OTELConfig{
		"unknown exporter": {ExporterType: "otlp-grpc"},
		"ratio above one":  {ExporterType: "stdout", SampleRatio: 1.5},
		"negative ratio":   {ExporterType: "stdout", SampleRatio: -0.1},
		"grpc endpoint":    {Endpoint: "grpc://collector:4317"},
	} {
		t.Run(name, func(t *testing.T) {
			if tp, err := SetupTrace(context.Background(), cfg); err == nil {
				_ = tp.Shutdown(context.Background())
				t.Fatal("want an error")
			}
		})
	}
}

func TestSetupTraceSampleRatio(t *testing.T) {
	ctx := context.Background()
	tp, err := SetupTrace(ctx, config.OTELConfig{ExporterType: "stdout", SampleRatio: 0.000001})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tp.Shutdown(ctx) })

	sampled := 0
	for range 1000 {
		_, span := tp.Tracer("test").Start(ctx, "op")
		if span.SpanContext().IsSampled() {
			sampled++
		}
		span.End()
	}
	if sampled > 5 {
		t.Fatalf("%d of 1000 root spans sampled at ratio 1e-6", sampled)
	}

	remote := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: oteltrace.TraceID{1}, SpanID: oteltrace.SpanID{1}, TraceFlags: oteltrace.FlagsSampled, Remote: true,
	})
	_, span := tp.Tracer("test").Start(oteltrace.ContextWithRemoteSpanContext(ctx, remote), "child")
	defer span.End()
	if !span.SpanContext().IsSampled() {
		t.Fatal("a sampled parent must keep its child sampled")
	}
}
