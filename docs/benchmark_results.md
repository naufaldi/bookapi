-- Benchmark results log for Epic 6.6.2 and 6.6.3
-- Run these after populating with 10k+ books

-- ============================================
-- RESULTS: 6.6.2 EXPLAIN ANALYZE Benchmarks
-- ============================================

-- Run each query and record the execution time:
-- \timing on
-- EXPLAIN ANALYZE [QUERY];

-- Template for documenting results:
/*
Query: [Description]
Planning Time: X.XXX ms
Execution Time: X.XXX ms

Notes:
*/

-- ============================================
-- RESULTS: 6.6.3 FTS Ranking Tuning
-- ============================================

-- Test different weight configurations:

-- Current (default):
-- setweight(title, 'A') || setweight(genre, 'B') || setweight(publisher, 'C') || setweight(description, 'D')

-- Alternative 1 (boost description):
-- setweight(title, 'A') || setweight(genre, 'B') || setweight(description, 'C') || setweight(publisher, 'D')

-- Alternative 2 (boost genre):
-- setweight(title, 'A') || setweight(genre, 'A') || setweight(description, 'B') || setweight(publisher, 'C')

-- Test query:
SELECT 
    b.id,
    b.title,
    b.genre,
    ts_rank(b.search_vector, plainto_tsquery('english', 'science')) as default_rank,
    ts_rank(
        setweight(to_tsvector('english', b.title), 'A') ||
        setweight(to_tsvector('english', b.genre), 'A') ||
        setweight(to_tsvector('english', coalesce(b.description, '')), 'B') ||
        setweight(to_tsvector('english', b.publisher), 'C'),
        plainto_tsquery('english', 'science')
    ) as boosted_rank
FROM books b
WHERE b.search_vector @@ plainto_tsquery('english', 'science')
ORDER BY default_rank DESC
LIMIT 20;

-- ============================================
-- PERFORMANCE TARGETS
-- ============================================

-- Target: < 100ms for any query
-- Target: < 50ms for basic list queries
-- Target: FTS search returns results in < 200ms

-- If queries are slow, consider:
-- 1. Add more indexes
-- 2. Optimize query structure
-- 3. Use cursor pagination instead of offset
-- 4. Add query result caching
