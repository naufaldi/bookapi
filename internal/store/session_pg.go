package store

import (
	"bookapi/internal/entity"
	"bookapi/internal/usecase"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionPG struct {
	db *pgxpool.Pool
}

func NewSessionPG(db *pgxpool.Pool) *SessionPG {
	return &SessionPG{db: db}
}

func (r *SessionPG) Create(ctx context.Context, session *entity.Session) error {
	const query = `
	INSERT INTO sessions (id, user_id, refresh_token_hash, user_agent, ip_address, remember_me, expires_at)
	VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
	RETURNING id, created_at, last_used_at
	`
	return r.db.QueryRow(ctx, query,
		session.UserID,
		session.RefreshTokenHash,
		session.UserAgent,
		session.IPAddress,
		session.RememberMe,
		session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt, &session.LastUsedAt)
}

func (r *SessionPG) GetByTokenHash(ctx context.Context, tokenHash string) (entity.Session, error) {
	const query = `
	SELECT id, user_id, refresh_token_hash, user_agent, ip_address, remember_me, expires_at, created_at, last_used_at
	FROM sessions
	WHERE refresh_token_hash = $1 AND expires_at > now()
	LIMIT 1
	`
	var session entity.Session
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.UserAgent,
		&session.IPAddress,
		&session.RememberMe,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Session{}, usecase.ErrNotFound
		}
		return entity.Session{}, err
	}
	return session, nil
}

func (r *SessionPG) ListByUserID(ctx context.Context, userID string) ([]entity.Session, error) {
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

	var sessions []entity.Session
	for rows.Next() {
		var session entity.Session
		if err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.RefreshTokenHash,
			&session.UserAgent,
			&session.IPAddress,
			&session.RememberMe,
			&session.ExpiresAt,
			&session.CreatedAt,
			&session.LastUsedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (r *SessionPG) Delete(ctx context.Context, sessionID string) error {
	const query = `DELETE FROM sessions WHERE id = $1`
	result, err := r.db.Exec(ctx, query, sessionID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return usecase.ErrNotFound
	}
	return nil
}

func (r *SessionPG) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	const query = `DELETE FROM sessions WHERE refresh_token_hash = $1`
	_, err := r.db.Exec(ctx, query, tokenHash)
	return err
}

func (r *SessionPG) UpdateLastUsed(ctx context.Context, sessionID string) error {
	const query = `UPDATE sessions SET last_used_at = now() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, sessionID)
	return err
}

func (r *SessionPG) CleanupExpired(ctx context.Context) error {
	const query = `DELETE FROM sessions WHERE expires_at < now()`
	_, err := r.db.Exec(ctx, query)
	return err
}
