package book

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

func (r *PostgresRepo) List(ctx context.Context, q Query) ([]Book, int, error) {
	clauses := []string{"1=1"}
	args := []any{}
	argn := 1

	if q.Genre != "" {
		clauses = append(clauses, fmt.Sprintf("genre = $%d", argn))
		args = append(args, q.Genre)
		argn++
	}

	if len(q.Genres) > 0 {
		clauses = append(clauses, fmt.Sprintf("genre = ANY($%d)", argn))
		args = append(args, q.Genres)
		argn++
	}

	if q.Publisher != "" {
		clauses = append(clauses, fmt.Sprintf("publisher = $%d", argn))
		args = append(args, q.Publisher)
		argn++
	}

	if q.Language != "" {
		clauses = append(clauses, fmt.Sprintf("language = $%d", argn))
		args = append(args, q.Language)
		argn++
	}

	if q.YearFrom != nil {
		clauses = append(clauses, fmt.Sprintf("publication_year >= $%d", argn))
		args = append(args, *q.YearFrom)
		argn++
	}

	if q.YearTo != nil {
		clauses = append(clauses, fmt.Sprintf("publication_year <= $%d", argn))
		args = append(args, *q.YearTo)
		argn++
	}

	if q.Search != "" {
		clauses = append(clauses, fmt.Sprintf("search_vector @@ plainto_tsquery('english', $%d)", argn))
		args = append(args, q.Search)
		argn++
	} else if q.Q != "" {
		clauses = append(clauses, fmt.Sprintf("(isbn ILIKE $%d OR title ILIKE $%d OR publisher ILIKE $%d OR description ILIKE $%d OR genre ILIKE $%d)", argn, argn+1, argn+2, argn+3, argn+4))
		pattern := "%" + q.Q + "%"
		args = append(args, pattern, pattern, pattern, pattern, pattern)
		argn += 5
	}

	where := "WHERE " + strings.Join(clauses, " AND ")

	ratingJoin := ""
	if q.MinRating != nil {
		ratingJoin = "JOIN (SELECT book_id, AVG(star) as avg_star FROM ratings GROUP BY book_id) r_stats ON r_stats.book_id = b.id"
		where += fmt.Sprintf(" AND r_stats.avg_star >= $%d", argn)
		args = append(args, *q.MinRating)
		argn++
	}

	sortCol := "b.title"
	if q.Search != "" && q.Sort == "relevance" {
		sortCol = fmt.Sprintf("ts_rank(b.search_vector, plainto_tsquery('english', $%d))", argn-1)
	} else {
		switch q.Sort {
		case "created_at":
			sortCol = "b.created_at"
		case "rating":
			if ratingJoin == "" {
				ratingJoin = "LEFT JOIN (SELECT book_id, AVG(star) as avg_star FROM ratings GROUP BY book_id) r_stats ON r_stats.book_id = b.id"
			}
			sortCol = "COALESCE(r_stats.avg_star, 0)"
		case "year":
			sortCol = "b.publication_year"
		default:
			sortCol = "b.title"
		}
	}

	order := "ASC"
	if q.Desc || q.Sort == "relevance" || q.Sort == "rating" {
		order = "DESC"
	}

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM books b %s %s", ratingJoin, where)
	var total int
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	if err := r.db.QueryRow(timeoutCtx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataSQL := fmt.Sprintf(`
		SELECT b.id, b.isbn, b.title, b.subtitle, b.genre, b.publisher, b.description, 
		       b.published_date, b.publication_year, b.page_count, b.language, b.cover_url,
		       b.created_at, b.updated_at
		FROM books b
		%s
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		ratingJoin, where, sortCol, order, argn, argn+1)

	argsWithPage := append([]any{}, args...)
	argsWithPage = append(argsWithPage, q.Limit, q.Offset)
	timeoutCtx2, cancel2 := r.withTimeout(ctx)
	defer cancel2()
	rows, err := r.db.Query(timeoutCtx2, dataSQL, argsWithPage...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(
			&b.ID, &b.ISBN, &b.Title, &b.Subtitle, &b.Genre, &b.Publisher, &b.Description,
			&b.PublishedDate, &b.PublicationYear, &b.PageCount, &b.Language, &b.CoverURL,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, b)
	}
	return out, total, rows.Err()
}

func (r *PostgresRepo) GetByISBN(ctx context.Context, isbn string) (Book, error) {
	const query = `
		SELECT id, isbn, title, subtitle, genre, publisher, description, 
		       published_date, publication_year, page_count, language, cover_url,
		       created_at, updated_at
		FROM books
		WHERE isbn = $1
		LIMIT 1
	`
	var b Book
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	err := r.db.QueryRow(timeoutCtx, query, isbn).Scan(
		&b.ID, &b.ISBN, &b.Title, &b.Subtitle, &b.Genre, &b.Publisher, &b.Description,
		&b.PublishedDate, &b.PublicationYear, &b.PageCount, &b.Language, &b.CoverURL,
		&b.CreatedAt, &b.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Book{}, ErrNotFound
		}
		return Book{}, err
	}
	return b, nil
}

func (r *PostgresRepo) UpsertFromIngest(ctx context.Context, book *Book) error {
	const sql = `
		INSERT INTO books (isbn, title, subtitle, genre, publisher, description, 
		                   published_date, publication_year, page_count, language, cover_url, 
		                   created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		ON CONFLICT (isbn) DO UPDATE SET
			title = EXCLUDED.title,
			subtitle = EXCLUDED.subtitle,
			genre = EXCLUDED.genre,
			publisher = EXCLUDED.publisher,
			description = EXCLUDED.description,
			published_date = EXCLUDED.published_date,
			publication_year = EXCLUDED.publication_year,
			page_count = EXCLUDED.page_count,
			language = EXCLUDED.language,
			cover_url = EXCLUDED.cover_url,
			updated_at = NOW()`

	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	_, err := r.db.Exec(timeoutCtx, sql,
		book.ISBN, book.Title, book.Subtitle, book.Genre, book.Publisher, book.Description,
		book.PublishedDate, book.PublicationYear, book.PageCount, book.Language, book.CoverURL,
	)
	return err
}
