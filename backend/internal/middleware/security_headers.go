package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders sets common HTTP security headers to mitigate XSS, clickjacking,
// and MIME sniffing attacks. Adjust values as needed for your application.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("Referrer-Policy", "no-referrer")
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'")
		// HSTS: only set if your app is served over HTTPS in production
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		c.Next()
	}
}
