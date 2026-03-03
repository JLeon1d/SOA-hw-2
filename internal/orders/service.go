package orders

import (
	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
)

type Service interface {
	Create(userID uuid.UUID, items []OrderItemRequest, promoCode *string) (*domain.OrderWithItems, error)
	GetByID(id uuid.UUID) (*domain.OrderWithItems, error)
	Update(orderID, userID uuid.UUID, items []OrderItemRequest) (*domain.OrderWithItems, error)
	Cancel(orderID, userID uuid.UUID) (*domain.OrderWithItems, error)
}

type OrderItemRequest struct {
	ProductID uuid.UUID
	Quantity  int
}
