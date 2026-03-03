package promos

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

func (h *Handler) CreatePromoCode(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r.Context())

	if role != string(domain.RoleAdmin) && role != string(domain.RoleSeller) {
		errors.NewAccessDenied().WriteJSON(w)
		return
	}

	var req api.PromoCodeCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	promo := &domain.PromoCode{
		Code:           req.Code,
		DiscountType:   domain.DiscountType(req.DiscountType),
		DiscountValue:  float64(req.DiscountValue),
		MinOrderAmount: float64(req.MinOrderAmount),
		MaxUses:        req.MaxUses,
		CurrentUses:    0,
		ValidFrom:      req.ValidFrom,
		ValidUntil:     req.ValidUntil,
		Active:         active,
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

	if role != string(domain.RoleAdmin) && role != string(domain.RoleSeller) {
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

	if role != string(domain.RoleAdmin) && role != string(domain.RoleSeller) {
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

	var req api.PromoCodeUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
		return
	}

	if req.Active != nil {
		existing.Active = *req.Active
	}
	if req.DiscountValue != nil {
		existing.DiscountValue = float64(*req.DiscountValue)
	}
	if req.MinOrderAmount != nil {
		existing.MinOrderAmount = float64(*req.MinOrderAmount)
	}
	if req.MaxUses != nil {
		existing.MaxUses = *req.MaxUses
	}
	if req.ValidUntil != nil {
		existing.ValidUntil = *req.ValidUntil
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

	if role != string(domain.RoleAdmin) && role != string(domain.RoleSeller) {
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

func mapPromoToResponse(p *domain.PromoCode) api.PromoCodeResponse {
	return api.PromoCodeResponse{
		Id:             p.ID.String(),
		Code:           p.Code,
		DiscountType:   api.DiscountType(p.DiscountType),
		DiscountValue:  float32(p.DiscountValue),
		MinOrderAmount: float32(p.MinOrderAmount),
		MaxUses:        p.MaxUses,
		CurrentUses:    p.CurrentUses,
		ValidFrom:      p.ValidFrom,
		ValidUntil:     p.ValidUntil,
		Active:         p.Active,
	}
}
