package main

import (
	"log"

	"github.com/FiraBro/local-go/internal/config"
	"github.com/FiraBro/local-go/internal/db"
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/FiraBro/local-go/internal/routes"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

const version = "/api/v1"

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("⚠ .env file not found, using system environment variables")
	}

	// Validate env
	config.ValidateConfig()

	// Init DB
	dbConn := db.InitDB()

	// ------------------------
	// Repositories
	// ------------------------
	userRepo := repositories.NewUserRepository(dbConn)
	staffRepo := repositories.NewStaffRepository(dbConn)
	serviceRepo := repositories.NewServiceRepository(dbConn)

	refreshRepo := repositories.NewRefreshTokenRepository(dbConn)
	resetRepo := repositories.NewResetTokenRepository(dbConn)

	// ------------------------
	// Services
	// ------------------------
	authService := services.NewAuthService(userRepo, refreshRepo, resetRepo)
	staffService := services.NewStaffService(staffRepo)
	serviceService := services.NewServiceService(serviceRepo)

	// ------------------------
	// Handlers
	// ------------------------
	authHandler := handlers.NewAuthHandler(authService)
	staffHandler := handlers.NewStaffHandler(staffService)
	serviceHandler := handlers.NewServiceHandler(serviceService)

	// ------------------------
	// Gin setup
	// ------------------------
	r := gin.Default()
	api := r.Group(version)

	// Auth routes
	routes.AuthRoutes(api, authHandler, userRepo)

	// User management routes
	routes.UserRoutes(api, authHandler, userRepo)

	// Staff routes (REPLACED event routes)
	routes.StaffRoutes(api, staffHandler, userRepo)

	// Service routes
	routes.ServiceRoutes(api, serviceHandler,userRepo)

	// ------------------------
	// Start server
	// ------------------------
	log.Println("✅ Server running on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("❌ Failed to start server:", err)
	}
}
