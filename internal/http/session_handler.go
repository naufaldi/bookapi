package http

import (
	"bookapi/internal/usecase"
	"encoding/json"
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

func (h *SessionHandler) ListSessionsHandler(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFrom(r)
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sessions, err := h.sessionRepo.ListByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    response,
	})
}

func (h *SessionHandler) DeleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFrom(r)
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] != "me" || parts[1] != "sessions" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sessionID := parts[2]

	sessions, err := h.sessionRepo.ListByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
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
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err := h.sessionRepo.Delete(r.Context(), sessionID); err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
