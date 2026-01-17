package user

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByEmail(ctx context.Context, email string) (User, error)
	GetByID(ctx context.Context, id string) (User, error)
	UpdateProfile(ctx context.Context, userID string, updates map[string]interface{}) error
	GetPublicProfile(ctx context.Context, userID string) (User, error)
}
