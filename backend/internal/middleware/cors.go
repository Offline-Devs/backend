package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins []string, allowAnyOrigin bool) gin.HandlerFunc {
	// Normalize allowed origins for case-insensitive comparison
	normalized := make([]string, 0, len(allowedOrigins))
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		normalized = append(normalized, strings.ToLower(strings.TrimRight(o, "/")))
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		originNorm := strings.ToLower(strings.TrimSpace(strings.TrimRight(origin, "/")))
		allowOrigin := ""
		if allowAnyOrigin && len(normalized) == 0 {
			allowOrigin = "*"
		} else {
			for _, allowed := range normalized {
				if strings.EqualFold(allowed, originNorm) {
					allowOrigin = origin
					break
				}
			}
		}

		if allowOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			c.Writer.Header().Set("Vary", "Origin")
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			if origin != "" && allowOrigin == "" {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
