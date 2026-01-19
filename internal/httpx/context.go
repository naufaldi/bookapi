package httpx

import (
	"context"
	"net/http"
)

type contextKey string

const (
	userIDKey    contextKey = "userID"
	roleKey      contextKey = "role"
	requestIDKey contextKey = "requestID"
)

// UserIDFrom retrieves the user ID from the request context.
func UserIDFrom(r *http.Request) string {
	if v, ok := r.Context().Value(userIDKey).(string); ok {
		return v
	}
	return ""
}

// RoleFrom retrieves the user role from the request context.
func RoleFrom(r *http.Request) string {
	if v, ok := r.Context().Value(roleKey).(string); ok {
		return v
	}
	return ""
}

// ContextWithUser returns a new context with the user ID and role.
func ContextWithUser(ctx context.Context, userID, role string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	return context.WithValue(ctx, roleKey, role)
}
