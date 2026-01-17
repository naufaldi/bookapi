package book

import (
	"context"
)

// Service provides book-related business logic.
type Service struct {
	repo Repository
}

// NewService creates a new book service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns a list of books matching the query.
func (s *Service) List(ctx context.Context, q Query) ([]Book, int, error) {
	return s.repo.List(ctx, q)
}

// GetByISBN returns a book by its ISBN.
func (s *Service) GetByISBN(ctx context.Context, isbn string) (Book, error) {
	return s.repo.GetByISBN(ctx, isbn)
}
