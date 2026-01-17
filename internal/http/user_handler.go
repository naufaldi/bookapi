package http

import (
	"bookapi/internal/auth"
	"bookapi/internal/entity"
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

type UserHandler struct {
	repo        usecase.UserRepository
	sessionRepo usecase.SessionRepository
	secret      string
}

func NewUserHandler(repo usecase.UserRepository, sessionRepo usecase.SessionRepository, secret string) *UserHandler {
	return &UserHandler{
		repo:        repo,
		sessionRepo: sessionRepo,
		secret:      secret,
	}
}

type registerReq struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,password_strength"`
}

// @Summary Register new user
// @Description Create a new user account
// @Tags users
// @Accept json
// @Produce json
// @Param user body registerReq true "User registration data"
// @Success 201 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /users/register [post]
func (handler *UserHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	req.Username = strings.TrimSpace(req.Username)

	if validationErrors := ValidateStruct(req); len(validationErrors) > 0 {
		JSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input", validationErrors)
		return
	}

	_, err := handler.repo.GetByEmail(r.Context(), req.Email)
	if err == nil {
		JSONError(w, http.StatusConflict, "ALREADY_EXISTS", "Email already exists", nil)
		return
	}

	if !errors.Is(err, usecase.ErrNotFound) {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	newUser := &entity.User{
		Email:    req.Email,
		Username: req.Username,
		Password: hashedPassword,
		Role:     "USER",
	}
	if err := handler.repo.Create(r.Context(), newUser); err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	JSONSuccessCreated(w, map[string]any{
		"id":       newUser.ID,
		"email":    newUser.Email,
		"username": newUser.Username,
		"role":     newUser.Role,
	})
}

type LoginReq struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required"`
	RememberMe bool   `json:"remember_me"`
}

// @Summary Login user
// @Description Authenticate user and create session
// @Tags users
// @Accept json
// @Produce json
// @Param login body LoginReq true "Login credentials"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /users/login [post]
func (h *UserHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var req LoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}
	req.Email = strings.TrimSpace(req.Email)

	if validationErrors := ValidateStruct(req); len(validationErrors) > 0 {
		JSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input", validationErrors)
		return
	}

	user, err := h.repo.GetByEmail(r.Context(), req.Email)
	if err != nil || !auth.VerifyPassword(user.Password, req.Password) {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid email or password", nil)
		return
	}

	const accessTokenTTL = 15 * time.Minute
	refreshTokenTTL := 30 * 24 * time.Hour
	if req.RememberMe {
		refreshTokenTTL = 90 * 24 * time.Hour
	}

	signedAccessToken, _, err := auth.GenerateToken(h.secret, user.ID, user.Role, accessTokenTTL)
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
	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	userAgent := r.Header.Get("User-Agent")
	ipAddress := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ipAddress = strings.Split(forwarded, ",")[0]
	}

	session := &entity.Session{
		UserID:          user.ID,
		RefreshTokenHash: tokenHash,
		UserAgent:       userAgent,
		IPAddress:       ipAddress,
		RememberMe:      req.RememberMe,
		ExpiresAt:       time.Now().Add(refreshTokenTTL),
	}

	if err := h.sessionRepo.Create(r.Context(), session); err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	JSONSuccess(w, map[string]any{
		"access_token":  signedAccessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(accessTokenTTL.Seconds()),
	}, nil)
}

// @Summary Get current user
// @Description Get currently authenticated user details
// @Tags users
// @Produce json
// @Security Bearer
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Router /me [get]
func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFrom(r)
	if userID == "" {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	user, err := h.repo.GetByID(r.Context(), userID)
	if err != nil {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	JSONSuccess(w, map[string]any{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
		"role":     user.Role,
	}, nil)
}
