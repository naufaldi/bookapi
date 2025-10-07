package usecase

import (
	"bookapi/internal/entity"
	"context"
)

// Repository interface
// Define the contract for fetching books.
type BookRepository interface {
	List(ctx context.Context, genre, publisher string, limit, offset int) ([]entity.Book, error)
}