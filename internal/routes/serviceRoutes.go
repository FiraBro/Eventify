package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
)

// ServiceRoutes sets up routes for services
func ServiceRoutes(api *gin.RouterGroup, handler *handlers.ServiceHandler, userRepo *repositories.UserRepository) {
	// Public routes
	api.GET("/services", handler.GetAll)
	api.GET("/services/:id", handler.GetByID)
	api.GET("/services/categories", handler.Categories)

	// Authenticated middleware
	authMW := middlewares.AuthMiddleware(userRepo)
	adminMW := middlewares.AdminOnly()

	// Admin-only routes
	api.POST("/services", authMW, adminMW, handler.Create)
	api.PATCH("/services/:id", authMW, adminMW, handler.Update)
	api.DELETE("/services/:id", authMW, adminMW, handler.Delete)
}
