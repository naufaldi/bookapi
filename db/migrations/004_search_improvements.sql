-- +goose Up

-- Migration for advanced search and book fields

-- Add new fields to books table
ALTER TABLE books ADD COLUMN IF NOT EXISTS publication_year INT;
ALTER TABLE books ADD COLUMN IF NOT EXISTS page_count INT;
ALTER TABLE books ADD COLUMN IF NOT EXISTS language VARCHAR(10) DEFAULT 'en';
ALTER TABLE books ADD COLUMN IF NOT EXISTS cover_url VARCHAR(500);

-- Full-text search support
ALTER TABLE books ADD COLUMN IF NOT EXISTS search_vector TSVECTOR;

-- Create GIN index on search_vector
CREATE INDEX IF NOT EXISTS books_search_idx ON books USING GIN(search_vector);

-- Trigger function to update search_vector
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION books_search_trigger() RETURNS trigger AS $$
BEGIN
  new.search_vector :=
    setweight(to_tsvector('english', coalesce(new.title, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(new.genre, '')), 'B') ||
    setweight(to_tsvector('english', coalesce(new.publisher, '')), 'C') ||
    setweight(to_tsvector('english', coalesce(new.description, '')), 'D');
  RETURN new;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- Create trigger
DROP TRIGGER IF EXISTS tsvector_update ON books;
CREATE TRIGGER tsvector_update BEFORE INSERT OR UPDATE
ON books FOR EACH ROW EXECUTE FUNCTION books_search_trigger();

-- Fuzzy search support for title
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS books_title_trgm_idx ON books USING GIN(title gin_trgm_ops);

-- Other indexes
CREATE INDEX IF NOT EXISTS books_publication_year_idx ON books(publication_year);
CREATE INDEX IF NOT EXISTS books_genre_idx ON books(genre);
CREATE INDEX IF NOT EXISTS books_language_idx ON books(language);

-- Populate search_vector for existing books
UPDATE books SET search_vector =
  setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
  setweight(to_tsvector('english', coalesce(genre, '')), 'B') ||
  setweight(to_tsvector('english', coalesce(publisher, '')), 'C') ||
  setweight(to_tsvector('english', coalesce(description, '')), 'D');

-- +goose Down

DROP INDEX IF EXISTS books_language_idx;
DROP INDEX IF EXISTS books_genre_idx;
DROP INDEX IF EXISTS books_publication_year_idx;
DROP INDEX IF EXISTS books_title_trgm_idx;
DROP INDEX IF EXISTS books_search_idx;

DROP TRIGGER IF EXISTS tsvector_update ON books;
DROP FUNCTION IF EXISTS books_search_trigger();

ALTER TABLE books DROP COLUMN IF EXISTS search_vector;
ALTER TABLE books DROP COLUMN IF EXISTS cover_url;
ALTER TABLE books DROP COLUMN IF EXISTS language;
ALTER TABLE books DROP COLUMN IF EXISTS page_count;
ALTER TABLE books DROP COLUMN IF EXISTS publication_year;
