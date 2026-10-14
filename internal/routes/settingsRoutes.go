package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
)

// SettingsRoutes sets up routes for settings
func SettingsRoutes(api *gin.RouterGroup, handler *handlers.SettingsHandler, userRepo *repositories.UserRepository) {
	authMW := middlewares.AuthMiddleware(userRepo)
	adminMW := middlewares.AdminOnly()

	// Any logged-in user can read (frontends need currency, booking rules, contact info)
	api.GET("/settings", authMW, handler.Get)

	// Admin-only
	api.PATCH("/settings", authMW, adminMW, handler.Update)
}