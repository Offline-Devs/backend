package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"github.com/yourusername/noshirvani-academy/backend/internal/infrastructure/auth"
	"gorm.io/gorm"
)

func AuthMiddleware(jwtService *auth.JWTService, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		parts := strings.Split(header, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}
		claims, err := jwtService.ValidateAccessToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		var user domain.User
		if err := db.First(&user, "id = ?", userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		if !user.IsActive {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user is inactive"})
			return
		}

		c.Set("user_id", user.ID)
		c.Set("role", user.Role)
		c.Set("user", user)
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

func RequireApprovedStudentOrAdmin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role == "admin" {
			c.Next()
			return
		}
		userID, ok := c.Get("user_id")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
			return
		}
		var student domain.Student
		if err := db.Where("user_id = ?", userID).First(&student).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "student approval required"})
			return
		}
		if !student.IsApproved {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "student approval required"})
			return
		}
		c.Set("student_id", student.ID)
		c.Set("student_approved", true)
		c.Next()
	}
}

func RequireApprovedStudent(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := c.Get("user_id")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
			return
		}

		var student domain.Student
		if err := db.Where("user_id = ?", userID).First(&student).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "student approval required"})
			return
		}
		if !student.IsApproved {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "student approval required"})
			return
		}

		c.Set("student_id", student.ID)
		c.Set("student_approved", true)
		c.Next()
	}
}
