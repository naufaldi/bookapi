package store

import (
	"bookapi/internal/entity"
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func setupSessionTestDB(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/booklibrary_test")
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test database: %v", err)
	}
	if err := db.Ping(ctx); err != nil {
		t.Skipf("Skipping test: cannot ping test database: %v", err)
	}
	return db
}

func TestSessionPG_Create(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionPG(db)
	ctx := context.Background()

	session := &entity.Session{
		UserID:          "test-user-id",
		RefreshTokenHash: "test-hash",
		UserAgent:       "test-agent",
		IPAddress:       "127.0.0.1",
		RememberMe:      false,
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}

	err := repo.Create(ctx, session)
	require.NoError(t, err)
	require.NotEmpty(t, session.ID)
	require.NotZero(t, session.CreatedAt)
}

func TestSessionPG_GetByTokenHash(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionPG(db)
	ctx := context.Background()

	session := &entity.Session{
		UserID:          "test-user-id",
		RefreshTokenHash: "test-hash-2",
		UserAgent:       "test-agent",
		IPAddress:       "127.0.0.1",
		RememberMe:      false,
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}

	err := repo.Create(ctx, session)
	require.NoError(t, err)

	found, err := repo.GetByTokenHash(ctx, "test-hash-2")
	require.NoError(t, err)
	require.Equal(t, session.ID, found.ID)
	require.Equal(t, session.UserID, found.UserID)
}

func TestSessionPG_ListByUserID(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionPG(db)
	ctx := context.Background()

	userID := "test-user-id-list"
	session1 := &entity.Session{
		UserID:          userID,
		RefreshTokenHash: "hash-1",
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}
	session2 := &entity.Session{
		UserID:          userID,
		RefreshTokenHash: "hash-2",
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}

	repo.Create(ctx, session1)
	repo.Create(ctx, session2)

	sessions, err := repo.ListByUserID(ctx, userID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(sessions), 2)
}

func TestSessionPG_Delete(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionPG(db)
	ctx := context.Background()

	session := &entity.Session{
		UserID:          "test-user-id",
		RefreshTokenHash: "test-hash-delete",
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}

	err := repo.Create(ctx, session)
	require.NoError(t, err)

	err = repo.Delete(ctx, session.ID)
	require.NoError(t, err)

	_, err = repo.GetByTokenHash(ctx, "test-hash-delete")
	require.Error(t, err)
}
