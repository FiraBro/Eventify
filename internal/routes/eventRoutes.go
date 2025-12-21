package routes

import (
	"local-go/internal/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, h *handlers.EventHandler) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Event Booking API 🚀"})
	})

	r.GET("/events", h.GetEvents)
	r.GET("/events/:id", h.GetEventByID)
	r.POST("/events", h.CreateEvent)
	r.PUT("/events/:id", h.UpdateEvent)
	r.DELETE("/events/:id", h.DeleteEvent)


}
