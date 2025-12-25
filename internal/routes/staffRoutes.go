package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/middlewares"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
)

func StaffRoutes(
	api *gin.RouterGroup,
	handler *handlers.StaffHandler,
	userRepo *repositories.UserRepository,
) {
	authMW := middlewares.AuthMiddleware(userRepo)
	adminMW := middlewares.AdminOnly()

	staff := api.Group("/staff")
	staff.Use(authMW) // all routes require authentication
	{
		// Public-ish routes (list and get staff details)
		staff.GET("", handler.ListStaff)           // list all staff
		staff.GET("/:id", handler.GetStaffDetails)       // get staff details
		staff.GET("/:id/services", handler.GetServices)
		staff.GET("/:id/schedule", handler.GetSchedule)

		// Admin-only routes
		staff.POST("", adminMW, handler.Create)                  // create staff
		staff.PATCH("/:id", adminMW, handler.Update)            // update staff
		staff.DELETE("/:id", adminMW, handler.Delete)           // delete staff
		staff.POST("/:id/services", adminMW, handler.AssignServices)
		staff.POST("/:id/schedule", adminMW, handler.SetSchedule)
		staff.POST("/:id/holidays", adminMW, handler.AddHoliday)
	}
}

