package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins []string, allowAnyOrigin bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowOrigin := ""
		if allowAnyOrigin && len(allowedOrigins) == 0 {
			allowOrigin = "*"
		} else {
			for _, allowed := range allowedOrigins {
				if allowed == origin {
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
