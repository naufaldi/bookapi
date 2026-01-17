package usecase

import (
	"bookapi/internal/entity"
	"context"
)

type SessionRepository interface {
	Create(ctx context.Context, session *entity.Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (entity.Session, error)
	ListByUserID(ctx context.Context, userID string) ([]entity.Session, error)
	Delete(ctx context.Context, sessionID string) error
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	UpdateLastUsed(ctx context.Context, sessionID string) error
	CleanupExpired(ctx context.Context) error
}

type BlacklistRepository interface {
	AddToken(ctx context.Context, jti string, userID string, expiresAt interface{}) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
	CleanupExpired(ctx context.Context) error
}
