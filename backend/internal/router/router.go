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
	// Configure trusted proxies if provided
	if len(cfg.TrustedProxies) > 0 {
		if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
			// If proxy configuration fails, continue but log to stdout to aid debugging
		}
	}

	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.CORSOrigins, cfg.Environment == "development" || cfg.Environment == "test"))
	r.Use(middleware.RateLimiter(cfg.RedisAddr))

	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTRefreshSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	otpStore := sms.NewOTPStore(
		cfg.RedisAddr,
		cfg.OTPProvider,
		cfg.SMSIRAPIKey,
		cfg.SMSIRTemplateID,
	)

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

		// Override basePath dynamically per-request using X-Forwarded-Prefix from nginx
		prefix := c.GetHeader("X-Forwarded-Prefix")
		if prefix != "" {
			swaggerMap["basePath"] = prefix
		} else {
			swaggerMap["basePath"] = "/"
		}

		c.JSON(http.StatusOK, swaggerMap)
	})

	// 2. Point ginSwagger to the new non-conflicting path
	r.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		// Use a relative path so it automatically includes Nginx's '/api/v1' prefix
		ginSwagger.URL("../swagger-doc/doc.json"),
	))

	api := r.Group("/")
	{
		authH := handler.NewAuthHandler(db, jwtService, otpStore, cfg.OTPProvider, cfg.ExposeMockOTP, cfg.AdminPhones)
		blogH := handler.NewBlogHandler(db)
		subjectsH := handler.NewSubjectsHandler()

		// Public routes
		api.POST("/auth/request-otp", authH.RequestOTP)
		api.POST("/auth/verify-otp", authH.VerifyOTP)
		api.POST("/auth/refresh", authH.RefreshToken)
		api.GET("/blog", blogH.PublicList)
		api.GET("/blog/*slug", blogH.PublicGet)
		api.GET("/subjects", subjectsH.GetSubjectsByMajor)
		api.GET("/majors", subjectsH.GetAllMajors)

		authenticated := api.Group("")
		authenticated.Use(middleware.AuthMiddleware(jwtService, db))
		{
			adminH := handler.NewAdminHandler(db)
			studentH := handler.NewStudentHandler(db)
			authenticated.POST("/students/profile", middleware.RequireRole("student"), studentH.CompleteProfile)
			authenticated.GET("/students/profile", studentH.GetProfile)
			authenticated.GET("/dynamic-fields", adminH.GetDynamicFields)

			uploadH := handler.NewUploadHandler(cfg.UploadPath)
			authenticated.POST("/upload", middleware.RequireApprovedStudentOrAdmin(db), uploadH.UploadFile)
			authenticated.POST("/upload/multiple", middleware.RequireApprovedStudentOrAdmin(db), uploadH.UploadMultiple)

			performanceH := handler.NewPerformanceHandler(db)
			notificationH := handler.NewNotificationHandler(db)
			statisticsH := handler.NewStatisticsHandler(db)

			studentProtected := authenticated.Group("")
			studentProtected.Use(middleware.RequireRole("student"), middleware.RequireApprovedStudent(db))
			{
				examH := handler.NewExamHandler(db)
				studentProtected.POST("/exams", examH.CreateExam)
				studentProtected.GET("/exams", examH.ListExams)
				studentProtected.GET("/exams/:id", examH.GetExam)
				studentProtected.PUT("/exams/:id", examH.UpdateExam)
				studentProtected.DELETE("/exams/:id", examH.DeleteExam)

				mistakeH := handler.NewMistakeHandler(db)
				studentProtected.POST("/mistakes", mistakeH.Create)
				studentProtected.GET("/mistakes", mistakeH.List)
				studentProtected.PUT("/mistakes/:id", mistakeH.Update)
				studentProtected.DELETE("/mistakes/:id", mistakeH.Delete)

				studentProtected.GET("/students/performance", performanceH.GetStudentPerformance)
				studentProtected.GET("/students/statistics", statisticsH.GetStudentStatistics)
				studentProtected.GET("/students/dashboard", statisticsH.GetDashboardSummary)
				studentProtected.GET("/notifications", notificationH.ListStudentNotifications)
				studentProtected.PUT("/notifications/:id/read", notificationH.MarkNotificationRead)

			}

			admin := authenticated.Group("/admin")
			admin.Use(middleware.RequireRole("admin"))
			{
				admin.GET("/students", adminH.ListStudents)
				admin.GET("/students/with-stats", adminH.GetAllStudentsWithStats)
				admin.GET("/students/:id", adminH.GetStudent)
				admin.GET("/students/:id/exams", adminH.GetStudentExams)
				admin.GET("/students/:id/mistakes", adminH.GetStudentMistakes)
				admin.PUT("/students/:id", adminH.UpdateStudent)
				admin.PUT("/students/:id/approve", adminH.ApproveStudent)
				admin.DELETE("/students/:id", adminH.DeleteStudent)
				admin.GET("/students/:id/performance", performanceH.AdminListStudentPerformance)
				admin.POST("/students/:id/performance", performanceH.AdminCreatePerformance)
				admin.PUT("/performance/:id", performanceH.AdminUpdatePerformance)
				admin.DELETE("/performance/:id", performanceH.AdminDeletePerformance)
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
