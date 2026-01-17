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
	if len(parts) == 3 && parts[0] == "books" {
		return parts[1], parts[2], true
	}
	return "", "", false
}

type createRatingRequest struct {
	Star int `json:"star" validate:"required,gte=1,lte=5"`
}

func (handler *RatingHandler) CreateRating(responseWriter http.ResponseWriter, request *http.Request) {
	isbn, action, ok := parseBookISBNAndAction(request.URL.Path)
	if !ok || action != "rating" {
		http.NotFound(responseWriter, request)
		return
	}

	userID := UserIDFrom(request)
	if userID == "" {
		http.Error(responseWriter, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body createRatingRequest
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		http.Error(responseWriter, "bad request", http.StatusBadRequest)
		return
	}
	if validationErrors := ValidateStruct(body); len(validationErrors) > 0 {
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(map[string]any{
			"success": false,
			"error": map[string]any{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid input",
				"details": validationErrors,
			},
		})
		return
	}

	if err := handler.ratingRepo.CreateOrUpdateRating(request.Context(), userID, isbn, body.Star); err != nil {
		switch {
		case errors.Is(err, usecase.ErrNotFound):
			http.Error(responseWriter, "book not found", http.StatusNotFound)
			return
		default:
			http.Error(responseWriter, "internal server error", http.StatusInternalServerError)
			return
		}
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(map[string]any{
		"message": "Rating saved",
		"data": map[string]any{
			"isbn":    isbn,
			"user_id": userID,
			"star":    body.Star,
		},
	})
}

func (handler *RatingHandler) GetRating(responseWriter http.ResponseWriter, request *http.Request) {
	isbn, action, ok := parseBookISBNAndAction(request.URL.Path)
	if !ok || action != "rating" {
		http.NotFound(responseWriter, request)
		return
	}
	var yourRating *int
	if userID := UserIDFrom(request); userID != "" {
		if star, err := handler.ratingRepo.GetUserRating(request.Context(), userID, isbn); err == nil {
			yourRating = &star
		}
	}
	average, count, err := handler.ratingRepo.GetBookRating(request.Context(), isbn)
	if err != nil {
		http.Error(responseWriter, "server error", http.StatusInternalServerError)
		return
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(map[string]any{
		"data": map[string]any{
			"average_rating": average,
			"total_ratings":  count,
			"your_rating":    yourRating,
		},
	})
}
