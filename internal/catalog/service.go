package catalog

import (
	"context"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Search(ctx context.Context, q SearchQuery) ([]Book, int, error) {
	return s.repo.List(ctx, q)
}

func (s *Service) GetByISBN(ctx context.Context, isbn13 string) (Book, error) {
	return s.repo.GetByISBN(ctx, isbn13)
}
