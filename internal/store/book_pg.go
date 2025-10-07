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
	if p.Genre != ""{
		clauses = append(clauses, fmt.Sprintf("genre = $%d", argn))
		args = append(args, p.Genre)
		argn++
	}

	if p.Publisher != ""{
		clauses = append(clauses, fmt.Sprintf("publisher = $%d", argn))
		args = append(args, p.Publisher)
		argn++
	}
	
	if p.Q != "" {
		// ilike for case-insensitive search on title or publisher
		clauses = append(clauses, fmt.Sprintf("(title ILIKE $%d OR publisher ILIKE $%d)", argn, argn+1))
		pattern := "%" + p.Q + "%"
		args = append(args, pattern, pattern)
		argn += 2
	}

	where := "WHERE " + strings.Join(clauses, " AND ")
// sort whitelist for avoid injection
	sortCol := "title"
	switch p.Sort {
	case "created_at":
		sortCol = "created_at"
	case "title":
		sortCol = "title"
	default:
		sortCol = "title"
	}
	order := "ASC"
	if p.Desc {
		order = "DESC"
	}
	// TOTAL Count
	countSQL := "SELECT COUNT(*) FROM books " + where
	var total int
	if err := r.db.QueryRow(ctx, countSQL).Scan(&total); err != nil {
		return nil, 0, err
	}
	// Data
	dataSQL := fmt.Sprintf(`
	SELECT id, isbn, title, genre, publisher, description, created_at, updated_at
	FROM books
	%s
	ORDER BY %s %s
	LIMIT $%d OFFSET $%d`,
		where, sortCol, order, argn, argn+1)

	argsWithPage := append([]any{}, args...)
	argsWithPage = append(argsWithPage, p.Limit, p.Offset)
	rows, err := r.db.Query(ctx, dataSQL, argsWithPage...)


	if err != nil {
		return nil,0, err
	}
	defer rows.Close()

	var out []entity.Book
	for rows.Next() {
		var b entity.Book
		// If you do have 'description', scan it too and add the column in SELECT.
		if err := rows.Scan(&b.ID, &b.ISBN, &b.Title, &b.Genre, &b.Publisher, &b.Description, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}


func ( r *BookPG) GetByISBN(ctx context.Context, isbn string) (entity.Book, error){
	const query = `
	SELECT id, isbn, title, genre, publisher, description, created_at, updated_at
	FROM books
	WHERE isbn = $1
	LIMIT 1
	`
	var b entity.Book
	err := r.db.QueryRow(ctx, query, isbn).Scan(&b.ID, &b.ISBN, &b.Title, &b.Genre, &b.Publisher, &b.Description, &b.CreatedAt, &b.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows){
			return entity.Book{}, usecase.ErrNotFound
		}
		return entity.Book{}, err
	}
	return b, nil
}