package config

import "os"

var (
	ServerPort = getEnv("SERVER_PORT", "8080")
	DBPath     = getEnv("DB_PATH", "./api.db")
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
