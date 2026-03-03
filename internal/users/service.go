package users

import (
	"github.com/golang-jwt/jwt/v5"

	"marketplace-backend/internal/domain"
)

type Service interface {
	Register(email, password string, role domain.UserRole) (*domain.User, error)
	Login(email, password string) (accessToken, refreshToken string, err error)
	RefreshToken(refreshToken string) (string, error)
	ValidateAccessToken(tokenString string) (*Claims, error)
}

type Claims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}
