package ingest

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateRun(ctx context.Context, run *Run) (string, error)
	UpdateRun(ctx context.Context, run *Run) error
	LinkBookToRun(ctx context.Context, runID string, isbn13 string) error
	LinkAuthorToRun(ctx context.Context, runID string, authorKey string) error
}

type PostgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepo(db *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) CreateRun(ctx context.Context, run *Run) (string, error) {
	const sql = `
		INSERT INTO ingest_runs (config_books_max, config_authors_max, config_subjects, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id string
	err := r.db.QueryRow(ctx, sql, run.ConfigBooksMax, run.ConfigAuthorsMax, run.ConfigSubjects, run.Status).Scan(&id)
	return id, err
}

func (r *PostgresRepo) UpdateRun(ctx context.Context, run *Run) error {
	const sql = `
		UPDATE ingest_runs SET
			finished_at = $1,
			status = $2,
			books_fetched = $3,
			books_upserted = $4,
			authors_fetched = $5,
			authors_upserted = $6,
			error = $7
		WHERE id = $8`

	_, err := r.db.Exec(ctx, sql, run.FinishedAt, run.Status, run.BooksFetched, run.BooksUpserted, run.AuthorsFetched, run.AuthorsUpserted, run.Error, run.ID)
	return err
}

func (r *PostgresRepo) LinkBookToRun(ctx context.Context, runID string, isbn13 string) error {
	const sql = `
		INSERT INTO ingest_run_books (run_id, isbn13)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING`
	_, err := r.db.Exec(ctx, sql, runID, isbn13)
	return err
}

func (r *PostgresRepo) LinkAuthorToRun(ctx context.Context, runID string, authorKey string) error {
	const sql = `
		INSERT INTO ingest_run_authors (run_id, author_key)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING`
	_, err := r.db.Exec(ctx, sql, runID, authorKey)
	return err
}
