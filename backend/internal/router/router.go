package router

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/config"
	"github.com/yourusername/noshirvani-academy/backend/internal/handler"
	"github.com/yourusername/noshirvani-academy/backend/internal/infrastructure/auth"
	"github.com/yourusername/noshirvani-academy/backend/internal/infrastructure/sms"
	"github.com/yourusername/noshirvani-academy/backend/internal/middleware"
	"gorm.io/gorm"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/yourusername/noshirvani-academy/backend/docs"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.Default()

	if cfg.UploadPath != "" {
		r.Static("/uploads", cfg.UploadPath)
	}

	r.Use(middleware.CORS(cfg.CORSOrigins))
	r.Use(middleware.RateLimiter())

	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTRefreshSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	otpStore := sms.NewOTPStore(cfg.RedisAddr, cfg.SMSIRAPIKey, cfg.SMSIRTemplateID)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/swagger-doc/doc.json", func(c *gin.Context) {
		doc := docs.SwaggerInfo.ReadDoc()
		if doc == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Swagger doc is empty. Did you import the docs package?"})
			return
		}

		var swaggerMap map[string]interface{}
		if err := json.Unmarshal([]byte(doc), &swaggerMap); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse swagger doc: " + err.Error()})
			return
		}

		swaggerMap["host"] = c.Request.Host
		prefix := c.GetHeader("X-Forwarded-Prefix")
		if prefix != "" {
			swaggerMap["basePath"] = prefix
		} else {
			swaggerMap["basePath"] = "/"
		}

		c.JSON(http.StatusOK, swaggerMap)
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("../swagger-doc/doc.json"),
	))

	api := r.Group("/")
	{
		authH := handler.NewAuthHandler(db, jwtService, otpStore, cfg.OTPProvider, cfg.AdminPhones)
		blogH := handler.NewBlogHandler(db)
		subjectsH := handler.NewSubjectsHandler()

		api.POST("/auth/request-otp", authH.RequestOTP)
		api.POST("/auth/verify-otp", authH.VerifyOTP)
		api.POST("/auth/refresh", authH.RefreshToken)
		api.GET("/blog", blogH.PublicList)
		api.GET("/blog/:slug", blogH.PublicGet)
		api.GET("/subjects", subjectsH.GetSubjectsByMajor)
		api.GET("/majors", subjectsH.GetAllMajors)

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
			protected.PUT("/exams/:id", examH.UpdateExam)
			protected.DELETE("/exams/:id", examH.DeleteExam)

			mistakeH := handler.NewMistakeHandler(db)
			protected.POST("/mistakes", mistakeH.Create)
			protected.GET("/mistakes", mistakeH.List)
			protected.PUT("/mistakes/:id", mistakeH.Update)
			protected.DELETE("/mistakes/:id", mistakeH.Delete)

			performanceH := handler.NewPerformanceHandler(db)
			protected.GET("/students/performance", performanceH.GetStudentPerformance)

			statisticsH := handler.NewStatisticsHandler(db)
			protected.GET("/students/statistics", statisticsH.GetStudentStatistics)
			protected.GET("/students/dashboard", statisticsH.GetDashboardSummary)

			uploadH := handler.NewUploadHandler(cfg.UploadPath)
			protected.POST("/upload", uploadH.UploadFile)
			protected.POST("/upload/multiple", uploadH.UploadMultiple)

			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRole("admin"))
			{
				adminH := handler.NewAdminHandler(db)
				admin.GET("/students", adminH.ListStudents)
				admin.GET("/students/with-stats", adminH.GetAllStudentsWithStats)
				admin.GET("/students/:id", adminH.GetStudent)
				admin.GET("/students/:id/exams", adminH.GetStudentExams)
				admin.GET("/students/:id/mistakes", adminH.GetStudentMistakes)
				admin.PUT("/students/:id", adminH.UpdateStudent)
				admin.PUT("/students/:id/approve", adminH.ApproveStudent)
				admin.DELETE("/students/:id", adminH.DeleteStudent)
				admin.GET("/students/:id/performance", performanceH.AdminListStudentPerformance)
				admin.POST("/students/:id/study-plans", performanceH.AdminCreateStudyPlan)
				admin.PUT("/study-plans/:id", performanceH.AdminUpdateStudyPlan)
				admin.DELETE("/study-plans/:id", performanceH.AdminDeleteStudyPlan)
				admin.POST("/students/:id/notes", performanceH.AdminCreateNote)
				admin.PUT("/notes/:id", performanceH.AdminUpdateNote)
				admin.DELETE("/notes/:id", performanceH.AdminDeleteNote)
				admin.GET("/students/:id/statistics", statisticsH.AdminGetStudentStatistics)
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
