package usecase_test

import (
	"bookapi/internal/entity"
	"bookapi/internal/store/mocks"
	"bookapi/internal/usecase"
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestProfileUsecase_GetPublicProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockRatingRepo := mocks.NewMockRatingRepository(ctrl)
	mockReadingListRepo := mocks.NewMockReadingListRepository(ctrl)

	uc := usecase.NewProfileUsecase(mockUserRepo, mockRatingRepo, mockReadingListRepo)
	ctx := context.Background()
	userID := "user-123"

	t.Run("success - public profile", func(t *testing.T) {
		user := entity.User{ID: userID, Username: "testuser", IsPublic: true}
		mockUserRepo.EXPECT().GetPublicProfile(ctx, userID).Return(user, nil)
		mockReadingListRepo.EXPECT().ListReadingListByStatus(ctx, userID, entity.ReadingListStatusFinished, 1, 0).Return([]entity.Book{}, 5, nil)
		mockRatingRepo.EXPECT().GetUserRatingStats(ctx, userID).Return(4.5, 10, nil)

		profile, err := uc.GetPublicProfile(ctx, userID)

		assert.NoError(t, err)
		assert.Equal(t, userID, profile.User.ID)
		assert.Equal(t, 5, profile.Stats.BooksRead)
		assert.Equal(t, 10, profile.Stats.RatingsCount)
		assert.Equal(t, 4.5, profile.Stats.AverageRating)
	})

	t.Run("error - private profile", func(t *testing.T) {
		user := entity.User{ID: userID, Username: "testuser", IsPublic: false}
		mockUserRepo.EXPECT().GetPublicProfile(ctx, userID).Return(user, nil)

		_, err := uc.GetPublicProfile(ctx, userID)

		assert.Error(t, err)
		assert.True(t, errors.Is(err, usecase.ErrNotFound))
	})

	t.Run("error - not found", func(t *testing.T) {
		mockUserRepo.EXPECT().GetPublicProfile(ctx, userID).Return(entity.User{}, usecase.ErrNotFound)

		_, err := uc.GetPublicProfile(ctx, userID)

		assert.Error(t, err)
		assert.True(t, errors.Is(err, usecase.ErrNotFound))
	})
}

func TestProfileUsecase_UpdateProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockRatingRepo := mocks.NewMockRatingRepository(ctrl)
	mockReadingListRepo := mocks.NewMockReadingListRepository(ctrl)

	uc := usecase.NewProfileUsecase(mockUserRepo, mockRatingRepo, mockReadingListRepo)
	ctx := context.Background()
	userID := "user-123"

	t.Run("success - valid update", func(t *testing.T) {
		updates := map[string]interface{}{
			"username": "updateduser",
			"website":  "https://example.com",
		}
		user := entity.User{ID: userID, Username: "updateduser", Website: func(s string) *string { return &s }("https://example.com")}

		mockUserRepo.EXPECT().UpdateProfile(ctx, userID, updates).Return(nil)
		mockUserRepo.EXPECT().GetByID(ctx, userID).Return(user, nil)
		mockReadingListRepo.EXPECT().ListReadingListByStatus(ctx, userID, entity.ReadingListStatusFinished, 1, 0).Return([]entity.Book{}, 0, nil)
		mockRatingRepo.EXPECT().GetUserRatingStats(ctx, userID).Return(0.0, 0, nil)

		profile, err := uc.UpdateProfile(ctx, userID, updates)

		assert.NoError(t, err)
		assert.Equal(t, "updateduser", profile.User.Username)
	})

	t.Run("error - invalid website", func(t *testing.T) {
		updates := map[string]interface{}{
			"website": "invalid-url",
		}

		_, err := uc.UpdateProfile(ctx, userID, updates)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid website URL")
	})

	t.Run("error - username too short", func(t *testing.T) {
		updates := map[string]interface{}{
			"username": "ab",
		}

		_, err := uc.UpdateProfile(ctx, userID, updates)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username too short")
	})
}
