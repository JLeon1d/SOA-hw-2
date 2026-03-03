package products

import (
	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
)

type Service interface {
	Create(product *domain.Product) error
	GetByID(id uuid.UUID) (*domain.Product, error)
	Update(product *domain.Product) error
	Delete(id uuid.UUID) error
	List(filter Filter) (*ListResult, error)
}
