package readinglist

import (
	"bookapi/internal/book"
	"context"
	"fmt"
)

const (
	StatusWishlist = "WISHLIST"
	StatusReading  = "READING"
	StatusFinished = "FINISHED"
)

func ValidateStatus(status string) error {
	switch status {
	case StatusWishlist, StatusReading, StatusFinished:
		return nil
	default:
		return fmt.Errorf("invalid status: %s", status)
	}
}

type Repository interface {
	UpsertReadingListItem(ctx context.Context, userID string, isbn string, status string) error
	ListReadingListByStatus(ctx context.Context, userID string, status string, limit, offset int) ([]book.Book, int, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Upsert(ctx context.Context, userID, isbn, status string) error {
	if err := ValidateStatus(status); err != nil {
		return err
	}
	return s.repo.UpsertReadingListItem(ctx, userID, isbn, status)
}

func (s *Service) List(ctx context.Context, userID, status string, limit, offset int) ([]book.Book, int, error) {
	if err := ValidateStatus(status); err != nil {
		return nil, 0, err
	}
	return s.repo.ListReadingListByStatus(ctx, userID, status, limit, offset)
}
