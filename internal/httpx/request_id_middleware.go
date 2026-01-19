package httpx

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const maxRequestIDLength = 128

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		
		if requestID == "" || !isValidRequestID(requestID) {
			requestID = uuid.New().String()
		} else {
			if len(requestID) > maxRequestIDLength {
				requestID = requestID[:maxRequestIDLength]
			}
		}

		w.Header().Set("X-Request-Id", requestID)
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestIDFrom(r *http.Request) string {
	if v, ok := r.Context().Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

func isValidRequestID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, r := range id {
		if r < 32 || r > 126 {
			return false
		}
	}
	return true
}
