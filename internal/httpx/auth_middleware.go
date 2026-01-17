package httpx

import (
	"bookapi/internal/platform/crypto"
	"context"
	"net/http"
	"strings"
)

// BlacklistRepository interface to decouple from usecase package later if needed
type BlacklistRepository interface {
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

func AuthMiddleware(secret string, blacklistRepo BlacklistRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := crypto.ParseToken(secret, token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if blacklistRepo != nil {
				isBlacklisted, err := blacklistRepo.IsBlacklisted(r.Context(), claims.ID)
				if err != nil || isBlacklisted {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
			}

			ctx := ContextWithUser(r.Context(), claims.Sub, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
