package readinglist

import (
	"bookapi/internal/book"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type PostgresRepo struct {
	db      *pgxpool.Pool
	timeout time.Duration
}

func NewPostgresRepo(db *pgxpool.Pool, timeout time.Duration) *PostgresRepo {
	return &PostgresRepo{db: db, timeout: timeout}
}

func (r *PostgresRepo) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, r.timeout)
}

func (r *PostgresRepo) UpsertReadingListItem(ctx context.Context, userID string, isbn string, status string) error {
	const upsertSQL = `
		INSERT INTO user_books (user_id, book_id, status, created_at, updated_at)
		SELECT $1, b.id, $3, NOW(), NOW()
		FROM books b
		WHERE b.isbn = $2
		ON CONFLICT (user_id, book_id)
		DO UPDATE SET status = EXCLUDED.status, updated_at = NOW()
	`
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	commandTag, err := r.db.Exec(timeoutCtx, upsertSQL, userID, isbn, status)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepo) ListReadingListByStatus(ctx context.Context, userID string, status string, limit, offset int) ([]book.Book, int, error) {
	const countSQL = `
		SELECT COUNT(*)
		FROM user_books ub
		JOIN books b ON b.id = ub.book_id
		WHERE ub.user_id = $1 AND ub.status = $2
	`
	var total int
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	if err := r.db.QueryRow(timeoutCtx, countSQL, userID, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	const dataSQL = `
		SELECT b.id, b.isbn, b.title, b.subtitle, b.genre, b.publisher, COALESCE(b.description, '') as description, 
		       b.published_date, b.publication_year, b.page_count, b.language, b.cover_url, b.created_at, b.updated_at
		FROM user_books ub
		JOIN books b ON b.id = ub.book_id
		WHERE ub.user_id = $1 AND ub.status = $2
		ORDER BY b.title ASC
		LIMIT $3 OFFSET $4
	`
	timeoutCtx2, cancel2 := r.withTimeout(ctx)
	defer cancel2()
	rows, err := r.db.Query(timeoutCtx2, dataSQL, userID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var books []book.Book
	for rows.Next() {
		var b book.Book
		if err := rows.Scan(
			&b.ID, &b.ISBN, &b.Title, &b.Subtitle, &b.Genre, &b.Publisher, &b.Description,
			&b.PublishedDate, &b.PublicationYear, &b.PageCount, &b.Language, &b.CoverURL,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		books = append(books, b)
	}
	return books, total, rows.Err()
}
