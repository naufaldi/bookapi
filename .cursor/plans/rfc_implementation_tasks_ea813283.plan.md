---
name: RFC Implementation Tasks
overview: "Break down RFC requirements from docs/brain.md into actionable tasks organized by 5 phases: Security Hardening, Authentication & Sessions, User Profiles, Advanced Search, and API Documentation."
todos:
  - id: phase1-cors
    content: "PHASE 1.1: Implement CORS middleware with whitelisted origins"
    status: completed
  - id: phase1-size
    content: "PHASE 1.2: Implement request size limits middleware (1MB JSON, 10MB CSV)"
    status: completed
  - id: phase1-security
    content: "PHASE 1.3: Implement security headers middleware (X-Content-Type-Options, X-Frame-Options, etc.)"
    status: completed
  - id: phase1-validator
    content: "PHASE 1.4: Add validator/v10 package and implement input validation"
    status: completed
  - id: phase1-password
    content: "PHASE 1.5: Implement password strength validation (min 8 chars, uppercase, lowercase, number, special char)"
    status: completed
  - id: phase2-migration
    content: "PHASE 2.1: Create database migration for sessions and token_blacklist tables"
    status: completed
  - id: phase2-session-entity
    content: "PHASE 2.2: Create session entity and repository (session_pg.go)"
    status: completed
  - id: phase2-blacklist
    content: "PHASE 2.3: Create token blacklist repository (blacklist_pg.go)"
    status: completed
  - id: phase2-jwt-jti
    content: "PHASE 2.4: Update JWT to support JTI (JWT ID) in claims"
    status: completed
  - id: phase2-auth-handler
    content: "PHASE 2.5: Create auth handler with logout and refresh token endpoints"
    status: completed
  - id: phase2-middleware-blacklist
    content: "PHASE 2.6: Update auth middleware to check token blacklist"
    status: completed
  - id: phase2-session-handler
    content: "PHASE 2.7: Create session management endpoints (GET /me/sessions, DELETE /me/sessions/{id})"
    status: completed
  - id: phase2-login-session
    content: "PHASE 2.8: Update login handler to create session and return refresh token"
    status: completed
  - id: phase3-migration-profile
    content: "PHASE 3.1: Create database migration to add profile fields to users table"
    status: completed
  - id: phase3-user-entity
    content: "PHASE 3.2: Update user entity with profile fields (bio, location, website, etc.)"
    status: completed
  - id: phase3-user-repo
    content: "PHASE 3.3: Update user repository with UpdateProfile and GetPublicProfile methods"
    status: completed
  - id: phase3-profile-usecase
    content: "PHASE 3.4: Create profile usecase with statistics computation logic"
    status: pending
  - id: phase3-profile-handler
    content: "PHASE 3.5: Create profile handler (GET /me/profile, GET /users/{id}/profile, PATCH /me/profile)"
    status: pending
  - id: phase4-migration-search
    content: "PHASE 4.1: Create database migration for full-text search (search_vector, publication_year, etc.)"
    status: pending
  - id: phase4-book-entity
    content: "PHASE 4.2: Update book entity with new fields (publication_year, page_count, language, cover_url)"
    status: pending
  - id: phase4-book-repo-search
    content: "PHASE 4.3: Update book repository List method with full-text search and advanced filters"
    status: pending
  - id: phase4-listparams
    content: "PHASE 4.4: Update ListParams with new filter fields (search, genres, min_rating, year_from, year_to)"
    status: pending
  - id: phase4-book-handler-query
    content: "PHASE 4.5: Update book handler to parse new query parameters"
    status: pending
  - id: phase5-response-helpers
    content: "PHASE 5.1: Create response helper functions for consistent JSON format"
    status: completed
  - id: phase5-update-handlers
    content: "PHASE 5.2: Update all handlers to use consistent response format"
    status: pending
  - id: phase5-swagger
    content: "PHASE 5.3: Complete Swagger annotations for all endpoints and regenerate docs"
    status: pending
isProject: false
---

# RFC Implementation Tasks Breakdown

This document breaks down the implementation tasks from `docs/brain.md` based on the current state of `cmd/api/main.go` and the codebase.

## Current Implementation Status

**Already Implemented:**

- ✅ Basic JWT authentication (24h token, no refresh)
- ✅ Basic auth middleware
- ✅ bcrypt password hashing
- ✅ Basic endpoints: register, login, /me, /books, /books/{isbn}, /books/{isbn}/rating
- ✅ Basic pagination and filtering (genre, publisher, simple ILIKE search)
- ✅ Swagger setup (partial annotations)
- ✅ Basic database schema (users, books, ratings, user_books)

**Missing:** All features from PHASE 1-5 as outlined below.

---

## PHASE 1: Basic Security Hardening

### Tasks

#### 1.1 CORS Middleware

- **User Story:**

> **As a** frontend developer

> **I want** the API to accept requests only from trusted origins

> **So that** unauthorized websites cannot access the API and my frontend can make cross-origin requests safely

- **File:** `internal/http/middleware_cors.go` (NEW)
- **Changes:** Implement CORS middleware with whitelisted origins
  - Development: `http://localhost:3000`, `http://localhost:5173`
  - Production: Configurable via `ALLOWED_ORIGINS` env var
  - Methods: GET, POST, PATCH, DELETE, OPTIONS
  - Headers: Content-Type, Authorization
  - Handle OPTIONS preflight requests
- **Update:** `cmd/api/main.go` - Apply CORS middleware to router
- **Unit Test:** `internal/http/middleware_cors_test.go` (NEW)
  - Test allowed origin returns CORS headers
  - Test disallowed origin returns 403 or no CORS headers
  - Test OPTIONS request returns 204 with CORS headers
  - Test CORS headers include correct methods and headers
  - Test credentials allowed for trusted origins

#### 1.2 Request Size Limits Middleware

- **User Story:**

> **As a** system administrator

> **I want** request body size limits enforced

> **So that** the API is protected from DOS attacks and resource exhaustion

- **File:** `internal/http/middleware_size.go` (NEW)
- **Changes:** Implement request size limit middleware
  - JSON endpoints: 1MB default
  - CSV upload: 10MB default
  - Configurable via `MAX_REQUEST_SIZE_MB` env var
  - Return 413 Payload Too Large when exceeded
- **Update:** `cmd/api/main.go` - Apply size limit middleware
- **Unit Test:** `internal/http/middleware_size_test.go` (NEW)
  - Test request under limit passes through
  - Test request over limit returns 413
  - Test different size limits for different endpoints
  - Test Content-Length header handling

#### 1.3 Security Headers Middleware

- **File:** `internal/http/middleware_security.go` (NEW)
- **Changes:** Implement security headers middleware
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `X-XSS-Protection: 1; mode=block`
  - `Content-Security-Policy: default-src 'self'`
  - HSTS (only in production, configurable)
- **Update:** `cmd/api/main.go` - Apply security headers middleware

#### 1.4 Input Validation with validator/v10

- **Dependency:** Add `github.com/go-playground/validator/v10` to `go.mod`
- **File:** `internal/http/validator.go` (NEW)
- **Changes:**
  - Initialize validator instance
  - Create custom validators (ISBN format, password strength)
  - Helper function to format validation errors
- **Update:** `internal/http/user_handler.go`
  - Add validation tags to `registerReq` struct
  - Validate email format, username (3-50 chars), password strength
- **Update:** `internal/http/rating_handler.go`
  - Add validation tags to rating request struct
  - Validate rating (1-5 range)

#### 1.5 Password Strength Validation

- **File:** `internal/auth/password.go` (MODIFY)
- **Changes:** Add `ValidatePasswordStrength` function
  - Minimum 8 characters
  - At least one uppercase letter
  - At least one lowercase letter
  - At least one number
  - At least one special character
- **Update:** `internal/http/user_handler.go` - Call validation in RegisterUser

---

## PHASE 2: Authentication & Session Management

### Database Migrations

#### 2.1 Sessions & Token Blacklist Tables

- **File:** `db/migrations/002_auth_enhancements.sql` (NEW)
- **Changes:** Create tables
  - `sessions` table (id, user_id, refresh_token_hash, user_agent, ip_address, remember_me, expires_at, created_at, last_used_at)
  - `token_blacklist` table (jti, user_id, expires_at, created_at)
  - Indexes on user_id, expires_at for both tables

### Tasks

#### 2.2 Session Entity & Repository

- **File:** `internal/entity/session.go` (NEW)
- **File:** `internal/store/session_pg.go` (NEW)
- **Changes:**
  - Session entity struct
  - Create, GetByTokenHash, ListByUserID, Delete, DeleteByTokenHash methods
  - Auto-cleanup expired sessions

#### 2.3 Token Blacklist Repository

- **File:** `internal/store/blacklist_pg.go` (NEW)
- **Changes:**
  - AddToken, IsBlacklisted, CleanupExpired methods

#### 2.4 JWT Enhancements (JTI support)

- **File:** `internal/auth/jwt.go` (MODIFY)
- **Changes:**
  - Add JTI (JWT ID) to Claims struct
  - GenerateToken should accept/return JTI
  - ParseToken should return JTI from claims

#### 2.5 Auth Handler (Logout, Refresh)

- **File:** `internal/http/auth_handler.go` (NEW)
- **Changes:**
  - `LogoutHandler` - Extract JTI from token, add to blacklist
  - `RefreshTokenHandler` - Validate refresh token, generate new access+refresh tokens
  - `RememberMe` parameter handling (30d vs 90d refresh token TTL)

#### 2.6 Auth Middleware Update (Blacklist Check)

- **File:** `internal/http/middleware_auth.go` (MODIFY)
- **Changes:**
  - Check token blacklist before accepting token
  - Query blacklist repository for JTI

#### 2.7 Session Management Endpoints

- **File:** `internal/http/session_handler.go` (NEW)
- **Changes:**
  - `ListSessionsHandler` - GET /me/sessions
  - `DeleteSessionHandler` - DELETE /me/sessions/{id}
  - Detect current session vs others

#### 2.8 Login Handler Update (Session Creation)

- **File:** `internal/http/user_handler.go` (MODIFY)
- **Changes:**
  - Generate refresh token on login
  - Hash refresh token
  - Create session record in database
  - Extract user_agent and IP from request
  - Return both access_token and refresh_token
  - Support "remember_me" checkbox

#### 2.9 Route Registration

- **File:** `cmd/api/main.go` (MODIFY)
- **Changes:**
  - Add `POST /auth/logout` route
  - Add `POST /auth/refresh` route
  - Add `GET /me/sessions` route (protected)
  - Add `DELETE /me/sessions/{id}` route (protected)
  - Wire up session and blacklist repositories

---

## PHASE 3: User Profile Management

### Database Migrations

#### 3.1 User Profile Fields

- **File:** `db/migrations/003_user_profiles.sql` (NEW)
- **Changes:** Add columns to users table
  - `bio TEXT`
  - `location VARCHAR(255)`
  - `website VARCHAR(500)`
  - `is_public BOOLEAN DEFAULT true`
  - `reading_preferences JSONB`
  - `last_login_at TIMESTAMPTZ`
  - Index on `is_public WHERE is_public = true`

### Tasks

#### 3.2 User Entity Update

- **File:** `internal/entity/user.go` (MODIFY)
- **Changes:** Add profile fields to User struct

#### 3.3 User Repository Update

- **File:** `internal/store/user_pg.go` (MODIFY)
- **Changes:**
  - Update GetByID to include profile fields
  - Add `UpdateProfile` method
  - Add `GetPublicProfile` method (excludes email)

#### 3.4 Profile Usecase

- **File:** `internal/usecase/profile.go` (NEW)
- **Changes:**
  - Business logic for profile updates
  - Statistics computation (books read, avg rating, reviews count)
  - Privacy check logic

#### 3.5 Profile Handler

- **File:** `internal/http/profile_handler.go` (NEW)
- **Changes:**
  - `GetOwnProfile` - GET /me/profile
  - `GetPublicProfile` - GET /users/{id}/profile
  - `UpdateProfile` - PATCH /me/profile
  - Include statistics in responses

#### 3.6 Route Registration

- **File:** `cmd/api/main.go` (MODIFY)
- **Changes:**
  - Add `GET /me/profile` route (protected)
  - Add `GET /users/{id}/profile` route (public)
  - Add `PATCH /me/profile` route (protected)

---

## PHASE 4: Advanced Search & Filtering

### Database Migrations

#### 4.1 Full-Text Search & Book Fields

- **File:** `db/migrations/004_search_improvements.sql` (NEW)
- **Changes:**
  - Add columns to books table: `publication_year INT`, `page_count INT`, `language VARCHAR(10) DEFAULT 'en'`, `cover_url VARCHAR(500)`
  - Add `search_vector TSVECTOR` column
  - Create trigger function `books_search_trigger()` to auto-update search_vector
  - Create trigger `tsvector_update` BEFORE INSERT OR UPDATE
  - Create GIN index on `search_vector`
  - Enable `pg_trgm` extension for fuzzy search
  - Create GIN index on `title gin_trgm_ops`
  - Indexes on `publication_year`, `genre`, `language`

### Tasks

#### 4.2 Book Entity Update

- **File:** `internal/entity/book.go` (MODIFY)
- **Changes:** Add new fields (publication_year, page_count, language, cover_url)

#### 4.3 Book Repository - Advanced Search

- **File:** `internal/store/book_pg.go` (MODIFY)
- **Changes:**
  - Update `List` method to support:
    - Full-text search using `search_vector @@ plainto_tsquery('english', $query)`
    - Multiple genres (comma-separated, OR logic)
    - `min_rating` filter (join with ratings, aggregate AVG)
    - Year range (`year_from`, `year_to`)
    - Language filter
  - Update sorting options:
    - `relevance` - when searching, use `ts_rank(search_vector, query)`
    - `rating_desc` - by average rating
    - `title_asc`, `created_desc` (existing)
  - Update query to include new book fields in SELECT

#### 4.4 ListParams Update

- **File:** `internal/usecase/book.go` (MODIFY)
- **Changes:** Add fields to ListParams struct
  - `Search` (full-text search query)
  - `Genres` (slice for multiple genres)
  - `MinRating` (float64)
  - `YearFrom`, `YearTo` (int)
  - `Language` (string)
  - `Sort` enum (relevance, rating_desc, title_asc, created_desc)

#### 4.5 Book Handler - Query Parsing

- **File:** `internal/http/book_handler.go` (MODIFY)
- **Changes:**
  - Parse new query parameters: `search`, `genre` (comma-separated), `min_rating`, `year_from`, `year_to`, `language`, `sort`
  - Build ListParams with new fields
  - Handle genre comma-separated list

#### 4.6 Migration Script for Existing Books

- **File:** `db/migrations/004b_populate_search_vector.sql` (NEW)
- **Changes:** Update search_vector for all existing books
  - `UPDATE books SET search_vector = ...` for existing rows

---

## PHASE 5: Better API Documentation

### Tasks

#### 5.1 Consistent Response Format

- **File:** `internal/http/response.go` (NEW)
- **Changes:** Helper functions for consistent responses
  - `JSONSuccess(w, data, meta)` - Standard success response
  - `JSONError(w, statusCode, code, message, details)` - Standard error response
  - Response structs: `SuccessResponse`, `ErrorResponse`, `ErrorDetail`

#### 5.2 Update All Handlers

- **Files:** All handler files in `internal/http/`
- **Changes:**
  - Replace direct `json.NewEncoder` calls with `JSONSuccess`/`JSONError`
  - Standardize error codes (VALIDATION_ERROR, NOT_FOUND, UNAUTHORIZED, etc.)
  - Add field-level error details for validation errors

#### 5.3 Enhanced Swagger Annotations

- **Files:** All handler files
- **Changes:** Complete Swagger annotations for all endpoints
  - `@Summary`, `@Description`, `@Tags`
  - `@Param` with all query/path/body parameters
  - `@Success` with example response structures
  - `@Failure` for all possible error responses (400, 401, 404, 422, 500)
  - `@Security` for protected endpoints
  - `@Router` with path and method

#### 5.4 Generate Swagger Docs

- **Command:** Run `swag init` to regenerate `docs/swagger.json` and `docs/swagger.yaml`
- **Verify:** Check Swagger UI at `/swagger/` shows all endpoints

---

## Dependencies to Add

Update `go.mod`:

```bash
go get github.com/go-playground/validator/v10
go mod tidy
```

---

## Testing Strategy

### Unit Tests

- Test all middleware functions
- Test validator functions
- Test password strength validation
- Test session/blacklist repository methods

### Integration Tests

- Test CORS headers in responses
- Test request size limits (413 response)
- Test authentication flow (login → refresh → logout)
- Test session management endpoints
- Test profile CRUD operations
- Test advanced search/filtering queries

### Manual Testing Checklist

- [ ] CORS headers present in browser DevTools
- [ ] Request size limits return 413
- [ ] Security headers present
- [ ] Validation errors return structured format
- [ ] Logout invalidates token
- [ ] Refresh token generates new tokens
- [ ] Session management works
- [ ] Profile endpoints functional
- [ ] Full-text search returns results
- [ ] Advanced filters work correctly
- [ ] Swagger UI shows all endpoints

---

## Implementation Order Recommendation

1. **PHASE 1** (Security Hardening) - Foundation, should be done first
2. **PHASE 2** (Auth & Sessions) - Core functionality enhancement
3. **PHASE 3** (Profiles) - User-facing features
4. **PHASE 4** (Search) - Feature enhancement
5. **PHASE 5** (Documentation) - Polish and consistency

This order ensures security is in place before adding features, and documentation comes last when all features are stable.