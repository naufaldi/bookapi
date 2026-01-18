-- Migration to add Open Library fields to books table for materialization

-- Add missing fields from Open Library
ALTER TABLE books ADD COLUMN IF NOT EXISTS subtitle TEXT;
ALTER TABLE books ADD COLUMN IF NOT EXISTS published_date TEXT;

-- Note: publication_year, page_count, language, cover_url already added in 004_search_improvements.sql
-- Note: search_vector trigger already exists and covers title/genre/publisher/description
