package orders

import (
	"encoding/json"
	"net/http"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/errors"
	"marketplace-backend/internal/generated/api"
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

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		errors.NewValidationError("Invalid user ID", nil).WriteJSON(w)
		return
	}

	var req api.OrderCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
		return
	}

	items := make([]OrderItemRequest, len(req.Items))
	for i, item := range req.Items {
		productID, err := uuid.Parse(item.ProductId)
		if err != nil {
			errors.NewValidationError("Invalid product ID", nil).WriteJSON(w)
			return
		}
		items[i] = OrderItemRequest{
			ProductID: productID,
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

	var req api.OrderUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
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

	items := make([]OrderItemRequest, len(req.Items))
	for i, item := range req.Items {
		productID, err := uuid.Parse(item.ProductId)
		if err != nil {
			errors.NewValidationError("Invalid product ID", nil).WriteJSON(w)
			return
		}
		items[i] = OrderItemRequest{
			ProductID: productID,
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

func mapOrderToResponse(order *domain.OrderWithItems) api.OrderResponse {
	items := make([]api.OrderItemResponse, len(order.Items))
	for i, item := range order.Items {
		items[i] = api.OrderItemResponse{
			Id:           item.ID.String(),
			ProductId:    item.ProductID.String(),
			Quantity:     item.Quantity,
			PriceAtOrder: float32(item.PriceAtOrder),
		}
	}

	resp := api.OrderResponse{
		Id:             order.Order.ID.String(),
		UserId:         order.Order.UserID.String(),
		Status:         api.OrderStatus(order.Order.Status),
		Items:          items,
		TotalAmount:    float32(order.Order.TotalAmount),
		DiscountAmount: float32(order.Order.DiscountAmount),
		CreatedAt:      order.Order.CreatedAt,
		UpdatedAt:      order.Order.UpdatedAt,
	}

	if order.Order.PromoCodeID != nil {
		promoCodeIDStr := order.Order.PromoCodeID.String()
		resp.PromoCodeId = &promoCodeIDStr
	}

	return resp
}
