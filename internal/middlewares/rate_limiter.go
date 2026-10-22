package middlewares

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

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

const (
	maxFallbackEntries = 10000
	cleanupInterval    = time.Minute
)

// fallbackEntry tracks a fixed-window counter for one client IP.
type fallbackEntry struct {
	count       int
	windowStart time.Time
}

// fallbackLimiter provides bounded, per-process rate limiting when Redis
// cannot be used. It is not a replacement for distributed rate limiting.
type fallbackLimiter struct {
	mu          sync.Mutex
	entries     map[string]fallbackEntry
	lastCleanup time.Time
}

func newFallbackLimiter() *fallbackLimiter {
	return &fallbackLimiter{
		entries: make(map[string]fallbackEntry),
	}
}

// allow returns whether this request is within the configured limit.
// When the map is full, expired entries are removed. If it remains full,
// unknown clients are rejected rather than bypassing the rate limit.
func (l *fallbackLimiter) allow(ip string, limit int, window time.Duration, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastCleanup) >= cleanupInterval {
		for key, entry := range l.entries {
			if now.Sub(entry.windowStart) >= window {
				delete(l.entries, key)
			}
		}
		l.lastCleanup = now
	}

	entry, exists := l.entries[ip]
	if exists && now.Sub(entry.windowStart) >= window {
		delete(l.entries, ip)
		exists = false
	}

	if !exists {
		if len(l.entries) >= maxFallbackEntries {
			return false
		}

		l.entries[ip] = fallbackEntry{
			count:       1,
			windowStart: now,
		}
		return limit >= 1
	}

	if entry.count >= limit {
		return false
	}

	entry.count++
	l.entries[ip] = entry
	return true
}

// RateLimiter uses Redis as the primary distributed limiter and falls back
// to bounded in-memory counters when Redis is absent or unavailable.
func RateLimiter(rdb *redis.Client, namespace string, limit int, window time.Duration) gin.HandlerFunc {
	fallback := newFallbackLimiter()

	// Avoid accepting nonsensical limits or windows.
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}

	windowSeconds := int(window / time.Second)
	if windowSeconds < 1 {
		windowSeconds = 1
	}

	var lastRedisWarning atomic.Int64

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		allowFallback := func() bool {
			return fallback.allow(ip, limit, window, now)
		}

		if rdb == nil {
			if !allowFallback() {
				rejectRateLimited(c, windowSeconds)
				return
			}
			c.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:%s:%s", namespace, ip)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 500*time.Millisecond)
		res, err := rdb.Eval(ctx, rateLimitLuaScript, []string{key}, limit, windowSeconds).Result()
		cancel()

		if err != nil {
			// Log at most once per minute per middleware instance.
			unixNow := now.Unix()
			last := lastRedisWarning.Load()
			if unixNow-last >= 60 && lastRedisWarning.CompareAndSwap(last, unixNow) {
				slog.Warn("Redis rate limiter unavailable; using local fallback",
					"namespace", namespace,
				)
			}

			if !allowFallback() {
				rejectRateLimited(c, windowSeconds)
				return
			}
			c.Next()
			return
		}

		currentRequests, ok := res.(int64)
		if !ok {
			// An unexpected Redis result must not silently bypass protection.
			if !allowFallback() {
				rejectRateLimited(c, windowSeconds)
				return
			}
			c.Next()
			return
		}

		if currentRequests > int64(limit) {
			rejectRateLimited(c, windowSeconds)
			return
		}

		c.Next()
	}
}

func rejectRateLimited(c *gin.Context, windowSeconds int) {
	c.Header("Retry-After", strconv.Itoa(windowSeconds))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"error": "Too many requests. Please try again later.",
	})
}
