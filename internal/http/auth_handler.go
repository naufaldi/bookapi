package http

import (
	"bookapi/internal/auth"
	"bookapi/internal/usecase"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type AuthHandler struct {
	secret           string
	sessionRepo      usecase.SessionRepository
	blacklistRepo    usecase.BlacklistRepository
	userRepo         usecase.UserRepository
}

func NewAuthHandler(secret string, sessionRepo usecase.SessionRepository, blacklistRepo usecase.BlacklistRepository, userRepo usecase.UserRepository) *AuthHandler {
	return &AuthHandler{
		secret:        secret,
		sessionRepo:   sessionRepo,
		blacklistRepo: blacklistRepo,
		userRepo:      userRepo,
	}
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := auth.ParseToken(h.secret, token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID := UserIDFrom(r)
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	if err := h.blacklistRepo.AddToken(r.Context(), claims.ID, userID, expiresAt); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (h *AuthHandler) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if validationErrors := ValidateStruct(req); len(validationErrors) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error": map[string]any{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid input",
				"details": validationErrors,
			},
		})
		return
	}

	tokenHash := hashToken(req.RefreshToken)
	session, err := h.sessionRepo.GetByTokenHash(r.Context(), tokenHash)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), session.UserID)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.sessionRepo.DeleteByTokenHash(r.Context(), tokenHash); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	accessTokenTTL := 15 * time.Minute
	refreshTokenTTL := 30 * 24 * time.Hour
	if session.RememberMe {
		refreshTokenTTL = 90 * 24 * time.Hour
	}

	accessToken, _, err := auth.GenerateToken(h.secret, user.ID, user.Role, accessTokenTTL)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	refreshToken := hex.EncodeToString(refreshTokenBytes)
	newTokenHash := hashToken(refreshToken)

	newSession := session
	newSession.RefreshTokenHash = newTokenHash
	newSession.ExpiresAt = time.Now().Add(refreshTokenTTL)
	newSession.ID = ""
	if err := h.sessionRepo.Create(r.Context(), &newSession); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data": RefreshTokenResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    int(accessTokenTTL.Seconds()),
		},
	})
}
