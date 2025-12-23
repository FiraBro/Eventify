package config

import (
	"log"
	"os"
	"time"
)

var (
	// Server
	ServerPort = getEnv("SERVER_PORT", "8080")

	// Database
	DBAddr = getDBAddr()

	// JWT
	JWTSecret = []byte(getEnv("JWT_SECRET", "my-first-go-jwt-secret-for-api-i-really-excited"))
	JWTExpiry = 24 * time.Hour

	// SMTP
	SMTPHost = getEnv("SMTP_HOST", "smtp.gmail.com")
	SMTPPort = getEnv("SMTP_PORT", "587")
	SMTPUser = getEnv("SMTP_USER", "")
	SMTPPass = getEnv("SMTP_PASS", "")
)

// ----------------------------
// Helper to get environment variable or fallback
// ----------------------------
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// ----------------------------
// Database address helper
// ----------------------------
func getDBAddr() string {
	dbHost := "localhost"
	if os.Getenv("DOCKER_ENV") == "true" {
		dbHost = "db"
	}
	return getEnv("DB_ADDR", "postgres://local_go_user:password@"+dbHost+":5432/local_go_db?sslmode=disable")
}

// ----------------------------
// Validate critical config
// ----------------------------
func ValidateConfig() {
	if len(JWTSecret) == 0 {
		log.Panic("❌ JWT_SECRET is not set")
	}

	if DBAddr == "" {
		log.Panic("❌ DB_ADDR is not set")
	}

	if SMTPUser == "" || SMTPPass == "" {
		log.Println("⚠ Warning: SMTP credentials are not set")
	}
}
