package store

import (
	"bookapi/internal/usecase"
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RatingPG struct {
	db *pgxpool.Pool
}

func NewRatingPG(db *pgxpool.Pool) *RatingPG {
	return &RatingPG{db: db}
}

func (repo *RatingPG) CreateOrUpdateRating(ctx context.Context, userID string, isbn string, star int) error {
	if star < 1 || star > 5 {
		return errors.New("Rating must be between 1 and 5")
	}
	var bookID string
	findBookSQL := ` select id from books where isbn = $1 limit 1`
	if err := repo.db.QueryRow(ctx, findBookSQL, isbn).Scan(&bookID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return usecase.ErrNotFound
		}
		return err
	}
	upsertSQL := `
		insert into ratings(user_id, book_id, star, created_at, updated_at)
		values($1, $2, $3, now(), now())
		on conflict(user_id, book_id)
		do update set star = excluded.star, updated_at = now();
	`
	_, err := repo.db.Exec(ctx, upsertSQL, userID, bookID, star)
	return err
}

func (repo *RatingPG) GetUserRating(ctx context.Context, userID, isbn string) (int, error) {
	query := `
		SELECT r.star
		FROM ratings r
		JOIN books b ON b.id = r.book_id
		WHERE r.user_id = $1 AND b.isbn = $2
		LIMIT 1
	`
	var star int
	if err := repo.db.QueryRow(ctx, query, userID, isbn).Scan(&star); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, usecase.ErrNotFound
		}
		return 0, err
	}
	return star, nil
}

func (repo *RatingPG) GetBookRating(ctx context.Context, isbn string) (float64, int, error) {
	query := `
		SELECT AVG(r.star)::FLOAT, COUNT(r.star)
		FROM ratings r
		JOIN books b ON b.id = r.book_id
		WHERE b.isbn = $1
	`
	var average sql.NullFloat64
	var count int
	if err := repo.db.QueryRow(ctx, query, isbn).Scan(&average, &count); err != nil {
		return 0, 0, err
	}
	if !average.Valid {
		return 0, 0, nil
	}
	return average.Float64, count, nil
}
