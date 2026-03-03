package promos

import (
	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
)

type Service interface {
	Create(promo *domain.PromoCode) error
	GetByCode(code string) (*domain.PromoCode, error)
	GetByID(id uuid.UUID) (*domain.PromoCode, error)
	Update(promo *domain.PromoCode) error
	Delete(id uuid.UUID) error
}
