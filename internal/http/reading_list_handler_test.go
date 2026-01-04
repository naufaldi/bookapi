package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bookapi/internal/entity"
	"bookapi/internal/store/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestReadingListHandler_AddOrUpdateReadingListItem(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockReadingListRepository(ctrl)
	handler := NewReadingListHandler(mockRepo)

	tests := []struct {
		name           string
		path           string
		body           map[string]string
		setupMock      func()
		expectedStatus int
	}{
		{
			name: "success - add to wishlist",
			path: "/users/test-user-id/wishlist",
			body: map[string]string{"isbn": "978-0-123456-78-9"},
			setupMock: func() {
				mockRepo.EXPECT().
					UpsertReadingListItem(gomock.Any(), "test-user-id", "978-0-123456-78-9", "WISHLIST").
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "success - add to reading",
			path: "/users/test-user-id/reading",
			body: map[string]string{"isbn": "978-0-123456-78-9"},
			setupMock: func() {
				mockRepo.EXPECT().
					UpsertReadingListItem(gomock.Any(), "test-user-id", "978-0-123456-78-9", "READING").
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "success - add to finished",
			path: "/users/test-user-id/finished",
			body: map[string]string{"isbn": "978-0-123456-78-9"},
			setupMock: func() {
				mockRepo.EXPECT().
					UpsertReadingListItem(gomock.Any(), "test-user-id", "978-0-123456-78-9", "FINISHED").
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found - invalid path",
			path:           "/users/test-user-id/invalid",
			body:           nil,
			setupMock:      func() {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "forbidden - wrong user",
			path:           "/users/other-user/wishlist",
			body:           map[string]string{"isbn": "978-0-123456-78-9"},
			setupMock:      func() {},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "bad request - empty ISBN",
			path:           "/users/test-user-id/wishlist",
			body:           map[string]string{"isbn": ""},
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, tt.path, nil)
			if tt.body != nil {
				body, _ := json.Marshal(tt.body)
				r = httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(body))
				r.Header.Set("Content-Type", "application/json")
			}

			// Set user ID and role in context
			ctx := context.WithValue(r.Context(), userIDKey, "test-user-id")
			ctx = context.WithValue(ctx, roleKey, "USER")
			r = r.WithContext(ctx)

			handler.AddOrUpdateReadingListItem(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestReadingListHandler_ListReadingListByStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockReadingListRepository(ctrl)
	handler := NewReadingListHandler(mockRepo)

	tests := []struct {
		name           string
		path           string
		setupMock      func() ([]entity.Book, int)
		expectedStatus int
	}{
		{
			name: "success - list wishlist",
			path: "/users/test-user-id/wishlist",
			setupMock: func() ([]entity.Book, int) {
				books := []entity.Book{TestBook}
				mockRepo.EXPECT().
					ListReadingListByStatus(gomock.Any(), "test-user-id", "WISHLIST", 20, 0).
					Return(books, 1, nil)
				return books, 1
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "success - list reading",
			path: "/users/test-user-id/reading",
			setupMock: func() ([]entity.Book, int) {
				mockRepo.EXPECT().
					ListReadingListByStatus(gomock.Any(), "test-user-id", "READING", 20, 0).
					Return([]entity.Book{}, 0, nil)
				return []entity.Book{}, 0
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "success - list finished",
			path: "/users/test-user-id/finished",
			setupMock: func() ([]entity.Book, int) {
				mockRepo.EXPECT().
					ListReadingListByStatus(gomock.Any(), "test-user-id", "FINISHED", 20, 0).
					Return([]entity.Book{}, 0, nil)
				return []entity.Book{}, 0
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found - invalid path",
			path:           "/users/test-user-id/invalid",
			setupMock:      func() ([]entity.Book, int) { return nil, 0 },
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "forbidden - wrong user",
			path:           "/users/other-user/wishlist",
			setupMock:      func() ([]entity.Book, int) { return nil, 0 },
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)

			// Set user ID and role in context
			ctx := context.WithValue(r.Context(), userIDKey, "test-user-id")
			ctx = context.WithValue(ctx, roleKey, "USER")
			r = r.WithContext(ctx)

			handler.ListReadingListByStatus(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				assert.Contains(t, w.Body.String(), "data")
				assert.Contains(t, w.Body.String(), "meta")
			}
		})
	}
}
