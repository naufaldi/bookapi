package entity

import "time"

type Book struct {
	ID        string    `json:"id"`
	ISBN      string    `json:"isbn"`
	Title     string    `json:"title"`
	Genre     string    `json:"genre,omitempty"`
	Publisher string    `json:"publisher,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}