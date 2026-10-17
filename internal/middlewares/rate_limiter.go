package middlewares

import (
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Lua script to atomically increment request counter and set an expiration window if new
const rateLimitLuaScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

local current = redis.call("INCR", key)
if current == 1 then
    redis.call("EXPIRE", key, window)
end
return current
`

// RateLimiter returns a Gin middleware that enforces an atomic request threshold per IP
func RateLimiter(rdb *redis.Client, namespace string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			// Fail-open strategy to protect business continuity if Redis configuration is entirely absent
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		key := fmt.Sprintf("ratelimit:%s:%s", namespace, clientIP)
		windowSeconds := int(window.Seconds())

		// Atomic execution via Lua script execution pool
		res, err := rdb.Eval(c.Request.Context(), rateLimitLuaScript, []string{key}, limit, windowSeconds).Result()
		if err != nil {
			// Fail-safe logic: Log the infrastructure error but allow the request to prevent service disruption
			log.Printf("[RATE_LIMITER_WARN] Redis connection down: %v. Falling back to fail-open policy.", err)
			c.Next()
			return
		}

		currentRequests, ok := res.(int64)
		if ok && currentRequests > int64(limit) {
			c.Header("Retry-After", strconv.Itoa(windowSeconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}

		c.Next()
	}
}
