package orders

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"marketplace-backend/internal/domain"
)

type repositoryImpl struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &repositoryImpl{db: db}
}

func (r *repositoryImpl) Create(tx *sqlx.Tx, order *domain.Order) error {
	query := `
		INSERT INTO orders (id, user_id, status, promo_code_id, total_amount, discount_amount, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	var err error
	if tx != nil {
		_, err = tx.Exec(query,
			order.ID,
			order.UserID,
			order.Status,
			order.PromoCodeID,
			order.TotalAmount,
			order.DiscountAmount,
			order.CreatedAt,
			order.UpdatedAt,
		)
	} else {
		_, err = r.db.Exec(query,
			order.ID,
			order.UserID,
			order.Status,
			order.PromoCodeID,
			order.TotalAmount,
			order.DiscountAmount,
			order.CreatedAt,
			order.UpdatedAt,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

func (r *repositoryImpl) GetByID(id uuid.UUID) (*domain.Order, error) {
	var order domain.Order

	query := `SELECT * FROM orders WHERE id = $1`

	err := r.db.Get(&order, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &order, nil
}

func (r *repositoryImpl) Update(tx *sqlx.Tx, order *domain.Order) error {
	query := `
		UPDATE orders 
		SET status = $2, promo_code_id = $3, total_amount = $4, 
		    discount_amount = $5, updated_at = $6
		WHERE id = $1
	`

	var result sql.Result
	var err error

	if tx != nil {
		result, err = tx.Exec(query,
			order.ID,
			order.Status,
			order.PromoCodeID,
			order.TotalAmount,
			order.DiscountAmount,
			order.UpdatedAt,
		)
	} else {
		result, err = r.db.Exec(query,
			order.ID,
			order.Status,
			order.PromoCodeID,
			order.TotalAmount,
			order.DiscountAmount,
			order.UpdatedAt,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("order not found")
	}

	return nil
}

func (r *repositoryImpl) GetActiveOrderByUserID(userID uuid.UUID) (*domain.Order, error) {
	var order domain.Order
	query := `
		SELECT * FROM orders 
		WHERE user_id = $1 AND status IN ($2, $3)
		LIMIT 1
	`
	err := r.db.Get(&order, query, userID, domain.OrderCreated, domain.OrderPaymentPending)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active order: %w", err)
	}
	return &order, nil
}

func (r *repositoryImpl) CreateItem(tx *sqlx.Tx, item *domain.OrderItem) error {
	query := `
		INSERT INTO order_items (id, order_id, product_id, quantity, price_at_order)
		VALUES ($1, $2, $3, $4, $5)
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(query,
			item.ID,
			item.OrderID,
			item.ProductID,
			item.Quantity,
			item.PriceAtOrder,
		)
	} else {
		_, err = r.db.Exec(query,
			item.ID,
			item.OrderID,
			item.ProductID,
			item.Quantity,
			item.PriceAtOrder,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to create order item: %w", err)
	}
	return nil
}

func (r *repositoryImpl) GetItemsByOrderID(orderID uuid.UUID) ([]domain.OrderItem, error) {
	var items []domain.OrderItem
	query := `SELECT * FROM order_items WHERE order_id = $1`
	err := r.db.Select(&items, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}
	return items, nil
}

func (r *repositoryImpl) DeleteItemsByOrderID(tx *sqlx.Tx, orderID uuid.UUID) error {
	query := `DELETE FROM order_items WHERE order_id = $1`
	var err error
	if tx != nil {
		_, err = tx.Exec(query, orderID)
	} else {
		_, err = r.db.Exec(query, orderID)
	}

	if err != nil {
		return fmt.Errorf("failed to delete order items: %w", err)
	}
	return nil
}

func (r *repositoryImpl) BeginTx() (*sqlx.Tx, error) {
	return r.db.Beginx()
}
