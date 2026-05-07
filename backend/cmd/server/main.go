package main

import (
    "log"
    "os"

    "github.com/joho/godotenv"

    "github.com/yourusername/noshirvani-academy/backend/internal/config"
    "github.com/yourusername/noshirvani-academy/backend/internal/infrastructure/database"
    "github.com/yourusername/noshirvani-academy/backend/internal/router"
)

func main() {
    _ = godotenv.Load()

    cfg := config.Load()

    db, err := database.NewPostgresDB(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("failed to connect to database: %v", err)
    }

    database.AutoMigrate(db)

    r := router.Setup(db, cfg)
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Printf("server running on :%s", port)
    if err := r.Run(":" + port); err != nil {
        log.Fatal(err)
    }
}
