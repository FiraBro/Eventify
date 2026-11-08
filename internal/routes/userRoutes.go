package routes

import (
	"time"

	"github.com/FiraBro/local-go/internal/config"
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
)

func AuthRoutes(api *gin.RouterGroup, authHandler *handlers.AuthHandler, userRepo *repositories.UserRepository) {
	rdb := config.RedisClient

	// Separate namespaces and windows to isolate registration abuse from standard credential requests
	loginLimiter := middlewares.RateLimiter(rdb, "login", 5, 1*time.Minute)
	registerLimiter := middlewares.RateLimiter(rdb, "register", 3, 1*time.Minute)
	resetLimiter := middlewares.RateLimiter(rdb, "reset", 3, 1*time.Minute)
	invitationLimiter := middlewares.RateLimiter(rdb, "staff-invitation-accept", 5, 1*time.Minute)
	resendInvitationLimiter := middlewares.RateLimiter(rdb, "staff-invitation-resend", 3, 1*time.Minute)

	// Public auth endpoints protected with isolated rate limit profiles
	api.POST("/auth/register", registerLimiter, authHandler.Register)
	api.POST("/auth/login", loginLimiter, authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)
	api.POST("/auth/logout", authHandler.Logout)
	api.POST("/auth/forgot-password", resetLimiter, authHandler.ForgotPassword)
	api.POST("/auth/reset-password", resetLimiter, authHandler.ResetPassword)
	api.POST("/auth/accept-invitation", invitationLimiter, authHandler.AcceptInvitation)
	// Retain the earlier partial implementation path as a compatibility alias.
	api.POST("/auth/invitation/complete", invitationLimiter, authHandler.CompleteInvitation)

	// Authenticated routes remain unthrottled here
	authMW := middlewares.AuthMiddleware(userRepo)
	adminMW := middlewares.AdminOnly()
	api.POST("/auth/resend-invitation", authMW, adminMW, resendInvitationLimiter, authHandler.ResendInvitation)
	api.GET("/auth/profile", authMW, authHandler.GetProfile)
	api.PATCH("/auth/profile", authMW, authHandler.UpdateProfile)
	api.PATCH("/auth/change-password", authMW, authHandler.ChangePassword)
	api.DELETE("/auth/delete-account", authMW, authHandler.DeleteUser)
	api.POST("/auth/restore-account", authMW, authHandler.RestoreUser)
}

func UserRoutes(api *gin.RouterGroup, authHandler *handlers.AuthHandler, userRepo *repositories.UserRepository) {
	authMW := middlewares.AuthMiddleware(userRepo)
	adminMW := middlewares.AdminOnly()

	// Admin-only routes under /users
	api.GET("/users", authMW, adminMW, authHandler.GetPaginatedUsers)       // GET /api/v1/users?page=1&limit=10
	api.POST("/users", authMW, adminMW, authHandler.CreateUserHandler)      // POST /api/v1/users
	api.PATCH("/users/:id", authMW, adminMW, authHandler.UpdateUserHandler) // PATCH /api/v1/users/:id
	api.PATCH("/users/:id/role", authMW, adminMW, authHandler.UpdateUserRole)
	api.DELETE("/users/:id", authMW, adminMW, authHandler.AdminDeleteUser)

	// Single user fetch (any authenticated user)
	api.GET("/users/:id", authMW, authHandler.GetUserByID)
}
