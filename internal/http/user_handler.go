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
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /users/register [post]
func (handler *UserHandler) RegisterUser(responseWriter http.ResponseWriter, request *http.Request) {
	var registerReq registerReq
	if err := json.NewDecoder(request.Body).Decode(&registerReq); err != nil {
		http.Error(responseWriter, "bad request", http.StatusBadRequest)
		return
	}
	registerReq.Email = strings.TrimSpace(registerReq.Email)
	registerReq.Username = strings.TrimSpace(registerReq.Username)

	if validationErrors := ValidateStruct(registerReq); len(validationErrors) > 0 {
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(map[string]any{
			"success": false,
			"error": map[string]any{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid input",
				"details": validationErrors,
			},
		})
		return
	}

	_, err := handler.repo.GetByEmail(request.Context(), registerReq.Email)
	if err == nil {
		http.Error(responseWriter, "Email already exists", http.StatusConflict)
		return
	}

	if !errors.Is(err, usecase.ErrNotFound) {
		http.Error(responseWriter, "Internal server error", http.StatusInternalServerError)
		return
	}

	hashedPassword, hashErr := auth.HashPassword(registerReq.Password)
	if hashErr != nil {
		http.Error(responseWriter, "server error", http.StatusInternalServerError)
		return
	}

	newUser := &entity.User{
		Email:    registerReq.Email,
		Username: registerReq.Username,
		Password: hashedPassword,
		Role:     "USER",
	}
	if createErr := handler.repo.Create(request.Context(), newUser); createErr != nil {
		http.Error(responseWriter, "server error", http.StatusInternalServerError)
		return
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(map[string]any{
		"data": map[string]any{
			"id":       newUser.ID,
			"email":    newUser.Email,
			"username": newUser.Username,
			"role":     newUser.Role,
		},
	})
}

type LoginReq struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

func (userHandler *UserHandler) LoginUser(responseWriter http.ResponseWriter, request *http.Request) {
	var loginReq LoginReq
	if err := json.NewDecoder(request.Body).Decode(&loginReq); err != nil {
		http.Error(responseWriter, "bad request", http.StatusBadRequest)
		return
	}
	loginReq.Email = strings.TrimSpace(loginReq.Email)

	if loginReq.Email == "" || len(loginReq.Password) < 6 {
		http.Error(responseWriter, "Invalid input", http.StatusBadRequest)
		return
	}

	foundUser, findErr := userHandler.repo.GetByEmail(request.Context(), loginReq.Email)
	if findErr != nil || !auth.VerifyPassword(foundUser.Password, loginReq.Password) {
		http.Error(responseWriter, "unauthorized", http.StatusUnauthorized)
		return
	}

	const accessTokenTTL = 15 * time.Minute
	refreshTokenTTL := 30 * 24 * time.Hour
	if loginReq.RememberMe {
		refreshTokenTTL = 90 * 24 * time.Hour
	}

	signedAccessToken, _, signErr := auth.GenerateToken(userHandler.secret, foundUser.ID, foundUser.Role, accessTokenTTL)
	if signErr != nil {
		http.Error(responseWriter, "server error", http.StatusInternalServerError)
		return
	}

	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		http.Error(responseWriter, "server error", http.StatusInternalServerError)
		return
	}
	refreshToken := hex.EncodeToString(refreshTokenBytes)
	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	userAgent := request.Header.Get("User-Agent")
	ipAddress := request.RemoteAddr
	if forwarded := request.Header.Get("X-Forwarded-For"); forwarded != "" {
		ipAddress = strings.Split(forwarded, ",")[0]
	}

	session := &entity.Session{
		UserID:          foundUser.ID,
		RefreshTokenHash: tokenHash,
		UserAgent:       userAgent,
		IPAddress:       ipAddress,
		RememberMe:      loginReq.RememberMe,
		ExpiresAt:       time.Now().Add(refreshTokenTTL),
	}

	if err := userHandler.sessionRepo.Create(request.Context(), session); err != nil {
		http.Error(responseWriter, "server error", http.StatusInternalServerError)
		return
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(map[string]any{
		"success": true,
		"data": map[string]any{
			"access_token":  signedAccessToken,
			"refresh_token": refreshToken,
			"expires_in":    int(accessTokenTTL.Seconds()),
		},
	})

}

func (userHandler *UserHandler) GetCurrentUser(responseWriter http.ResponseWriter, request *http.Request) {
	userID := UserIDFrom(request)
	if userID == "" {
		http.Error(responseWriter, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := userHandler.repo.GetByID(request.Context(), userID)
	if err != nil {
		http.Error(responseWriter, "unauthorized", http.StatusUnauthorized)
		return
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(map[string]any{
		"data": map[string]any{
			"id":       user.ID,
			"email":    user.Email,
			"username": user.Username,
		},
	})
}
