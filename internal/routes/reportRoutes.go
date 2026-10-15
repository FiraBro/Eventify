package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
)

// ReportRoutes sets up routes for reports (all admin-only)
func ReportRoutes(api *gin.RouterGroup, handler *handlers.ReportHandler, userRepo *repositories.UserRepository) {
	authMW := middlewares.AuthMiddleware(userRepo)
	adminMW := middlewares.AdminOnly()

	reports := api.Group("/reports", authMW, adminMW)
	{
		reports.GET("/bookings", handler.Bookings)
		reports.GET("/revenue", handler.Revenue)
		reports.GET("/services", handler.Services)
		reports.GET("/staff-performance", handler.StaffPerformance)
		reports.GET("/customers", handler.Customers)
	}
}
