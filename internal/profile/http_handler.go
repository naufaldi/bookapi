package profile

import (
	"bookapi/internal/httpx"
	"encoding/json"
	"errors"
	"net/http"

	"bookapi/internal/user"
)

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

// GetOwnProfile handles GET /me/profile
// @Summary Get own profile
// @Description Get the authenticated user's complete profile
// @Tags profiles
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} httpx.SuccessResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /me/profile [get]
func (h *HTTPHandler) GetOwnProfile(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserIDFrom(r)
	if userID == "" {
		httpx.JSONError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	p, err := h.service.GetOwnProfile(r.Context(), userID)
	if err != nil {
		httpx.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccess(w, r, p, nil)
}

// GetPublicProfile handles GET /users/{id}/profile
// @Summary Get public profile
// @Description Get a user's public profile information
// @Tags profiles
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} httpx.SuccessResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /users/{id}/profile [get]
func (h *HTTPHandler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		httpx.JSONError(w, r, http.StatusBadRequest, "BAD_REQUEST", "Invalid user ID", nil)
		return
	}

	p, err := h.service.GetPublicProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			httpx.JSONError(w, r, http.StatusNotFound, "NOT_FOUND", "Profile not found or private", nil)
			return
		}
		httpx.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccess(w, r, p, nil)
}

// UpdateProfile handles PATCH /me/profile
// @Summary Update own profile
// @Description Update the authenticated user's profile
// @Tags profiles
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body UpdateCommand true "Profile update request"
// @Success 200 {object} httpx.SuccessResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /me/profile [patch]
func (h *HTTPHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserIDFrom(r)
	if userID == "" {
		httpx.JSONError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	var cmd UpdateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		httpx.JSONError(w, r, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}

	p, err := h.service.UpdateProfile(r.Context(), userID, cmd)
	if err != nil {
		httpx.JSONError(w, r, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	httpx.JSONSuccess(w, r, p, nil)
}
