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

   # Run migrations in order
   psql -d booklibrary -f db/migrations/002_auth_enhancements.sql
   psql -d booklibrary -f db/migrations/003_user_profiles.sql
   psql -d booklibrary -f db/migrations/004_search_improvements.sql
   psql -d booklibrary -f db/migrations/004b_populate_search_vector.sql
   psql -d booklibrary -f db/migrations/005_catalog_and_ingestion.sql
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

The project includes an ingestion job to populate a local catalog with data from Open Library. This is designed to be triggered by an external cron job.

### Triggering Manually

You can trigger the ingestion process using `curl`. Replace `your-internal-cron-secret` with the value of your `INTERNAL_JOBS_SECRET` env var:

```bash
curl -X POST http://localhost:8080/internal/jobs/ingest \
  -H "X-Internal-Secret: your-internal-cron-secret"
```

### Ingestion Features
- **Incremental**: Only fetches what is needed to reach the target set in `INGEST_BOOKS_MAX`.
- **Run History**: Tracks every ingestion run in `ingest_runs` for audit and comparison.
- **Safe**: Implements rate limiting (1 req/s) and exponential backoff.

### Verification Queries
```sql
-- Check ingestion summary
SELECT * FROM ingest_runs ORDER BY started_at DESC LIMIT 1;

-- See newly added books
SELECT isbn13, title FROM catalog_books ORDER BY updated_at DESC LIMIT 10;
```

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

## 🤝 Contributing

This is a learning project. Feel free to report issues or suggest improvements.

## 📄 License

Educational purpose only.
