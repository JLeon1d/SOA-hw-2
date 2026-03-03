package orders

import (
	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(tx *sqlx.Tx, order *domain.Order) error
	GetByID(id uuid.UUID) (*domain.Order, error)
	Update(tx *sqlx.Tx, order *domain.Order) error
	GetActiveOrderByUserID(userID uuid.UUID) (*domain.Order, error)
	CreateItem(tx *sqlx.Tx, item *domain.OrderItem) error
	GetItemsByOrderID(orderID uuid.UUID) ([]domain.OrderItem, error)
	DeleteItemsByOrderID(tx *sqlx.Tx, orderID uuid.UUID) error
	BeginTx() (*sqlx.Tx, error)
}
