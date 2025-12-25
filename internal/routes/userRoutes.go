package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
)

// AuthRoutes sets up authentication routes
func AuthRoutes(api *gin.RouterGroup, authHandler *handlers.AuthHandler, userRepo *repositories.UserRepository) {
	// Public auth endpoints
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)
	api.POST("/auth/logout", authHandler.Logout)
	api.POST("/auth/forgot-password", authHandler.ForgotPassword)
	api.POST("/auth/reset-password", authHandler.ResetPassword)

	// Authenticated routes
	authMW := middlewares.AuthMiddleware(userRepo)
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
	api.DELETE("/users/:id", authMW, adminMW, authHandler.DeleteUser)

	// Single user fetch (any authenticated user)
	api.GET("/users/:id", authMW, authHandler.GetUserByID)
}

