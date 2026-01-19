# 📖 Personal Book Tracking API

A REST API for tracking books you've read, want to read, or are currently reading. This project demonstrates clean architecture, PostgreSQL integration, and secure API design in Go.

## 🛠 Tech Stack

- **Language**: Go 1.25.1
- **Database**: PostgreSQL with `pgx/v5`
- **Authentication**: JWT (JSON Web Tokens) with session-based refresh and blacklist support
- **Documentation**: Swagger/OpenAPI 2.0 via `swaggo`
- **Validation**: `github.com/go-playground/validator/v10`
- **Architecture**: Clean Architecture (Entities, Usecases, Handlers, Repositories)

## ✨ Features

- **Books Management**: Advanced search (Full-Text Search + Fuzzy matching), filtering by genre/publisher/rating/year, and pagination.
- **Authentication**: Secure registration, login with "remember me", refresh tokens, and logout with token blacklisting.
- **Session Management**: Track and revoke active sessions across devices.
- **User Profiles**: Manage personal profiles (bio, location, preferences) with privacy controls.
- **Reading Lists**: Track books in WISHLIST, READING, or FINISHED lists.
- **Ratings**: Rate books (1-5 stars) and view average ratings.
- **Security**: CORS protection, request size limits, and standard security headers (CSP, HSTS, etc.).

## 🚀 Getting Started

### Prerequisites

- Go 1.25.1 or later
- PostgreSQL 12+
- `swag` (for documentation): `go install github.com/swaggo/swag/cmd/swag@latest`

### Local Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/naufaldi/bookapi.git
   cd bookapi
   ```

2. **Configure environment variables**
   Create a `.env.local` file:
   ```env
   APP_ADDR=:8080
   DB_DSN=postgres://postgres:postgres@localhost:5432/booklibrary
   JWT_SECRET=your-super-secret-key-at-least-32-chars
   ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
   MAX_REQUEST_SIZE_MB=1
   ENABLE_HSTS=false

   # Ingestion
   INGEST_ENABLED=true
   INGEST_BOOKS_MAX=100
   INGEST_AUTHORS_MAX=100
   INTERNAL_JOBS_SECRET=your-internal-cron-secret
   ```

3. **Database Setup**
   ```bash
   # Create database
   createdb booklibrary

   # Initialize base schema
   psql -d booklibrary -f db/schema.sql

   # Run migrations using goose CLI
   go run ./cmd/migrate -command=up
   
   # Or manually run migrations in order:
   # psql -d booklibrary -f db/migrations/002_auth_enhancements.sql
   # psql -d booklibrary -f db/migrations/003_user_profiles.sql
   # psql -d booklibrary -f db/migrations/004_search_improvements.sql
   # psql -d booklibrary -f db/migrations/004b_populate_search_vector.sql
   # psql -d booklibrary -f db/migrations/005_catalog_and_ingestion.sql
   # psql -d booklibrary -f db/migrations/006_books_openlibrary_fields.sql
   # psql -d booklibrary -f db/migrations/007_add_missing_indexes.sql
   ```

4. **Run the application**
   ```bash
   go run ./cmd/api
   ```

## 📋 API Documentation

The API uses Swagger for documentation. Once the server is running, you can access the interactive UI at:
`http://localhost:8080/swagger/`

### Regenerate Documentation
If you modify handler annotations, regenerate the Swagger files:
```bash
swag init -g cmd/api/main.go -o docs
```

## 📥 Catalog Ingestion (Open Library)

The project includes an ingestion job to populate a local catalog with data from Open Library. Follow these steps to run it locally:

### 1. Configure Environment Variables
Ensure your `.env.local` has ingestion enabled and a secret set:
```env
INGEST_ENABLED=true
INGEST_BOOKS_MAX=100
INTERNAL_JOBS_SECRET=your-internal-cron-secret
```

### 2. Run the API Server
Start the server if it's not already running:
```bash
go run ./cmd/api
```

### 3. Trigger Ingestion
In a new terminal, use `curl` to trigger the job. Replace `your-internal-cron-secret` with your actual secret:
```bash
curl -X POST http://localhost:8080/internal/jobs/ingest \
  -H "X-Internal-Secret: your-internal-cron-secret"
```

### 4. Check if it worked
Verify the data has been persisted in your database:
```sql
-- Check if catalog_books is populated
SELECT count(*) FROM catalog_books;

-- Check the status of the latest ingestion run
SELECT status, books_upserted, authors_upserted, error 
FROM ingest_runs 
ORDER BY started_at DESC LIMIT 1;
```

### Ingestion Features
- **Incremental**: Only fetches what is needed to reach the target set in `INGEST_BOOKS_MAX`.
- **Run History**: Tracks every ingestion run in `ingest_runs` for audit and comparison.
- **Safe**: Implements rate limiting (1 req/s) and exponential backoff.

## 📁 Project Structure

```text
bookapi/
├── cmd/api/            # Application entry point
├── db/                 # Database schema and migrations
├── docs/               # Swagger documentation and RFCs
├── internal/           # Private application code
│   ├── auth/           # JWT and password logic
│   ├── entity/         # Domain models
│   ├── http/           # HTTP handlers and middleware
│   ├── store/          # Data access (Postgres)
│   ├── usecase/        # Business logic
│   └── testutil/       # Test helpers
└── go.mod              # Dependencies
```

## 🏗 Development

### Testing
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...
```

### Linting
```bash
go vet ./...
```

## 🔧 Database Migrations

The project uses [goose](https://github.com/pressly/goose) for migration management. Use the migration CLI:

```bash
# Apply all pending migrations
go run ./cmd/migrate -command=up

# Rollback last migration
go run ./cmd/migrate -command=down

# Check migration status
go run ./cmd/migrate -command=status

# Create a new migration
go run ./cmd/migrate -command=create -name=add_new_feature
```

### VPS Deployment

For production deployments on VPS with Docker:

```bash
# In your deployment script or docker-compose
docker exec bookapi-api go run ./cmd/migrate -command=up
```

## 📊 Observability & Monitoring

The API includes built-in observability features for production monitoring:

### Health & Readiness Checks

- **Liveness**: `GET /healthz` - Returns `200 OK` with "ok" body
- **Readiness**: `GET /readyz` - Returns `200 OK` if DB is reachable, `503` otherwise

Use these endpoints for Docker healthchecks and load balancer probes.

### Request Tracing

Every request includes a unique `X-Request-Id` header and `meta.request_id` in JSON responses. This enables easy log correlation:

```bash
# View API logs (Docker Compose)
docker compose logs -f api

# Filter by request ID
docker compose logs api | grep "request_id=abc123-def456"

# Find errors (4xx/5xx)
docker compose logs api | grep -E "status=[45][0-9][0-9]"

# Find slow requests (>1000ms)
docker compose logs api | grep "duration_ms=[0-9][0-9][0-9][0-9]"

# Find panics
docker compose logs api | grep "panic recovered"
```

### Caddy Configuration

To preserve request IDs when using Caddy as reverse proxy, configure:

```caddyfile
your-api.example.com {
    reverse_proxy api:8080 {
        header_up X-Request-Id {http.request.id}
    }
    
    log {
        output stdout
        format json
    }
}
```

### Access Logs

Access logs include:
- `method` - HTTP method
- `path` - Request path
- `status` - HTTP status code
- `duration_ms` - Request duration in milliseconds
- `request_id` - Unique request identifier
- `user_id` - Authenticated user ID (if present)

## 🤝 Contributing

This is a learning project. Feel free to report issues or suggest improvements.

## 📄 License

Educational purpose only.
