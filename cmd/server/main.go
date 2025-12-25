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
	// 1. Load Environment Variables
	if err := godotenv.Load(); err != nil {
		log.Println("⚠ .env file not found, using system environment variables")
	}

	// 2. Configuration & Database
	config.ValidateConfig()
	dbConn := db.InitDB()

	// ------------------------
	// 3. Repositories
	// ------------------------
	userRepo := repositories.NewUserRepository(dbConn)
	staffRepo := repositories.NewStaffRepository(dbConn)
	serviceRepo := repositories.NewServiceRepository(dbConn)
	refreshRepo := repositories.NewRefreshTokenRepository(dbConn)
	resetRepo := repositories.NewResetTokenRepository(dbConn)

	// ------------------------
	// 4. Services (FIXED DEPENDENCIES)
	// ------------------------
	authService := services.NewAuthService(userRepo, refreshRepo, resetRepo)
	
	// StaffService needs BOTH staffRepo and serviceRepo to manage relationships
	staffService := services.NewStaffService(staffRepo, serviceRepo) 
	
	serviceService := services.NewServiceService(serviceRepo)

	// ------------------------
	// 5. Handlers
	// ------------------------
	authHandler := handlers.NewAuthHandler(authService)
	staffHandler := handlers.NewStaffHandler(staffService)
	serviceHandler := handlers.NewServiceHandler(serviceService)

	// ------------------------
	// 6. Router Setup
	// ------------------------
	r := gin.Default()
	
	// Apply Global Middlewares (CORS, etc.) if you have them
	// r.Use(middlewares.CORS())

	api := r.Group(version)

	// Initialize Routes
	routes.AuthRoutes(api, authHandler, userRepo)
	routes.UserRoutes(api, authHandler, userRepo)
	routes.StaffRoutes(api, staffHandler, userRepo)
	routes.ServiceRoutes(api, serviceHandler, userRepo)

	// ------------------------
	// 7. Start Server
	// ------------------------
	log.Println("✅ Server running on http://localhost:8080" + version)
	if err := r.Run(":8080"); err != nil {
		log.Fatal("❌ Failed to start server:", err)
	}
}