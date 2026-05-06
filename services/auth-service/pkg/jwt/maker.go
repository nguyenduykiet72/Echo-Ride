package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenMaker struct {
	secretKey string
}

func NewTokenMaker(secretKey string) *TokenMaker {
	return &TokenMaker{secretKey: secretKey}
}

func (maker *TokenMaker) GenerateToken(userID uuid.UUID, role, status string, duration time.Duration) (string, error) {
	claims := CustomClaims{
		Key:    "echo-auth-service",
		UserID: userID,
		Role:   role,
		Status: status,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()), // Token is valid immediately
			Issuer:    "auth-service",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(maker.secretKey))
}
