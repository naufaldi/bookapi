package catalog

import (
	"time"
)

type Book struct {
	ISBN13        string
	Title         string
	Subtitle      string
	Description   string
	CoverURL      string
	PublishedDate string
	Publisher     string
	Language      string
	PageCount     int
	UpdatedAt     time.Time
}

type Author struct {
	Key       string
	Name      string
	BirthDate string
	Bio       string
	UpdatedAt time.Time
}

type Source struct {
	ID         string
	EntityType string // BOOK, AUTHOR
	EntityKey  string
	Provider   string
	RawJSON    []byte
	FetchedAt  time.Time
}
