package users

import (
	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
)

type Repository interface {
	Create(user *domain.User) error
	GetByID(id uuid.UUID) (*domain.User, error)
	GetByEmail(email string) (*domain.User, error)
}
