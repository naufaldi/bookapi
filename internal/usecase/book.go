package usecase

import (
	"bookapi/internal/entity"
	"context"
)

// object for filter
type ListParams struct {
	Genre string
	Publisher string
	Q string
	Sort string
	Desc bool
	Limit int
	Offset int
}

// Repository interface
// Define the contract for fetching books.
type BookRepository interface {
	// List Books with pagination and filters
	List(ctx context.Context, p ListParams) ([]entity.Book, int, error)
	// Get Book by ISBN
	GetByISBN(ctx context.Context, isbn string) (entity.Book, error)
}


