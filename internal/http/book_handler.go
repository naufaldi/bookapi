package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"bookapi/internal/usecase"
)

type BookHandler struct {
	repo usecase.BookRepository
}

func NewBookHandler(repo usecase.BookRepository) *BookHandler {
	return &BookHandler{repo: repo}
}

// @Summary List books
// @Description Get all books with advanced filters, full-text search and pagination
// @Tags books
// @Accept json
// @Produce json
// @Param genre query string false "Filter by single genre"
// @Param genres query string false "Filter by multiple genres (comma-separated)"
// @Param publisher query string false "Filter by publisher"
// @Param q query string false "Legacy search query (ILIKE)"
// @Param search query string false "Full-text search query"
// @Param min_rating query number false "Minimum average rating"
// @Param year_from query int false "Publication year from"
// @Param year_to query int false "Publication year to"
// @Param language query string false "Filter by language code (e.g. en, id)"
// @Param sort query string false "Sort by (title, created_at, rating, year, relevance)"
// @Param desc query bool false "Sort in descending order"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /books [get]
func (h *BookHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := r.URL.Query()

	// Build ListParams from query parameters
	params := usecase.ListParams{
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

	//pagination
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

	books, total, err := h.repo.List(ctx, params)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	JSONSuccess(w, books, map[string]interface{}{
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": (total + pageSize - 1) / pageSize,
	})
}

// @Summary Get book by ISBN
// @Description Get a single book's details by ISBN
// @Tags books
// @Produce json
// @Param isbn path string true "Book ISBN"
// @Success 200 {object} SuccessResponse
// @Failure 404 {object} ErrorResponse
// @Router /books/{isbn} [get]
func (h *BookHandler) GetByISBN(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// crude path param extraction with net/http's ServeMux
	// /books/{isbn}
	const prefix = "/books/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	isbn := strings.TrimPrefix(r.URL.Path, prefix)
	if isbn == "" || strings.Contains(isbn, "/") {
		http.NotFound(w, r)
		return
	}
	book, err := h.repo.GetByISBN(ctx, isbn)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrNotFound):
			JSONError(w, http.StatusNotFound, "NOT_FOUND", "ISBN not found", nil)
		default:
			JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		}
		return

	}
	JSONSuccess(w, book, nil)
}
