# BookAPI - Personal Book Tracking API

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.25-blue?logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?logo=postgresql)
![License](https://img.shields.io/badge/License-MIT-green)
![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen)

A production-ready REST API for tracking books you've read, want to read, or are currently reading. Built with Go, PostgreSQL, and Docker.

[Features](#-features) • [Quick Start](#-quick-start) • [API Documentation](#-api-documentation) • [Deployment](#-deployment) • [Contributing](#-contributing)

</div>

## 📚 About

BookAPI is a personal book tracking application that demonstrates clean architecture principles, secure API design, and modern DevOps practices. It features advanced search capabilities, user authentication with session management, and integration with Open Library for catalog ingestion.

## ✨ Features

| Category | Features |
|----------|----------|
| **Authentication** | JWT with refresh tokens, session tracking, token blacklisting |
| **Books** | ISBN-based lookup, advanced search, fuzzy matching, filtering |
| **User Profiles** | Bio, location, privacy controls, reading statistics |
| **Reading Lists** | WISHLIST, READING, FINISHED status tracking |
| **Ratings** | 1-5 star ratings with average calculation |
| **Catalog** | Full-text search via PostgreSQL GIN index |
| **Ingestion** | Open Library API integration with rate limiting |
| **Security** | CORS, rate limiting, security headers, bcrypt passwords |
| **Observability** | Health checks, request tracing, access logging, Prometheus metrics |

## 🛠 Tech Stack

- **Language**: Go 1.25
- **Database**: PostgreSQL 15 with pgx/v5
- **Authentication**: JWT (golang-jwt/jwt) with session blacklist
- **Documentation**: Swagger/OpenAPI via swaggo
- **Validation**: go-playground/validator
- **Architecture**: Feature-based Clean Architecture
- **Deployment**: Docker + Docker Compose + GitHub Actions
- **Reverse Proxy**: Caddy Docker Proxy

## 📁 Project Structure

```
bookapi/
├── cmd/
│   ├── api/              # API server entry point
│   └── migrate/          # Database migration tool
├── db/
│   ├── schema.sql        # Initial database schema
│   └── migrations/       # Goose migrations (002-007)
├── internal/             # Feature-based modules
│   ├── auth/             # JWT authentication
│   ├── book/             # Book management
│   ├── catalog/          # Book catalog with search
│   ├── httpx/            # HTTP utilities & middleware
│   ├── ingest/           # Open Library ingestion
│   ├── platform/         # Infrastructure
│   │   ├── crypto/       # Password & JWT
│   │   └── openlibrary/  # Open Library client
│   ├── profile/          # User profiles
│   ├── rating/           # Book ratings
│   ├── readinglist/      # Reading lists
│   ├── session/          # Session management
│   └── user/             # User management
├── .github/workflows/    # CI/CD pipeline
├── Dockerfile            # Multi-stage Docker build
├── docker-compose.yml    # Docker services
└── README.md             # This file
```

## 🚀 Quick Start

### Prerequisites

- Go 1.25 or later
- PostgreSQL 15+
- Docker (optional)

### Local Development

```bash
# Clone the repository
git clone https://github.com/naufaldi/bookapi.git
cd bookapi

# Create environment file
cp .env.local .env
# Edit .env with your configuration

# Create database
createdb booklibrary

# Initialize schema
psql -d booklibrary -f db/schema.sql

# Run migrations
go run ./cmd/migrate -command=up

# Start the server
go run ./cmd/api
```

The API will be available at `http://localhost:8080`

### Environment Variables

```env
# Server
APP_ADDR=:8080

# Database
DB_DSN=postgres://postgres:postgres@localhost:5432/booklibrary

# Security
JWT_SECRET=your-super-secret-key-at-least-32-chars

# CORS
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

# Optional
MAX_REQUEST_SIZE_MB=1
ENABLE_HSTS=false

# Ingestion (optional)
INGEST_ENABLED=false
INGEST_BOOKS_MAX=100
INGEST_AUTHORS_MAX=100
INTERNAL_JOBS_SECRET=your-internal-cron-secret
```

## 📋 API Documentation

### Base URL

```
http://localhost:8080
```

### Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| **Infrastructure** |
| GET | `/healthz` | Liveness check | No |
| GET | `/readyz` | Readiness check (DB) | No |
| GET | `/swagger/` | Swagger UI | No |
| **Books** |
| GET | `/v1/books` | List books with filters | No |
| GET | `/v1/books/{isbn}` | Get book by ISBN | No |
| GET | `/v1/books/{isbn}/rating` | Get book rating | No |
| POST | `/v1/books/{isbn}/rating` | Rate book | Yes |
| **Auth** |
| POST | `/v1/users/register` | Register user | No |
| POST | `/v1/users/login` | Login | No |
| POST | `/v1/auth/refresh` | Refresh token | No |
| POST | `/v1/auth/logout` | Logout | Yes |
| **User** |
| GET | `/v1/me` | Get current user | Yes |
| GET | `/v1/me/profile` | Get own profile | Yes |
| PATCH | `/v1/me/profile` | Update profile | Yes |
| GET | `/v1/me/sessions` | List sessions | Yes |
| DELETE | `/v1/me/sessions/{id}` | Delete session | Yes |
| GET | `/v1/users/{id}/profile` | Public profile | No |
| **Reading Lists** |
| POST | `/v1/users/readinglist` | Add/update item | Yes |
| GET | `/v1/users/{id}/{status}` | List by status | No |
| **Catalog** |
| GET | `/v1/catalog/search` | Search catalog | No |
| GET | `/v1/catalog/books/{isbn}` | Get catalog book | No |

### Example Requests

```bash
# Register a user
curl -X POST http://localhost:8080/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","username":"john","password":"secure123"}'

# Login
curl -X POST http://localhost:8080/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secure123"}'

# Get books with filters
curl "http://localhost:8080/v1/books?genre=fiction&limit=10&offset=0"

# Search catalog
curl "http://localhost:8080/v1/catalog/search?q=harry+potter"
```

## 🐳 Docker Deployment

### Prerequisites

- Docker
- Docker Compose

### Quick Start

```bash
# Clone and setup
git clone https://github.com/naufaldi/bookapi.git
cd bookapi

# Create environment file
cp .env.production .env
# Edit .env with production values

# Start services
docker compose up -d --build

# Run migrations
docker compose exec api ./migrate -command=up

# Verify
curl https://your-domain/healthz
```

### Environment Variables (Production)

```env
DOMAIN=book-api.example.com
DB_PASSWORD=secure-password生成
JWT_SECRET=your-32-char-secret
APP_ADDR=:8080
DB_DSN=postgres://bookapi:${DB_PASSWORD}@db:5432/bookapi
ALLOWED_ORIGINS=https://book-api.example.com
MAX_REQUEST_SIZE_MB=1
ENABLE_HSTS=true
```

### Docker Compose Services

| Service | Port | Description |
|---------|------|-------------|
| api | 8080 | API server (internal) |
| db | 5432 | PostgreSQL (internal) |

The API is accessed via Caddy reverse proxy on port 443 (HTTPS).

## 🔄 CI/CD Pipeline

This project uses GitHub Actions for automated deployment:

1. **Test**: Runs on every push
   - Unit tests with race detection
   - Coverage report

2. **Deploy**: Runs on push to `main`
   - Builds Docker image
   - Pushes to GitHub Container Registry
   - Deploys to VPS via SSH

### Required GitHub Secrets

| Secret | Description |
|--------|-------------|
| `VPS_HOST` | VPS IP address |
| `VPS_USER` | SSH username |
| `VPS_SSH_KEY` | Private SSH key |

## 📊 Database Schema

```
users
├── id (UUID)
├── email (UNIQUE)
├── username (UNIQUE)
├── password (bcrypt hash)
├── role (USER/ADMIN)
├── bio, location, website
├── is_public
├── reading_preferences (JSONB)
└── timestamps

books
├── id (UUID)
├── isbn (UNIQUE)
├── title, genre, publisher
├── description
├── publication_year, page_count
├── language, cover_url
├── search_vector (tsvector)
└── timestamps

user_books
├── user_id + book_id (PK)
├── status (WISHLIST/READING/FINISHED)
└── timestamps

ratings
├── user_id + book_id (PK)
├── star (1-5)
└── timestamps

sessions
├── id (UUID)
├── user_id (FK)
├── refresh_token_hash
├── user_agent, ip_address
├── remember_me, expires_at
└── timestamps

catalog_books, catalog_authors, ingest_runs, ...
└── Open Library catalog tables
```

## 🧪 Testing

```bash
# Run all tests
go test ./... -race -cover

# Run specific package
go test ./internal/book -v

# With coverage report
go test -cover ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## 📝 Database Migrations

This project uses [goose](https://github.com/pressly/goose) for migrations:

```bash
# Apply all migrations
go run ./cmd/migrate -command=up

# Rollback one migration
go run ./cmd/migrate -command=down

# Check status
go run ./cmd/migrate -command=status

# Create new migration
go run ./cmd/migrate -command=create -name=add_new_feature
```

## 🔒 Security

- **Passwords**: Bcrypt hashing (cost factor 10)
- **Tokens**: JWT with RS256 signing
- **Sessions**: Refresh tokens with device tracking
- **Blacklisting**: Immediate token invalidation on logout
- **Rate Limiting**: 5 requests/second, burst of 10
- **Input Validation**: Struct tags with go-playground/validator
- **SQL Injection**: Parameterized queries via pgx
- **CORS**: Configurable allowed origins
- **Headers**: X-Content-Type-Options, HSTS, CSP

## 📈 Observability

### Health Endpoints

- `GET /healthz` → Returns "ok" (liveness)
- `GET /readyz` → Returns "ready" or 503 (readiness)

### Request Tracing

All requests include:
- `X-Request-Id` header
- Access logs with: method, path, status, duration_ms, request_id, user_id

### Prometheus Metrics

`GET /metrics` - Standard Prometheus metrics

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Open Library](https://openlibrary.org/) for the book data API
- [Go](https://golang.org/) for the excellent programming language
- [PostgreSQL](https://www.postgresql.org/) for the robust database
- [Caddy](https://caddyserver.com/) for the automatic HTTPS

---

<div align="center">
Built with ❤️ by [naufaldi](https://github.com/naufaldi)
</div>
