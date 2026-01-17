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

func (h *HTTPHandler) GetOwnProfile(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserIDFrom(r)
	if userID == "" {
		httpx.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	p, err := h.service.GetOwnProfile(r.Context(), userID)
	if err != nil {
		httpx.JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccess(w, p, nil)
}

func (h *HTTPHandler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		httpx.JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid user ID", nil)
		return
	}

	p, err := h.service.GetPublicProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			httpx.JSONError(w, http.StatusNotFound, "NOT_FOUND", "Profile not found or private", nil)
			return
		}
		httpx.JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccess(w, p, nil)
}

func (h *HTTPHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserIDFrom(r)
	if userID == "" {
		httpx.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	var cmd UpdateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		httpx.JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}

	p, err := h.service.UpdateProfile(r.Context(), userID, cmd)
	if err != nil {
		httpx.JSONError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	httpx.JSONSuccess(w, p, nil)
}
