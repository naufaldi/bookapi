package store

//Repository implementation (Postgres)

import (
	"bookapi/internal/entity"
	"bookapi/internal/usecase"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)
type BookPG struct {
	db * pgxpool.Pool
}

func NewBookPG(db * pgxpool.Pool) * BookPG {
	return &BookPG{db: db}
}

func (r * BookPG) List(ctx context.Context, genre,publisher string, limit, offset int) ([]entity.Book, error){
	query := `
	SELECT id, isbn, title, genre, publisher, created_at, updated_at
	FROM books
	WHERE ($1 = '' OR genre = $1)
	AND ($2 = '' OR publisher = $2)
	ORDER BY title
	LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx,query,genre,publisher,limit,offset)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []entity.Book
	for rows.Next(){
		var b entity.Book
		if err := rows.Scan(&b.ID, &b.ISBN,  &b.Title, &b.Genre, &b.Publisher, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return books, nil
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