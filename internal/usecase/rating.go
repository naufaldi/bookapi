package usecase

import "context"

type RatingRepository interface {
	CreateOrUpdateRating(ctx context.Context, userID string, isbn string, star int) error
	GetUserRating(ctx context.Context, userID string, isbn string) (int, error)
	GetBookRating(ctx context.Context, isbn string) (average float64, count int, err error)
}
