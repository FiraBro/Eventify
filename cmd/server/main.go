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

	// Debug log: confirm SMTP values before using them
	log.Println("SMTP_HOST:", config.SMTPHost)
	log.Println("SMTP_PORT:", config.SMTPPort)
	log.Println("SMTP_USER:", config.SMTPUser)

	// Validate that all required env variables are set
	config.ValidateConfig()

	log.Println("📦 SQLite DB Path:", config.DBAddr)

	// Initialize SQLite DB
	dbConn := db.InitDB() // returns *sql.DB

	// ------------------------
	// Event routes setup
	// ------------------------
	eventRepo := repositories.NewEventRepository(dbConn)
	eventService := services.NewEventService(eventRepo)
	eventHandler := handlers.NewEventHandler(eventService)

	// ------------------------
	// Auth routes setup
	// ------------------------
	authRepo := repositories.NewUserRepository(dbConn)
	refreshRepo := repositories.NewRefreshTokenRepository(dbConn)
	resetRepo := repositories.NewResetTokenRepository(dbConn)
	authService := services.NewAuthService(authRepo, refreshRepo, resetRepo)
	authHandler := handlers.NewAuthHandler(authService)

	// ------------------------
	// Gin router setup
	// ------------------------
	r := gin.Default()

	// Create API versioning group
	api := r.Group(version)

	// Register routes
	routes.SetupEventRoutes(api, eventHandler, authRepo) // ✅ pass authRepo for AuthMiddleware
	routes.AuthRoutes(api, authHandler, authRepo)        // ✅ same as before

	// Start server
	log.Println("✅ Server running on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
