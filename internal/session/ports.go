package session

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, s *Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (Session, error)
	ListByUserID(ctx context.Context, userID string) ([]Session, error)
	Delete(ctx context.Context, sessionID string) error
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	UpdateLastUsed(ctx context.Context, sessionID string) error
	CleanupExpired(ctx context.Context) error
}

type BlacklistRepository interface {
	AddToken(ctx context.Context, jti string, userID string, expiresAt any) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
	CleanupExpired(ctx context.Context) error
}
