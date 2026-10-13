package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
)

// PaymentRoutes sets up routes for payments
func PaymentRoutes(api *gin.RouterGroup, handler *handlers.PaymentHandler, userRepo *repositories.UserRepository) {
	authMW := middlewares.AuthMiddleware(userRepo)
	adminMW := middlewares.AdminOnly()

	// Public route: called by Chapa, protected by signature + server-side verification
	api.POST("/payments/webhook", handler.Webhook)

	// Authenticated routes
	api.POST("/payments/checkout", authMW, handler.Checkout)
	api.GET("/payments/:bookingId/status", authMW, handler.GetStatus)

	// Admin-only routes
	api.POST("/payments/refund/:bookingId", authMW, adminMW, handler.Refund)
}