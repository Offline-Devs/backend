package middleware

import (
    "net/http"
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
)

func cleanupExpiredVisitors(now time.Time) {
    for ip, v := range visitors {
        if now.After(v.resetAt) {
            delete(visitors, ip)
        }
    }
}

func RateLimiter() gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()
        now := time.Now()

        visitorsMu.Lock()
        // lightweight opportunistic cleanup to avoid unbounded map growth
        if len(visitors) > 1000 {
            cleanupExpiredVisitors(now)
        }

        v, ok := visitors[ip]
        if !ok || now.After(v.resetAt) {
            v = &visitor{remaining: rateLimitMax, resetAt: now.Add(rateLimitWindow)}
            visitors[ip] = v
        }

        if v.remaining <= 0 {
            visitorsMu.Unlock()
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
            return
        }
        v.remaining--
        visitorsMu.Unlock()

        c.Next()
    }
}
