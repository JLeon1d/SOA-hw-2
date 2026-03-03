package middleware

import (
	"context"
	"net/http"
	"strings"

	"marketplace-backend/internal/errors"
	"marketplace-backend/internal/users"
)

const claimsKey contextKey = "claims"

func AuthMiddleware(authService users.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				errors.NewTokenInvalid().WriteJSON(w)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				errors.NewTokenInvalid().WriteJSON(w)
				return
			}

			token := parts[1]

			claims, err := authService.ValidateAccessToken(token)
			if err != nil {
				if err.Error() == "token expired" {
					errors.NewTokenExpired().WriteJSON(w)
				} else {
					errors.NewTokenInvalid().WriteJSON(w)
				}
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetClaims(ctx context.Context) *users.Claims {
	if claims, ok := ctx.Value(claimsKey).(*users.Claims); ok {
		return claims
	}
	return nil
}

func GetUserID(ctx context.Context) string {
	claims := GetClaims(ctx)
	if claims != nil {
		return claims.UserID
	}
	return ""
}

func GetUserRole(ctx context.Context) string {
	claims := GetClaims(ctx)
	if claims != nil {
		return claims.Role
	}
	return ""
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				errors.NewTokenInvalid().WriteJSON(w)
				return
			}

			hasRole := false
			for _, role := range roles {
				if claims.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				errors.NewAccessDenied().WriteJSON(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
