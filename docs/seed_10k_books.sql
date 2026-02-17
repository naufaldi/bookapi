-- Seed script to populate database with 10k+ books
-- Run this directly against the database

-- First, check current count
SELECT 'Current books count: ' || COUNT(*)::text FROM books;

-- Generate 10,000 books using set-returning functions
-- This approach uses generate_series to create data without needing external tools

DO $$
DECLARE
    i INT;
    batch_size INT := 1000;
    total INT := 10000;
    genres TEXT[] := ARRAY['Fiction', 'Science Fiction', 'History', 'Science', 'Technology', 'Romance', 'Mystery', 'Biography', 'Philosophy', 'Art'];
    publishers TEXT[] := ARRAY['Penguin', 'HarperCollins', 'Oxford', 'Cambridge', 'MIT Press', 'Springer', 'Wiley', 'Elsevier'];
    languages TEXT[] := ARRAY['en', 'es', 'fr', 'de', 'it', 'pt'];
    titles TEXT[] := ARRAY['The Great Adventure', 'Journey to the Unknown', 'Secrets of the Past', 'Dreams of Tomorrow', 'The Art of Living', 'Beyond the Horizon', 'The Last Chapter', 'A New Beginning', ' Truth', 'WThe Hiddeninds of Change'];
    rand_genre TEXT;
    rand_publisher TEXT;
    rand_lang TEXT;
    rand_title TEXT;
    rand_year INT;
    rand_pages INT;
BEGIN
    -- Loop to insert books in batches
    FOR i IN 1..(total / batch_size) LOOP
        INSERT INTO books (id, isbn, title, subtitle, genre, publisher, description, published_date, publication_year, page_count, language, cover_url, created_at, updated_at)
        SELECT 
            gen_random_uuid(),
            '978-' || LPAD((i * batch_size + generate_series)::TEXT, 10, '0'),
            titles[1 + floor(random() * 10)::int] || ' ' || (i * batch_size + generate_series)::TEXT,
            'A ' || CASE floor(random() * 5)::int WHEN 0 THEN 'fascinating' WHEN 1 THEN 'compelling' WHEN 2 THEN 'thought-provoking' WHEN 3 THEN 'inspiring' ELSE 'remarkable' END || ' tale',
            genres[1 + floor(random() * 10)::int],
            publishers[1 + floor(random() * 8)::int],
            'This book explores the depths of knowledge and presents ideas that challenge conventional thinking. A must-read for curious minds.',
            (2000 + floor(random() * 25)::int)::text || '-01-01',
            2000 + floor(random() * 25)::int,
            100 + floor(random() * 700)::int,
            languages[1 + floor(random() * 6)::int],
            NULL,
            NOW(),
            NOW()
        FROM generate_series(1, batch_size)
        ON CONFLICT (isbn) DO NOTHING;
        
        RAISE NOTICE 'Inserted batch % (%.%)', i, (i-1)*batch_size + 1, i*batch_size;
    END LOOP;
END $$;

-- Verify the count
SELECT 'New books count: ' || COUNT(*)::text FROM books;

-- Update search_vector for the new books (if the column exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'books' AND column_name = 'search_vector') THEN
        UPDATE books 
        SET search_vector = setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
                           setweight(to_tsvector('english', coalesce(genre, '')), 'B') ||
                           setweight(to_tsvector('english', coalesce(publisher, '')), 'C') ||
                           setweight(to_tsvector('english', coalesce(description, '')), 'D')
        WHERE search_vector IS NULL;
        RAISE NOTICE 'Updated search_vector for new books';
    ELSE
        RAISE NOTICE 'search_vector column not found, skipping';
    END IF;
END $$;

-- Show sample data
SELECT id, isbn, title, genre, publisher, publication_year FROM books ORDER BY created_at DESC LIMIT 5;
