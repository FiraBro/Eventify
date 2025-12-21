package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupEventRoutes(r *gin.Engine, eventHandler *handlers.EventHandler) {
	// Public routes
	r.GET("/events", eventHandler.GetEvents)
	r.GET("/events/:id", eventHandler.GetEventByID)

	// Protected routes
	authGroup := r.Group("/")
	authGroup.Use(middlewares.JWTAuth())
	{
		authGroup.POST("/events", eventHandler.CreateEvent)
		authGroup.PUT("/events/:id", eventHandler.UpdateEvent)
		authGroup.DELETE("/events/:id", eventHandler.DeleteEvent)
	}
}
