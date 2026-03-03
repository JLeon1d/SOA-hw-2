package promos

import (
	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(promo *domain.PromoCode) error
	GetByCode(code string) (*domain.PromoCode, error)
	GetByID(id uuid.UUID) (*domain.PromoCode, error)
	Update(promo *domain.PromoCode) error
	Delete(id uuid.UUID) error
	IncrementUses(tx *sqlx.Tx, id uuid.UUID) error
	DecrementUses(tx *sqlx.Tx, id uuid.UUID) error
}
