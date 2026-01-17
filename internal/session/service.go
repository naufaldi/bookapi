package session

import (
	"context"
)

type Service struct {
	repo          Repository
	blacklistRepo BlacklistRepository
}

func NewService(repo Repository, blacklistRepo BlacklistRepository) *Service {
	return &Service{
		repo:          repo,
		blacklistRepo: blacklistRepo,
	}
}

func (s *Service) ListByUserID(ctx context.Context, userID string) ([]Session, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, sessionID string) error {
	return s.repo.Delete(ctx, sessionID)
}

func (s *Service) Create(ctx context.Context, session *Session) error {
	return s.repo.Create(ctx, session)
}

func (s *Service) GetByTokenHash(ctx context.Context, hash string) (Session, error) {
	return s.repo.GetByTokenHash(ctx, hash)
}

func (s *Service) DeleteByTokenHash(ctx context.Context, hash string) error {
	return s.repo.DeleteByTokenHash(ctx, hash)
}

func (s *Service) AddToBlacklist(ctx context.Context, jti, userID string, expiresAt any) error {
	return s.blacklistRepo.AddToken(ctx, jti, userID, expiresAt)
}
