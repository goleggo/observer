// Package nethttp instruments a net/http server: a server span per request (named after the matched route), the
// OTEL HTTP server metrics, the caller's traceparent, and one log line per request.
package nethttp

import (
	"bufio"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/goleggo/observer/middleware"
)

type options struct {
	skip  map[string]struct{}
	noLog bool
}

type Option func(*options)

// WithSkip serves these exact paths (health checks, for example) without a span, metrics or a log line.
func WithSkip(paths ...string) Option {
	return func(o *options) {
		for _, p := range paths {
			o.skip[p] = struct{}{}
		}
	}
}

// WithoutRequestLog keeps spans and metrics but drops the per-request log line.
func WithoutRequestLog() Option {
	return func(o *options) { o.noLog = true }
}

// Handler wraps next. Use it around the mux, so the span name and the http.route attribute carry the route pattern.
func Handler(next http.Handler, opts ...Option) http.Handler {
	o := options{skip: map[string]struct{}{}}
	for _, opt := range opts {
		opt(&o)
	}

	inner := next
	if !o.noLog {
		inner = logRequests(next)
	}
	instrumented := otelhttp.NewHandler(inner, "http.server")
	if len(o.skip) == 0 {
		return instrumented
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := o.skip[r.URL.Path]; ok {
			next.ServeHTTP(w, r)
			return
		}
		instrumented.ServeHTTP(w, r)
	})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)

		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		middleware.LogRequest(r.Context(), middleware.RequestLog{
			Method:    r.Method,
			Path:      r.URL.Path,
			Route:     r.Pattern,
			Status:    status,
			Latency:   time.Since(start),
			ClientIP:  ip,
			UserAgent: r.UserAgent(),
		})
	})
}

// statusWriter records the status code. Unwrap lets http.ResponseController reach the connection (write deadlines
// on a streaming response), and Flush and Hijack keep the type assertions of older code working.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 && code >= http.StatusOK {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusWriter) Flush() {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w.ResponseWriter).Hijack()
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
