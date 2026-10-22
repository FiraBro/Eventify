package middlewares

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders injects OWASP-compliant security response headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content-Security-Policy restricted to api data delivery scopes
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; sandbox")

		// Apply HSTS only if explicitly running in verified production environments
		if os.Getenv("APP_ENV") == "production" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// ExplicitCORS validates requests against an allowed array list of origins
func ExplicitCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		// Read allow-list configurations from the system environment
		allowedOriginsStr := os.Getenv("CORS_ALLOWED_ORIGINS")
		if allowedOriginsStr == "" {
			allowedOriginsStr = "http://localhost:3000,http://localhost:5173" // Default safe development fallbacks
		}
		allowedOrigins := strings.Split(allowedOriginsStr, ",")

		var isAllowed bool
		for _, allowed := range allowedOrigins {
			if origin == strings.TrimSpace(allowed) {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CORS policy: Origin not allowed"})
			return
		}

		// Configure valid response parameters cleanly
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		// Instantly intercept preflight OPTIONS transactions
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestBodyLimiter blocks bloated requests before body decoding occurs
func RequestBodyLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Default JSON threshold constraint cap of 2MB to protect memory pools
		var limit int64 = 2 << 20

		// Read dynamic size limits if specified in environment configurations
		if envLimit := os.Getenv("MAX_REQUEST_BODY_BYTES"); envLimit != "" {
			var parsed int64
			if _, err := fmt.Sscanf(envLimit, "%d", &parsed); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		// Wrap the input stream inside an explicit MaxBytesReader wrapper
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}
