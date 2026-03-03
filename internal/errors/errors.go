package errors

import (
	"encoding/json"
	"net/http"
)

type ErrorCode string

const (
	ProductNotFound         ErrorCode = "PRODUCT_NOT_FOUND"
	ProductInactive         ErrorCode = "PRODUCT_INACTIVE"
	OrderNotFound           ErrorCode = "ORDER_NOT_FOUND"
	OrderLimitExceeded      ErrorCode = "ORDER_LIMIT_EXCEEDED"
	OrderHasActive          ErrorCode = "ORDER_HAS_ACTIVE"
	InvalidStateTransition  ErrorCode = "INVALID_STATE_TRANSITION"
	InsufficientStock       ErrorCode = "INSUFFICIENT_STOCK"
	PromoCodeInvalid        ErrorCode = "PROMO_CODE_INVALID"
	PromoCodeMinAmount      ErrorCode = "PROMO_CODE_MIN_AMOUNT"
	OrderOwnershipViolation ErrorCode = "ORDER_OWNERSHIP_VIOLATION"
	ValidationError         ErrorCode = "VALIDATION_ERROR"
	TokenExpired            ErrorCode = "TOKEN_EXPIRED"
	TokenInvalid            ErrorCode = "TOKEN_INVALID"
	RefreshTokenInvalid     ErrorCode = "REFRESH_TOKEN_INVALID"
	AccessDenied            ErrorCode = "ACCESS_DENIED"
)

type AppError struct {
	Code       ErrorCode              `json:"error_code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	HTTPStatus int                    `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(code ErrorCode, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Details:    make(map[string]interface{}),
	}
}

func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

func (e *AppError) WriteJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.HTTPStatus)
	json.NewEncoder(w).Encode(e)
}

func NewProductNotFound(productID string) *AppError {
	return NewAppError(ProductNotFound, "Product not found", http.StatusNotFound).
		WithDetails(map[string]interface{}{"product_id": productID})
}

func NewProductInactive(productID string) *AppError {
	return NewAppError(ProductInactive, "Cannot order inactive product", http.StatusConflict).
		WithDetails(map[string]interface{}{"product_id": productID})
}

func NewOrderNotFound(orderID string) *AppError {
	return NewAppError(OrderNotFound, "Order not found", http.StatusNotFound).
		WithDetails(map[string]interface{}{"order_id": orderID})
}

func NewOrderLimitExceeded(retryAfterSeconds int) *AppError {
	return NewAppError(OrderLimitExceeded, "Order operation rate limit exceeded", http.StatusTooManyRequests).
		WithDetails(map[string]interface{}{"retry_after_seconds": retryAfterSeconds})
}

func NewOrderHasActive() *AppError {
	return NewAppError(OrderHasActive, "User already has an active order", http.StatusConflict)
}

func NewInvalidStateTransition(from, to string) *AppError {
	return NewAppError(InvalidStateTransition, "Invalid order status transition", http.StatusConflict).
		WithDetails(map[string]interface{}{"from": from, "to": to})
}

func NewInsufficientStock(products []map[string]interface{}) *AppError {
	return NewAppError(InsufficientStock, "Insufficient stock for requested products", http.StatusConflict).
		WithDetails(map[string]interface{}{"products": products})
}

func NewPromoCodeInvalid(reason string) *AppError {
	return NewAppError(PromoCodeInvalid, "Promo code is invalid, expired, or exhausted", http.StatusUnprocessableEntity).
		WithDetails(map[string]interface{}{"reason": reason})
}

func NewPromoCodeMinAmount(minAmount, currentAmount float64) *AppError {
	return NewAppError(PromoCodeMinAmount, "Order amount below minimum for promo code", http.StatusUnprocessableEntity).
		WithDetails(map[string]interface{}{
			"min_amount":     minAmount,
			"current_amount": currentAmount,
		})
}

func NewOrderOwnershipViolation() *AppError {
	return NewAppError(OrderOwnershipViolation, "Order belongs to another user", http.StatusForbidden)
}

func NewValidationError(message string, details map[string]interface{}) *AppError {
	return NewAppError(ValidationError, message, http.StatusBadRequest).
		WithDetails(details)
}

func NewTokenExpired() *AppError {
	return NewAppError(TokenExpired, "Access token has expired", http.StatusUnauthorized)
}

func NewTokenInvalid() *AppError {
	return NewAppError(TokenInvalid, "Invalid access token", http.StatusUnauthorized)
}

func NewRefreshTokenInvalid() *AppError {
	return NewAppError(RefreshTokenInvalid, "Invalid or expired refresh token", http.StatusUnauthorized)
}

func NewAccessDenied() *AppError {
	return NewAppError(AccessDenied, "Insufficient permissions to access this resource", http.StatusForbidden)
}
