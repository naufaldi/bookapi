package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBlacklistPG_AddToken(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewBlacklistPG(db)
	ctx := context.Background()

	jti := "test-jti-1"
	userID := "test-user-id"
	expiresAt := time.Now().Add(24 * time.Hour)

	err := repo.AddToken(ctx, jti, userID, expiresAt)
	require.NoError(t, err)
}

func TestBlacklistPG_IsBlacklisted(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewBlacklistPG(db)
	ctx := context.Background()

	jti := "test-jti-2"
	userID := "test-user-id"
	expiresAt := time.Now().Add(24 * time.Hour)

	err := repo.AddToken(ctx, jti, userID, expiresAt)
	require.NoError(t, err)

	isBlacklisted, err := repo.IsBlacklisted(ctx, jti)
	require.NoError(t, err)
	require.True(t, isBlacklisted)

	isBlacklisted, err = repo.IsBlacklisted(ctx, "non-existent-jti")
	require.NoError(t, err)
	require.False(t, isBlacklisted)
}

func TestBlacklistPG_CleanupExpired(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewBlacklistPG(db)
	ctx := context.Background()

	jti := "test-jti-expired"
	userID := "test-user-id"
	expiresAt := time.Now().Add(-1 * time.Hour)

	err := repo.AddToken(ctx, jti, userID, expiresAt)
	require.NoError(t, err)

	err = repo.CleanupExpired(ctx)
	require.NoError(t, err)

	isBlacklisted, err := repo.IsBlacklisted(ctx, jti)
	require.NoError(t, err)
	require.False(t, isBlacklisted)
}
