package products

import (
	"encoding/json"
	"fmt"
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

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	role := middleware.GetUserRole(r.Context())

	if role != string(domain.RoleSeller) && role != string(domain.RoleAdmin) {
		errors.NewAccessDenied().WriteJSON(w)
		return
	}

	var req api.ProductCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
		return
	}

	sellerUUID, err := uuid.Parse(userID)
	if err != nil {
		errors.NewValidationError("Invalid user ID", nil).WriteJSON(w)
		return
	}

	status := domain.ProductActive
	if req.Status != nil {
		status = domain.ProductStatus(*req.Status)
	}

	product := &domain.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       float64(req.Price),
		Stock:       req.Stock,
		Category:    req.Category,
		SellerID:    &sellerUUID,
		Status:      status,
	}

	if err := h.service.Create(product); err != nil {
		errors.NewAppError(errors.ValidationError, err.Error(), http.StatusBadRequest).WriteJSON(w)
		return
	}

	response := mapProductToResponse(product)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.NewValidationError("Invalid product ID", nil).WriteJSON(w)
		return
	}

	product, err := h.service.GetByID(id)
	if err != nil {
		errors.NewProductNotFound(idStr).WriteJSON(w)
		return
	}

	response := mapProductToResponse(product)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	page := 1
	if p := query.Get("page"); p != "" {
		var parsed int
		if _, err := fmt.Sscanf(p, "%d", &parsed); err == nil && parsed > 0 {
			page = parsed
		}
	}

	size := 20
	if s := query.Get("size"); s != "" {
		var parsed int
		if _, err := fmt.Sscanf(s, "%d", &parsed); err == nil && parsed > 0 && parsed <= 100 {
			size = parsed
		}
	}

	filter := Filter{
		Page: page - 1, // Convert to 0-based for repository
		Size: size,
	}

	if category := query.Get("category"); category != "" {
		filter.Category = &category
	}

	if status := query.Get("status"); status != "" {
		productStatus := domain.ProductStatus(status)
		filter.Status = &productStatus
	}

	result, err := h.service.List(filter)
	if err != nil {
		errors.NewAppError(errors.ValidationError, err.Error(), http.StatusInternalServerError).WriteJSON(w)
		return
	}

	items := make([]api.ProductResponse, len(result.Products))
	for i, p := range result.Products {
		items[i] = mapProductToResponse(&p)
	}

	response := api.ProductListResponse{
		Items:         items,
		TotalElements: result.TotalElements,
		Page:          page - 1,
		Size:          size,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	role := middleware.GetUserRole(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.NewValidationError("Invalid product ID", nil).WriteJSON(w)
		return
	}

	existing, err := h.service.GetByID(id)
	if err != nil {
		errors.NewProductNotFound(idStr).WriteJSON(w)
		return
	}

	if role != string(domain.RoleAdmin) && (existing.SellerID == nil || existing.SellerID.String() != userID) {
		errors.NewAccessDenied().WriteJSON(w)
		return
	}

	var req api.ProductUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = req.Description
	}
	if req.Price != nil {
		existing.Price = float64(*req.Price)
	}
	if req.Stock != nil {
		existing.Stock = *req.Stock
	}
	if req.Category != nil {
		existing.Category = *req.Category
	}
	if req.Status != nil {
		existing.Status = domain.ProductStatus(*req.Status)
	}

	if err := h.service.Update(existing); err != nil {
		errors.NewAppError(errors.ValidationError, err.Error(), http.StatusBadRequest).WriteJSON(w)
		return
	}

	response := mapProductToResponse(existing)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	role := middleware.GetUserRole(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.NewValidationError("Invalid product ID", nil).WriteJSON(w)
		return
	}

	existing, err := h.service.GetByID(id)
	if err != nil {
		errors.NewProductNotFound(idStr).WriteJSON(w)
		return
	}

	if role != string(domain.RoleAdmin) && (existing.SellerID == nil || existing.SellerID.String() != userID) {
		errors.NewAccessDenied().WriteJSON(w)
		return
	}

	if err := h.service.Delete(id); err != nil {
		errors.NewAppError(errors.ValidationError, err.Error(), http.StatusInternalServerError).WriteJSON(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func mapProductToResponse(p *domain.Product) api.ProductResponse {
	resp := api.ProductResponse{
		Id:        p.ID.String(),
		Name:      p.Name,
		Price:     float32(p.Price),
		Stock:     p.Stock,
		Category:  p.Category,
		Status:    api.ProductStatus(p.Status),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}

	if p.Description != nil {
		resp.Description = p.Description
	}

	if p.SellerID != nil {
		sellerIDStr := p.SellerID.String()
		resp.SellerId = &sellerIDStr
	}

	return resp
}
