package store

//Repository implementation (Postgres)

import (
	"bookapi/internal/entity"
	"bookapi/internal/usecase"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)
type BookPG struct {
	db * pgxpool.Pool
}

func NewBookPG(db * pgxpool.Pool) * BookPG {
	return &BookPG{db: db}
}

func (r *BookPG) List(ctx context.Context, p usecase.ListParams) ([]entity.Book, int, error) {
	clauses := []string{"1=1"}
	args := []any{}
	argn := 1

	if p.Genre != "" {
		clauses = append(clauses, fmt.Sprintf("genre = $%d", argn))
		args = append(args, p.Genre)
		argn++
	}

	if len(p.Genres) > 0 {
		clauses = append(clauses, fmt.Sprintf("genre = ANY($%d)", argn))
		args = append(args, p.Genres)
		argn++
	}

	if p.Publisher != "" {
		clauses = append(clauses, fmt.Sprintf("publisher = $%d", argn))
		args = append(args, p.Publisher)
		argn++
	}

	if p.Language != "" {
		clauses = append(clauses, fmt.Sprintf("language = $%d", argn))
		args = append(args, p.Language)
		argn++
	}

	if p.YearFrom != nil {
		clauses = append(clauses, fmt.Sprintf("publication_year >= $%d", argn))
		args = append(args, *p.YearFrom)
		argn++
	}

	if p.YearTo != nil {
		clauses = append(clauses, fmt.Sprintf("publication_year <= $%d", argn))
		args = append(args, *p.YearTo)
		argn++
	}

	if p.Search != "" {
		clauses = append(clauses, fmt.Sprintf("search_vector @@ plainto_tsquery('english', $%d)", argn))
		args = append(args, p.Search)
		argn++
	} else if p.Q != "" {
		// fallback to ILIKE if no FTS search provided but Q is
		clauses = append(clauses, fmt.Sprintf("(title ILIKE $%d OR publisher ILIKE $%d)", argn, argn+1))
		pattern := "%" + p.Q + "%"
		args = append(args, pattern, pattern)
		argn += 2
	}

	where := "WHERE " + strings.Join(clauses, " AND ")

	// Min rating filter requires joining with ratings
	ratingJoin := ""
	if p.MinRating != nil {
		ratingJoin = "JOIN (SELECT book_id, AVG(star) as avg_star FROM ratings GROUP BY book_id) r_stats ON r_stats.book_id = b.id"
		where += fmt.Sprintf(" AND r_stats.avg_star >= $%d", argn)
		args = append(args, *p.MinRating)
		argn++
	}

	// sort whitelist
	sortCol := "b.title"
	if p.Search != "" && p.Sort == "relevance" {
		sortCol = fmt.Sprintf("ts_rank(b.search_vector, plainto_tsquery('english', $%d))", argn-1) // use the same search arg
	} else {
		switch p.Sort {
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
	if p.Desc || p.Sort == "relevance" || p.Sort == "rating" {
		order = "DESC"
	}

	// TOTAL Count
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM books b %s %s", ratingJoin, where)
	var total int
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data
	dataSQL := fmt.Sprintf(`
		SELECT b.id, b.isbn, b.title, b.genre, b.publisher, b.description, 
		       b.publication_year, b.page_count, b.language, b.cover_url,
		       b.created_at, b.updated_at
		FROM books b
		%s
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		ratingJoin, where, sortCol, order, argn, argn+1)

	argsWithPage := append([]any{}, args...)
	argsWithPage = append(argsWithPage, p.Limit, p.Offset)
	rows, err := r.db.Query(ctx, dataSQL, argsWithPage...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []entity.Book
	for rows.Next() {
		var b entity.Book
		if err := rows.Scan(
			&b.ID, &b.ISBN, &b.Title, &b.Genre, &b.Publisher, &b.Description,
			&b.PublicationYear, &b.PageCount, &b.Language, &b.CoverURL,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *BookPG) GetByISBN(ctx context.Context, isbn string) (entity.Book, error) {
	const query = `
		SELECT id, isbn, title, genre, publisher, description, 
		       publication_year, page_count, language, cover_url,
		       created_at, updated_at
		FROM books
		WHERE isbn = $1
		LIMIT 1
	`
	var b entity.Book
	err := r.db.QueryRow(ctx, query, isbn).Scan(
		&b.ID, &b.ISBN, &b.Title, &b.Genre, &b.Publisher, &b.Description,
		&b.PublicationYear, &b.PageCount, &b.Language, &b.CoverURL,
		&b.CreatedAt, &b.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Book{}, usecase.ErrNotFound
		}
		return entity.Book{}, err
	}
	return b, nil
}
