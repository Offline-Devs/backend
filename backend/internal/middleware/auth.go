package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/infrastructure/auth"
)

func AuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		var tokenString string

		// Handle both "Bearer <token>" and direct "<token>" formats
		// This supports both standard requests and Swagger UI testing
		if strings.HasPrefix(header, "Bearer ") {
			// Standard format: "Bearer <token>"
			tokenString = strings.TrimPrefix(header, "Bearer ")
		} else if strings.HasPrefix(header, "Bearer") {
			// Handle case where there's no space after Bearer
			tokenString = strings.TrimPrefix(header, "Bearer")
		} else {
			// Assume the entire header is the token (for Swagger UI compatibility)
			tokenString = header
		}

		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		claims, err := jwtService.ValidateAccessToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("user_id", claims["user_id"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, _ := c.Get("role")
		if userRole != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}
