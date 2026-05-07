package config

import (
    "os"
    "strconv"
)

type Config struct {
    DatabaseURL   string
    JWTSecret     string
    JWTRefreshSecret string
    JWTAccessTTL  int64
    JWTRefreshTTL int64
    OTPProvider   string
    UploadPath    string
    ServerAddr    string
}

func Load() *Config {
    refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
    if refreshSecret == "" {
        refreshSecret = os.Getenv("JWT_SECRET")
    }

    return &Config{
        DatabaseURL:   os.Getenv("DATABASE_URL"),
        JWTSecret:     os.Getenv("JWT_SECRET"),
        JWTRefreshSecret: refreshSecret,
        JWTAccessTTL:  getEnvInt("JWT_ACCESS_TTL", 3600),
        JWTRefreshTTL: getEnvInt("JWT_REFRESH_TTL", 15*24*3600),
        OTPProvider:   os.Getenv("OTP_PROVIDER"),
        UploadPath:    os.Getenv("UPLOAD_PATH"),
        ServerAddr:    os.Getenv("SERVER_ADDR"),
    }
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
