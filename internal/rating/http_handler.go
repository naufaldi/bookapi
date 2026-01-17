package rating

import (
	"bookapi/internal/httpx"
	"encoding/json"
	"errors"
	"net/http"
)

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

type createRatingReq struct {
	Star int `json:"star" validate:"required,min=1,max=5"`
}

func (h *HTTPHandler) CreateRating(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserIDFrom(r)
	if userID == "" {
		httpx.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	isbn := r.PathValue("isbn")
	if isbn == "" {
		httpx.JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid ISBN", nil)
		return
	}

	var req createRatingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}

	if validationErrors := httpx.ValidateStruct(req); len(validationErrors) > 0 {
		httpx.JSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input", validationErrors)
		return
	}

	if err := h.service.CreateOrUpdate(r.Context(), userID, isbn, req.Star); err != nil {
		if errors.Is(err, ErrInternalNotFound) {
			httpx.JSONError(w, http.StatusNotFound, "NOT_FOUND", "Book not found", nil)
			return
		}
		httpx.JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccessNoContent(w)
}

func (h *HTTPHandler) GetRating(w http.ResponseWriter, r *http.Request) {
	isbn := r.PathValue("isbn")
	if isbn == "" {
		httpx.JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid ISBN", nil)
		return
	}

	average, count, err := h.service.GetBookRating(r.Context(), isbn)
	if err != nil {
		httpx.JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccess(w, map[string]any{
		"average_rating": average,
		"ratings_count":  count,
	}, nil)
}
