package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
)

// LogRoutes sets up routes for activity logs (admin-only)
func LogRoutes(api *gin.RouterGroup, handler *handlers.LogHandler, userRepo *repositories.UserRepository) {
	authMW := middlewares.AuthMiddleware(userRepo)
	adminMW := middlewares.AdminOnly()

	api.GET("/logs", authMW, adminMW, handler.List)
}