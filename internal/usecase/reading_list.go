package usecase

import (
	"bookapi/internal/entity"
	"context"
)

type ReadingListRepository interface {
	// Tambah atau ubah status untuk (user, isbn) → upsert.
	UpsertReadingListItem(ctx context.Context, userID string, isbn string, status string) error

	// Ambil daftar buku untuk user + status, dengan pagination, sekaligus total.
	ListReadingListByStatus(ctx context.Context, userID string, status string, limit, offset int) ([]entity.Book, int, error)
}