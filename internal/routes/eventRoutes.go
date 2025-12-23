package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/FiraBro/local-go/internal/repositories"

	"github.com/gin-gonic/gin"
)

// SetupEventRoutes sets up public and protected event routes
func SetupEventRoutes(rg *gin.RouterGroup, eventHandler *handlers.EventHandler, userRepo *repositories.UserRepository) {
	// Public routes
	rg.GET("/events", eventHandler.GetEvents)
	rg.GET("/events/:id", eventHandler.GetEventByID)

	// Protected routes (requires authentication)
	authGroup := rg.Group("/")
	authGroup.Use(middlewares.AuthMiddleware(userRepo))
	{
		authGroup.POST("/events", eventHandler.CreateEvent)
		authGroup.PUT("/events/:id", eventHandler.UpdateEvent)
		authGroup.DELETE("/events/:id", eventHandler.DeleteEvent)
	}
}
