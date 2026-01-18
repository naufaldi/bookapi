package ingest

import (
	"time"
)

type Run struct {
	ID               string
	StartedAt        time.Time
	FinishedAt       *time.Time
	Status           string // RUNNING, COMPLETED, FAILED
	ConfigBooksMax   int
	ConfigAuthorsMax int
	ConfigSubjects   string
	BooksFetched     int
	BooksUpserted    int
	AuthorsFetched   int
	AuthorsUpserted  int
	Error            string
}
