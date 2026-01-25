package catalog

import (
	"bookapi/internal/httpx"
	"net/http"
	"strconv"
)

type HTTPHandler struct {
	svc *Service
}

func NewHTTPHandler(svc *Service) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// Search handles GET /v1/catalog/search
// @Summary Search master catalog
// @Description Search the global master catalog (Open Library data)
// @Tags catalog
// @Accept json
// @Produce json
// @Param q query string false "Search query"
// @Param publisher query string false "Filter by publisher"
// @Param language query string false "Filter by language"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Success 200 {object} httpx.SuccessResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/catalog/search [get]
func (h *HTTPHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(query.Get("page_size"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	q := SearchQuery{
		Q:         query.Get("q"),
		Publisher: query.Get("publisher"),
		Language:  query.Get("language"),
		Limit:     pageSize,
		Offset:    (page - 1) * pageSize,
	}

	books, total, err := h.svc.Search(r.Context(), q)
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

// GetByISBN handles GET /v1/catalog/books/{isbn}
// @Summary Get catalog book by ISBN
// @Description Retrieve master metadata for a book by its ISBN
// @Tags catalog
// @Accept json
// @Produce json
// @Param isbn path string true "Book ISBN"
// @Success 200 {object} httpx.SuccessResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/catalog/books/{isbn} [get]
func (h *HTTPHandler) GetByISBN(w http.ResponseWriter, r *http.Request) {
	isbn := r.PathValue("isbn")
	if isbn == "" {
		httpx.JSONError(w, r, http.StatusBadRequest, "BAD_REQUEST", "ISBN is required", nil)
		return
	}

	book, err := h.svc.GetByISBN(r.Context(), isbn)
	if err != nil {
		httpx.JSONError(w, r, http.StatusNotFound, "NOT_FOUND", "Book not found in catalog", nil)
		return
	}

	httpx.JSONSuccess(w, r, book, nil)
}
