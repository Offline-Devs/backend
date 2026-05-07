package auth

import (
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
    claims := jwt.MapClaims{
        "user_id": userID,
        "role":    role,
        "exp":     time.Now().Add(j.accessTTL).Unix(),
        "iat":     time.Now().Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(j.accessSecret))
}

func (j *JWTService) GenerateRefreshToken(userID string) (string, error) {
    claims := jwt.MapClaims{
        "user_id": userID,
        "exp":     time.Now().Add(j.refreshTTL).Unix(),
        "iat":     time.Now().Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(j.refreshSecret))
}

func (j *JWTService) ValidateAccessToken(tokenString string) (jwt.MapClaims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return []byte(j.accessSecret), nil
    })
    if err != nil {
        return nil, err
    }
    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        return claims, nil
    }
    return nil, jwt.ErrSignatureInvalid
}

func (j *JWTService) ValidateRefreshToken(tokenString string) (jwt.MapClaims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return []byte(j.refreshSecret), nil
    })
    if err != nil {
        return nil, err
    }
    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        return claims, nil
    }
    return nil, jwt.ErrSignatureInvalid
}
