-- +goose Up

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  username TEXT NOT NULL UNIQUE,
  password TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'USER',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT role_valid CHECK (role IN ('USER', 'ADMIN'))
);

CREATE TABLE IF NOT EXISTS books (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  isbn TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  genre TEXT NOT NULL,
  publisher TEXT NOT NULL,
  description TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS book_title_idx ON books(title);
CREATE INDEX IF NOT EXISTS books_created_idx ON books(created_at);
CREATE INDEX IF NOT EXISTS books_publisher_idx ON books(publisher);

CREATE TABLE IF NOT EXISTS user_books (
  user_id UUID NOT NULL,
  book_id UUID NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT user_books_pk PRIMARY KEY (user_id, book_id),
  CONSTRAINT user_books_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT user_books_book_fk FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE,
  CONSTRAINT user_books_status_valid CHECK (status IN ('WISHLIST', 'READING', 'FINISHED'))
);

CREATE INDEX IF NOT EXISTS user_books_user_status_idx ON user_books (user_id, status);

CREATE TABLE IF NOT EXISTS ratings (
  user_id UUID NOT NULL,
  book_id UUID NOT NULL,
  star SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT ratings_pk PRIMARY KEY (user_id, book_id),
  CONSTRAINT ratings_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT ratings_book_fk FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE,
  CONSTRAINT star_range CHECK (star BETWEEN 1 AND 5)
);

CREATE INDEX IF NOT EXISTS ratings_book_idx ON ratings (book_id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS users_set_updated_at ON users;
CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS books_set_updated_at ON books;
CREATE TRIGGER books_set_updated_at
BEFORE UPDATE ON books
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS ratings_set_update_at ON ratings;
CREATE TRIGGER ratings_set_update_at
BEFORE UPDATE ON ratings
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS user_books_set_update_at ON user_books;
CREATE TRIGGER user_books_set_update_at
BEFORE UPDATE ON user_books
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down

DROP TRIGGER IF EXISTS users_set_updated_at ON users;
DROP TRIGGER IF EXISTS books_set_updated_at ON books;
DROP TRIGGER IF EXISTS ratings_set_update_at ON ratings;
DROP TRIGGER IF EXISTS user_books_set_update_at ON user_books;

DROP FUNCTION IF EXISTS set_updated_at();

DROP TABLE IF EXISTS ratings;
DROP TABLE IF EXISTS user_books;
DROP TABLE IF EXISTS books;
DROP TABLE IF EXISTS users;

DROP EXTENSION IF EXISTS pgcrypto;
