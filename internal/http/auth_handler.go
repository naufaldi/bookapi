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

// @Summary Logout user
// @Description Invalidate current access token by adding it to blacklist
// @Tags auth
// @Produce json
// @Security Bearer
// @Success 204 "No Content"
// @Failure 401 {object} ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := auth.ParseToken(h.secret, token)
	if err != nil {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	userID := UserIDFrom(r)
	if userID == "" {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	if err := h.blacklistRepo.AddToken(r.Context(), claims.ID, userID, expiresAt); err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	JSONSuccessNoContent(w)
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// @Summary Refresh access token
// @Description Get a new access token using a valid refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}

	if validationErrors := ValidateStruct(req); len(validationErrors) > 0 {
		JSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input", validationErrors)
		return
	}

	tokenHash := hashToken(req.RefreshToken)
	session, err := h.sessionRepo.GetByTokenHash(r.Context(), tokenHash)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired refresh token", nil)
			return
		}
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), session.UserID)
	if err != nil {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found", nil)
		return
	}

	if err := h.sessionRepo.DeleteByTokenHash(r.Context(), tokenHash); err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	accessTokenTTL := 15 * time.Minute
	refreshTokenTTL := 30 * 24 * time.Hour
	if session.RememberMe {
		refreshTokenTTL = 90 * 24 * time.Hour
	}

	accessToken, _, err := auth.GenerateToken(h.secret, user.ID, user.Role, accessTokenTTL)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}
	refreshToken := hex.EncodeToString(refreshTokenBytes)
	newTokenHash := hashToken(refreshToken)

	newSession := session
	newSession.RefreshTokenHash = newTokenHash
	newSession.ExpiresAt = time.Now().Add(refreshTokenTTL)
	newSession.ID = ""
	if err := h.sessionRepo.Create(r.Context(), &newSession); err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	JSONSuccess(w, RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	}, nil)
}
