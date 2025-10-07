package usecase

import (
	"bookapi/internal/entity"
	"context"
)

// Repository interface
// Define the contract for fetching books.
type BookRepository interface {
	// List Books with pagination and filters
	List(ctx context.Context, genre, publisher string, limit, offset int) ([]entity.Book, error)
	// Get Book by ISBN
	GetByISBN(ctx context.Context, isbn string) (entity.Book, error)
}


