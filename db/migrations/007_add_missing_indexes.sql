-- +goose Up

-- Add missing indexes for performance

-- Index on users.email (already has UNIQUE constraint, but explicit index helps with lookups)
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Index on user_books.user_id (for reading list queries)
-- Note: user_books_user_status_idx already exists, but a simple user_id index helps with general queries
CREATE INDEX IF NOT EXISTS idx_user_books_user_id ON user_books(user_id);

-- +goose Down

DROP INDEX IF EXISTS idx_user_books_user_id;
DROP INDEX IF EXISTS idx_users_email;
