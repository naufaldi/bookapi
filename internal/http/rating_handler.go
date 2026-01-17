package http

import (
	"bookapi/internal/usecase"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type RatingHandler struct {
	ratingRepo usecase.RatingRepository
}

func NewRatingHandler(ratingRepo usecase.RatingRepository) *RatingHandler {
	return &RatingHandler{ratingRepo: ratingRepo}
}

func parseBookISBNAndAction(path string) (isbn, action string, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) >= 3 && parts[0] == "books" {
		return parts[1], parts[2], true
	}
	return "", "", false
}

type createRatingRequest struct {
	Star int `json:"star" validate:"required,gte=1,lte=5"`
}

// @Summary Create or update rating
// @Description Rate a book (1-5 stars)
// @Tags ratings
// @Accept json
// @Produce json
// @Security Bearer
// @Param isbn path string true "Book ISBN"
// @Param rating body createRatingRequest true "Rating value"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /books/{isbn}/rating [post]
func (h *RatingHandler) CreateRating(w http.ResponseWriter, r *http.Request) {
	isbn, action, ok := parseBookISBNAndAction(r.URL.Path)
	if !ok || action != "rating" {
		http.NotFound(w, r)
		return
	}

	userID := UserIDFrom(r)
	if userID == "" {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	var req createRatingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}

	if validationErrors := ValidateStruct(req); len(validationErrors) > 0 {
		JSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input", validationErrors)
		return
	}

	if err := h.ratingRepo.CreateOrUpdateRating(r.Context(), userID, isbn, req.Star); err != nil {
		switch {
		case errors.Is(err, usecase.ErrNotFound):
			JSONError(w, http.StatusNotFound, "NOT_FOUND", "Book not found", nil)
		default:
			JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		}
		return
	}

	JSONSuccess(w, map[string]any{
		"isbn":    isbn,
		"user_id": userID,
		"star":    req.Star,
	}, map[string]string{"message": "Rating saved"})
}

// @Summary Get book rating
// @Description Get average rating and total count for a book
// @Tags ratings
// @Produce json
// @Param isbn path string true "Book ISBN"
// @Success 200 {object} SuccessResponse
// @Failure 404 {object} ErrorResponse
// @Router /books/{isbn}/rating [get]
func (h *RatingHandler) GetRating(w http.ResponseWriter, r *http.Request) {
	isbn, action, ok := parseBookISBNAndAction(r.URL.Path)
	if !ok || action != "rating" {
		http.NotFound(w, r)
		return
	}

	var yourRating *int
	if userID := UserIDFrom(r); userID != "" {
		if star, err := h.ratingRepo.GetUserRating(r.Context(), userID, isbn); err == nil {
			yourRating = &star
		}
	}

	average, count, err := h.ratingRepo.GetBookRating(r.Context(), isbn)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	JSONSuccess(w, map[string]any{
		"average_rating": average,
		"total_ratings":  count,
		"your_rating":    yourRating,
	}, nil)
}
