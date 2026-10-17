//go:build integration

package integration

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHTTPHardeningPolicies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	os.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	r.Use(middlewares.SecurityHeaders())
	r.Use(middlewares.ExplicitCORS())
	r.Use(middlewares.RequestBodyLimiter())

	r.POST("/api/v1/test", func(c *gin.Context) {
		var body map[string]string
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "payload too large"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "allowed"})
	})

	t.Run("OWASP security headers are present on data replies", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/test", bytes.NewBuffer([]byte(`{}`)))
		r.ServeHTTP(w, req)

		if w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Error("missing X-Content-Type-Options protective response header")
		}
		if w.Header().Get("X-Frame-Options") != "DENY" {
			t.Error("missing X-Frame-Options protection header")
		}
	})

	t.Run("Allowed CORS origins pass validation rules", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/test", bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Origin", "https://example.com")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK for valid origin, got %d", w.Code)
		}
	})

	t.Run("Unregistered malicious origins are blocked with 403 Forbidden", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/test", bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Origin", "https://malicious.site")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for untrusted context, got %d", w.Code)
		}
	})
}
