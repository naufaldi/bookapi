-- Migration for Catalog and Ingestion history

CREATE TABLE IF NOT EXISTS catalog_books (
    isbn13 VARCHAR(13) PRIMARY KEY,
    title TEXT NOT NULL,
    subtitle TEXT,
    description TEXT,
    cover_url TEXT,
    published_date TEXT,
    publisher TEXT,
    language VARCHAR(10),
    page_count INT,
    search_vector TSVECTOR,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_catalog_books_search_vector ON catalog_books USING GIN(search_vector);

-- Trigger function to update search_vector for catalog_books
CREATE OR REPLACE FUNCTION catalog_books_search_trigger() RETURNS trigger AS $$
BEGIN
  new.search_vector :=
    setweight(to_tsvector('english', coalesce(new.title, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(new.subtitle, '')), 'B') ||
    setweight(to_tsvector('english', coalesce(new.publisher, '')), 'C') ||
    setweight(to_tsvector('english', coalesce(new.description, '')), 'D');
  RETURN new;
END
$$ LANGUAGE plpgsql;


DROP TRIGGER IF EXISTS tsvector_update_catalog ON catalog_books;
CREATE TRIGGER tsvector_update_catalog BEFORE INSERT OR UPDATE
ON catalog_books FOR EACH ROW EXECUTE FUNCTION catalog_books_search_trigger();

CREATE TABLE IF NOT EXISTS catalog_authors (
    key VARCHAR(50) PRIMARY KEY,
    name TEXT NOT NULL,
    birth_date TEXT,
    bio TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(20) NOT NULL, -- 'BOOK' or 'AUTHOR'
    entity_key VARCHAR(50) NOT NULL,
    provider VARCHAR(20) NOT NULL DEFAULT 'OPEN_LIBRARY',
    raw_json JSONB NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(entity_type, entity_key, provider)
);

CREATE TABLE IF NOT EXISTS ingest_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'RUNNING',
    config_books_max INT NOT NULL,
    config_authors_max INT NOT NULL,
    config_subjects TEXT NOT NULL,
    books_fetched INT DEFAULT 0,
    books_upserted INT DEFAULT 0,
    authors_fetched INT DEFAULT 0,
    authors_upserted INT DEFAULT 0,
    error TEXT
);

CREATE TABLE IF NOT EXISTS ingest_run_books (
    run_id UUID REFERENCES ingest_runs(id) ON DELETE CASCADE,
    isbn13 VARCHAR(13) REFERENCES catalog_books(isbn13) ON DELETE CASCADE,
    PRIMARY KEY (run_id, isbn13)
);

CREATE TABLE IF NOT EXISTS ingest_run_authors (
    run_id UUID REFERENCES ingest_runs(id) ON DELETE CASCADE,
    author_key VARCHAR(50) REFERENCES catalog_authors(key) ON DELETE CASCADE,
    PRIMARY KEY (run_id, author_key)
);
