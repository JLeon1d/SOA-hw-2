package products

import (
	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(product *domain.Product) error
	GetByID(id uuid.UUID) (*domain.Product, error)
	Update(product *domain.Product) error
	Delete(id uuid.UUID) error
	List(filter Filter) (*ListResult, error)
	UpdateStock(tx *sqlx.Tx, productID uuid.UUID, delta int) error
}

type Filter struct {
	Status   *domain.ProductStatus
	Category *string
	Page     int
	Size     int
}

type ListResult struct {
	Products      []domain.Product
	TotalElements int
}
