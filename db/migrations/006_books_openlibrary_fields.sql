-- +goose Up

-- Migration to add Open Library fields to books table for materialization

-- Add missing fields from Open Library
ALTER TABLE books ADD COLUMN IF NOT EXISTS subtitle TEXT;
ALTER TABLE books ADD COLUMN IF NOT EXISTS published_date TEXT;

-- +goose Down

ALTER TABLE books DROP COLUMN IF EXISTS published_date;
ALTER TABLE books DROP COLUMN IF EXISTS subtitle;
