package middlewares

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ActivityRecorder is satisfied by *services.LogService.
type ActivityRecorder interface {
	Record(ctx context.Context, entry *models.ActivityLog) error
}

// ActivityLogger records every write request (POST, PUT, PATCH, DELETE) after it has finished.
// It never stores request bodies, headers or query strings, and a failure to log never affects the response.
// Register it BEFORE the routes: api.Use(middlewares.ActivityLogger(logService))
func ActivityLogger(rec ActivityRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next() // run the handler first: the auth middleware sets the user during this call

		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			return
		}

		entry := &models.ActivityLog{
			Role:       ctxString(c, "role"),
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			Route:      c.FullPath(),
			StatusCode: c.Writer.Status(),
			IP:         c.ClientIP(),
			DurationMs: int(time.Since(start).Milliseconds()),
		}
		if id := ctxString(c, "user_id"); id != "" {
			if _, err := uuid.Parse(id); err == nil {
				entry.UserID = id
			}
		}
		// Persist before returning so completed requests are immediately auditable. Logging
		// remains fail-open: a logging failure is reported, but does not change the response.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := rec.Record(ctx, entry); err != nil {
			slog.Error("activity log could not be recorded", "method", entry.Method, "path", entry.Path, "error", err)
		}
	}
}

// ctxString reads a context value whether it was stored as a string, uuid.UUID, etc.
func ctxString(c *gin.Context, key string) string {
	v, ok := c.Get(key)
	if !ok || v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
