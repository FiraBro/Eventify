package main

import (
	"github.com/FiraBro/local-go/internal/db"
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/FiraBro/local-go/internal/routes"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
)

const version = "/api/v1"

func main() {
	dbConn := db.InitDB() // must return *sql.DB

	// Event routes
	eventRepo := repositories.NewEventRepository(dbConn)
	eventService := services.NewEventService(eventRepo)
	eventHandler := handlers.NewEventHandler(eventService)

	// Auth routes
	authRepo := repositories.NewUserRepository(dbConn)
	refreshRepo := repositories.NewRefreshTokenRepository(dbConn)
	resetRepo := repositories.NewResetTokenRepository(dbConn)
	authService := services.NewAuthService(authRepo, refreshRepo, resetRepo)
	authHandler := handlers.NewAuthHandler(authService)

	r := gin.Default()

	// Create a group for API versioning
	api := r.Group(version)

	// Register routes under the group
	routes.SetupEventRoutes(api, eventHandler) // all event routes prefixed with /api/v1
	routes.AuthRoutes(api, authHandler)       // all auth routes prefixed with /api/v1

	r.Run(":8080")
}
