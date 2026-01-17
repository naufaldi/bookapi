package session

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepo(db *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) Create(ctx context.Context, s *Session) error {
	const query = `
	INSERT INTO sessions (id, user_id, refresh_token_hash, user_agent, ip_address, remember_me, expires_at)
	VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
	RETURNING id, created_at, last_used_at
	`
	return r.db.QueryRow(ctx, query,
		s.UserID,
		s.RefreshTokenHash,
		s.UserAgent,
		s.IPAddress,
		s.RememberMe,
		s.ExpiresAt,
	).Scan(&s.ID, &s.CreatedAt, &s.LastUsedAt)
}

func (r *PostgresRepo) GetByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	const query = `
	SELECT id, user_id, refresh_token_hash, user_agent, ip_address, remember_me, expires_at, created_at, last_used_at
	FROM sessions
	WHERE refresh_token_hash = $1 AND expires_at > now()
	LIMIT 1
	`
	var s Session
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&s.ID,
		&s.UserID,
		&s.RefreshTokenHash,
		&s.UserAgent,
		&s.IPAddress,
		&s.RememberMe,
		&s.ExpiresAt,
		&s.CreatedAt,
		&s.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrNotFound
		}
		return Session{}, err
	}
	return s, nil
}

func (r *PostgresRepo) ListByUserID(ctx context.Context, userID string) ([]Session, error) {
	const query = `
	SELECT id, user_id, refresh_token_hash, user_agent, ip_address, remember_me, expires_at, created_at, last_used_at
	FROM sessions
	WHERE user_id = $1 AND expires_at > now()
	ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.RefreshTokenHash,
			&s.UserAgent,
			&s.IPAddress,
			&s.RememberMe,
			&s.ExpiresAt,
			&s.CreatedAt,
			&s.LastUsedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *PostgresRepo) Delete(ctx context.Context, sessionID string) error {
	const query = `DELETE FROM sessions WHERE id = $1`
	result, err := r.db.Exec(ctx, query, sessionID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepo) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	const query = `DELETE FROM sessions WHERE refresh_token_hash = $1`
	_, err := r.db.Exec(ctx, query, tokenHash)
	return err
}

func (r *PostgresRepo) UpdateLastUsed(ctx context.Context, sessionID string) error {
	const query = `UPDATE sessions SET last_used_at = now() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, sessionID)
	return err
}

func (r *PostgresRepo) CleanupExpired(ctx context.Context) error {
	const query = `DELETE FROM sessions WHERE expires_at < now()`
	_, err := r.db.Exec(ctx, query)
	return err
}

type BlacklistPostgresRepo struct {
	db *pgxpool.Pool
}

func NewBlacklistPostgresRepo(db *pgxpool.Pool) *BlacklistPostgresRepo {
	return &BlacklistPostgresRepo{db: db}
}

func (r *BlacklistPostgresRepo) AddToken(ctx context.Context, jti string, userID string, expiresAt any) error {
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

func (r *BlacklistPostgresRepo) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
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

func (r *BlacklistPostgresRepo) CleanupExpired(ctx context.Context) error {
	const query = `DELETE FROM token_blacklist WHERE expires_at < now()`
	_, err := r.db.Exec(ctx, query)
	return err
}
