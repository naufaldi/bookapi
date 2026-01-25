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

// CreateRating handles POST /books/{isbn}/rating
// @Summary Create or update book rating
// @Description Rate a book (1-5 stars)
// @Tags ratings
// @Accept json
// @Produce json
// @Security Bearer
// @Param isbn path string true "Book ISBN"
// @Param request body createRatingReq true "Rating request"
// @Success 204 "No Content"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /books/{isbn}/rating [post]
func (h *HTTPHandler) CreateRating(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserIDFrom(r)
	if userID == "" {
		httpx.JSONError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	isbn := r.PathValue("isbn")
	if isbn == "" {
		httpx.JSONError(w, r, http.StatusBadRequest, "BAD_REQUEST", "Invalid ISBN", nil)
		return
	}

	var req createRatingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSONError(w, r, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}

	if validationErrors := httpx.ValidateStruct(req); len(validationErrors) > 0 {
		httpx.JSONError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input", validationErrors)
		return
	}

	if err := h.service.CreateOrUpdate(r.Context(), userID, isbn, req.Star); err != nil {
		if errors.Is(err, ErrInternalNotFound) {
			httpx.JSONError(w, r, http.StatusNotFound, "NOT_FOUND", "Book not found", nil)
			return
		}
		httpx.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccessNoContent(w)
}

// GetRating handles GET /books/{isbn}/rating
// @Summary Get book rating
// @Description Get average rating and total count for a book
// @Tags ratings
// @Accept json
// @Produce json
// @Param isbn path string true "Book ISBN"
// @Success 200 {object} httpx.SuccessResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /books/{isbn}/rating [get]
func (h *HTTPHandler) GetRating(w http.ResponseWriter, r *http.Request) {
	isbn := r.PathValue("isbn")
	if isbn == "" {
		httpx.JSONError(w, r, http.StatusBadRequest, "BAD_REQUEST", "Invalid ISBN", nil)
		return
	}

	average, count, err := h.service.GetBookRating(r.Context(), isbn)
	if err != nil {
		httpx.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccess(w, r, map[string]any{
		"average_rating": average,
		"ratings_count":  count,
	}, nil)
}
