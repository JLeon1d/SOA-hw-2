package users

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"marketplace-backend/internal/domain"
)

type serviceImpl struct {
	repo          Repository
	jwtSecret     []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewService(
	repo Repository,
	jwtSecret string,
	accessExpiry time.Duration,
	refreshExpiry time.Duration,
) Service {
	return &serviceImpl{
		repo:          repo,
		jwtSecret:     []byte(jwtSecret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (s *serviceImpl) Register(email, password string, role domain.UserRole) (*domain.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         role,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *serviceImpl) Login(email, password string) (accessToken, refreshToken string, err error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		return "", "", fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", fmt.Errorf("invalid credentials")
	}

	accessToken, err = s.generateToken(user, s.accessExpiry)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err = s.generateToken(user, s.refreshExpiry)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *serviceImpl) RefreshToken(refreshToken string) (string, error) {
	claims, err := s.validateToken(refreshToken)
	if err != nil {
		return "", err
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID in token")
	}

	user, err := s.repo.GetByID(userID)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}

	accessToken, err := s.generateToken(user, s.accessExpiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}

	return accessToken, nil
}

func (s *serviceImpl) ValidateAccessToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString)
}

func (s *serviceImpl) generateToken(user *domain.User, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *serviceImpl) validateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		if err == jwt.ErrTokenExpired {
			return nil, fmt.Errorf("token expired")
		}
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}
