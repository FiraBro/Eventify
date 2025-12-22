package config

import (
	"os"
	"time"
)

var (
	ServerPort = getEnv("SERVER_PORT", "8080")
	DBPath     = getEnv("DB_PATH", "./api.db")
   JWTSecret = []byte(getEnv("JWT_SECRET", "my-first-go-jwt-secret-for-api-i-really-excited"))
	JWTExpiry  = 24 * time.Hour  // ✅ FIXED: Use time.Hour directly
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func ValidateConfig() {
	if len(JWTSecret) == 0 {
		panic("❌ JWT_SECRET is not set")
	}
}
