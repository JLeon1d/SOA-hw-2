package users

import (
	"encoding/json"
	"net/http"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/errors"
	"marketplace-backend/internal/generated/api"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req api.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
		return
	}

	role := domain.UserRole(req.Role)
	_, err := h.service.Register(string(req.Email), req.Password, role)
	if err != nil {
		errors.NewAppError(errors.ValidationError, err.Error(), http.StatusBadRequest).WriteJSON(w)
		return
	}

	accessToken, refreshToken, err := h.service.Login(string(req.Email), req.Password)
	if err != nil {
		errors.NewAppError(errors.TokenInvalid, err.Error(), http.StatusInternalServerError).WriteJSON(w)
		return
	}

	response := api.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    1800,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req api.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
		return
	}

	accessToken, refreshToken, err := h.service.Login(string(req.Email), req.Password)
	if err != nil {
		errors.NewTokenInvalid().WriteJSON(w)
		return
	}

	response := api.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    1800,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req api.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.NewValidationError("Invalid request body", nil).WriteJSON(w)
		return
	}

	accessToken, err := h.service.RefreshToken(req.RefreshToken)
	if err != nil {
		errors.NewRefreshTokenInvalid().WriteJSON(w)
		return
	}

	response := api.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: req.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    1800,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
