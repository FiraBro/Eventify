package middlewares

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/FiraBro/local-go/internal/observability"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

// RequestID validates an incoming canonical UUID request ID or creates a new one.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		parsed, err := uuid.Parse(requestID)
		if err != nil || parsed.String() != requestID {
			requestID = uuid.NewString()
		}
		c.Header(RequestIDHeader, requestID)
		c.Set("request_id", requestID)
		c.Request = c.Request.WithContext(observability.ContextWithRequestID(c.Request.Context(), requestID))
		c.Next()
	}
}

// HTTPObservability records one bounded-cardinality metric observation and one
// structured completion log per non-health request. It never logs URLs or bodies.
func HTTPObservability(logger *slog.Logger, metrics *observability.Metrics) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = observability.NewMetrics()
	}
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()
		finish := metrics.Begin()
		c.Next()
		duration := time.Since(start)
		status := c.Writer.Status()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		finish(c.Request.Method, route, status, duration)
		if route == "/healthz" || route == "/readyz" {
			return
		}
		attrs := []any{
			"method", c.Request.Method,
			"route", route,
			"status_code", status,
			"duration_ms", float64(duration.Microseconds()) / 1000,
		}
		if status >= http.StatusInternalServerError {
			logger.ErrorContext(c.Request.Context(), "http request completed", attrs...)
		} else if status >= http.StatusBadRequest {
			logger.WarnContext(c.Request.Context(), "http request completed", attrs...)
		} else {
			logger.InfoContext(c.Request.Context(), "http request completed", attrs...)
		}
	}
}

// Recovery logs a correlated panic event without logging the panic value or
// exposing implementation details to the caller. The HTTP completion middleware
// observes the resulting 500 once after this middleware recovers.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c *gin.Context) {
		defer func() {
			if recover() != nil {
				if logger.Enabled(c.Request.Context(), slog.LevelError) {
					logger.ErrorContext(c.Request.Context(), "panic recovered",
						"method", c.Request.Method,
						"route", routePattern(c),
						"stack", string(debug.Stack()),
					)
				}
				if !c.Writer.Written() {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				} else {
					c.Abort()
				}
			}
		}()
		c.Next()
	}
}

func routePattern(c *gin.Context) string {
	if route := c.FullPath(); route != "" {
		return route
	}
	return "unmatched"
}
