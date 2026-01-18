package book

import (
	"context"
)

// Repository defines the contract for book data storage.
type Repository interface {
	List(ctx context.Context, q Query) ([]Book, int, error)
	GetByISBN(ctx context.Context, isbn string) (Book, error)
	UpsertFromIngest(ctx context.Context, book *Book) error
}
