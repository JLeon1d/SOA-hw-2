package products

import (
	"encoding/json"
	"fmt"
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

type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Category    string  `json:"category"`
}

type UpdateProductRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Price       *float64 `json:"price"`
	Stock       *int     `json:"stock"`
	Category    *string  `json:"category"`
}

type ProductResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	Price       float64    `json:"price"`
	Stock       int        `json:"stock"`
	Category    string     `json:"category"`
	Status      string     `json:"status"`
	SellerID    *uuid.UUID `json:"seller_id"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
}

type ProductListResponse struct {
	Items  []ProductResponse `json:"items"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	role := middleware.GetUserRole(r.Context())

	if role != string(domain.RoleSeller) && role != string(domain.RoleAdmin) {
		errors.NewAccessDenied().WriteJSON(w)
		return
	}

	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
		return
	}

	sellerUUID, err := uuid.Parse(userID)
	if err != nil {
		errors.NewValidationError("Invalid user ID", nil).WriteJSON(w)
		return
	}

	product := &domain.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
		SellerID:    &sellerUUID,
		Status:      domain.ProductActive,
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

	items := make([]ProductResponse, len(result.Products))
	for i, p := range result.Products {
		items[i] = mapProductToResponse(&p)
	}

	response := ProductListResponse{
		Items:  items,
		Total:  result.TotalElements,
		Limit:  size,
		Offset: (page - 1) * size,
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

	var req UpdateProductRequest
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
		existing.Price = *req.Price
	}
	if req.Stock != nil {
		existing.Stock = *req.Stock
	}
	if req.Category != nil {
		existing.Category = *req.Category
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

func mapProductToResponse(p *domain.Product) ProductResponse {
	return ProductResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
		Category:    p.Category,
		Status:      string(p.Status),
		SellerID:    p.SellerID,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
