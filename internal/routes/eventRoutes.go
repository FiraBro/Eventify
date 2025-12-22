package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"

	"github.com/gin-gonic/gin"
)

// Accept *gin.RouterGroup instead of *gin.Engine
func SetupEventRoutes(rg *gin.RouterGroup, eventHandler *handlers.EventHandler) {
	// Public routes
	rg.GET("/events", eventHandler.GetEvents)
	rg.GET("/events/:id", eventHandler.GetEventByID)

	// Protected routes
	authGroup := rg.Group("/")
	authGroup.Use(middlewares.AuthMiddleware())
	{
		authGroup.POST("/events", eventHandler.CreateEvent)
		authGroup.PUT("/events/:id", eventHandler.UpdateEvent)
		authGroup.DELETE("/events/:id", eventHandler.DeleteEvent)
	}
}
