package middlewares

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type activityRecorderFunc func(context.Context, *models.ActivityLog) error

func (f activityRecorderFunc) Record(ctx context.Context, entry *models.ActivityLog) error {
	return f(ctx, entry)
}

func TestActivityLoggerRecordsWriteAfterHandlerWithoutRequestSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.NewString()
	var recorded *models.ActivityLog
	r := gin.New()
	r.Use(ActivityLogger(activityRecorderFunc(func(_ context.Context, entry *models.ActivityLog) error {
		copy := *entry
		recorded = &copy
		return nil
	})))
	r.POST("/private", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("role", "admin")
		c.JSON(201, gin.H{"ok": true})
	})

	req := httptest.NewRequest("POST", "/private?token=never-log", nil)
	req.Header.Set("Authorization", "Bearer never-log")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if recorded == nil {
		t.Fatal("write request was not recorded")
	}
	if recorded.UserID != userID || recorded.Role != "admin" || recorded.Method != "POST" || recorded.StatusCode != 201 {
		t.Fatalf("unexpected audit record: %+v", recorded)
	}
	if recorded.Path != "/private" || recorded.Route != "/private" {
		t.Fatalf("unexpected request path metadata: %+v", recorded)
	}
	if recorded.Details != nil {
		t.Fatalf("request details must not be copied: %s", recorded.Details)
	}
}

func TestActivityLoggerSkipsReadMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	calls := 0
	r := gin.New()
	r.Use(ActivityLogger(activityRecorderFunc(func(context.Context, *models.ActivityLog) error { calls++; return nil })))
	r.GET("/read", func(c *gin.Context) { c.Status(200) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/read", nil))
	if calls != 0 {
		t.Fatalf("GET generated %d audit records, want 0", calls)
	}
}

func TestActivityLoggerFailureDoesNotChangeResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ActivityLogger(activityRecorderFunc(func(context.Context, *models.ActivityLog) error { return errors.New("database unavailable") })))
	r.DELETE("/resource", func(c *gin.Context) { c.Status(204) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/resource", nil))
	if w.Code != 204 {
		t.Fatalf("logging failure changed response to %d", w.Code)
	}
}
