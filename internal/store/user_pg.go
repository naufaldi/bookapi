package store

import (
	"context"
	"errors"
	"strconv"
	"strings"

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
	INSERT INTO users (id, email, username, password_hash, role, is_public)
	VALUES (gen_random_uuid(), $1, $2, $3, COALESCE($4, 'USER'), COALESCE($5, true))
	RETURNING id, role, is_public, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query, user.Email, user.Username, user.Password, user.Role, user.IsPublic).Scan(&user.ID, &user.Role, &user.IsPublic, &user.CreatedAt, &user.UpdatedAt)
}

func (r * UserPG) GetByEmail(ctx context.Context, email string) (entity.User, error) {
	const query = `
	SELECT id, email, username, password_hash, role, bio, location, website, is_public, reading_preferences, last_login_at, created_at, updated_at
	FROM users
	WHERE email = $1
	LIMIT 1
	`
	var user entity.User
	var readingPrefs []byte
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Username, &user.Password, &user.Role,
		&user.Bio, &user.Location, &user.Website, &user.IsPublic,
		&readingPrefs, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows){
			return entity.User{}, usecase.ErrNotFound
		}
		return entity.User{}, err
	}
	if len(readingPrefs) > 0 {
		user.ReadingPreferences = readingPrefs
	}
	return user, nil
}

func (r *UserPG) GetByID(ctx context.Context, id string) (entity.User, error) {
	const query = `
	SELECT id, email, username, password_hash, role, bio, location, website, is_public, reading_preferences, last_login_at, created_at, updated_at
	FROM users WHERE id = $1 LIMIT 1
	`
	var user entity.User
	var readingPrefs []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Username, &user.Password, &user.Role,
		&user.Bio, &user.Location, &user.Website, &user.IsPublic,
		&readingPrefs, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if len(readingPrefs) > 0 {
		user.ReadingPreferences = readingPrefs
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.User{}, usecase.ErrNotFound
		}
		return entity.User{}, err
	}
	return user, nil
}

func (r *UserPG) GetPublicProfile(ctx context.Context, id string) (entity.User, error) {
	const query = `
	SELECT id, username, bio, location, website, is_public, reading_preferences, created_at
	FROM users WHERE id = $1 AND is_public = true LIMIT 1
	`
	var user entity.User
	var readingPrefs []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Bio, &user.Location, &user.Website,
		&user.IsPublic, &readingPrefs, &user.CreatedAt,
	)
	if len(readingPrefs) > 0 {
		user.ReadingPreferences = readingPrefs
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.User{}, usecase.ErrNotFound
		}
		return entity.User{}, err
	}
	return user, nil
}

func (r *UserPG) UpdateProfile(ctx context.Context, userID string, updates map[string]interface{}) error {
	fields := []string{}
	args := []interface{}{}
	argn := 1

	for key, value := range updates {
		switch key {
		case "username", "bio", "location", "website", "is_public", "reading_preferences":
			fields = append(fields, key+" = $"+strconv.Itoa(argn))
			args = append(args, value)
			argn++
		}
	}

	if len(fields) == 0 {
		return nil
	}

	fields = append(fields, "updated_at = now()")
	args = append(args, userID)

	query := "UPDATE users SET " + strings.Join(fields, ", ") + " WHERE id = $" + strconv.Itoa(argn)
	_, err := r.db.Exec(ctx, query, args...)
	return err
}