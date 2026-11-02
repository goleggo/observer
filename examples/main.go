package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/goleggo/observer"
	"github.com/goleggo/observer/config"
	"github.com/goleggo/observer/log"
	"github.com/goleggo/observer/middleware/nethttp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	shutdown, err := observer.Setup(ctx, config.Config{
		Log: config.LogConfig{Level: "debug", Format: "text"},
		OTEL: config.OTELConfig{
			ServiceName:  "observer-example",
			ExporterType: "stdout", // "otlp" sends to Endpoint (or OTEL_EXPORTER_OTLP_ENDPOINT)
			Endpoint:     "http://localhost:4318",
			SampleRatio:  1,
			Resource:     map[string]string{"deployment.environment": "dev"},
		},
	})
	if err != nil {
		log.Error(ctx, "observer setup failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			log.Error(ctx, "observer shutdown failed", "error", err)
		}
	}()

	hits, _ := otel.Meter("observer-example").Int64Counter("example_hits")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello/{name}", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(r.Context(), 1, metric.WithAttributes(attribute.String("name", r.PathValue("name"))))
		log.Info(r.Context(), "saying hello", "name", r.PathValue("name"))
		_, _ = w.Write([]byte("hello " + r.PathValue("name") + "\n"))
	})
	mux.HandleFunc("GET /healthz", func(http.ResponseWriter, *http.Request) {})

	srv := &http.Server{Addr: ":8080", Handler: nethttp.Handler(mux, nethttp.WithSkip("/healthz")), ReadHeaderTimeout: 2 * time.Second}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	log.Info(ctx, "listening on :8080; try curl localhost:8080/hello/world")
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Error(ctx, "server failed", "error", err)
	}
}
