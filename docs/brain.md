# 🧠 Brainstorming: Advanced Book API Features

## 🎯 **FOCUSED PRIORITIES** (Database-Driven, No Cloud Dependencies Yet)

Focus on building robust API with database-backed features. Skip Redis and R2 for now - we'll add those later when scaling.

---

## 📋 **PHASE 1: Basic Security Hardening** (🔥 HIGHEST PRIORITY)

### 1. **CORS Configuration**
- Configure proper CORS headers
- **DON'T** use `Access-Control-Allow-Origin: *` in production
- Allow credentials only for trusted domains
- Whitelist specific origins:
  - Development: `http://localhost:3000`, `http://localhost:5173`
  - Production: Your actual frontend domain
- Configure allowed methods: `GET, POST, PATCH, DELETE, OPTIONS`
- Configure allowed headers: `Content-Type, Authorization`

```go
// Example middleware
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            if isAllowedOrigin(origin, allowedOrigins) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Credentials", "true")
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
            }
            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusNoContent)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### 2. **Request Size Limits**
- Limit request body size to prevent DOS attacks
- **JSON endpoints**: Max 1MB
- **CSV upload endpoint**: Max 10MB
- Implement middleware to check `Content-Length` header
- Return `413 Payload Too Large` for oversized requests

```go
func requestSizeLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
            next.ServeHTTP(w, r)
        })
    }
}
```

### 3. **Security Headers Middleware**
Implement standard security headers to protect against common attacks:

```go
func securityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Prevent MIME sniffing
        w.Header().Set("X-Content-Type-Options", "nosniff")
        
        // Prevent clickjacking
        w.Header().Set("X-Frame-Options", "DENY")
        
        // XSS protection (legacy but still useful)
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        // Content Security Policy
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        
        // HSTS - force HTTPS (only enable in production with HTTPS)
        // w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        
        next.ServeHTTP(w, r)
    })
}
```

### 4. **Input Validation**
- Use `go-playground/validator/v10` library
- Validate **ALL** user inputs
- Common validations:
  - Email format: `validate:"required,email"`
  - ISBN format: Custom validator (10 or 13 digits)
  - Rating range: `validate:"min=1,max=5"`
  - URL format: `validate:"url"`
  - String lengths: `validate:"min=3,max=255"`
  - Required fields: `validate:"required"`

```go
// Example validation setup
import "github.com/go-playground/validator/v10"

var validate = validator.New()

type RegisterRequest struct {
    Username string `json:"username" validate:"required,min=3,max=50"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

func validateRequest(req interface{}) error {
    if err := validate.Struct(req); err != nil {
        return err // Return validation errors
    }
    return nil
}
```

### 5. **Password Security** (Enhance Current Implementation)
- ✅ Verify bcrypt is already being used
- Add password strength validation:
  - Minimum 8 characters
  - At least one uppercase letter
  - At least one lowercase letter
  - At least one number
  - At least one special character
- Implement custom validator:

```go
func validatePasswordStrength(password string) error {
    if len(password) < 8 {
        return errors.New("password must be at least 8 characters")
    }
    // Add regex checks for uppercase, lowercase, number, special char
    // ...
    return nil
}
```

### 6. **SQL Injection Prevention**
- ✅ Already using parameterized queries with pgx ✅
- **Never** concatenate user input into SQL
- Always use placeholders: `$1, $2, $3`
- Whitelist any dynamic columns (sorting, filtering)

---

## 📋 **PHASE 2: Authentication & Session Management** (HIGH PRIORITY)

### 7. **Enhanced Login/Logout System**
- ✅ Login already exists (JWT-based)
- ❌ Need to add:
  - **Logout endpoint**: Token invalidation using **database blacklist**
    - Create `token_blacklist` table
    - Store JWT ID (jti) and expiration
    - Middleware checks blacklist on protected routes
  - **Refresh tokens**: Long-lived refresh + short-lived access tokens
    - Access token: 15 minutes
    - Refresh token: 30 days (stored in database)
    - Endpoint: `POST /auth/refresh`
  - **Session management**: Track active sessions in **database**
    - Create `sessions` table
    - Track: user_id, refresh_token_hash, user_agent, IP, created_at, expires_at
    - Endpoints: `GET /me/sessions`, `DELETE /me/sessions/{id}`
  - **Remember me** functionality
    - Longer refresh token expiry (90 days) when enabled
    - Checkbox on login

---

## 📋 **PHASE 3: User Profile Management** (HIGH PRIORITY)

### 8. **User Profile CRUD**
- **View profile**: 
  - `GET /me/profile` - Own profile
  - `GET /users/{id}/profile` - Public profiles (respects privacy settings)
- **Update profile**: 
  - `PATCH /me/profile` - Update own profile
  - Fields: username, email, bio, location, website, privacy settings
- **Profile fields to add**:
  - `bio` TEXT
  - `location` VARCHAR(255)
  - `website` VARCHAR(500)
  - `is_public` BOOLEAN (privacy toggle)
  - `reading_preferences` JSONB (favorite genres, etc.)
- **Statistics** (computed):
  - Books read count
  - Average rating given
  - Reviews written
  - Join date, last active

---
---

## 📋 **PHASE 4: Advanced Search & Filtering** (HIGH PRIORITY)

### 9. **Full-Text Search with PostgreSQL**
- Use PostgreSQL's built-in full-text search (no external dependencies!)
- Add `search_vector` column to books table (TSVECTOR)
- Enable search on: title, description, author names, genre, publisher
- Features:
  - **Fuzzy matching**: Handle typos with `pg_trgm` extension
  - **Ranking**: Order results by relevance (ts_rank)
  - **Highlighting**: Show matched terms in results
  - **Autocomplete**: Suggest books as user types
  
```sql
-- Add full-text search column
ALTER TABLE books ADD COLUMN search_vector TSVECTOR;

-- Create trigger to auto-update search vector
CREATE FUNCTION books_search_trigger() RETURNS trigger AS $$
BEGIN
  NEW.search_vector := 
    setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(NEW.genre, '')), 'C');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tsvector_update BEFORE INSERT OR UPDATE
  ON books FOR EACH ROW EXECUTE FUNCTION books_search_trigger();

-- Create GIN index for fast search
CREATE INDEX idx_books_search ON books USING GIN(search_vector);

-- Enable fuzzy search extension
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_books_title_trgm ON books USING GIN(title gin_trgm_ops);
```

### 10. **Advanced Filters & Sorting**
- **Filter Endpoints**:
  - `GET /books?genre=fiction,mystery` - Multiple genres (OR logic)
  - `GET /books?min_rating=4.0` - Minimum average rating
  - `GET /books?year_from=2020&year_to=2024` - Publication year range
  - `GET /books?search=harry+potter` - Full-text search
  - `GET /books?publisher=penguin` - Filter by publisher
- **Sorting**:
  - `?sort=rating_desc` - By average rating (high to low)
  - `?sort=title_asc` - Alphabetical
  - `?sort=created_desc` - Recently added
  - `?sort=relevance` - Search relevance (when searching)
- **Pagination**:
  - Cursor-based: `?cursor=uuid&limit=20` (better for large datasets)
  - Offset-based: `?page=1&per_page=20` (simpler, for small datasets)
  - Include metadata: `total_count`, `has_next`, `has_prev`

---

## 📋 **PHASE 5: Better API Documentation** (HIGH PRIORITY)

### 11. **Enhanced Swagger/OpenAPI**
- ✅ Swagger already set up
- Improvements needed:
  - **Complete annotations** for all endpoints
  - **Request/response examples** with realistic data
  - **Error responses** documented (400, 401, 404, 422, 500)
  - **Authentication** flow documented clearly
  - **Search/filter parameters** well-documented
  - **Pagination** examples
  - **Rate limit headers** (when we add them later)

### 12. **API Standards**
- **Consistent response format**:
```json
{
  "success": true,
  "data": { /* payload */ },
  "meta": {
    "pagination": { "page": 1, "per_page": 20, "total": 100 }
  }
}

// Error format
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input",
    "details": [
      { "field": "email", "message": "Invalid email format" }
    ]
  }
}
```

- **HTTP Status Codes**:
  - `200 OK` - Successful GET
  - `201 Created` - Resource created
  - `204 No Content` - Successful DELETE
  - `400 Bad Request` - Validation errors
  - `401 Unauthorized` - Not authenticated
  - `403 Forbidden` - Not authorized
  - `404 Not Found` - Resource doesn't exist
  - `422 Unprocessable Entity` - Business logic error
  - `429 Too Many Requests` - Rate limited (future)
  - `500 Internal Server Error` - Server error

---

## 📋 **PHASE 6: New Database Tables** (HIGH PRIORITY)

### 13. **Enhanced Schema Design**

#### Authors Table (Many-to-Many with Books)
```sql
CREATE TABLE authors (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  bio TEXT,
  born_date DATE,
  died_date DATE,
  nationality VARCHAR(100),
  website VARCHAR(500),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_authors_name ON authors(name);

CREATE TABLE book_authors (
  book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  author_id UUID NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
  author_role VARCHAR(50) DEFAULT 'AUTHOR' CHECK (author_role IN ('AUTHOR', 'CO_AUTHOR', 'EDITOR', 'TRANSLATOR')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (book_id, author_id)
);

CREATE INDEX idx_book_authors_book_id ON book_authors(book_id);
CREATE INDEX idx_book_authors_author_id ON book_authors(author_id);
```

#### Reviews Table (Separate from Ratings)
```sql
CREATE TABLE reviews (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  title VARCHAR(255),
  content TEXT NOT NULL,
  is_spoiler BOOLEAN DEFAULT false,
  helpful_count INT DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, book_id)
);

CREATE INDEX idx_reviews_book_id ON reviews(book_id, created_at DESC);
CREATE INDEX idx_reviews_user_id ON reviews(user_id, created_at DESC);
```

#### Reading Progress Table
```sql
CREATE TABLE reading_progress (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  current_page INT NOT NULL DEFAULT 0,
  total_pages INT,
  percentage DECIMAL(5,2) GENERATED ALWAYS AS (
    CASE 
      WHEN total_pages > 0 THEN (current_page::decimal / total_pages * 100)
      ELSE 0
    END
  ) STORED,
  started_at TIMESTAMPTZ,
  last_read_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, book_id)
);

CREATE INDEX idx_reading_progress_user_id ON reading_progress(user_id, last_read_at DESC);
```

#### Sessions & Token Management (Database-Backed)
```sql
CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  refresh_token_hash VARCHAR(255) NOT NULL UNIQUE,
  user_agent VARCHAR(500),
  ip_address INET,
  remember_me BOOLEAN DEFAULT false,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_used_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id, created_at DESC);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Token blacklist for JWT invalidation (logout)
CREATE TABLE token_blacklist (
  jti VARCHAR(255) PRIMARY KEY,
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_token_blacklist_expires_at ON token_blacklist(expires_at);

-- Cleanup job: DELETE FROM token_blacklist WHERE expires_at < now();
```

#### Extended User Table
```sql
ALTER TABLE users ADD COLUMN bio TEXT;
ALTER TABLE users ADD COLUMN location VARCHAR(255);
ALTER TABLE users ADD COLUMN website VARCHAR(500);
ALTER TABLE users ADD COLUMN is_public BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN reading_preferences JSONB;
ALTER TABLE users ADD COLUMN last_login_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN email_verified BOOLEAN DEFAULT false;
```

#### Add Publication Year to Books
```sql
ALTER TABLE books ADD COLUMN publication_year INT;
ALTER TABLE books ADD COLUMN page_count INT;
ALTER TABLE books ADD COLUMN language VARCHAR(10) DEFAULT 'en';
ALTER TABLE books ADD COLUMN cover_url VARCHAR(500);

CREATE INDEX idx_books_publication_year ON books(publication_year);
CREATE INDEX idx_books_language ON books(language);
```

---

## 📋 **PHASE 7: CSV Import Feature** (HIGH PRIORITY)

### 14. **Bulk Import Books from CSV**
- **Endpoint**: `POST /admin/books/import`
  - Requires admin role
  - Accept CSV file upload
  - Parse and validate data
  - Insert/update books in batch
  - Return summary (success count, errors)
  
- **CSV Format Expected**:
```csv
isbn,title,genre,publisher,description,publication_year,page_count,language,author_name
978-0-123456-78-9,Sample Book,Fiction,Penguin,A great book,2020,350,en,John Doe
```

- **Features**:
  - Validate ISBN format
  - Check for duplicates (skip or update)
  - Create authors if they don't exist
  - Link books to authors automatically
  - Transaction-based (all or nothing, or partial with error log)
  - Progress reporting for large files

- **Implementation**:
  - Use Go's `encoding/csv` package
  - Stream processing for large files (don't load all in memory)
  - Use database transactions
  - Batch inserts (insert 100-500 rows at a time)

---

## 📋 **PHASE 8: Performance Optimizations** (HIGH PRIORITY)

### 15. **Database Optimizations**

#### Indexing Strategy
```sql
-- Books table indexes (beyond what we already have)
CREATE INDEX idx_books_genre_year ON books(genre, publication_year DESC);
CREATE INDEX idx_books_rating ON books((
  SELECT AVG(star) FROM ratings WHERE ratings.book_id = books.id
));

-- User books indexes
CREATE INDEX idx_user_books_status_updated ON user_books(user_id, status, updated_at DESC);

-- Ratings indexes
CREATE INDEX idx_ratings_star ON ratings(star); -- For filtering by rating

-- Composite index for common queries
CREATE INDEX idx_books_genre_lang_year ON books(genre, language, publication_year DESC);
```

#### Query Optimization
- Use `EXPLAIN ANALYZE` for all queries
- Avoid N+1 queries:
  - Load authors with books in single query (JOIN)
  - Preload ratings when listing books
- Use CTEs (Common Table Expressions) for complex queries
- Materialized views for expensive aggregations (if needed)

#### Connection Pooling
- ✅ Already using `pgxpool` 
- Tune pool settings:
  - Max connections: 25-50
  - Min connections: 5
  - Max idle time: 30 minutes
  - Health check period: 1 minute

### 16. **API Performance**
- **Response Compression**: gzip middleware
- **Pagination**: Limit max page size (e.g., 100 items)
- **Field Selection**: Allow clients to request specific fields (`?fields=id,title,genre`)
- **HTTP Caching Headers**:
  - `Cache-Control` for public endpoints
  - `ETag` for conditional requests
  - `Last-Modified` headers

---

## 📋 **PHASE 9: Advanced Security & Audit** (MEDIUM PRIORITY)

### 17. **HTML Sanitization**
- Sanitize HTML if accepting rich text (use `bluemonday`)
- Prevent XSS attacks in reviews/comments

### 18. **Audit Logging**
- Log all authentication attempts (success/failure)
- Log all data modifications (who changed what, when)
- Log admin actions
- Store in separate `audit_logs` table

---

## 📋 **PHASE 10: Testing & Quality** (HIGH PRIORITY)

### 19. **Testing Strategy**

#### Unit Tests
- Test all business logic in `internal/usecase`
- Test all database queries in `internal/store`
- Aim for 80%+ code coverage
- Use table-driven tests

#### Integration Tests
- Test full HTTP request/response cycle
- Use `testcontainers-go` for real PostgreSQL in tests
- Test authentication flow
- Test error scenarios (404, 401, validation errors)

#### Testing Tools
- `testing` (standard library)
- `testify/assert` for assertions
- `testify/mock` for mocking
- `testcontainers-go` for database tests
- `httptest` for HTTP handler tests

#### Test Organization
```
internal/
  store/
    book_pg.go
    book_pg_test.go  # Unit tests for DB layer
  usecase/
    book_service.go
    book_service_test.go  # Unit tests for business logic
  http/
    book_handler.go
    book_handler_test.go  # Integration tests
```

### 20. **Code Quality**
- Run `go vet ./...` (already in AGENTS.md ✅)
- Run `go fmt ./...` (already in AGENTS.md ✅)
- Optional: Use `golangci-lint run` for advanced linting
- Pre-commit hooks to run tests and linters

---

## 📋 **PHASE 11: Observability** (HIGH PRIORITY)

### 21. **Structured Logging**
- Use Go's `slog` package (stdlib in Go 1.21+)
- OR use `zerolog` for better performance
- Log levels: DEBUG, INFO, WARN, ERROR
- Include context:
  - Request ID (generate UUID for each request)
  - User ID (from JWT)
  - Endpoint path
  - Duration
  - Status code

```go
logger.Info("user logged in",
    slog.String("request_id", requestID),
    slog.String("user_id", userID),
    slog.String("ip", clientIP),
    slog.Duration("duration", elapsed),
)
```

### 22. **Request/Response Logging Middleware**
- Log all incoming requests:
  - Method, Path, Query params
  - User agent, IP address
  - Request ID
- Log responses:
  - Status code
  - Response time
  - Error messages (if any)

### 23. **Health Checks**
- ✅ `/healthz` already exists
- ✅ `/readyz` already exists (checks DB)
- Enhance `/readyz`:
  - Check database connection
  - Check database migration status
  - Return 503 if not ready

### 24. **Metrics** (Future: Prometheus)
- For now, log important metrics:
  - Request count per endpoint
  - Request duration percentiles
  - Error rates
  - Database query duration
  - Active sessions count
- Later: Export to Prometheus format

---

## 🎯 **IMPLEMENTATION PRIORITY ORDER**

### Week 1: Basic Security
1. ✅ Implement CORS middleware
2. ✅ Add request size limit middleware
3. ✅ Add security headers middleware
4. ✅ Set up input validation with validator/v10
5. ✅ Add password strength validation
6. ✅ Test all security measures

### Week 2-3: Authentication & Sessions
7. ✅ Create migration for sessions and token_blacklist tables
8. ✅ Implement refresh token logic
9. ✅ Build logout endpoint with blacklist
10. ✅ Add session management endpoints
11. ✅ Add remember me functionality
12. ✅ Write tests for auth flow

### Week 4: User Profiles
13. ✅ Migrate users table (add bio, location, etc.)
14. ✅ Build profile endpoints (GET, PATCH)
15. ✅ Add privacy settings logic
16. ✅ Write tests for profile management

### Week 5: Database Schema Expansion
17. ✅ Create authors table
18. ✅ Migrate existing books to have authors
19. ✅ Create reviews table
20. ✅ Create reading_progress table
21. ✅ Add new columns to books table

### Week 6: Search & Filtering
22. ✅ Add full-text search to books
23. ✅ Implement advanced filters
24. ✅ Add sorting options
25. ✅ Improve pagination

### Week 7: CSV Import
26. ✅ Build CSV parser
27. ✅ Implement bulk import endpoint
28. ✅ Add validation and error handling
29. ✅ Test with large datasets

### Week 8: Performance & Advanced Security
30. ✅ Add database indexes
31. ✅ Optimize queries with EXPLAIN ANALYZE
32. ✅ Add audit logging
33. ✅ Test performance with large datasets

### Week 9-10: Testing & Observability
34. ✅ Write comprehensive tests
35. ✅ Set up structured logging
36. ✅ Improve error handling
37. ✅ Enhance API documentation

---

## 🛠️ **TECHNOLOGY STACK** (Database-Focused)

### Core (Keep Current ✅)
- **Language**: Go 1.21+
- **Web Framework**: `net/http` (stdlib)
- **Database**: PostgreSQL 15+
- **DB Driver**: `pgx/v5`
- **Auth**: JWT

### New Packages to Add
- **Validation**: `github.com/go-playground/validator/v10`
- **Logging**: `log/slog` (stdlib) or `github.com/rs/zerolog`
- **Password**: Already using bcrypt? (verify)
- **CSV**: `encoding/csv` (stdlib)
- **Testing**: `github.com/stretchr/testify`
- **Test DB**: `github.com/testcontainers/testcontainers-go`
- **Security**: `github.com/microcosm-cc/bluemonday` (HTML sanitization)
- **Config**: Current approach with env vars is fine, or add `github.com/spf13/viper`

### Deferred (For Later Scaling)
- ❌ Redis - defer until we need caching/rate limiting
- ❌ Cloudflare R2 - defer until we need image storage
- ❌ Message Queue - defer until we need async jobs
- ❌ Email Service - defer until we need notifications

---

## 📊 **EXPECTED DATABASE SIZE**

For learning purposes, let's populate:
- **Books**: 10,000 - 50,000 records (realistic library)
- **Authors**: 5,000 - 20,000 records
- **Users**: 100 - 1,000 (for testing)
- **Ratings**: 50,000 - 200,000
- **Reviews**: 10,000 - 50,000
- **User Books**: 100,000+ (wishlists, reading, finished)

This will allow testing:
- Query performance at scale
- Search relevance
- Pagination edge cases
- Index effectiveness

---

## 🎓 **LEARNING OUTCOMES**

By building this database-first API, you'll learn:

1. **JWT Auth** - Refresh tokens, logout, session management
2. **PostgreSQL Advanced** - Full-text search, indexes, JSONB, generated columns
3. **Clean Architecture** - Proper layering, testability
4. **API Design** - RESTful patterns, pagination, filtering, error handling
5. **Database Design** - Normalization, relationships, constraints
6. **Performance** - Query optimization, indexing strategies, EXPLAIN ANALYZE
7. **Security** - Input validation, SQL injection prevention, password policies
8. **Testing** - Unit, integration, table-driven tests
9. **CSV Processing** - Bulk imports, streaming, transactions
10. **Observability** - Structured logging, health checks, metrics planning
11. **Documentation** - OpenAPI/Swagger, code comments

---

## ✅ **NEXT STEPS**

1. **Review this updated brain.md** - Confirm these are the right priorities
2. **Create detailed RFC** for the new features (Phase 1-10)
3. **Design database migrations** - SQL scripts for all new tables/columns
4. **Create implementation plan** with:
   - User stories for each feature
   - Acceptance criteria
   - Task breakdown
   - Test plan
5. **Set up testing infrastructure** - testcontainers, test helpers
6. **Start with Phase 1** - Authentication enhancements

**Let me know when you're ready to proceed, and I'll create the detailed RFC and implementation plan! 🚀**
