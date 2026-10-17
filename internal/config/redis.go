package config

import (
	"context"
	"github.com/redis/go-redis/v9"
	"os"
	"time"
)

var RedisClient *redis.Client

// InitRedis configures and initializes the global Redis connection pool.
func InitRedis() error {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		// Fallback to basic string initialization if not a full redis:// URI
		opts = &redis.Options{
			Addr: redisURL,
		}
	}

	RedisClient = redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return RedisClient.Ping(ctx).Err()
}
