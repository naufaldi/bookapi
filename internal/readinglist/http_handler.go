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

// AddOrUpdate handles POST /users/readinglist
// @Summary Add or update reading list entry
// @Description Add a book to reading list or update its status (WISHLIST, READING, FINISHED)
// @Tags reading-lists
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body upsertReq true "Reading list request"
// @Success 204 "No Content"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /users/readinglist [post]
func (h *HTTPHandler) AddOrUpdate(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserIDFrom(r)
	if userID == "" {
		httpx.JSONError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	var req upsertReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSONError(w, r, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}

	if err := h.service.Upsert(r.Context(), userID, req.ISBN, req.Status); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.JSONError(w, r, http.StatusNotFound, "NOT_FOUND", "Book not found", nil)
			return
		}
		httpx.JSONError(w, r, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	httpx.JSONSuccessNoContent(w)
}

// ListByStatus handles GET /users/{id}/{status}
// @Summary List books by status
// @Description Get a user's reading list filtered by status (WISHLIST, READING, FINISHED)
// @Tags reading-lists
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param status path string true "Reading list status" Enums(WISHLIST, READING, FINISHED)
// @Param limit query int false "Limit results" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} httpx.SuccessResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /users/{id}/{status} [get]
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
		httpx.JSONError(w, r, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	httpx.JSONSuccess(w, r, books, map[string]any{
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
