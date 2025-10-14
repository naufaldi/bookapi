package http

import (
	"bookapi/internal/auth"
	"bookapi/internal/entity"
	"bookapi/internal/usecase"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type UserHandler struct {
	repo usecase.UserRepository
	secret string
}

func NewUserHandler(repo usecase.UserRepository, secret string) *UserHandler {
	return &UserHandler{
		repo: repo,
		secret: secret,
	}
}

type registerReq struct {
	Email string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (handler *UserHandler) RegisterUser(responseWriter http.ResponseWriter, request *http.Request) {
	var registerReq registerReq
	if err := json.NewDecoder(request.Body).Decode(&registerReq); err != nil {
		http.Error(responseWriter, "bad request", http.StatusBadRequest)
		return
	}
	registerReq.Email = strings.TrimSpace(registerReq.Email)
	registerReq.Username = strings.TrimSpace(registerReq.Username)
	
	if registerReq.Email == "" || registerReq.Username == "" || len(registerReq.Password) < 6 {
		http.Error(responseWriter, "Invalid request", http.StatusBadRequest)
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
		Email: registerReq.Email,
		Username: registerReq.Username,
		Password: hashedPassword,
		Role: "USER",
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
	Email string `json:"email"`
	Password string `json:"password"`
}

func (userHandler *UserHandler) LoginUser(responseWriter http.ResponseWriter, request *http.Request){
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
	if findErr != nil || !auth.VerifyPassword(foundUser.Password, loginReq.Password){
		http.Error(responseWriter, "unauthorized", http.StatusUnauthorized)
		return
	}

	const accessTokenTTL = 24 * time.Hour
	signedAccessToken, signErr := auth.GenerateToken(userHandler.secret, foundUser.ID, foundUser.Role, accessTokenTTL)

	if signErr != nil {
		http.Error(responseWriter, "server error", http.StatusInternalServerError)
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(map[string]any{
		"data": map[string]any{
			"access_token": signedAccessToken,
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
			"id": user.ID,
			"email": user.Email,
			"username": user.Username,
		},
	})
}