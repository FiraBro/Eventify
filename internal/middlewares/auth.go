package middlewares

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/FiraBro/local-go/internal/config"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// Typed JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware verifies JWT and checks user existence & soft delete
func AuthMiddleware(userRepo *repositories.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(config.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Optional: check token expiry
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			return
		}

		// Fetch user from DB to validate soft deletion
		user, err := userRepo.GetActiveByID(claims.UserID)
		fmt.Println("USER FROM DB =", user.Role)
		if err != nil || user.DeletedAt != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User no longer active"})
			return
		}

		// Store info in context
		c.Set("user_id", user.ID)
		c.Set("role", user.Role)

		c.Next()
	}
}

// AdminOnly ensures only admin users can access
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := getStringFromContext(c, "role")
     fmt.Println("ROLE FROM DB =", role)

		if !exists || role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		c.Next()
	}
}

// OwnerOrAdmin ensures only owner or admin can access
func OwnerOrAdmin(getOwnerID func(c *gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := getStringFromContext(c, "user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		role, _ := getStringFromContext(c, "role")
     fmt.Println("ROLE FROM DB =", role)

		ownerID := getOwnerID(c)

		if role != "admin" && userID != ownerID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		c.Next()
	}
}

// Helper to get string from context
func getStringFromContext(c *gin.Context, key string) (string, bool) {
	val, exists := c.Get(key)
	if !exists {
		return "", false
	}
	strVal, ok := val.(string)
	return strVal, ok
}
