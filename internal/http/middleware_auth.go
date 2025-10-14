package http

import (
	"bookapi/internal/auth"
	"context"
	"net/http"
	"strings"
)

type contextKey string
const userIDKey contextKey = "userID"
const roleKey contextKey = "role"

func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			authHeader := request.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(responseWriter, "unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := auth.ParseToken(secret, token)
			if err != nil {
				http.Error(responseWriter, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(request.Context(), userIDKey, claims.Sub)
			ctx = context.WithValue(ctx, roleKey, claims.Role)
			next.ServeHTTP(responseWriter, request.WithContext(ctx))
		})
	}
}

// helper utk ambil user id/role dari context
func UserIDFrom(request *http.Request) string {
	contextValue := request.Context().Value(userIDKey)
	if contextValue != nil {
		if userID, ok := contextValue.(string); ok {
			return userID
		}
	}
	return ""
}
func RoleFrom(request *http.Request) string {
	contextValue := request.Context().Value(roleKey)
	if contextValue != nil {
		if role, ok := contextValue.(string); ok {
			return role
		}
	}
	return ""
}