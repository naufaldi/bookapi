package usecase

import (
	"bookapi/internal/entity"
	"context"
	"errors"
)

type UserRepository interface {
	Create(ctx context.Context, u *entity.User) error
	GetByEmail(ctx context.Context, email string) (entity.User, error)
	GetByID(ctx context.Context, id string) (entity.User, error)
}


func NewAlreadyExists() error {
	return errors.New("user already exists")
}

var (
	ErrAlreadyExists = NewAlreadyExists() 
)