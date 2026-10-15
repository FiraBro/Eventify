package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
)

// BookingRoutes sets up routes for bookings
func BookingRoutes(api *gin.RouterGroup, handler *handlers.BookingHandler, userRepo *repositories.UserRepository) {
	authMW := middlewares.AuthMiddleware(userRepo)
	adminMW := middlewares.AdminOnly()

	// Authenticated routes (ownership / role checks happen in the service layer)
	api.POST("/bookings", authMW, handler.Create)
	api.GET("/bookings/my", authMW, handler.GetMine) // must be before /bookings/:id
	api.GET("/bookings/:id", authMW, handler.GetByID)
	api.PATCH("/bookings/:id", authMW, handler.Update)
	api.DELETE("/bookings/:id", authMW, handler.Delete)
	api.PATCH("/bookings/:id/status", authMW, handler.UpdateStatus)

	// Admin-only routes
	api.GET("/bookings", authMW, adminMW, handler.GetAll)
}
