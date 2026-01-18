# RFC: Phase 2 - Production Readiness & Open Library Catalog

## Context
The `bookapi` project has successfully implemented its first 5 phases of security hardening and basic authentication. As we move towards a production-ready system and look to provide a richer experience for users, we need to address two major areas:
1. **Production Hygiene & Observability**: Ensuring the system is robust, observable, and follows API best practices.
2. **Book Catalog Integration**: Integrating with Open Library to provide a massive, searchable book database, which will also serve as a learning ground for advanced SQL queries, full-text search, and performance tuning at scale.

## Goals
- Achieve "production-ready" runtime hygiene (graceful shutdown, timeouts, structured config).
- Implement a robust observability baseline (request IDs, access logs, panic recovery).
- Standardize the API contract with versioning (`/v1`) and consistent response envelopes.
- Implement a **Catalog** feature that fetches and caches book metadata from Open Library.
- Build a **Cron Ingestion Job** to populate the local database with realistic book data for learning purposes.

## Non-Goals
- Building a full Administrative UI.
- Implementing a distributed asynchronous job system (like Celery or Temporal) - a simple cron/internal trigger is sufficient for now.
- Full OpenTelemetry tracing (Otel) - logs and basic metrics are the priority.
- Generating client SDKs.

## User Stories
### Reader / API Developer
- As a **Reader**, I want to search for books by title, author, or genre across a large catalog so I can find new books to read.
- As an **API Developer**, I want a consistent API versioning and response format so I can easily integrate with the `bookapi`.
- As a **Reader**, I want to see detailed metadata (page counts, languages, cover images) for books I find.

### Operator / System Developer
- As an **Operator**, I want the server to shut down gracefully so that in-flight requests are not abruptly terminated during deployments.
- As a **Developer**, I want to see a unique Request-ID in every log entry and error response so I can easily trace issues in production.
- As an **Operator**, I want the ingestion job to be idempotent and respect Open Library's rate limits so our service remains a good citizen of the web.

## Architecture

### System Flow: Cron Ingestion
The ingestion process will discover books via the Open Library Search API and hydrate detailed metadata via the Books API.

```mermaid
graph TD
    Cron[Cron Job / Trigger] -->|POST /internal/jobs/ingest| API[BookAPI]
    API -->|Search Books| OL_Search[Open Library Search API]
    OL_Search -->|ISBNs / Keys| API
    API -->|Fetch Details| OL_Books[Open Library Books API]
    OL_Books -->|Raw JSON| API
    API -->|Upsert| DB[(PostgreSQL)]
    DB -->|Catalog Data| API
```

### Request Flow: Read-Through Cache
When a user requests a book that isn't in our local catalog, we fetch it from Open Library and cache it.

```mermaid
sequenceDiagram
    participant User
    participant Handler as Catalog Handler
    participant Service as Catalog Service
    participant Repo as Postgres Repo
    participant OL as Open Library API
    participant DB as Postgres

    User->>Handler: GET /v1/catalog/books/{isbn}
    Handler->>Service: GetByISBN(isbn)
    Service->>Repo: FindByISBN(isbn)
    Repo->>DB: SELECT ...
    DB-->>Repo: Result (Found/Not Found)
    
    alt Not Found or Stale
        Repo-->>Service: Not Found / Stale
        Service->>OL: GET /api/books?bibkeys=ISBN:{isbn}...
        OL-->>Service: Book Data (JSON)
        Service->>Repo: Upsert(book)
        Repo->>DB: INSERT/UPDATE ...
    else Found and Fresh
        Repo-->>Service: Book Data
    end
    
    Service-->>Handler: Book Data
    Handler-->>User: JSON Response
```

### Module Design (Aligned to docs/architecture.md)
We will introduce the following modules:
- `internal/catalog/`: Handles local caching and serving of book metadata.
- `internal/platform/openlibrary/`: A dedicated client for Open Library APIs.
- `internal/ingest/`: Logic for the batch ingestion process.
- `internal/httpx/`: Extensions for request IDs, logging, and recovery middleware.

## Data Model
We will add two primary tables to support the catalog:

### `catalog_books`
Stores normalized book data for fast searching and display.
- `isbn13` (PK)
- `title`, `subtitle`, `description`
- `cover_url`
- `published_date`, `publisher`
- `language`, `page_count`
- `search_vector` (TSVECTOR for FTS)
- `updated_at` (to track cache freshness)

### `catalog_sources`
Stores the raw JSON from providers for future re-processing without re-fetching.
- `isbn13` (PK)
- `provider` (e.g., 'OPEN_LIBRARY')
- `raw_json` (JSONB)
- `fetched_at`

## API Design

### Versioning
All new endpoints will be prefixed with `/v1`. Existing endpoints will eventually be migrated or deprecated.

### Standard Response Envelope
All responses will follow this format:
```json
{
  "success": true,
  "data": { ... },
  "meta": { "request_id": "..." }
}
```

### Endpoints
- `GET /v1/catalog/books/{isbn}`: Fetch book details from the catalog.
- `GET /v1/catalog/search?q=...&limit=...&offset=...`: Search the catalog using PostgreSQL Full-Text Search.
- `POST /internal/jobs/ingest`: (Protected) Manually trigger the Open Library ingestion job.

## Open Library Ingestion Spec

### API References
We will use the following Open Library endpoints:
- **Search API**: `https://openlibrary.org/search.json?q={subject}&fields=key,title,author_name,isbn,first_publish_year,language&limit=100`
- **Books API (Data)**: `https://openlibrary.org/api/books?bibkeys=ISBN:{isbn1},ISBN:{isbn2}&jscmd=data&format=json`

### Strategy
1. **Discovery**: Use the Search API to find books by popular subjects (e.g., 'fiction', 'science', 'history').
2. **Hydration**: Collect ISBNs and fetch full details in batches of 50-100 using the Books API.
3. **Storage**: Upsert into `catalog_books` and cache the raw JSON in `catalog_sources`.
4. **Rate Limiting**: 
   - Set a descriptive `User-Agent`: `BookAPI/1.0 (contact: your-email@example.com)`.
   - Implement exponential backoff for 429/5xx errors.
   - Limit to 1 request per second to remain a good citizen.
5. **Freshness**: Use an `updated_at` column to ensure we don't re-fetch a book more than once every 7 days unless forced.

### Incremental Ingestion Semantics (100 today → 200 tomorrow)
We will interpret ingestion limits as **desired total unique rows** in our local database (not "fetch this many every run").

- Example behavior:
  - Day 1 config: `INGEST_BOOKS_MAX=100` → ingest until `catalog_books` contains ~100 unique books.
  - Day 2 config: `INGEST_BOOKS_MAX=200` → ingest only the *missing* unique books to reach ~200 total.
- If the configured max decreases, we **do not delete** existing data; it only affects future ingestion runs.

This makes ingestion deterministic for learning purposes while allowing the dataset to grow over time.

### Run History (compare runs / learning SQL)
In addition to the canonical catalog tables, we will record ingestion runs so we can compare runs and analyze ingestion behavior.

- Add tables:
  - `ingest_runs`: one row per run (started_at, finished_at, config snapshot, counters).
  - `ingest_run_books`: join table `(run_id, isbn13)`.
  - `ingest_run_authors`: join table `(run_id, author_key)`.
- Practical uses:
  - Find which books were newly added in the latest run.
  - Compare two runs (what changed, what was skipped due to freshness).
  - Measure ingestion performance over time (duration, error rates).

### Testing Requirements (Cron/Job Ingestion)
We will add unit tests to ensure the ingestion job works reliably and remains safe to run repeatedly.

- Minimum unit test coverage:
  - **Incremental target behavior**: when max increases (100 → 200), only missing unique rows are added.
  - **Dedupe**: repeated ISBNs/author keys do not create duplicates.
  - **Batching**: Books API bibkeys batching logic is correct.
  - **Freshness**: recently fetched records are skipped unless forced.
  - **Run history**: `ingest_runs` and join tables are recorded consistently.

## Security & Observability

### Runtime Hygiene
- **Graceful Shutdown**: Handle `SIGTERM` and `SIGINT` to allow the server to finish processing requests.
- **HTTP Hardening**: Set `ReadHeaderTimeout: 5 * time.Second` and `MaxHeaderBytes: 1 << 20` (1MB).
- **Structured Config**: Move from scattered `os.Getenv` calls to a single `Config` struct with validation.

### Observability
- **Request ID**: Every request gets a UUID in the `X-Request-Id` header, propagated through logs and responses.
- **Access Logs**: Log `method`, `path`, `status`, `duration_ms`, `request_id`, and `user_id`.
- **Panic Recovery**: Catch panics and return a clean JSON 500 error with the `request_id`.

## Epics and Task Breakdown

### Epic 0: Runtime Hygiene & Config
- [ ] Implement `Config` struct and validation in `cmd/api/main.go`.
- [ ] Implement graceful shutdown using `context.WithCancel` and `http.Server.Shutdown`.
- [ ] Configure `ReadHeaderTimeout` and `MaxHeaderBytes` for the HTTP server.

### Epic 1: Observability Baseline
- [ ] Create `httpx.RequestIDMiddleware`.
- [ ] Create `httpx.AccessLogMiddleware`.
- [ ] Create `httpx.RecoveryMiddleware`.
- [ ] Integrate middlewares into `main.go`.

### Epic 2: API Contract & v1
- [ ] Update `httpx.JSONSuccess` and `httpx.JSONError` to include the standard envelope.
- [ ] Wrap all routes in a `/v1` router or prefix.
- [ ] Fix routing bug: Ensure `PATCH /me/profile` is correctly registered.

### Epic 3: Data Layer Hardening
- [ ] Add context timeouts to all `PostgresRepo` methods.
- [ ] Implement `goose` or similar for migration management.
- [ ] Add missing indexes for `users(email)`, `sessions(user_id)`, and `reading_list(user_id)`.

### Epic 4: Catalog & Open Library Client
- [ ] Implement the `openlibrary.Client` with `User-Agent` and timeout settings.
- [ ] Create migrations for `catalog_books` and `catalog_sources`.
- [ ] Implement `catalog.Service` with read-through caching logic.
- [ ] Implement `GET /v1/catalog/books/{isbn}`.

### Epic 5: Ingestion Job (Cron)
- [ ] Create the ingestion job logic (Discovery -> Hydration -> Upsert).
- [ ] Implement batching and rate-limiting/backoff in the ingest service.
- [ ] Add `POST /internal/jobs/ingest` endpoint (protected by internal secret or admin role).

### Epic 6: SQL & Search Learning (Exercises)
- [ ] Populate the DB with 10k+ books from Open Library.
- [ ] Experiment with `EXPLAIN ANALYZE` on complex search queries.
- [ ] Tune PostgreSQL Full-Text Search weights and ranking (`ts_rank`).
- [ ] Compare performance of offset-based vs cursor-based pagination on large datasets.
