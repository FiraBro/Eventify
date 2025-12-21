package main

import (
	"github.com/FiraBro/local-go/internal/db"
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/FiraBro/local-go/internal/routes"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
)

func main() {
    db.InitDB()

    // Event routes
    eventRepo := repositories.NewEventRepository()
    eventService := services.NewEventService(eventRepo)
    eventHandler := handlers.NewEventHandler(eventService)

    // Auth routes - SEPARATE service needed
    authRepo := repositories.NewUserRepository()  // or whatever your auth repo is
    authService := services.NewAuthService(authRepo)
    authHandler := handlers.NewAuthHandler(authService)  // ✅ Correct

    r := gin.Default()
    routes.SetupEventRoutes(r, eventHandler)
    routes.SetupAuthRoutes(r, authHandler)

    r.Run(":8080")
}
