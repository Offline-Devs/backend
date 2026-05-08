package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/config"
	"github.com/yourusername/noshirvani-academy/backend/internal/handler"
	"github.com/yourusername/noshirvani-academy/backend/internal/infrastructure/auth"
	"github.com/yourusername/noshirvani-academy/backend/internal/infrastructure/sms"
	"github.com/yourusername/noshirvani-academy/backend/internal/middleware"
	"gorm.io/gorm"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.Default()

	if cfg.UploadPath != "" {
		r.Static("/uploads", cfg.UploadPath)
	}

	r.Use(middleware.CORS(cfg.CORSOrigins))
	r.Use(middleware.RateLimiter())

	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTRefreshSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	otpStore := sms.NewOTPStore()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		authH := handler.NewAuthHandler(db, jwtService, otpStore, cfg.OTPProvider, cfg.AdminPhones)
		blogH := handler.NewBlogHandler(db)
		api.POST("/auth/request-otp", authH.RequestOTP)
		api.POST("/auth/verify-otp", authH.VerifyOTP)
		api.POST("/auth/refresh", authH.RefreshToken)
		api.GET("/blog", blogH.PublicList)
		api.GET("/blog/:slug", blogH.PublicGet)

		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(jwtService))
		{
			studentH := handler.NewStudentHandler(db)
			protected.POST("/students/profile", studentH.CompleteProfile)
			protected.GET("/students/profile", studentH.GetProfile)

			examH := handler.NewExamHandler(db)
			protected.POST("/exams", examH.CreateExam)
			protected.GET("/exams", examH.ListExams)
			protected.GET("/exams/:id", examH.GetExam)
			protected.DELETE("/exams/:id", examH.DeleteExam)

			mistakeH := handler.NewMistakeHandler(db)
			protected.POST("/mistakes", mistakeH.Create)
			protected.GET("/mistakes", mistakeH.List)
			protected.DELETE("/mistakes/:id", mistakeH.Delete)

			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRole("admin"))
			{
				adminH := handler.NewAdminHandler(db)
				admin.GET("/students", adminH.ListStudents)
				admin.GET("/students/:id", adminH.GetStudent)
				admin.PUT("/students/:id", adminH.UpdateStudent)
				admin.PUT("/students/:id/approve", adminH.ApproveStudent)
				admin.DELETE("/students/:id", adminH.DeleteStudent)
				admin.GET("/dynamic-fields", adminH.GetDynamicFields)
				admin.POST("/dynamic-fields", adminH.CreateDynamicField)
				admin.PUT("/dynamic-fields/:id", adminH.UpdateDynamicField)
				admin.DELETE("/dynamic-fields/:id", adminH.DeleteDynamicField)
				admin.GET("/blog", blogH.List)
				admin.POST("/blog", blogH.Create)
				admin.PUT("/blog/:id", blogH.Update)
				admin.PUT("/blog/:id/publish", blogH.Publish)
				admin.DELETE("/blog/:id", blogH.Delete)
			}
		}
	}

	return r
}
