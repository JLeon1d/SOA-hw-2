package orders

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"marketplace-backend/internal/domain"
)

// Force rebuild - Update and Cancel implementation

type serviceImpl struct {
	orderRepo        Repository
	productRepo      ProductRepository
	promoRepo        PromoRepository
	operationRepo    OperationRepository
	rateLimitMinutes int
}

type ProductRepository interface {
	GetByID(id uuid.UUID) (*domain.Product, error)
	UpdateStock(tx *sqlx.Tx, productID uuid.UUID, delta int) error
}

type PromoRepository interface {
	GetByCode(code string) (*domain.PromoCode, error)
	IncrementUses(tx *sqlx.Tx, id uuid.UUID) error
	DecrementUses(tx *sqlx.Tx, id uuid.UUID) error
}

type OperationRepository interface {
	Create(tx *sqlx.Tx, operation *domain.UserOperation) error
	CheckRateLimit(userID uuid.UUID, operationType domain.OperationType, windowMinutes int) (bool, time.Duration, error)
}

func NewService(
	orderRepo Repository,
	productRepo ProductRepository,
	promoRepo PromoRepository,
	operationRepo OperationRepository,
	rateLimitMinutes int,
) Service {
	return &serviceImpl{
		orderRepo:        orderRepo,
		productRepo:      productRepo,
		promoRepo:        promoRepo,
		operationRepo:    operationRepo,
		rateLimitMinutes: rateLimitMinutes,
	}
}

func (s *serviceImpl) Create(userID uuid.UUID, items []OrderItemRequest, promoCode *string) (*domain.OrderWithItems, error) {
	allowed, remaining, err := s.operationRepo.CheckRateLimit(userID, domain.OperationCreateOrder, s.rateLimitMinutes)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("rate limit exceeded, retry after %v", remaining)
	}

	activeOrder, err := s.orderRepo.GetActiveOrderByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active orders: %w", err)
	}
	if activeOrder != nil {
		return nil, fmt.Errorf("user already has an active order")
	}

	tx, err := s.orderRepo.BeginTx()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var totalAmount float64
	var orderItems []domain.OrderItem

	for _, item := range items {
		product, err := s.productRepo.GetByID(item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product not found: %w", err)
		}

		if product.Status != domain.ProductActive {
			return nil, fmt.Errorf("product %s is not active", item.ProductID)
		}

		if product.Stock < item.Quantity {
			return nil, fmt.Errorf("insufficient stock for product %s", item.ProductID)
		}

		if err := s.productRepo.UpdateStock(tx, item.ProductID, -item.Quantity); err != nil {
			return nil, fmt.Errorf("failed to reserve stock: %w", err)
		}

		orderItem := domain.OrderItem{
			ID:           uuid.New(),
			ProductID:    item.ProductID,
			Quantity:     item.Quantity,
			PriceAtOrder: product.Price,
		}
		orderItems = append(orderItems, orderItem)
		totalAmount += product.Price * float64(item.Quantity)
	}

	var discountAmount float64
	var promoCodeID *uuid.UUID

	if promoCode != nil && *promoCode != "" {
		promo, err := s.promoRepo.GetByCode(*promoCode)
		if err != nil {
			return nil, fmt.Errorf("promo code not found: %w", err)
		}

		if !promo.IsValid() {
			return nil, fmt.Errorf("promo code is invalid")
		}

		if totalAmount < promo.MinOrderAmount {
			return nil, fmt.Errorf("order amount below minimum for promo code")
		}

		discountAmount = promo.CalculateDiscount(totalAmount)
		promoCodeID = &promo.ID

		if err := s.promoRepo.IncrementUses(tx, promo.ID); err != nil {
			return nil, fmt.Errorf("failed to increment promo uses: %w", err)
		}
	}

	order := &domain.Order{
		ID:             uuid.New(),
		UserID:         userID,
		Status:         domain.OrderCreated,
		PromoCodeID:    promoCodeID,
		TotalAmount:    totalAmount - discountAmount,
		DiscountAmount: discountAmount,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.orderRepo.Create(tx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	for i := range orderItems {
		orderItems[i].OrderID = order.ID
		if err := s.orderRepo.CreateItem(tx, &orderItems[i]); err != nil {
			return nil, fmt.Errorf("failed to create order item: %w", err)
		}
	}

	operation := &domain.UserOperation{
		ID:            uuid.New(),
		UserID:        userID,
		OperationType: domain.OperationCreateOrder,
		CreatedAt:     time.Now(),
	}
	if err := s.operationRepo.Create(tx, operation); err != nil {
		return nil, fmt.Errorf("failed to record operation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &domain.OrderWithItems{
		Order: *order,
		Items: orderItems,
	}, nil
}

func (s *serviceImpl) GetByID(id uuid.UUID) (*domain.OrderWithItems, error) {
	order, err := s.orderRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	items, err := s.orderRepo.GetItemsByOrderID(id)
	if err != nil {
		return nil, err
	}

	return &domain.OrderWithItems{
		Order: *order,
		Items: items,
	}, nil
}

func (s *serviceImpl) Update(orderID, userID uuid.UUID, items []OrderItemRequest) (*domain.OrderWithItems, error) {
	// Get existing order
	existing, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// Check ownership
	if existing.UserID != userID {
		return nil, fmt.Errorf("unauthorized: order belongs to different user")
	}

	// Begin transaction
	tx, err := s.orderRepo.BeginTx()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get existing items to restore stock
	existingItems, err := s.orderRepo.GetItemsByOrderID(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing items: %w", err)
	}

	// Restore stock for old items
	for _, item := range existingItems {
		if err := s.productRepo.UpdateStock(tx, item.ProductID, item.Quantity); err != nil {
			return nil, fmt.Errorf("failed to restore stock: %w", err)
		}
	}

	// Calculate new total and reserve stock for new items
	var totalAmount float64
	var orderItems []domain.OrderItem

	for _, item := range items {
		product, err := s.productRepo.GetByID(item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product not found: %w", err)
		}

		if product.Status != domain.ProductActive {
			return nil, fmt.Errorf("product %s is not active", item.ProductID)
		}

		if product.Stock < item.Quantity {
			return nil, fmt.Errorf("insufficient stock for product %s", item.ProductID)
		}

		if err := s.productRepo.UpdateStock(tx, item.ProductID, -item.Quantity); err != nil {
			return nil, fmt.Errorf("failed to reserve stock: %w", err)
		}

		orderItem := domain.OrderItem{
			ID:           uuid.New(),
			OrderID:      orderID,
			ProductID:    item.ProductID,
			Quantity:     item.Quantity,
			PriceAtOrder: product.Price,
		}
		orderItems = append(orderItems, orderItem)
		totalAmount += product.Price * float64(item.Quantity)
	}

	// Update order
	existing.TotalAmount = totalAmount
	existing.UpdatedAt = time.Now()

	if err := s.orderRepo.Update(tx, existing); err != nil {
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	// Delete old items and create new ones
	if err := s.orderRepo.DeleteItemsByOrderID(tx, orderID); err != nil {
		return nil, fmt.Errorf("failed to delete old items: %w", err)
	}

	for i := range orderItems {
		if err := s.orderRepo.CreateItem(tx, &orderItems[i]); err != nil {
			return nil, fmt.Errorf("failed to create order item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &domain.OrderWithItems{
		Order: *existing,
		Items: orderItems,
	}, nil
}

func (s *serviceImpl) Cancel(orderID, userID uuid.UUID) (*domain.OrderWithItems, error) {
	// Get existing order
	existing, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// Check ownership
	if existing.UserID != userID {
		return nil, fmt.Errorf("unauthorized: order belongs to different user")
	}

	// Check if order can be canceled
	if !existing.Status.CanTransitionTo(domain.OrderCanceled) {
		return nil, fmt.Errorf("order cannot be canceled in current status: %s", existing.Status)
	}

	// Begin transaction
	tx, err := s.orderRepo.BeginTx()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get items to restore stock
	items, err := s.orderRepo.GetItemsByOrderID(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	// Restore stock
	for _, item := range items {
		if err := s.productRepo.UpdateStock(tx, item.ProductID, item.Quantity); err != nil {
			return nil, fmt.Errorf("failed to restore stock: %w", err)
		}
	}

	// Decrement promo code uses if applicable
	if existing.PromoCodeID != nil {
		if err := s.promoRepo.DecrementUses(tx, *existing.PromoCodeID); err != nil {
			return nil, fmt.Errorf("failed to decrement promo uses: %w", err)
		}
	}

	// Update order status
	existing.Status = domain.OrderCanceled
	existing.UpdatedAt = time.Now()

	if err := s.orderRepo.Update(tx, existing); err != nil {
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &domain.OrderWithItems{
		Order: *existing,
		Items: items,
	}, nil
}
