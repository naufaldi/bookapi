package http

import (
	"bookapi/internal/usecase"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type ProfileHandler struct {
	usecase *usecase.ProfileUsecase
}

func NewProfileHandler(usecase *usecase.ProfileUsecase) *ProfileHandler {
	return &ProfileHandler{usecase: usecase}
}

// @Summary Get own profile
// @Description Retrieve the authenticated user's complete profile and statistics
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security Bearer
// @Router /me/profile [get]
func (h *ProfileHandler) GetOwnProfile(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFrom(r)
	if userID == "" {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	profile, err := h.usecase.GetOwnProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			JSONError(w, http.StatusNotFound, "NOT_FOUND", "Profile not found", nil)
			return
		}
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	JSONSuccess(w, profile, nil)
}

// @Summary Get public profile
// @Description Retrieve public profile of another user by their ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /users/{id}/profile [get]
func (h *ProfileHandler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	// Expected path: /users/{id}/profile
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] != "users" || parts[2] != "profile" {
		JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid path", nil)
		return
	}

	userID := parts[1]
	profile, err := h.usecase.GetPublicProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			JSONError(w, http.StatusNotFound, "NOT_FOUND", "Profile not found or is private", nil)
			return
		}
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	JSONSuccess(w, profile, nil)
}

// @Summary Update profile
// @Description Update the authenticated user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Param updates body map[string]interface{} true "Profile updates (username, bio, location, website, reading_preferences, is_public)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security Bearer
// @Router /me/profile [patch]
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFrom(r)
	if userID == "" {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}

	// We don't want to allow updating certain fields through this endpoint
	delete(updates, "id")
	delete(updates, "email")
	delete(updates, "role")
	delete(updates, "created_at")
	delete(updates, "updated_at")

	profile, err := h.usecase.UpdateProfile(r.Context(), userID, updates)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	JSONSuccess(w, profile, nil)
}
