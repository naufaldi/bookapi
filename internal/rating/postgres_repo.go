package rating

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInternalNotFound = errors.New("not found")

type PostgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepo(db *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (repo *PostgresRepo) CreateOrUpdateRating(ctx context.Context, userID string, isbn string, star int) error {
	if star < 1 || star > 5 {
		return errors.New("rating must be between 1 and 5")
	}
	var bookID string
	findBookSQL := `SELECT id FROM books WHERE isbn = $1 LIMIT 1`
	if err := repo.db.QueryRow(ctx, findBookSQL, isbn).Scan(&bookID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInternalNotFound
		}
		return err
	}
	upsertSQL := `
		INSERT INTO ratings(user_id, book_id, star, created_at, updated_at)
		VALUES($1, $2, $3, now(), now())
		ON CONFLICT(user_id, book_id)
		DO UPDATE SET star = excluded.star, updated_at = now();
	`
	_, err := repo.db.Exec(ctx, upsertSQL, userID, bookID, star)
	return err
}

func (repo *PostgresRepo) GetUserRating(ctx context.Context, userID, isbn string) (int, error) {
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
			return 0, ErrInternalNotFound
		}
		return 0, err
	}
	return star, nil
}

func (repo *PostgresRepo) GetBookRating(ctx context.Context, isbn string) (float64, int, error) {
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

func (repo *PostgresRepo) GetUserRatingStats(ctx context.Context, userID string) (float64, int, error) {
	query := `
		SELECT AVG(star)::FLOAT, COUNT(star)
		FROM ratings
		WHERE user_id = $1
	`
	var average sql.NullFloat64
	var count int
	if err := repo.db.QueryRow(ctx, query, userID).Scan(&average, &count); err != nil {
		return 0, 0, err
	}
	if !average.Valid {
		return 0, 0, nil
	}
	return average.Float64, count, nil
}
