package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

    "github.com/gin-gonic/gin"
)

type visitor struct {
    remaining int
    resetAt   time.Time
}

const (
    rateLimitWindow = time.Minute
    rateLimitMax    = 60
)

var (
	visitors   = map[string]*visitor{}
	visitorsMu sync.Mutex
	lastCleanup time.Time
)

func cleanupExpiredVisitors(now time.Time) {
	for ip, v := range visitors {
        if now.After(v.resetAt) {
			delete(visitors, ip)
		}
	}
	lastCleanup = now
}

func RateLimiter() gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()
        now := time.Now()

		visitorsMu.Lock()
		if lastCleanup.IsZero() || now.Sub(lastCleanup) >= rateLimitWindow {
			cleanupExpiredVisitors(now)
		}

        v, ok := visitors[ip]
		if !ok || now.After(v.resetAt) {
			v = &visitor{remaining: rateLimitMax, resetAt: now.Add(rateLimitWindow)}
			visitors[ip] = v
		}
		resetAt := v.resetAt

		if v.remaining <= 0 {
			visitorsMu.Unlock()
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
		v.remaining--
		remaining := v.remaining
		visitorsMu.Unlock()

		c.Header("X-RateLimit-Limit", strconv.Itoa(rateLimitMax))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))

		c.Next()
	}
}
