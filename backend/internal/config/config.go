package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/yourusername/noshirvani-academy/backend/pkg"
)

type Config struct {
	DatabaseURL      string
	JWTSecret        string
	JWTRefreshSecret string
	JWTAccessTTL     int64
	JWTRefreshTTL    int64
	OTPProvider      string
	UploadPath       string
	ServerAddr       string
	CORSOrigins      []string
	AdminPhones      map[string]bool
	SMSIRAPIKey      string
	SMSIRTemplateID  string
	RedisAddr        string
}

func Load() *Config {
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		refreshSecret = os.Getenv("JWT_SECRET")
	}

	cfg := &Config{
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		JWTRefreshSecret: refreshSecret,
		JWTAccessTTL:     getEnvInt("JWT_ACCESS_TTL", 3600),
		JWTRefreshTTL:    getEnvInt("JWT_REFRESH_TTL", 15*24*3600),
		OTPProvider:      getEnv("OTP_PROVIDER", "mock"),
		UploadPath:       getEnv("UPLOAD_PATH", "./uploads"),
		ServerAddr:       getEnv("SERVER_ADDR", ":8080"),
		CORSOrigins:      splitCSV(os.Getenv("CORS_ORIGINS")),
		AdminPhones:      phoneSet(os.Getenv("ADMIN_PHONES")),
		SMSIRAPIKey:      os.Getenv("SMSIR_API_KEY"),
		SMSIRTemplateID:  os.Getenv("SMSIR_TEMPLATE_ID"),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" || cfg.JWTRefreshSecret == "" {
		log.Fatal("JWT_SECRET and JWT_REFRESH_SECRET are required")
	}

	return cfg
}

func getEnv(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(name string, fallback int64) int64 {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	output := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			output = append(output, trimmed)
		}
	}
	return output
}

func phoneSet(value string) map[string]bool {
	phones := map[string]bool{}
	for _, phone := range splitCSV(value) {
		// Normalize phone number to handle different formats (09xxx, +989xxx, 989xxx)
		normalizedPhone := pkg.NormalizePhone(phone)
		if normalizedPhone != "" {
			phones[normalizedPhone] = true
		}
	}
	return phones
}
