package profile

import (
	"bookapi/internal/rating"
	"bookapi/internal/readinglist"
	"bookapi/internal/user"
	"context"
	"fmt"
	"net/url"
	"strings"
)

type Service struct {
	userService        *user.Service
	ratingService      *rating.Service
	readingListService *readinglist.Service
}

func NewService(userService *user.Service, ratingService *rating.Service, readingListService *readinglist.Service) *Service {
	return &Service{
		userService:        userService,
		ratingService:      ratingService,
		readingListService: readingListService,
	}
}

func (s *Service) GetOwnProfile(ctx context.Context, userID string) (Profile, error) {
	u, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		return Profile{}, err
	}

	stats, err := s.computeStats(ctx, userID)
	if err != nil {
		return Profile{}, err
	}

	return Profile{
		User:  u,
		Stats: stats,
	}, nil
}

func (s *Service) GetPublicProfile(ctx context.Context, userID string) (Profile, error) {
	// For public profile, we use the user service to get public info
	// Actually, user service doesn't have a GetPublicProfile method yet, let's add it or use GetByID and check IsPublic
	u, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		return Profile{}, err
	}

	if !u.IsPublic {
		return Profile{}, user.ErrNotFound
	}

	stats, err := s.computeStats(ctx, userID)
	if err != nil {
		return Profile{}, err
	}

	return Profile{
		User:  u,
		Stats: stats,
	}, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, cmd UpdateCommand) (Profile, error) {
	updates := cmd.ToMap()

	if username, ok := updates["username"].(string); ok {
		updates["username"] = strings.TrimSpace(username)
		if len(updates["username"].(string)) < 3 {
			return Profile{}, fmt.Errorf("username too short")
		}
	}

	if website, ok := updates["website"].(string); ok && website != "" {
		if _, err := url.ParseRequestURI(website); err != nil {
			return Profile{}, fmt.Errorf("invalid website URL")
		}
	}

	if err := s.userService.UpdateProfile(ctx, userID, updates); err != nil {
		return Profile{}, err
	}

	return s.GetOwnProfile(ctx, userID)
}

func (s *Service) computeStats(ctx context.Context, userID string) (Stats, error) {
	_, finishedCount, err := s.readingListService.List(ctx, userID, readinglist.StatusFinished, 1, 0)
	if err != nil {
		return Stats{}, err
	}

	avgRating, ratingsCount, err := s.ratingService.GetUserRatingStats(ctx, userID)
	if err != nil {
		return Stats{}, err
	}

	return Stats{
		BooksRead:     finishedCount,
		RatingsCount:  ratingsCount,
		AverageRating: avgRating,
	}, nil
}
