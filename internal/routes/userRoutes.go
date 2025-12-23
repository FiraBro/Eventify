package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
)

func AuthRoutes(api *gin.RouterGroup, authHandler *handlers.AuthHandler, userRepo *repositories.UserRepository) {
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)
	api.POST("/auth/logout", authHandler.Logout)

	api.POST("/auth/forgot-password", authHandler.ForgotPassword)
	api.POST("/auth/reset-password", authHandler.ResetPassword)

	// Pass userRepo to middleware
	authMW := middlewares.AuthMiddleware(userRepo)

	api.GET("/auth/profile", authMW, authHandler.GetProfile)
	api.PATCH("/auth/profile", authMW, authHandler.UpdateProfile)
	api.PATCH("/auth/change-password", authMW, authHandler.ChangePassword)

	// 🔥 Optional: soft-delete aware routes
	api.DELETE("/auth/delete-account", authMW, authHandler.DeleteUser)
	api.POST("/auth/restore-account", authMW, authHandler.RestoreUser)
}
