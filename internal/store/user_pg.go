package store

import (
	"context"
	"errors"

	"bookapi/internal/entity"
	"bookapi/internal/usecase"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserPG struct {
	db * pgxpool.Pool
}

func NewUserPG(db * pgxpool.Pool) * UserPG {
	return &UserPG{db: db}
}

func (r * UserPG) Create(ctx context.Context, user * entity.User) error {
	const query = `
	INSERT INTO users (id, email, username, password, role)
	VALUES (gen_random_uuid(), $1, $2, $3, COALESCE($4, 'USER'))
	RETURNING id, role, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query, user.Email, user.Username, user.Password, user.Role).Scan(&user.ID, &user.Role, &user.CreatedAt, &user.UpdatedAt)
}

func (r * UserPG) GetByEmail(ctx context.Context, email string) (entity.User, error) {
	const query = `
	SELECT id, email, username, password, role, created_at, updated_at
	FROM users
	WHERE email = $1
	LIMIT 1
	`
	var user entity.User
	err := r.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.Username, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows){
			return entity.User{}, usecase.ErrNotFound
		}
		return entity.User{}, err
	}
	return user, nil
}

func (r *UserPG) GetByID(ctx context.Context, id string) (entity.User, error) {
	const query = `
	SELECT id, email, username, password, role, created_at, updated_at
	FROM users WHERE id  = $1 LIMIT 1
	`
	var user entity.User
	err := r.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Email, &user.Username, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows){
			return entity.User{}, usecase.ErrNotFound
		}
		return entity.User{}, err
	}
	return user, nil
}