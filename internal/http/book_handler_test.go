package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bookapi/internal/entity"
	"bookapi/internal/store/mocks"
	"bookapi/internal/usecase"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var TestBook = entity.Book{
	ID:          "test-book-id-789",
	ISBN:        "978-0-123456-78-9",
	Title:       "Test Book Title",
	Genre:       "Fiction",
	Publisher:   "Test Publisher",
	Description: "A test book description",
	CreatedAt:   time.Now(),
	UpdatedAt:   time.Now(),
}

func TestBookHandler_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockBookRepository(ctrl)
	handler := NewBookHandler(mockRepo)

	tests := []struct {
		name           string
		queryParams    string
		setupMock      func()
		expectedStatus int
	}{
		{
			name:        "success - empty list",
			queryParams: "?page=1&page_size=20",
			setupMock: func() {
				mockRepo.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return([]entity.Book{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "success - with books",
			queryParams: "?page=1&page_size=20",
			setupMock: func() {
				books := []entity.Book{TestBook}
				mockRepo.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(books, 1, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "success - with genre filter",
			queryParams: "?genre=Fiction&page=1&page_size=20",
			setupMock: func() {
				mockRepo.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return([]entity.Book{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "success - with publisher filter",
			queryParams: "?publisher=Test+Publisher&page=1&page_size=20",
			setupMock: func() {
				mockRepo.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return([]entity.Book{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "success - with search query",
			queryParams: "?q=test&page=1&page_size=20",
			setupMock: func() {
				mockRepo.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return([]entity.Book{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "server error",
			queryParams: "?page=1&page_size=20",
			setupMock: func() {
				mockRepo.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil, 0, context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/books"+tt.queryParams, nil)

			handler.List(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestBookHandler_GetByISBN(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockBookRepository(ctrl)
	handler := NewBookHandler(mockRepo)

	tests := []struct {
		name           string
		path           string
		setupMock      func()
		expectedStatus int
	}{
		{
			name: "success - book found",
			path: "/books/978-0-123456-78-9",
			setupMock: func() {
				mockRepo.EXPECT().
					GetByISBN(gomock.Any(), "978-0-123456-78-9").
					Return(TestBook, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "not found - invalid ISBN",
			path: "/books/",
			setupMock: func() {
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "not found - book not in DB",
			path: "/books/999-9-999999-99-9",
			setupMock: func() {
				mockRepo.EXPECT().
					GetByISBN(gomock.Any(), "999-9-999999-99-9").
					Return(entity.Book{}, usecase.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "server error",
			path: "/books/978-0-123456-78-9",
			setupMock: func() {
				mockRepo.EXPECT().
					GetByISBN(gomock.Any(), "978-0-123456-78-9").
					Return(entity.Book{}, context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)

			handler.GetByISBN(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
