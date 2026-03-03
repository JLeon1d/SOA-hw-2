package promos

import (
	"encoding/json"
	"net/http"
	"time"

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

type CreatePromoCodeRequest struct {
	Code           string  `json:"code"`
	DiscountType   string  `json:"discount_type"`
	DiscountValue  float64 `json:"discount_value"`
	MinOrderAmount float64 `json:"min_order_amount"`
	MaxUses        int     `json:"max_uses"`
	ValidFrom      string  `json:"valid_from"`
	ValidUntil     string  `json:"valid_until"`
}

type UpdatePromoCodeRequest struct {
	Active         *bool    `json:"active"`
	DiscountValue  *float64 `json:"discount_value"`
	MinOrderAmount *float64 `json:"min_order_amount"`
	MaxUses        *int     `json:"max_uses"`
	ValidUntil     *string  `json:"valid_until"`
}

type PromoCodeResponse struct {
	ID             uuid.UUID `json:"id"`
	Code           string    `json:"code"`
	DiscountType   string    `json:"discount_type"`
	DiscountValue  float64   `json:"discount_value"`
	MinOrderAmount float64   `json:"min_order_amount"`
	MaxUses        int       `json:"max_uses"`
	CurrentUses    int       `json:"current_uses"`
	ValidFrom      string    `json:"valid_from"`
	ValidUntil     string    `json:"valid_until"`
	Active         bool      `json:"active"`
}

func (h *Handler) CreatePromoCode(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r.Context())

	if role != string(domain.RoleAdmin) {
		errors.NewAccessDenied().WriteJSON(w)
		return
	}

	var req CreatePromoCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
		return
	}

	validFrom, err := time.Parse(time.RFC3339, req.ValidFrom)
	if err != nil {
		errors.NewValidationError("Invalid valid_from format", nil).WriteJSON(w)
		return
	}

	validUntil, err := time.Parse(time.RFC3339, req.ValidUntil)
	if err != nil {
		errors.NewValidationError("Invalid valid_until format", nil).WriteJSON(w)
		return
	}

	promo := &domain.PromoCode{
		Code:           req.Code,
		DiscountType:   domain.DiscountType(req.DiscountType),
		DiscountValue:  req.DiscountValue,
		MinOrderAmount: req.MinOrderAmount,
		MaxUses:        req.MaxUses,
		CurrentUses:    0,
		ValidFrom:      validFrom,
		ValidUntil:     validUntil,
		Active:         true,
	}

	if err := h.service.Create(promo); err != nil {
		errors.NewAppError(errors.ValidationError, err.Error(), http.StatusBadRequest).WriteJSON(w)
		return
	}

	response := mapPromoToResponse(promo)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetPromoCode(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r.Context())

	if role != string(domain.RoleAdmin) {
		errors.NewAccessDenied().WriteJSON(w)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.NewValidationError("Invalid promo code ID", nil).WriteJSON(w)
		return
	}

	promo, err := h.service.GetByID(id)
	if err != nil {
		errors.NewAppError(errors.ValidationError, "Promo code not found", http.StatusNotFound).WriteJSON(w)
		return
	}

	response := mapPromoToResponse(promo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) UpdatePromoCode(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r.Context())

	if role != string(domain.RoleAdmin) {
		errors.NewAccessDenied().WriteJSON(w)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.NewValidationError("Invalid promo code ID", nil).WriteJSON(w)
		return
	}

	existing, err := h.service.GetByID(id)
	if err != nil {
		errors.NewAppError(errors.ValidationError, "Promo code not found", http.StatusNotFound).WriteJSON(w)
		return
	}

	var req UpdatePromoCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
		return
	}

	if req.Active != nil {
		existing.Active = *req.Active
	}
	if req.DiscountValue != nil {
		existing.DiscountValue = *req.DiscountValue
	}
	if req.MinOrderAmount != nil {
		existing.MinOrderAmount = *req.MinOrderAmount
	}
	if req.MaxUses != nil {
		existing.MaxUses = *req.MaxUses
	}
	if req.ValidUntil != nil {
		validUntil, err := time.Parse(time.RFC3339, *req.ValidUntil)
		if err != nil {
			errors.NewValidationError("Invalid valid_until format", nil).WriteJSON(w)
			return
		}
		existing.ValidUntil = validUntil
	}

	if err := h.service.Update(existing); err != nil {
		errors.NewAppError(errors.ValidationError, err.Error(), http.StatusBadRequest).WriteJSON(w)
		return
	}

	response := mapPromoToResponse(existing)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) DeletePromoCode(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r.Context())

	if role != string(domain.RoleAdmin) {
		errors.NewAccessDenied().WriteJSON(w)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.NewValidationError("Invalid promo code ID", nil).WriteJSON(w)
		return
	}

	if err := h.service.Delete(id); err != nil {
		errors.NewAppError(errors.ValidationError, err.Error(), http.StatusInternalServerError).WriteJSON(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func mapPromoToResponse(p *domain.PromoCode) PromoCodeResponse {
	return PromoCodeResponse{
		ID:             p.ID,
		Code:           p.Code,
		DiscountType:   string(p.DiscountType),
		DiscountValue:  p.DiscountValue,
		MinOrderAmount: p.MinOrderAmount,
		MaxUses:        p.MaxUses,
		CurrentUses:    p.CurrentUses,
		ValidFrom:      p.ValidFrom.Format("2006-01-02T15:04:05Z07:00"),
		ValidUntil:     p.ValidUntil.Format("2006-01-02T15:04:05Z07:00"),
		Active:         p.Active,
	}
}
