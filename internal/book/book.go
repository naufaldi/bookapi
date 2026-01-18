package book

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a book is not found.
var ErrNotFound = errors.New("book not found")

// Book represents a book entity.
type Book struct {
	ID              string    `json:"id"`
	ISBN            string    `json:"isbn"`
	Title           string    `json:"title"`
	Subtitle        string    `json:"subtitle,omitempty"`
	Genre           string    `json:"genre,omitempty"`
	Publisher       string    `json:"publisher,omitempty"`
	Description     string    `json:"description,omitempty"`
	PublishedDate   string    `json:"published_date,omitempty"`
	PublicationYear *int      `json:"publication_year,omitempty"`
	PageCount       *int      `json:"page_count,omitempty"`
	Language        string    `json:"language,omitempty"`
	CoverURL        *string   `json:"cover_url,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Query defines filters and pagination for listing books.
type Query struct {
	Genre     string
	Genres    []string
	Publisher string
	Q         string
	Search    string
	MinRating *float64
	YearFrom  *int
	YearTo    *int
	Language  string
	Sort      string
	Desc      bool
	Limit     int
	Offset    int
}
