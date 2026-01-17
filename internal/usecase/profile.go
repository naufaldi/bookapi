package usecase

import (
	"bookapi/internal/entity"
	"context"
	"fmt"
	"net/url"
	"strings"
)

type ProfileStats struct {
	BooksRead    int     `json:"books_read"`
	RatingsCount int     `json:"ratings_count"`
	AverageRating float64 `json:"average_rating"`
}

type ProfileWithStats struct {
	User  entity.User  `json:"user"`
	Stats ProfileStats `json:"stats"`
}

type ProfileUsecase struct {
	userRepo        UserRepository
	ratingRepo      RatingRepository
	readingListRepo ReadingListRepository
}

func NewProfileUsecase(userRepo UserRepository, ratingRepo RatingRepository, readingListRepo ReadingListRepository) *ProfileUsecase {
	return &ProfileUsecase{
		userRepo:        userRepo,
		ratingRepo:      ratingRepo,
		readingListRepo: readingListRepo,
	}
}

func (u *ProfileUsecase) GetOwnProfile(ctx context.Context, userID string) (ProfileWithStats, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ProfileWithStats{}, err
	}

	stats, err := u.computeStats(ctx, userID)
	if err != nil {
		return ProfileWithStats{}, err
	}

	return ProfileWithStats{
		User:  user,
		Stats: stats,
	}, nil
}

func (u *ProfileUsecase) GetPublicProfile(ctx context.Context, userID string) (ProfileWithStats, error) {
	user, err := u.userRepo.GetPublicProfile(ctx, userID)
	if err != nil {
		return ProfileWithStats{}, err
	}

	if !user.IsPublic {
		return ProfileWithStats{}, ErrNotFound
	}

	stats, err := u.computeStats(ctx, userID)
	if err != nil {
		return ProfileWithStats{}, err
	}

	return ProfileWithStats{
		User:  user,
		Stats: stats,
	}, nil
}

func (u *ProfileUsecase) UpdateProfile(ctx context.Context, userID string, updates map[string]interface{}) (ProfileWithStats, error) {
	// Normalize and validate updates
	if username, ok := updates["username"].(string); ok {
		updates["username"] = strings.TrimSpace(username)
		if len(updates["username"].(string)) < 3 {
			return ProfileWithStats{}, fmt.Errorf("username too short")
		}
	}

	if website, ok := updates["website"].(string); ok && website != "" {
		if _, err := url.ParseRequestURI(website); err != nil {
			return ProfileWithStats{}, fmt.Errorf("invalid website URL")
		}
	}

	err := u.userRepo.UpdateProfile(ctx, userID, updates)
	if err != nil {
		return ProfileWithStats{}, err
	}

	return u.GetOwnProfile(ctx, userID)
}

func (u *ProfileUsecase) computeStats(ctx context.Context, userID string) (ProfileStats, error) {
	// Books read count
	_, finishedCount, err := u.readingListRepo.ListReadingListByStatus(ctx, userID, entity.ReadingListStatusFinished, 1, 0)
	if err != nil {
		return ProfileStats{}, err
	}

	// Rating stats
	avgRating, ratingsCount, err := u.ratingRepo.GetUserRatingStats(ctx, userID)
	if err != nil {
		return ProfileStats{}, err
	}

	return ProfileStats{
		BooksRead:    finishedCount,
		RatingsCount: ratingsCount,
		AverageRating: avgRating,
	}, nil
}
