package store

import (
	"bookapi/internal/entity"
	"bookapi/internal/usecase"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReadingListPG struct {
	db *pgxpool.Pool
}

func NewReadingListPG(db *pgxpool.Pool) *ReadingListPG {
	return &ReadingListPG{
		db: db,
	}
}

// Logika: cari book_id lewat ISBN, lalu INSERT ... ON CONFLICT (user_id, book_id) DO UPDATE SET status=...

func (repo *ReadingListPG) UpsertReadingListItem(ctx context.Context, userID string, isbn string, status string) error {
	const upsertSQL = `
		INSERT INTO user_books (user_id, book_id, status, created_at, updated_at)
		SELECT $1, $2, $3, NOW(), NOW()
		FROM books b
		WHERE b.isbn =$2
		ON CONFLICT (user_id, book_id)
		DO UPDATE SET status = EXCLUDED.status, updated_at = NOW()
	`
	commandTag, err := repo.db.Exec(ctx, upsertSQL, userID, isbn, status)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return usecase.ErrNotFound
	}
	return nil
}

// ListReadingListByStatus mengembalikan daftar buku + total untuk pagination.
func (repo *ReadingListPG) ListReadingListByStatus(ctx context.Context, userID string, status string, limit, offset int) ([]entity.Book, int, error){
	const countSQL = `
		SELECT COUNT(*) 
		FROM user_books ub
		JOIN books b on b.id = ub.book_id
		WHERE ub.user_id = $1 AND ub.status = $2
	`
	var total int 
	if err := repo.db.QueryRow(ctx, countSQL, userID,status).Scan(&total); err != nil{
		return nil, 0, err
	}

	const dataSQL = `
		SEELCT b.id, b.isbn, b.title, b.genre, b.publisher, COALESCE(b.description, '') as description, b.created_at, b.updated_at
		FROM user_books ub
		JOIN books b ON b.id = ub.book_id
		WHERE ub.user_id = $1 AND ub.status = $2
		ORDER by b.title ASC
		LIMIT $3 OFFSET $4  
	`
	rows, err := repo.db.Query(ctx, dataSQL, userID, status, limit, offset) 
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var books []entity.Book
	for rows.Next(){
		var book entity.Book
		if err := rows.Scan(
			&book.ID, &book.ISBN, &book.Title, &book.Genre, &book.Publisher, &book.Description, &book.CreatedAt, &book.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		books = append(books, book)
	}
	if err := rows.Err(); err != nil {
		return nil,0, err
	}
	return books, total, nil
}

func ValidasiReadingListStatus(status string) error {
	switch status {
		case entity.ReadingListStatusWishlist, entity.ReadingListStatusReading, entity.ReadingListStatusFinished:
			return nil
		default:
		return fmt.Errorf("invalid status: %s", status)
	}
}