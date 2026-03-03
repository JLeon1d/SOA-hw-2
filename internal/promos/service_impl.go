package promos

import (
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain"
)

// Force rebuild - promo validation fix

type serviceImpl struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &serviceImpl{
		repo: repo,
	}
}

func (s *serviceImpl) Create(promo *domain.PromoCode) error {
	promo.ID = uuid.New()
	promo.CurrentUses = 0
	return s.repo.Create(promo)
}

func (s *serviceImpl) GetByCode(code string) (*domain.PromoCode, error) {
	promo, err := s.repo.GetByCode(code)
	if err != nil {
		return nil, err
	}

	if !promo.IsValid() {
		return nil, fmt.Errorf("promo code is invalid or expired")
	}

	return promo, nil
}

func (s *serviceImpl) GetByID(id uuid.UUID) (*domain.PromoCode, error) {
	return s.repo.GetByID(id)
}

func (s *serviceImpl) Update(promo *domain.PromoCode) error {
	return s.repo.Update(promo)
}

func (s *serviceImpl) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}
