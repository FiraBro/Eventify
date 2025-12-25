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

    // 1. Staff Resource Routes (Under /staff)
    staff := api.Group("/staff")
    staff.Use(authMW) 
    {
        staff.GET("", handler.ListStaff)           
        staff.GET("/:id", handler.GetStaffDetails)       
        staff.GET("/:id/services", handler.GetServices)
        staff.GET("/:id/schedule", handler.GetSchedule)

        staff.POST("", adminMW, handler.Create)                  
        staff.PATCH("/:id", adminMW, handler.Update)            
        staff.DELETE("/:id", adminMW, handler.Delete)           
        staff.POST("/:id/services", adminMW, handler.AssignServices)
        staff.POST("/:id/schedule", adminMW, handler.SetSchedule)
        staff.POST("/:id/holidays", adminMW, handler.AddHoliday)
    }

    // 2. Availability Routes (Under /availability)
    // These are usually public or require basic auth
    availability := api.Group("/availability")
    availability.Use(authMW) 
    {
        // GET /api/v1/availability/services/:serviceId
        availability.GET("/services/:serviceId", handler.GetServiceAvailability)
        
        // GET /api/v1/availability/staff/:staffId
        availability.GET("/staff/:staffId", handler.GetStaffAvailability)
    }
}

