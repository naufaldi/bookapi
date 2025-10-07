package http

import (
	"encoding/json"
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

func (h *BookHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Build ListParams from query parameters
	params := usecase.ListParams{
		Genre:     r.URL.Query().Get("genre"),
		Publisher: r.URL.Query().Get("publisher"),
		Q:         r.URL.Query().Get("q"), // search query
		Sort:      r.URL.Query().Get("sort"),
		Desc:      r.URL.Query().Get("desc") == "true",
	}

	//pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	params.Limit = pageSize
	params.Offset = (page - 1) * pageSize

	books, total, err := h.repo.List(ctx, params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "server error"})
		return
	}

	resp := map[string]interface{}{
		"data": books,
		"meta": map[string]interface{}{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_pages": (total + pageSize - 1) / pageSize, // ceiling division
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *BookHandler) GetByISBN(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// crude path param extraction with net/http's ServeMux
	// /books/{isbn}
 const prefix ="/books/"
 if !strings.HasPrefix(r.URL.Path, prefix) {
	http.NotFound(w,r)
	return 
 }
 isbn := strings.TrimPrefix(r.URL.Path, prefix)
 if isbn == "" || strings.Contains(isbn, "/") {
	http.NotFound(w,r)
	return
 }
 book, err := h.repo.GetByISBN(ctx, isbn)
 if err != nil {
	switch {
		case errors.Is(err, usecase.ErrNotFound):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "ISBN not found"})
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "server error"})
	}
	return
	
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": book,
	})
}

