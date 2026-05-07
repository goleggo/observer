package gin

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goleggo/observer/middleware"
)

// Logger returns a gin middleware that logs each request/response
// with method, path, status, latency, and client IP.
// Trace and span IDs are automatically included via context propagation.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		r := middleware.RequestLog{
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Status:    c.Writer.Status(),
			Latency:   time.Since(start),
			ClientIP:  c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		}
		if len(c.Errors) > 0 {
			r.Error = c.Errors.String()
		}

		middleware.LogRequest(c.Request.Context(), r)
	}
}
