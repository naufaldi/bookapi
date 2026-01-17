package http

import (
	"bookapi/internal/entity"
	"bookapi/internal/store/mocks"
	"bookapi/internal/usecase"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestProfileHandler_GetOwnProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockRatingRepo := mocks.NewMockRatingRepository(ctrl)
	mockReadingListRepo := mocks.NewMockReadingListRepository(ctrl)

	uc := usecase.NewProfileUsecase(mockUserRepo, mockRatingRepo, mockReadingListRepo)
	handler := NewProfileHandler(uc)

	userID := "user-123"
	ctx := context.WithValue(context.Background(), userIDKey, userID)

	t.Run("success", func(t *testing.T) {
		user := entity.User{ID: userID, Username: "testuser"}
		mockUserRepo.EXPECT().GetByID(gomock.Any(), userID).Return(user, nil)
		mockReadingListRepo.EXPECT().ListReadingListByStatus(gomock.Any(), userID, entity.ReadingListStatusFinished, 1, 0).Return([]entity.Book{}, 5, nil)
		mockRatingRepo.EXPECT().GetUserRatingStats(gomock.Any(), userID).Return(4.5, 10, nil)

		req := httptest.NewRequest(http.MethodGet, "/me/profile", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetOwnProfile(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp SuccessResponse
		json.NewDecoder(w.Body).Decode(&resp)
		assert.True(t, resp.Success)
		data := resp.Data.(map[string]interface{})
		assert.Equal(t, userID, data["user"].(map[string]interface{})["id"])
		stats := data["stats"].(map[string]interface{})
		assert.Equal(t, float64(5), stats["books_read"])
	})

	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/me/profile", nil) // no context
		w := httptest.NewRecorder()

		handler.GetOwnProfile(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestProfileHandler_GetPublicProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockRatingRepo := mocks.NewMockRatingRepository(ctrl)
	mockReadingListRepo := mocks.NewMockReadingListRepository(ctrl)

	uc := usecase.NewProfileUsecase(mockUserRepo, mockRatingRepo, mockReadingListRepo)
	handler := NewProfileHandler(uc)

	userID := "user-123"

	t.Run("success", func(t *testing.T) {
		user := entity.User{ID: userID, Username: "testuser", IsPublic: true}
		mockUserRepo.EXPECT().GetPublicProfile(gomock.Any(), userID).Return(user, nil)
		mockReadingListRepo.EXPECT().ListReadingListByStatus(gomock.Any(), userID, entity.ReadingListStatusFinished, 1, 0).Return([]entity.Book{}, 5, nil)
		mockRatingRepo.EXPECT().GetUserRatingStats(gomock.Any(), userID).Return(4.5, 10, nil)

		req := httptest.NewRequest(http.MethodGet, "/users/user-123/profile", nil)
		w := httptest.NewRecorder()

		handler.GetPublicProfile(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found or private", func(t *testing.T) {
		mockUserRepo.EXPECT().GetPublicProfile(gomock.Any(), userID).Return(entity.User{}, usecase.ErrNotFound)

		req := httptest.NewRequest(http.MethodGet, "/users/user-123/profile", nil)
		w := httptest.NewRecorder()

		handler.GetPublicProfile(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestProfileHandler_UpdateProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockRatingRepo := mocks.NewMockRatingRepository(ctrl)
	mockReadingListRepo := mocks.NewMockReadingListRepository(ctrl)

	uc := usecase.NewProfileUsecase(mockUserRepo, mockRatingRepo, mockReadingListRepo)
	handler := NewProfileHandler(uc)

	userID := "user-123"
	ctx := context.WithValue(context.Background(), userIDKey, userID)

	t.Run("success", func(t *testing.T) {
		updates := map[string]interface{}{"bio": "New bio"}
		body, _ := json.Marshal(updates)

		mockUserRepo.EXPECT().UpdateProfile(gomock.Any(), userID, updates).Return(nil)
		mockUserRepo.EXPECT().GetByID(gomock.Any(), userID).Return(entity.User{ID: userID, Bio: func(s string) *string { return &s }("New bio")}, nil)
		mockReadingListRepo.EXPECT().ListReadingListByStatus(gomock.Any(), userID, entity.ReadingListStatusFinished, 1, 0).Return([]entity.Book{}, 0, nil)
		mockRatingRepo.EXPECT().GetUserRatingStats(gomock.Any(), userID).Return(0.0, 0, nil)

		req := httptest.NewRequest(http.MethodPatch, "/me/profile", bytes.NewReader(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateProfile(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
