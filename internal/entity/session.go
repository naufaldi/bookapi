package entity

import "time"

type Session struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	RefreshTokenHash string    `json:"-"`
	UserAgent       string    `json:"user_agent"`
	IPAddress       string    `json:"ip_address"`
	RememberMe      bool      `json:"remember_me"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
	LastUsedAt      time.Time `json:"last_used_at"`
}
