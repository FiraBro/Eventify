package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/gin-gonic/gin"
)

// Accept *gin.RouterGroup instead of *gin.Engine
func AuthRoutes(rg *gin.RouterGroup, authHandler *handlers.AuthHandler) {
	rg.POST("/auth/register", authHandler.Register)
	rg.POST("/auth/login", authHandler.Login)
	rg.POST("/auth/refresh", authHandler.Refresh)
	rg.POST("/auth/logout", authHandler.Logout)
	rg.POST("/auth/forgot-password", authHandler.ForgotPassword)
	rg.POST("/auth/reset-password", authHandler.ResetPassword)

	// Protected routes
	authProtected := rg.Group("/")
	authProtected.Use(middlewares.AuthMiddleware())
	{
		authProtected.GET("/auth/profile", authHandler.GetProfile)
		authProtected.PATCH("/auth/profile", authHandler.UpdateProfile)
		authProtected.PATCH("/auth/change-password", authHandler.ChangePassword)
	}
}
