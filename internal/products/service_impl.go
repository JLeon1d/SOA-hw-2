package products

import (
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain"
)

type serviceImpl struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &serviceImpl{
		repo: repo,
	}
}

func (s *serviceImpl) Create(product *domain.Product) error {
	product.ID = uuid.New()
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()
	return s.repo.Create(product)
}

func (s *serviceImpl) GetByID(id uuid.UUID) (*domain.Product, error) {
	return s.repo.GetByID(id)
}

func (s *serviceImpl) Update(product *domain.Product) error {
	product.UpdatedAt = time.Now()
	return s.repo.Update(product)
}

func (s *serviceImpl) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}

func (s *serviceImpl) List(filter Filter) (*ListResult, error) {
	return s.repo.List(filter)
}
