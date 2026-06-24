package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	rateLimitWindow = time.Minute
	rateLimitMax    = 60
)

// RateLimiter returns a Redis-backed rate limiter middleware. If redisAddr is
// empty or Redis is unreachable, it falls back to a best-effort in-memory
// limiter for single-instance use.
func RateLimiter(redisAddr string) gin.HandlerFunc {
	var rdb *redis.Client
	var useRedis bool
	if redisAddr != "" {
		rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := rdb.Ping(ctx).Err(); err == nil {
			useRedis = true
		}
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("rate:%s", ip)

		if useRedis {
			ctx := context.Background()
			cnt, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				// Redis error: do not block request, but do not award headers
				c.Next()
				return
			}
			if cnt == 1 {
				rdb.Expire(ctx, key, rateLimitWindow)
			}

			remaining := rateLimitMax - int(cnt)
			ttl, _ := rdb.TTL(ctx, key).Result()
			resetAt := time.Now().Add(ttl)

			if remaining < 0 {
				retryAfter := int(time.Until(resetAt).Seconds())
				if retryAfter < 0 {
					retryAfter = 0
				}
				c.Header("Retry-After", strconv.Itoa(retryAfter))
				c.Header("X-RateLimit-Limit", strconv.Itoa(rateLimitMax))
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
				return
			}

			c.Header("X-RateLimit-Limit", strconv.Itoa(rateLimitMax))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))
			c.Next()
			return
		}

		// Fallback: best-effort single-instance in-memory limiter
		// Keep behavior minimal to avoid panics if Redis not configured
		now := time.Now()
		// Simple in-memory key with timestamp-based window
		// Use Gin context to store per-process map
		visitorsAny, _ := c.Get("__visitors_map")
		var visitors map[string]struct {
			Count   int
			ResetAt time.Time
		}
		if visitorsAny == nil {
			visitors = map[string]struct {
				Count   int
				ResetAt time.Time
			}{}
			c.Set("__visitors_map", visitors)
		} else {
			visitors = visitorsAny.(map[string]struct {
				Count   int
				ResetAt time.Time
			})
		}

		v, ok := visitors[ip]
		if !ok || now.After(v.ResetAt) {
			v = struct {
				Count   int
				ResetAt time.Time
			}{Count: 1, ResetAt: now.Add(rateLimitWindow)}
			visitors[ip] = v
		} else {
			v.Count++
			visitors[ip] = v
		}

		remaining := rateLimitMax - v.Count
		if remaining < 0 {
			retryAfter := int(time.Until(v.ResetAt).Seconds())
			if retryAfter < 0 {
				retryAfter = 0
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.Header("X-RateLimit-Limit", strconv.Itoa(rateLimitMax))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", v.ResetAt.Unix()))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(rateLimitMax))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", v.ResetAt.Unix()))
		c.Next()
	}
}
