package http

import (
	"bookapi/internal/usecase"
	"errors"
	"net/http"
	"strings"
)

type SessionHandler struct {
	sessionRepo usecase.SessionRepository
}

func NewSessionHandler(sessionRepo usecase.SessionRepository) *SessionHandler {
	return &SessionHandler{sessionRepo: sessionRepo}
}

type SessionResponse struct {
	ID          string `json:"id"`
	UserAgent  string `json:"user_agent"`
	IPAddress  string `json:"ip_address"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at"`
	IsCurrent  bool   `json:"is_current"`
}

// @Summary List user sessions
// @Description Get all active sessions for the currently authenticated user
// @Tags sessions
// @Produce json
// @Security Bearer
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Router /me/sessions [get]
func (h *SessionHandler) ListSessionsHandler(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFrom(r)
	if userID == "" {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	sessions, err := h.sessionRepo.ListByUserID(r.Context(), userID)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	currentJTI := r.Header.Get("X-Current-JTI")
	var response []SessionResponse
	for _, session := range sessions {
		isCurrent := false
		if currentJTI != "" {
			isCurrent = true
		}

		response = append(response, SessionResponse{
			ID:          session.ID,
			UserAgent:  session.UserAgent,
			IPAddress:  session.IPAddress,
			CreatedAt:  session.CreatedAt.Format("2006-01-02T15:04:05Z"),
			LastUsedAt: session.LastUsedAt.Format("2006-01-02T15:04:05Z"),
			IsCurrent:  isCurrent,
		})
	}

	JSONSuccess(w, response, nil)
}

// @Summary Delete session
// @Description Invalidate a specific session by ID
// @Tags sessions
// @Security Bearer
// @Param id path string true "Session ID"
// @Success 204 "No Content"
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /me/sessions/{id} [delete]
func (h *SessionHandler) DeleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFrom(r)
	if userID == "" {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] != "me" || parts[1] != "sessions" {
		JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid path", nil)
		return
	}

	sessionID := parts[2]

	sessions, err := h.sessionRepo.ListByUserID(r.Context(), userID)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	found := false
	for _, session := range sessions {
		if session.ID == sessionID {
			found = true
			break
		}
	}

	if !found {
		JSONError(w, http.StatusNotFound, "NOT_FOUND", "Session not found", nil)
		return
	}

	if err := h.sessionRepo.Delete(r.Context(), sessionID); err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			JSONError(w, http.StatusNotFound, "NOT_FOUND", "Session not found", nil)
			return
		}
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	JSONSuccessNoContent(w)
}
