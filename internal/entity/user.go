package entity

import (
	"time"
)

type User struct {
	ID                string          `json:"id"`
	Username          string          `json:"username"`
	Role              string          `json:"role"` // USER, ADMIN
	Email             string          `json:"email"`
	Password          string          `json:"-"`
	Bio               *string         `json:"bio,omitempty"`
	Location          *string         `json:"location,omitempty"`
	Website           *string         `json:"website,omitempty"`
	IsPublic          bool            `json:"is_public"`
	ReadingPreferences []byte          `json:"reading_preferences,omitempty"`
	LastLoginAt       *time.Time      `json:"last_login_at,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}