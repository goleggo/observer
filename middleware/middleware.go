package middleware

import (
	"context"
	"time"

	"github.com/goleggo/observer/log"
)

// RequestLog holds the data logged for each HTTP request/response cycle.
type RequestLog struct {
	Method    string
	Path      string
	Route     string // the matched pattern, e.g. "GET /v1/items/{id}"
	Status    int
	Latency   time.Duration
	ClientIP  string
	UserAgent string
	Error     string
}

// LogRequest logs a completed request: at error level for a 5xx, at info level otherwise.
func LogRequest(ctx context.Context, r RequestLog) {
	args := make([]any, 0, 16)
	args = append(args,
		"method", r.Method,
		"path", r.Path,
		"status", r.Status,
		"latency_ms", r.Latency.Milliseconds(),
		"client_ip", r.ClientIP,
	)
	if r.Route != "" {
		args = append(args, "route", r.Route)
	}
	if r.UserAgent != "" {
		args = append(args, "user_agent", r.UserAgent)
	}
	if r.Error != "" {
		args = append(args, "error", r.Error)
	}

	if r.Status >= 500 {
		log.Error(ctx, "request completed", args...)
		return
	}
	log.Info(ctx, "request completed", args...)
}
