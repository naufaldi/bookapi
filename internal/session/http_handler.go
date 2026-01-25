package session

import (
	"bookapi/internal/httpx"
	"errors"
	"net/http"
)

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

type SessionResponse struct {
	ID         string `json:"id"`
	UserAgent  string `json:"user_agent"`
	IPAddress  string `json:"ip_address"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at"`
	IsCurrent  bool   `json:"is_current"`
}

// ListSessions handles GET /me/sessions
// @Summary List user sessions
// @Description Get all active sessions for the authenticated user
// @Tags sessions
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} httpx.SuccessResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /me/sessions [get]
func (h *HTTPHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserIDFrom(r)
	if userID == "" {
		httpx.JSONError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	sessions, err := h.service.ListByUserID(r.Context(), userID)
	if err != nil {
		httpx.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	currentJTI := r.Header.Get("X-Current-JTI")
	var response []SessionResponse
	for _, s := range sessions {
		isCurrent := false
		if currentJTI != "" {
			// In a real app, you'd compare the JTI of the current session
			isCurrent = true
		}

		response = append(response, SessionResponse{
			ID:         s.ID,
			UserAgent:  s.UserAgent,
			IPAddress:  s.IPAddress,
			CreatedAt:  s.CreatedAt.Format("2006-01-02T15:04:05Z"),
			LastUsedAt: s.LastUsedAt.Format("2006-01-02T15:04:05Z"),
			IsCurrent:  isCurrent,
		})
	}

	httpx.JSONSuccess(w, r, response, nil)
}

// DeleteSession handles DELETE /me/sessions/{id}
// @Summary Delete session
// @Description Delete a specific session for the authenticated user
// @Tags sessions
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "Session ID"
// @Success 204 "No Content"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /me/sessions/{id} [delete]
func (h *HTTPHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserIDFrom(r)
	if userID == "" {
		httpx.JSONError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		httpx.JSONError(w, r, http.StatusBadRequest, "BAD_REQUEST", "Invalid session ID", nil)
		return
	}

	// Security check: ensure session belongs to user
	sessions, err := h.service.ListByUserID(r.Context(), userID)
	if err != nil {
		httpx.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	found := false
	for _, s := range sessions {
		if s.ID == sessionID {
			found = true
			break
		}
	}

	if !found {
		httpx.JSONError(w, r, http.StatusNotFound, "NOT_FOUND", "Session not found", nil)
		return
	}

	if err := h.service.Delete(r.Context(), sessionID); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.JSONError(w, r, http.StatusNotFound, "NOT_FOUND", "Session not found", nil)
			return
		}
		httpx.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccessNoContent(w)
}
