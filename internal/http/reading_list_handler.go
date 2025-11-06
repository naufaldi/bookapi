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
	ISBN string `json:"isbn"`
}

func (handler *ReadingListHandler) AddOrUpdateReadingListItem(responseWriter http.ResponseWriter, request *http.Request) {
	pathUserID, listName, ok := parseReadingListPath(request.URL.Path)

	if !ok {
		http.NotFound(responseWriter, request)
		return
	}
	if !isSelfOrAdmin(request, pathUserID) {
		http.Error(responseWriter, "forbidden", http.StatusForbidden)
		return
	}

	var input addReadingListRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil || strings.TrimSpace(input.ISBN) == "" {
		http.Error(responseWriter, "bad request", http.StatusBadRequest)
		return
	}

	status := statusFromListName(listName)
	if status == "" {
		http.Error(responseWriter, "bad request", http.StatusBadRequest)
		return
	}

	err := handler.readingListRepository.UpsertReadingListItem(request.Context(), pathUserID, input.ISBN, status)
	if err != nil {
		http.Error(responseWriter, "server error", http.StatusInternalServerError)
		return
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(map[string]any{"message": "book added to reading list"})
}

func (handler *ReadingListHandler) ListReadingListByStatus(responseWriter http.ResponseWriter, request *http.Request) {
	pathUserID, listName, ok := parseReadingListPath(request.URL.Path)
	if !ok {
		http.NotFound(responseWriter, request)
		return
	}
	if !isSelfOrAdmin(request, pathUserID) {
		http.Error(responseWriter, "forbidden", http.StatusForbidden)
		return
	}
	page, _ := strconv.Atoi(request.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(request.URL.Query().Get("page_size"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	status := statusFromListName(listName)
	items, total, err := handler.readingListRepository.ListReadingListByStatus(request.Context(), pathUserID, status, pageSize, offset)
	if err != nil {
		http.Error(responseWriter, "server error", http.StatusInternalServerError)
		return
	}
	totalPages := (total + pageSize - 1) / pageSize

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(map[string]any{
		"data": items,
		"meta": map[string]any{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
			"status":      status,
		},
	})
}
