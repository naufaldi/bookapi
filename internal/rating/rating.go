package rating

import "context"

type Rating struct {
	UserID string `json:"user_id"`
	ISBN   string `json:"isbn"`
	Star   int    `json:"star"`
}

type Repository interface {
	CreateOrUpdateRating(ctx context.Context, userID string, isbn string, star int) error
	GetUserRating(ctx context.Context, userID string, isbn string) (int, error)
	GetBookRating(ctx context.Context, isbn string) (average float64, count int, err error)
	GetUserRatingStats(ctx context.Context, userID string) (average float64, count int, err error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateOrUpdate(ctx context.Context, userID, isbn string, star int) error {
	return s.repo.CreateOrUpdateRating(ctx, userID, isbn, star)
}

func (s *Service) GetUserRating(ctx context.Context, userID, isbn string) (int, error) {
	return s.repo.GetUserRating(ctx, userID, isbn)
}

func (s *Service) GetBookRating(ctx context.Context, isbn string) (float64, int, error) {
	return s.repo.GetBookRating(ctx, isbn)
}

func (s *Service) GetUserRatingStats(ctx context.Context, userID string) (float64, int, error) {
	return s.repo.GetUserRatingStats(ctx, userID)
}
