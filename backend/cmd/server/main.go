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
