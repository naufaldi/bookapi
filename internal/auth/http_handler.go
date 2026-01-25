package auth

import (
	"bookapi/internal/httpx"
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

type LoginReq struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required"`
	RememberMe bool   `json:"remember_me"`
}

// Login handles POST /users/login
// @Summary User login
// @Description Authenticate user and receive access and refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginReq true "Login request"
// @Success 200 {object} httpx.SuccessResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /users/login [post]
func (h *HTTPHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSONError(w, r, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}
	req.Email = strings.TrimSpace(req.Email)

	if validationErrors := httpx.ValidateStruct(req); len(validationErrors) > 0 {
		httpx.JSONError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input", validationErrors)
		return
	}

	userAgent := r.Header.Get("User-Agent")
	ipAddress := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ipAddress = strings.Split(forwarded, ",")[0]
	}

	accessToken, refreshToken, expiresIn, err := h.service.Login(r.Context(), req.Email, req.Password, req.RememberMe, userAgent, ipAddress)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			httpx.JSONError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid email or password", nil)
			return
		}
		httpx.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccess(w, r, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    expiresIn,
	}, nil)
}

type RefreshReq struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshToken handles POST /auth/refresh
// @Summary Refresh access token
// @Description Get a new access token using a refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshReq true "Refresh token request"
// @Success 200 {object} httpx.SuccessResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /auth/refresh [post]
func (h *HTTPHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSONError(w, r, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}

	if validationErrors := httpx.ValidateStruct(req); len(validationErrors) > 0 {
		httpx.JSONError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input", validationErrors)
		return
	}

	accessToken, refreshToken, expiresIn, err := h.service.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			httpx.JSONError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired refresh token", nil)
			return
		}
		httpx.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccess(w, r, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    expiresIn,
	}, nil)
}

// Logout handles POST /auth/logout
// @Summary User logout
// @Description Logout and invalidate the current access token
// @Tags auth
// @Accept json
// @Produce json
// @Security Bearer
// @Success 204 "No Content"
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /auth/logout [post]
func (h *HTTPHandler) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		httpx.JSONError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	userID := httpx.UserIDFrom(r)
	if userID == "" {
		httpx.JSONError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	if err := h.service.Logout(r.Context(), token, userID); err != nil {
		httpx.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccessNoContent(w)
}
