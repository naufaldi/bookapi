package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BlacklistPG struct {
	db *pgxpool.Pool
}

func NewBlacklistPG(db *pgxpool.Pool) *BlacklistPG {
	return &BlacklistPG{db: db}
}

func (r *BlacklistPG) AddToken(ctx context.Context, jti string, userID string, expiresAt interface{}) error {
	var expTime time.Time
	switch v := expiresAt.(type) {
	case time.Time:
		expTime = v
	default:
		return nil
	}
	const query = `
	INSERT INTO token_blacklist (jti, user_id, expires_at)
	VALUES ($1, $2, $3)
	ON CONFLICT (jti) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, jti, userID, expTime)
	return err
}

func (r *BlacklistPG) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	const query = `
	SELECT EXISTS(
		SELECT 1 FROM token_blacklist
		WHERE jti = $1 AND expires_at > now()
	)
	`
	var exists bool
	err := r.db.QueryRow(ctx, query, jti).Scan(&exists)
	return exists, err
}

func (r *BlacklistPG) CleanupExpired(ctx context.Context) error {
	const query = `DELETE FROM token_blacklist WHERE expires_at < now()`
	_, err := r.db.Exec(ctx, query)
	return err
}
