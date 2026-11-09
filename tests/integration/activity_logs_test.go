//go:build integration

package integration

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
)

func TestActivityLogMiddlewareRepositoryAndAdminListing(t *testing.T) {
	db := newPostgresDatabase(t)
	repo := repositories.NewLogRepository(db)
	service := services.NewLogService(repo)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middlewares.ActivityLogger(service))
	router.POST("/audit-fixture", func(c *gin.Context) {
		c.Set("user_id", "")
		c.Set("role", "admin")
		c.Status(201)
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", "/audit-fixture", nil))
	if response.Code != 201 {
		t.Fatalf("write request status = %d, want 201", response.Code)
	}

	ctx := context.Background()
	first := &models.ActivityLog{Role: "admin", Method: "POST", Path: "/audit-search-alpha", Route: "/audit-search-alpha", StatusCode: 201}
	if err := service.Record(ctx, first); err != nil {
		t.Fatalf("persist first record: %v", err)
	}
	second := &models.ActivityLog{Role: "admin", Method: "PATCH", Path: "/audit-search-beta", Route: "/audit-search-beta", StatusCode: 204}
	if err := service.Record(ctx, second); err != nil {
		t.Fatalf("persist second record: %v", err)
	}

	page, err := service.List(ctx, "admin", &models.LogListParams{Q: "audit-search", Page: 1, Limit: 1})
	if err != nil {
		t.Fatalf("admin list: %v", err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Page != 1 || page.Limit != 1 {
		t.Fatalf("unexpected paginated result: %+v", page)
	}

	filtered, err := service.List(ctx, "admin", &models.LogListParams{Method: "PATCH", Q: "audit-search"})
	if err != nil {
		t.Fatalf("filtered admin list: %v", err)
	}
	if filtered.Total != 1 || len(filtered.Items) != 1 || filtered.Items[0].Path != second.Path {
		t.Fatalf("filter did not select PATCH record: %+v", filtered)
	}

	if _, err := service.List(ctx, "user", &models.LogListParams{}); err != services.ErrLogForbidden {
		t.Fatalf("non-admin access error = %v, want forbidden", err)
	}
}
