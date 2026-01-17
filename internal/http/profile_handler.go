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
