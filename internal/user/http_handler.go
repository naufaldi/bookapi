package user

import (
	"bookapi/internal/httpx"
	"bookapi/internal/platform/crypto"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

type registerReq struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,password_strength"`
}

// RegisterUser handles POST /users/register
// @Summary Register a new user
// @Description Create a new user account
// @Tags users
// @Accept json
// @Produce json
// @Param request body registerReq true "Registration request"
// @Success 201 {object} httpx.SuccessResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /users/register [post]
func (h *HTTPHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	req.Username = strings.TrimSpace(req.Username)

	if validationErrors := httpx.ValidateStruct(req); len(validationErrors) > 0 {
		// Convert httpx.ErrorDetail to httpx.ErrorDetail (it's the same type now)
		httpx.JSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input", validationErrors)
		return
	}

	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		httpx.JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	newUser, err := h.service.Register(r.Context(), req.Email, req.Username, hashedPassword)
	if err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			httpx.JSONError(w, http.StatusConflict, "ALREADY_EXISTS", "Email already exists", nil)
			return
		}
		httpx.JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccessCreated(w, map[string]any{
		"id":       newUser.ID,
		"email":    newUser.Email,
		"username": newUser.Username,
		"role":     newUser.Role,
	})
}

// GetCurrentUser handles GET /me
// @Summary Get current user
// @Description Get the authenticated user's information
// @Tags users
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} httpx.SuccessResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /me [get]
func (h *HTTPHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserIDFrom(r)
	if userID == "" {
		httpx.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	user, err := h.service.GetByID(r.Context(), userID)
	if err != nil {
		httpx.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	httpx.JSONSuccess(w, map[string]any{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
		"role":     user.Role,
	}, nil)
}
