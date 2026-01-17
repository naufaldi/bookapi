package entity

import "time"

type Book struct {
	ID              string    `json:"id"`
	ISBN            string    `json:"isbn"`
	Title           string    `json:"title"`
	Genre           string    `json:"genre,omitempty"`
	Publisher       string    `json:"publisher,omitempty"`
	Description     string    `json:"description,omitempty"`
	PublicationYear *int      `json:"publication_year,omitempty"`
	PageCount       *int      `json:"page_count,omitempty"`
	Language        string    `json:"language,omitempty"`
	CoverURL        *string   `json:"cover_url,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}