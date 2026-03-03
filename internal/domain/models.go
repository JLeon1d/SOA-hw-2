package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleUser   UserRole = "USER"
	RoleSeller UserRole = "SELLER"
	RoleAdmin  UserRole = "ADMIN"
)

type User struct {
	ID           uuid.UUID `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Role         UserRole  `db:"role"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type ProductStatus string

const (
	ProductActive   ProductStatus = "ACTIVE"
	ProductInactive ProductStatus = "INACTIVE"
	ProductArchived ProductStatus = "ARCHIVED"
)

type Product struct {
	ID          uuid.UUID     `db:"id"`
	Name        string        `db:"name"`
	Description *string       `db:"description"`
	Price       float64       `db:"price"`
	Stock       int           `db:"stock"`
	Category    string        `db:"category"`
	Status      ProductStatus `db:"status"`
	SellerID    *uuid.UUID    `db:"seller_id"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
}

type OrderStatus string

const (
	OrderCreated        OrderStatus = "CREATED"
	OrderPaymentPending OrderStatus = "PAYMENT_PENDING"
	OrderPaid           OrderStatus = "PAID"
	OrderShipped        OrderStatus = "SHIPPED"
	OrderCompleted      OrderStatus = "COMPLETED"
	OrderCanceled       OrderStatus = "CANCELED"
)

var ValidTransitions = map[OrderStatus][]OrderStatus{
	OrderCreated:        {OrderPaymentPending, OrderCanceled},
	OrderPaymentPending: {OrderPaid, OrderCanceled},
	OrderPaid:           {OrderShipped},
	OrderShipped:        {OrderCompleted},
	OrderCompleted:      {},
	OrderCanceled:       {},
}

func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	validTargets, exists := ValidTransitions[s]
	if !exists {
		return false
	}
	for _, validTarget := range validTargets {
		if validTarget == target {
			return true
		}
	}
	return false
}

type Order struct {
	ID             uuid.UUID   `db:"id"`
	UserID         uuid.UUID   `db:"user_id"`
	Status         OrderStatus `db:"status"`
	PromoCodeID    *uuid.UUID  `db:"promo_code_id"`
	TotalAmount    float64     `db:"total_amount"`
	DiscountAmount float64     `db:"discount_amount"`
	CreatedAt      time.Time   `db:"created_at"`
	UpdatedAt      time.Time   `db:"updated_at"`
}

type OrderItem struct {
	ID           uuid.UUID `db:"id"`
	OrderID      uuid.UUID `db:"order_id"`
	ProductID    uuid.UUID `db:"product_id"`
	Quantity     int       `db:"quantity"`
	PriceAtOrder float64   `db:"price_at_order"`
}

type OrderWithItems struct {
	Order Order
	Items []OrderItem
}

type DiscountType string

const (
	DiscountPercentage  DiscountType = "PERCENTAGE"
	DiscountFixedAmount DiscountType = "FIXED_AMOUNT"
)

type PromoCode struct {
	ID             uuid.UUID    `db:"id"`
	Code           string       `db:"code"`
	DiscountType   DiscountType `db:"discount_type"`
	DiscountValue  float64      `db:"discount_value"`
	MinOrderAmount float64      `db:"min_order_amount"`
	MaxUses        int          `db:"max_uses"`
	CurrentUses    int          `db:"current_uses"`
	ValidFrom      time.Time    `db:"valid_from"`
	ValidUntil     time.Time    `db:"valid_until"`
	Active         bool         `db:"active"`
}

func (p *PromoCode) IsValid() bool {
	now := time.Now()
	return p.Active &&
		p.CurrentUses < p.MaxUses &&
		!now.Before(p.ValidFrom) &&
		!now.After(p.ValidUntil)
}

func (p *PromoCode) CalculateDiscount(orderTotal float64) float64 {
	if orderTotal < p.MinOrderAmount {
		return 0
	}

	var discount float64
	if p.DiscountType == DiscountPercentage {
		discount = orderTotal * p.DiscountValue / 100
		maxDiscount := orderTotal * 0.7
		if discount > maxDiscount {
			discount = orderTotal
		}
	} else {
		discount = p.DiscountValue
		if discount > orderTotal {
			discount = orderTotal
		}
	}

	return discount
}

type OperationType string

const (
	OperationCreateOrder OperationType = "CREATE_ORDER"
	OperationUpdateOrder OperationType = "UPDATE_ORDER"
)

type UserOperation struct {
	ID            uuid.UUID     `db:"id"`
	UserID        uuid.UUID     `db:"user_id"`
	OperationType OperationType `db:"operation_type"`
	CreatedAt     time.Time     `db:"created_at"`
}
