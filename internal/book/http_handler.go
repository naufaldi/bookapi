package book

import (
	"bookapi/internal/httpx"
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

// List handles GET /books
// @Summary List books
// @Description Get a paginated list of books with optional filtering and search
// @Tags books
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Param q query string false "Simple search query"
// @Param search query string false "Full-text search query"
// @Param genre query string false "Filter by genre"
// @Param genres query string false "Filter by multiple genres (comma-separated)"
// @Param publisher query string false "Filter by publisher"
// @Param min_rating query number false "Minimum rating (0-5)"
// @Param year_from query int false "Filter by publication year from"
// @Param year_to query int false "Filter by publication year to"
// @Param language query string false "Filter by language"
// @Param sort query string false "Sort field"
// @Param desc query boolean false "Sort descending" default(false)
// @Success 200 {object} httpx.SuccessResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /books [get]
func (h *HTTPHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	params := Query{
		Genre:     query.Get("genre"),
		Publisher: query.Get("publisher"),
		Q:         query.Get("q"),
		Search:    query.Get("search"),
		Sort:      query.Get("sort"),
		Desc:      query.Get("desc") == "true",
		Language:  query.Get("language"),
	}

	if genres := query.Get("genres"); genres != "" {
		params.Genres = strings.Split(genres, ",")
	}

	if minRatingStr := query.Get("min_rating"); minRatingStr != "" {
		if val, err := strconv.ParseFloat(minRatingStr, 64); err == nil {
			params.MinRating = &val
		}
	}

	if yearFromStr := query.Get("year_from"); yearFromStr != "" {
		if val, err := strconv.Atoi(yearFromStr); err == nil {
			params.YearFrom = &val
		}
	}

	if yearToStr := query.Get("year_to"); yearToStr != "" {
		if val, err := strconv.Atoi(yearToStr); err == nil {
			params.YearTo = &val
		}
	}

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(query.Get("page_size"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	params.Limit = pageSize
	params.Offset = (page - 1) * pageSize

	books, total, err := h.service.List(r.Context(), params)
	if err != nil {
		httpx.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	httpx.JSONSuccess(w, r, books, map[string]any{
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": (total + pageSize - 1) / pageSize,
	})
}

// GetByISBN handles GET /books/{isbn}
// @Summary Get book by ISBN
// @Description Retrieve a book by its ISBN
// @Tags books
// @Accept json
// @Produce json
// @Param isbn path string true "Book ISBN"
// @Success 200 {object} httpx.SuccessResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /books/{isbn} [get]
func (h *HTTPHandler) GetByISBN(w http.ResponseWriter, r *http.Request) {
	// Go 1.22+ routing: use r.PathValue
	isbn := r.PathValue("isbn")
	if isbn == "" {
		// Fallback for old routing if needed, but we aim for modern
		const prefix = "/books/"
		isbn = strings.TrimPrefix(r.URL.Path, prefix)
	}

	if isbn == "" || strings.Contains(isbn, "/") {
		http.NotFound(w, r)
		return
	}

	book, err := h.service.GetByISBN(r.Context(), isbn)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.JSONError(w, r, http.StatusNotFound, "NOT_FOUND", "ISBN not found", nil)
			return
		}
		httpx.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}
	httpx.JSONSuccess(w, r, book, nil)
}
