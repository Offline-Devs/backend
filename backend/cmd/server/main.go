package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	_ "github.com/yourusername/noshirvani-academy/backend/docs"
	"github.com/yourusername/noshirvani-academy/backend/internal/config"
	"github.com/yourusername/noshirvani-academy/backend/internal/infrastructure/database"
	"github.com/yourusername/noshirvani-academy/backend/internal/router"
)

// @title           نوشیروانی آکادمی API
// @version         1.0
// @description     API برای سیستم مدیریت آکادمی نوشیروانی
// @description     شامل خدماتی برای احراز هویت، مدیریت دانشجویان، آزمون‌ها، اشتباهات و مقالات بلاگ
// @termsOfService  http://swagger.io/terms/

// @contact.name   تیم توسعه اپلیکیشن
// @contact.url    https://github.com/Offline-Devs
// @contact.email  support@noshirvaniacademy.com

// @license.name  MIT License
// @license.url   https://opensource.org/licenses/MIT

// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description "توکن JWT - فرمت: Bearer <token>"

// @externalDocs.description توضیحات API
// @externalDocs.url https://github.com/Offline-Devs/noshirvani-academy

// HealthResponse پاسخ بررسی وضعیت سرور
// @Summary بررسی وضعیت سرور
// @Description بررسی می‌کند که آیا سرور در حال اجرا است
// @Tags عمومی
// @Produce json
// @Success 200 {object} map[string]string "سرور فعال است"
// @Router /health [get]
func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	db, err := database.NewPostgresDB(cfg.DatabaseURL, cfg.Environment)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	r := router.Setup(db, cfg)
	addr := cfg.ServerAddr
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	log.Printf("server running on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
