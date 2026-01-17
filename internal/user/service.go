package user

import (
	"context"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, email, username, hashedPassword string) (User, error) {
	_, err := s.repo.GetByEmail(ctx, email)
	if err == nil {
		return User{}, ErrAlreadyExists
	}

	newUser := &User{
		Email:    email,
		Username: username,
		Password: hashedPassword,
		Role:     "USER",
	}

	if err := s.repo.Create(ctx, newUser); err != nil {
		return User{}, err
	}

	return *newUser, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetByEmail(ctx context.Context, email string) (User, error) {
	return s.repo.GetByEmail(ctx, email)
}

func (s *Service) GetPublicProfile(ctx context.Context, id string) (User, error) {
	return s.repo.GetPublicProfile(ctx, id)
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, updates map[string]any) error {
	return s.repo.UpdateProfile(ctx, userID, updates)
}
