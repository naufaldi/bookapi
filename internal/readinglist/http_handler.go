package readinglist

import (
	"bookapi/internal/httpx"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

type upsertReq struct {
	ISBN   string `json:"isbn" validate:"required"`
	Status string `json:"status" validate:"required"`
}

func (h *HTTPHandler) AddOrUpdate(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserIDFrom(r)
	if userID == "" {
		httpx.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	var req upsertReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}

	if err := h.service.Upsert(r.Context(), userID, req.ISBN, req.Status); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.JSONError(w, http.StatusNotFound, "NOT_FOUND", "Book not found", nil)
			return
		}
		httpx.JSONError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	httpx.JSONSuccessNoContent(w)
}

func (h *HTTPHandler) ListByStatus(w http.ResponseWriter, r *http.Request) {
	// Pattern: /users/{id}/{status}
	userID := r.PathValue("id")
	status := strings.ToUpper(r.PathValue("status"))

	if userID == "" || status == "" {
		// Fallback for old manual parsing if needed
		path := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			userID = parts[1]
		}
		if len(parts) >= 3 {
			status = strings.ToUpper(parts[2])
		}
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	books, total, err := h.service.List(r.Context(), userID, status, limit, offset)
	if err != nil {
		httpx.JSONError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	httpx.JSONSuccess(w, books, map[string]any{
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
