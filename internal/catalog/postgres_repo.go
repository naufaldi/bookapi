package catalog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	UpsertBook(ctx context.Context, book *Book, rawJSON []byte) error
	UpsertAuthor(ctx context.Context, author *Author, rawJSON []byte) error
	GetTotalBooks(ctx context.Context) (int, error)
	GetTotalAuthors(ctx context.Context) (int, error)
	GetBookUpdatedAt(ctx context.Context, isbn13 string) (time.Time, error)
	GetAuthorUpdatedAt(ctx context.Context, key string) (time.Time, error)
	List(ctx context.Context, q SearchQuery) ([]Book, int, error)
	GetByISBN(ctx context.Context, isbn13 string) (Book, error)
}

type PostgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepo(db *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) UpsertBook(ctx context.Context, b *Book, rawJSON []byte) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const bookSQL = `
		INSERT INTO catalog_books (isbn13, title, subtitle, description, cover_url, published_date, publisher, language, page_count, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		ON CONFLICT (isbn13) DO UPDATE SET
			title = EXCLUDED.title,
			subtitle = EXCLUDED.subtitle,
			description = EXCLUDED.description,
			cover_url = EXCLUDED.cover_url,
			published_date = EXCLUDED.published_date,
			publisher = EXCLUDED.publisher,
			language = EXCLUDED.language,
			page_count = EXCLUDED.page_count,
			updated_at = now()`

	_, err = tx.Exec(ctx, bookSQL, b.ISBN13, b.Title, b.Subtitle, b.Description, b.CoverURL, b.PublishedDate, b.Publisher, b.Language, b.PageCount)
	if err != nil {
		return fmt.Errorf("upsert book: %w", err)
	}

	const sourceSQL = `
		INSERT INTO catalog_sources (entity_type, entity_key, provider, raw_json, fetched_at)
		VALUES ('BOOK', $1, 'OPEN_LIBRARY', $2, now())
		ON CONFLICT (entity_type, entity_key, provider) DO UPDATE SET
			raw_json = EXCLUDED.raw_json,
			fetched_at = now()`

	_, err = tx.Exec(ctx, sourceSQL, b.ISBN13, rawJSON)
	if err != nil {
		return fmt.Errorf("upsert book source: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepo) UpsertAuthor(ctx context.Context, a *Author, rawJSON []byte) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const authorSQL = `
		INSERT INTO catalog_authors (key, name, birth_date, bio, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (key) DO UPDATE SET
			name = EXCLUDED.name,
			birth_date = EXCLUDED.birth_date,
			bio = EXCLUDED.bio,
			updated_at = now()`

	_, err = tx.Exec(ctx, authorSQL, a.Key, a.Name, a.BirthDate, a.Bio)
	if err != nil {
		return fmt.Errorf("upsert author: %w", err)
	}

	const sourceSQL = `
		INSERT INTO catalog_sources (entity_type, entity_key, provider, raw_json, fetched_at)
		VALUES ('AUTHOR', $1, 'OPEN_LIBRARY', $2, now())
		ON CONFLICT (entity_type, entity_key, provider) DO UPDATE SET
			raw_json = EXCLUDED.raw_json,
			fetched_at = now()`

	_, err = tx.Exec(ctx, sourceSQL, a.Key, rawJSON)
	if err != nil {
		return fmt.Errorf("upsert author source: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepo) GetTotalBooks(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM catalog_books").Scan(&count)
	return count, err
}

func (r *PostgresRepo) GetTotalAuthors(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM catalog_authors").Scan(&count)
	return count, err
}

func (r *PostgresRepo) GetBookUpdatedAt(ctx context.Context, isbn13 string) (time.Time, error) {
	var t time.Time
	err := r.db.QueryRow(ctx, "SELECT updated_at FROM catalog_books WHERE isbn13 = $1", isbn13).Scan(&t)
	if err == pgx.ErrNoRows {
		return time.Time{}, nil
	}
	return t, err
}

func (r *PostgresRepo) GetAuthorUpdatedAt(ctx context.Context, key string) (time.Time, error) {
	var t time.Time
	err := r.db.QueryRow(ctx, "SELECT updated_at FROM catalog_authors WHERE key = $1", key).Scan(&t)
	if err == pgx.ErrNoRows {
		return time.Time{}, nil
	}
	return t, err
}

func (r *PostgresRepo) List(ctx context.Context, q SearchQuery) ([]Book, int, error) {
	clauses := []string{"1=1"}
	args := []any{}
	argn := 1

	if q.Publisher != "" {
		clauses = append(clauses, fmt.Sprintf("publisher ILIKE $%d", argn))
		args = append(args, "%"+q.Publisher+"%")
		argn++
	}

	if q.Language != "" {
		clauses = append(clauses, fmt.Sprintf("language = $%d", argn))
		args = append(args, q.Language)
		argn++
	}

	if q.Q != "" {
		clauses = append(clauses, fmt.Sprintf("search_vector @@ plainto_tsquery('english', $%d)", argn))
		args = append(args, q.Q)
		argn++
	}

	where := "WHERE " + strings.Join(clauses, " AND ")

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM catalog_books %s", where)
	var total int
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataSQL := fmt.Sprintf(`
		SELECT isbn13, title, subtitle, description, cover_url, published_date, publisher, language, page_count, updated_at
		FROM catalog_books
		%s
		ORDER BY title ASC
		LIMIT $%d OFFSET $%d`,
		where, argn, argn+1)

	argsWithPage := append([]any{}, args...)
	argsWithPage = append(argsWithPage, q.Limit, q.Offset)
	rows, err := r.db.Query(ctx, dataSQL, argsWithPage...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(
			&b.ISBN13, &b.Title, &b.Subtitle, &b.Description, &b.CoverURL,
			&b.PublishedDate, &b.Publisher, &b.Language, &b.PageCount, &b.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, b)
	}
	return out, total, rows.Err()
}

func (r *PostgresRepo) GetByISBN(ctx context.Context, isbn13 string) (Book, error) {
	const query = `
		SELECT isbn13, title, subtitle, description, cover_url, published_date, publisher, language, page_count, updated_at
		FROM catalog_books
		WHERE isbn13 = $1
	`
	var b Book
	err := r.db.QueryRow(ctx, query, isbn13).Scan(
		&b.ISBN13, &b.Title, &b.Subtitle, &b.Description, &b.CoverURL,
		&b.PublishedDate, &b.Publisher, &b.Language, &b.PageCount, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Book{}, fmt.Errorf("book not found: %s", isbn13)
		}
		return Book{}, err
	}
	return b, nil
}
