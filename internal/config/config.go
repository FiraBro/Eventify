package config

import (
	"os"
	"time"
)

var (
	ServerPort = getEnv("SERVER_PORT", "8080")
	DBPath     = getEnv("DB_PATH", "./api.db")
	JWTSecret  = getEnv("JWT_SECRET", "your-super-secret-key-min-32-chars-long")
	JWTExpiry  = 24 * time.Hour  // ✅ FIXED: Use time.Hour directly
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
