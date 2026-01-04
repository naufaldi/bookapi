package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bookapi/internal/store/mocks"
	"bookapi/internal/usecase"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestRatingHandler_CreateRating(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		body           map[string]int
		setupMock      func(ctrl *gomock.Controller) *mocks.MockRatingRepository
		expectedStatus int
	}{
		{
			name: "success - new rating",
			path: "/books/978-0-123456-78-9/rating",
			body: map[string]int{"star": 4},
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRatingRepository {
				mockRepo := mocks.NewMockRatingRepository(ctrl)
				mockRepo.EXPECT().
					CreateOrUpdateRating(gomock.Any(), "test-user-id", "978-0-123456-78-9", 4).
					Return(nil)
				return mockRepo
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "success - update existing rating",
			path: "/books/978-0-123456-78-9/rating",
			body: map[string]int{"star": 5},
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRatingRepository {
				mockRepo := mocks.NewMockRatingRepository(ctrl)
				mockRepo.EXPECT().
					CreateOrUpdateRating(gomock.Any(), "test-user-id", "978-0-123456-78-9", 5).
					Return(nil)
				return mockRepo
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found - invalid path",
			path:           "/books/978-0-123456-78-9/invalid",
			body:           nil,
			setupMock:      func(ctrl *gomock.Controller) *mocks.MockRatingRepository { return mocks.NewMockRatingRepository(ctrl) },
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "unauthorized - no token",
			path: "/books/978-0-123456-78-9/rating",
			body: map[string]int{"star": 4},
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRatingRepository {
				mockRepo := mocks.NewMockRatingRepository(ctrl)
				// Don't set up any expectations since we shouldn't reach the repo
				return mockRepo
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "bad request - star < 1",
			path:           "/books/978-0-123456-78-9/rating",
			body:           map[string]int{"star": 0},
			setupMock:      func(ctrl *gomock.Controller) *mocks.MockRatingRepository { return nil },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "bad request - star > 5",
			path:           "/books/978-0-123456-78-9/rating",
			body:           map[string]int{"star": 6},
			setupMock:      func(ctrl *gomock.Controller) *mocks.MockRatingRepository { return nil },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "not found - book not in DB",
			path: "/books/999-9-999999-99-9/rating",
			body: map[string]int{"star": 4},
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRatingRepository {
				mockRepo := mocks.NewMockRatingRepository(ctrl)
				mockRepo.EXPECT().
					CreateOrUpdateRating(gomock.Any(), "test-user-id", "999-9-999999-99-9", 4).
					Return(usecase.ErrNotFound)
				return mockRepo
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := tt.setupMock(ctrl)
			var handler *RatingHandler
			if mockRepo != nil {
				handler = NewRatingHandler(mockRepo)
			} else {
				handler = NewRatingHandler(nil)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, tt.path, nil)
			if tt.body != nil {
				body, _ := json.Marshal(tt.body)
				r = httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(body))
				r.Header.Set("Content-Type", "application/json")
			}

			// Only set user ID if we expect to reach the repo (not unauthorized cases)
			if tt.expectedStatus != http.StatusUnauthorized {
				ctx := context.WithValue(r.Context(), userIDKey, "test-user-id")
				r = r.WithContext(ctx)
			}

			handler.CreateRating(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRatingHandler_GetRating(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setupMock      func(ctrl *gomock.Controller) *mocks.MockRatingRepository
		expectedStatus int
	}{
		{
			name: "success - with ratings",
			path: "/books/978-0-123456-78-9/rating",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRatingRepository {
				mockRepo := mocks.NewMockRatingRepository(ctrl)
				mockRepo.EXPECT().
					GetBookRating(gomock.Any(), "978-0-123456-78-9").
					Return(float64(4.5), 10, nil)
				return mockRepo
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "success - no ratings",
			path: "/books/978-0-123456-78-9/rating",
			setupMock: func(ctrl *gomock.Controller) *mocks.MockRatingRepository {
				mockRepo := mocks.NewMockRatingRepository(ctrl)
				mockRepo.EXPECT().
					GetBookRating(gomock.Any(), "978-0-123456-78-9").
					Return(float64(0), 0, nil)
				return mockRepo
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found - invalid path",
			path:           "/books/978-0-123456-78-9/invalid",
			setupMock:      func(ctrl *gomock.Controller) *mocks.MockRatingRepository { return mocks.NewMockRatingRepository(ctrl) },
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := tt.setupMock(ctrl)
			var handler *RatingHandler
			if mockRepo != nil {
				handler = NewRatingHandler(mockRepo)
			} else {
				handler = NewRatingHandler(nil)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)

			handler.GetRating(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				assert.Contains(t, w.Body.String(), "average_rating")
				assert.Contains(t, w.Body.String(), "total_ratings")
			}
		})
	}
}
