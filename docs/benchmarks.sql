-- Benchmark queries for Epic 6.6.2 and 6.6.3
-- Run these queries with EXPLAIN ANALYZE to measure performance

-- ============================================
-- 6.6.2: EXPLAIN ANALYZE Benchmarks
-- ============================================

-- Benchmark 1: Basic book list (no filters)
EXPLAIN ANALYZE 
SELECT b.id, b.isbn, b.title, b.genre, b.publisher, b.publication_year
FROM books b
ORDER BY b.created_at DESC
LIMIT 20;

-- Benchmark 2: Full-text search
EXPLAIN ANALYZE 
SELECT b.id, b.isbn, b.title, b.genre
FROM books b
WHERE b.search_vector @@ plainto_tsquery('english', 'science fiction')
ORDER BY ts_rank(b.search_vector, plainto_tsquery('english', 'science fiction')) DESC
LIMIT 20;

-- Benchmark 3: Filter by genre
EXPLAIN ANALYZE 
SELECT b.id, b.isbn, b.title, b.genre
FROM books b
WHERE b.genre = 'Fiction'
ORDER BY b.title ASC
LIMIT 20;

-- Benchmark 4: Filter by year range
EXPLAIN ANALYZE 
SELECT b.id, b.isbn, b.title, b.publication_year
FROM books b
WHERE b.publication_year >= 2000 AND b.publication_year <= 2020
ORDER BY b.publication_year DESC
LIMIT 20;

-- Benchmark 5: With rating join
EXPLAIN ANALYZE 
SELECT b.id, b.title, AVG(r.star) as avg_rating, COUNT(r.id) as rating_count
FROM books b
LEFT JOIN ratings r ON r.book_id = b.id
GROUP BY b.id, b.title
HAVING COUNT(r.id) > 0
ORDER BY avg_rating DESC
LIMIT 20;

-- Benchmark 6: Offset pagination (slow for large offsets)
EXPLAIN ANALYZE 
SELECT b.id, b.title
FROM books b
ORDER BY b.created_at DESC
LIMIT 20 OFFSET 1000;

-- Benchmark 7: Cursor-based pagination
EXPLAIN ANALYZE 
SELECT b.id, b.title, b.created_at
FROM books b
WHERE b.created_at < (SELECT created_at FROM books WHERE id = 'some-uuid-here')
ORDER BY b.created_at DESC
LIMIT 20;

-- ============================================
-- 6.6.3: FTS Ranking Tuning
-- ============================================

-- Test current ranking with weights
SELECT 
    b.id,
    b.title,
    b.author,
    ts_rank(b.search_vector, plainto_tsquery('english', 'artificial intelligence')) as rank
FROM books b
WHERE b.search_vector @@ plainto_tsquery('english', 'artificial intelligence')
ORDER BY rank DESC
LIMIT 10;

-- Test with custom weights (title more important)
SELECT 
    b.id,
    b.title,
    ts_rank(
        setweight(to_tsvector('english', b.title), 'A') ||
        setweight(to_tsvector('english', b.author), 'B') ||
        setweight(to_tsvector('english', COALESCE(b.description, '')), 'C'),
        plainto_tsquery('english', 'artificial intelligence')
    ) as rank
FROM books b
WHERE b.search_vector @@ plainto_tsquery('english', 'artificial intelligence')
ORDER BY rank DESC
LIMIT 10;

-- Compare relevance scoring across different query terms
SELECT 
    b.id,
    b.title,
    b.genre,
    ts_rank(b.search_vector, plainto_tsquery('english', 'love')) as rank_love,
    ts_rank(b.search_vector, plainto_tsquery('english', 'war')) as rank_war,
    ts_rank(b.search_vector, plainto_tsquery('english', 'technology')) as rank_tech
FROM books b
WHERE b.search_vector @@ plainto_tsquery('english', 'love')
   OR b.search_vector @@ plainto_tsquery('english', 'war')
   OR b.search_vector @@ plainto_tsquery('english', 'technology')
ORDER BY (rank_love + rank_war + rank_tech) DESC
LIMIT 20;

-- Check index usage
SELECT 
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
AND tablename IN ('books', 'ratings', 'user_books')
ORDER BY tablename, indexname;
