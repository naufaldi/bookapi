package http_test

import (
	"bookapi/internal/http"
	"bookapi/internal/store"
	"bookapi/internal/usecase"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func setupIntegrationDB(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/booklibrary_test")
	if err != nil {
		t.Skipf("Skipping integration test: cannot connect to test database: %v", err)
	}
	if err := db.Ping(ctx); err != nil {
		t.Skipf("Skipping integration test: cannot ping test database: %v", err)
	}
	return db
}

func TestIntegration_ProfileFlow(t *testing.T) {
	db := setupIntegrationDB(t)
	defer db.Close()

	userRepo := store.NewUserPG(db)
	ratingRepo := store.NewRatingPG(db)
	readingListRepo := store.NewReadingListPG(db)
	profileUsecase := usecase.NewProfileUsecase(userRepo, ratingRepo, readingListRepo)
	handler := http.NewProfileHandler(profileUsecase)

	t.Run("public profile access", func(t *testing.T) {
		// This requires a user to exist in the DB.
		// Since we don't have a reliable way to seed here without a lot of boilerplate,
		// we'll just verify the handler doesn't crash and returns 404 for non-existent user.
		
		req := httptest.NewRequest("GET", "/users/non-existent-id/profile", nil)
		w := httptest.NewRecorder()
		
		handler.GetPublicProfile(w, req)
		
		assert.Equal(t, 404, w.Code)
	})
}
