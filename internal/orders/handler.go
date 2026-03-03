package orders

import (
	"encoding/json"
	"net/http"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/errors"
	"marketplace-backend/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

type OrderItemHTTP struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

type CreateOrderRequest struct {
	Items     []OrderItemHTTP `json:"items"`
	PromoCode *string         `json:"promo_code"`
}

type UpdateOrderRequest struct {
	Status string `json:"status"`
}

type OrderItemResponse struct {
	ID           uuid.UUID `json:"id"`
	ProductID    uuid.UUID `json:"product_id"`
	Quantity     int       `json:"quantity"`
	PriceAtOrder float64   `json:"price_at_order"`
}

type OrderResponse struct {
	ID             uuid.UUID           `json:"id"`
	UserID         uuid.UUID           `json:"user_id"`
	Status         string              `json:"status"`
	Items          []OrderItemResponse `json:"items"`
	TotalAmount    float64             `json:"total_amount"`
	DiscountAmount float64             `json:"discount_amount"`
	CreatedAt      string              `json:"created_at"`
	UpdatedAt      string              `json:"updated_at"`
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		errors.NewValidationError("Invalid user ID", nil).WriteJSON(w)
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
		return
	}

	items := make([]OrderItemRequest, len(req.Items))
	for i, item := range req.Items {
		items[i] = OrderItemRequest{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	order, err := h.service.Create(userUUID, items, req.PromoCode)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			appErr.WriteJSON(w)
		} else {
			errors.NewAppError(errors.ValidationError, err.Error(), http.StatusBadRequest).WriteJSON(w)
		}
		return
	}

	response := mapOrderToResponse(order)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	role := middleware.GetUserRole(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.NewValidationError("Invalid order ID", nil).WriteJSON(w)
		return
	}

	order, err := h.service.GetByID(id)
	if err != nil {
		errors.NewOrderNotFound(idStr).WriteJSON(w)
		return
	}

	if role != string(domain.RoleAdmin) && order.Order.UserID.String() != userID {
		errors.NewOrderOwnershipViolation().WriteJSON(w)
		return
	}

	response := mapOrderToResponse(order)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	role := middleware.GetUserRole(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.NewValidationError("Invalid order ID", nil).WriteJSON(w)
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		errors.NewValidationError("Invalid user ID", nil).WriteJSON(w)
		return
	}

	var req UpdateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
		return
	}

	newStatus := domain.OrderStatus(req.Status)

	existing, err := h.service.GetByID(id)
	if err != nil {
		errors.NewOrderNotFound(idStr).WriteJSON(w)
		return
	}

	if role != string(domain.RoleAdmin) && existing.Order.UserID != userUUID {
		errors.NewOrderOwnershipViolation().WriteJSON(w)
		return
	}

	if !existing.Order.Status.CanTransitionTo(newStatus) {
		errors.NewInvalidStateTransition(string(existing.Order.Status), string(newStatus)).WriteJSON(w)
		return
	}

	items := make([]OrderItemRequest, len(existing.Items))
	for i, item := range existing.Items {
		items[i] = OrderItemRequest{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	order, err := h.service.Update(id, userUUID, items)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			appErr.WriteJSON(w)
		} else {
			errors.NewAppError(errors.ValidationError, err.Error(), http.StatusBadRequest).WriteJSON(w)
		}
		return
	}

	response := mapOrderToResponse(order)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	role := middleware.GetUserRole(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.NewValidationError("Invalid order ID", nil).WriteJSON(w)
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		errors.NewValidationError("Invalid user ID", nil).WriteJSON(w)
		return
	}

	existing, err := h.service.GetByID(id)
	if err != nil {
		errors.NewOrderNotFound(idStr).WriteJSON(w)
		return
	}

	if role != string(domain.RoleAdmin) && existing.Order.UserID != userUUID {
		errors.NewOrderOwnershipViolation().WriteJSON(w)
		return
	}

	order, err := h.service.Cancel(id, userUUID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			appErr.WriteJSON(w)
		} else {
			errors.NewAppError(errors.ValidationError, err.Error(), http.StatusBadRequest).WriteJSON(w)
		}
		return
	}

	response := mapOrderToResponse(order)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func mapOrderToResponse(order *domain.OrderWithItems) OrderResponse {
	items := make([]OrderItemResponse, len(order.Items))
	for i, item := range order.Items {
		items[i] = OrderItemResponse{
			ID:           item.ID,
			ProductID:    item.ProductID,
			Quantity:     item.Quantity,
			PriceAtOrder: item.PriceAtOrder,
		}
	}

	return OrderResponse{
		ID:             order.Order.ID,
		UserID:         order.Order.UserID,
		Status:         string(order.Order.Status),
		Items:          items,
		TotalAmount:    order.Order.TotalAmount,
		DiscountAmount: order.Order.DiscountAmount,
		CreatedAt:      order.Order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      order.Order.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
