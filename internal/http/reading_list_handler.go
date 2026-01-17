package http

import (
	"bookapi/internal/entity"
	"bookapi/internal/usecase"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type ReadingListHandler struct {
	readingListRepository usecase.ReadingListRepository
}

func NewReadingListHandler(repo usecase.ReadingListRepository) *ReadingListHandler {
	return &ReadingListHandler{
		readingListRepository: repo,
	}
}

func parseReadingListPath(path string) (userID string, listName string, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 3 {
		return "", "", false
	}
	if parts[0] != "users" {
		return "", "", false
	}
	list := strings.ToUpper(parts[len(parts)-1])
	switch list {
	case "WISHLIST", "READING", "FINISHED":
		return parts[1], list, true
	default:
		return "", "", false
	}

}

func statusFromListName(listName string) string {
	switch strings.ToUpper(listName) {
	case "WISHLIST":
		return entity.ReadingListStatusWishlist
	case "READING":
		return entity.ReadingListStatusReading
	case "FINISHED":
		return entity.ReadingListStatusFinished
	default:
		return ""
	}
}

func isSelfOrAdmin(request *http.Request, pathUserId string) bool {
	authenticatedUserId := UserIDFrom(request)
	role := RoleFrom(request)
	return authenticatedUserId == pathUserId || role == "ADMIN"
}

type addReadingListRequest struct {
	ISBN string `json:"isbn" validate:"required,isbn"`
}

// @Summary Add or update reading list item
// @Description Add a book to a user's reading list (WISHLIST, READING, FINISHED)
// @Tags reading-lists
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "User ID"
// @Param list path string true "List name (WISHLIST, READING, FINISHED)"
// @Param item body addReadingListRequest true "Book ISBN"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /users/{id}/{list} [post]
func (h *ReadingListHandler) AddOrUpdateReadingListItem(w http.ResponseWriter, r *http.Request) {
	pathUserID, listName, ok := parseReadingListPath(r.URL.Path)

	if !ok {
		http.NotFound(w, r)
		return
	}
	if !isSelfOrAdmin(r, pathUserID) {
		JSONError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden", nil)
		return
	}

	var input addReadingListRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", nil)
		return
	}

	if validationErrors := ValidateStruct(input); len(validationErrors) > 0 {
		JSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input", validationErrors)
		return
	}

	status := statusFromListName(listName)
	if status == "" {
		JSONError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid list name", nil)
		return
	}

	err := h.readingListRepository.UpsertReadingListItem(r.Context(), pathUserID, input.ISBN, status)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	JSONSuccess(w, nil, map[string]string{"message": "Book added to reading list"})
}

// @Summary List reading list items
// @Description Get a user's reading list items by status
// @Tags reading-lists
// @Produce json
// @Security Bearer
// @Param id path string true "User ID"
// @Param list path string true "List name (WISHLIST, READING, FINISHED)"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Success 200 {object} SuccessResponse
// @Failure 403 {object} ErrorResponse
// @Router /users/{id}/{list} [get]
func (h *ReadingListHandler) ListReadingListByStatus(w http.ResponseWriter, r *http.Request) {
	pathUserID, listName, ok := parseReadingListPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !isSelfOrAdmin(r, pathUserID) {
		JSONError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden", nil)
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	status := statusFromListName(listName)
	items, total, err := h.readingListRepository.ListReadingListByStatus(r.Context(), pathUserID, status, pageSize, offset)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}
	totalPages := (total + pageSize - 1) / pageSize

	JSONSuccess(w, items, map[string]interface{}{
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"status":      status,
	})
}
