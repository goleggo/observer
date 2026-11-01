package nethttp

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/goleggo/observer/config"
	"github.com/goleggo/observer/log"
)

func setup(t *testing.T) (*tracetest.SpanRecorder, *bytes.Buffer) {
	t.Helper()
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	oldTP, oldProp, oldLog := otel.GetTracerProvider(), otel.GetTextMapPropagator(), slog.Default()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	var buf bytes.Buffer
	log.SetLogger(slog.New(log.NewConsoleHandler(config.LogConfig{Format: "json"}, &buf)))
	t.Cleanup(func() {
		otel.SetTracerProvider(oldTP)
		otel.SetTextMapPropagator(oldProp)
		slog.SetDefault(oldLog)
	})
	return rec, &buf
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("GET /stream", func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)
		if err := rc.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("data: 1\n\n"))
		w.(http.Flusher).Flush()
	})
	return mux
}

func TestHandlerTracesAndLogsWithTheRoute(t *testing.T) {
	rec, buf := setup(t)
	srv := httptest.NewServer(Handler(newMux()))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/items/42", nil)
	req.Header.Set("Traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	spans := rec.Ended()
	if len(spans) != 1 {
		t.Fatalf("got %d spans", len(spans))
	}
	s := spans[0]
	if s.Name() != "GET /items/{id}" {
		t.Errorf("span name %q", s.Name())
	}
	if got := s.Parent().TraceID().String(); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("parent trace %s: traceparent not extracted", got)
	}
	line := buf.String()
	for _, want := range []string{`"status":418`, `"route":"GET /items/{id}"`, `"path":"/items/42"`, s.SpanContext().TraceID().String()} {
		if !strings.Contains(line, want) {
			t.Errorf("log misses %s: %s", want, line)
		}
	}
}

func TestHandlerSkipsPaths(t *testing.T) {
	rec, buf := setup(t)
	srv := httptest.NewServer(Handler(newMux(), WithSkip("/healthz")))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK || len(rec.Ended()) != 0 || buf.Len() != 0 {
		t.Fatalf("status %d, %d spans, log %q", res.StatusCode, len(rec.Ended()), buf.String())
	}
}

func TestHandlerKeepsStreamingWorking(t *testing.T) {
	_, buf := setup(t)
	srv := httptest.NewServer(Handler(newMux(), WithoutRequestLog()))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d: the write deadline did not reach the connection", res.StatusCode)
	}
	if buf.Len() != 0 {
		t.Fatalf("request log not disabled: %s", buf.String())
	}
}

func TestStatusWriterUnwraps(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec}
	if http.NewResponseController(sw).Flush() != nil || !rec.Flushed || sw.status != http.StatusOK {
		t.Fatal("flush did not reach the recorder")
	}
}
