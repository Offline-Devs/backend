package auth

import (
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
    accessSecret  string
    refreshSecret string
    accessTTL     time.Duration
    refreshTTL    time.Duration
}

func NewJWTService(accessSecret, refreshSecret string, accessTTLSeconds, refreshTTLSeconds int64) *JWTService {
    return &JWTService{
        accessSecret:  accessSecret,
        refreshSecret: refreshSecret,
        accessTTL:     time.Second * time.Duration(accessTTLSeconds),
        refreshTTL:    time.Second * time.Duration(refreshTTLSeconds),
    }
}

func (j *JWTService) GenerateAccessToken(userID, role string) (string, error) {
    now := time.Now()
    claims := jwt.MapClaims{
        "user_id": userID,
        "role":    role,
        "exp":     now.Add(j.accessTTL).Unix(),
        "iat":     now.Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(j.accessSecret))
}

func (j *JWTService) GenerateRefreshToken(userID string) (string, error) {
    now := time.Now()
    claims := jwt.MapClaims{
        "user_id": userID,
        "exp":     now.Add(j.refreshTTL).Unix(),
        "iat":     now.Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(j.refreshSecret))
}

func (j *JWTService) ValidateAccessToken(tokenString string) (jwt.MapClaims, error) {
    return parseToken(tokenString, j.accessSecret)
}

func (j *JWTService) ValidateRefreshToken(tokenString string) (jwt.MapClaims, error) {
    return parseToken(tokenString, j.refreshSecret)
}

func parseToken(tokenString, secret string) (jwt.MapClaims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(secret), nil
    })
    if err != nil {
        return nil, err
    }
    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        return claims, nil
    }
    return nil, jwt.ErrSignatureInvalid
}
