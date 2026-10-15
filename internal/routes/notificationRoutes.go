package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
)

// NotificationRoutes sets up routes for notifications
func NotificationRoutes(api *gin.RouterGroup, handler *handlers.NotificationHandler, userRepo *repositories.UserRepository) {
	authMW := middlewares.AuthMiddleware(userRepo)
	adminMW := middlewares.AdminOnly()

	// Authenticated routes (a user only ever sees their own notifications)
	api.GET("/notifications", authMW, handler.GetMine)
	api.PATCH("/notifications/:id/read", authMW, handler.MarkRead)

	// Admin-only routes
	api.POST("/notifications/send", authMW, adminMW, handler.Send)
}
