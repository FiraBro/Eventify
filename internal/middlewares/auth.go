package middlewares

import (
	"net/http"
	"strings"

	"github.com/FiraBro/local-go/internal/config"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// AuthMiddleware verifies JWT and checks user existence & soft delete
func AuthMiddleware(userRepo *repositories.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			return
		}

		// Parse JWT
		token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(config.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing user_id"})
			return
		}

		role, ok := claims["role"].(string)
		if !ok || role == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing role"})
			return
		}

		// ✅ Fetch user from DB and check soft delete
		user, err := userRepo.GetActiveByID(userID)
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
		if !exists || role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		c.Next()
	}
}

// OwnerOrAdmin ensures only owner of the resource or admin can access
func OwnerOrAdmin(getOwnerID func(c *gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := getStringFromContext(c, "user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		role, _ := getStringFromContext(c, "role")
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
