package middleware

import (
	"context"
	"time"

	"github.com/goleggo/observer/log"
)

// RequestLog holds the data logged for each HTTP request/response cycle.
type RequestLog struct {
	Method     string
	Path       string
	Status     int
	Latency    time.Duration
	ClientIP   string
	UserAgent  string
	Error      string
}

// LogRequest logs a completed request using the observer log package.
func LogRequest(ctx context.Context, r RequestLog) {
	args := []any{
		"method", r.Method,
		"path", r.Path,
		"status", r.Status,
		"latency_ms", r.Latency.Milliseconds(),
		"client_ip", r.ClientIP,
	}
	if r.UserAgent != "" {
		args = append(args, "user_agent", r.UserAgent)
	}
	if r.Error != "" {
		args = append(args, "error", r.Error)
	}

	if r.Status >= 500 {
		log.Error(ctx, "request completed", args...)
	} else {
		log.Info(ctx, "request completed", args...)
	}
}
