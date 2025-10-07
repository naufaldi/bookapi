# 📖 Personal Book Tracking API

A REST API for tracking books you've read, want to read, or are currently reading. This project serves as a learning exercise for backend development and SQL database design, inspired by Goodreads but built as a lightweight, personal alternative.

## 🎯 Learning Objectives

This repository is designed to learn and demonstrate:
- **Backend Development**: Building REST APIs with Go using clean architecture principles
- **SQL Database Design**: PostgreSQL schema design, indexing, and query optimization
- **Go Best Practices**: Clean code, dependency injection, error handling, and testing

## 🛠 Tech Stack

- **Language**: Go 1.25.1
- **Database**: PostgreSQL
- **Database Driver**: pgx/v5 (PostgreSQL driver for Go)
- **Architecture**: Clean Architecture (entities, usecases, handlers, repositories)
- **HTTP Server**: Standard library (net/http)

## ✨ Features

### Currently Implemented
- ✅ Book listing with filtering and pagination
- ✅ Get book details by ISBN
- ✅ Health check endpoint
- ✅ Clean architecture with dependency injection
- ✅ PostgreSQL integration with connection pooling

### Planned Features (from RFC)
- 🔄 User registration and authentication (JWT)
- 🔄 Wishlist management
- 🔄 Reading progress tracking
- 🔄 Book ratings and reviews
- 🔄 User-specific book lists (wishlist, reading, finished)

## 📋 API Endpoints

### Public Endpoints

#### Health Check
```http
GET /healthz
```
Returns: `200 OK` with `"ok"`

#### List Books
```http
GET /books?page=1&page_size=20&genre=Fantasy&publisher=Penguin
```
**Query Parameters:**
- `page` (int): Page number (default: 1)
- `page_size` (int): Items per page (default: 20, max: 100)
- `genre` (string): Filter by genre
- `publisher` (string): Filter by publisher

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "isbn": "978-0-123456-78-9",
      "title": "Book Title",
      "genre": "Fantasy",
      "publisher": "Penguin",
      "description": "Book description...",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20
  }
}
```

#### Get Book by ISBN
```http
GET /books/{isbn}
```
**Response:**
```json
{
  "data": {
    "id": "uuid",
    "isbn": "978-0-123456-78-9",
    "title": "Book Title",
    "genre": "Fantasy",
    "publisher": "Penguin",
    "description": "Book description...",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

## 🗄 Database Schema

The application uses PostgreSQL with the following schema:

### Tables

#### users
```sql
CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  username varchar UNIQUE NOT NULL,
  role varchar NOT NULL CHECK (role IN ('USER','ADMIN')),
  email varchar(255) UNIQUE NOT NULL,
  password_hash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
```

#### books
```sql
CREATE TABLE books (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  isbn varchar(20) UNIQUE NOT NULL,
  title varchar NOT NULL,
  genre varchar,
  publisher varchar,
  description text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
```

#### ratings
```sql
CREATE TABLE ratings (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  book_id uuid NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  star integer NOT NULL CHECK (star BETWEEN 1 AND 5),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, book_id)
);

CREATE INDEX idx_ratings_book_id ON ratings(book_id);
```

#### user_books
```sql
CREATE TABLE user_books (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  book_id uuid NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  status varchar(16) NOT NULL CHECK (status IN ('WISHLIST','READING','FINISHED')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, book_id)
);

CREATE INDEX idx_user_books_user_status ON user_books (user_id, status);
```

## 🚀 Getting Started

### Prerequisites
- Go 1.25.1 or later
- PostgreSQL 12+
- Git

### Local Development Setup

1. **Clone the repository**
```bash
git clone http://github.com/naufaldi
cd bookapi
```

2. **Set up PostgreSQL database**
```bash
# Create database
createdb booklibrary

# Run the schema (copy from docs/rfc.md or create schema.sql)
psql -d booklibrary -f schema.sql
```

3. **Configure environment variables**
Create a `.env.local` file in the project root:
```env
APP_ADDR=:8080
DB_DSN=postgres://postgres:postgres@localhost:5432/booklibrary
```

4. **Install dependencies**
```bash
go mod download
```

5. **Run the application**
```bash
go run cmd/api/main.go
```

The server will start on `http://localhost:8080`

### Testing the API

```bash
# Health check
curl http://localhost:8080/healthz

# List books
curl "http://localhost:8080/books?page=1&page_size=10"

# Get book by ISBN (if data exists)
curl http://localhost:8080/books/978-0-123456-78-9
```

## 📁 Project Structure

```
bookapi/
├── cmd/api/              # Application entry point
│   └── main.go          # Server setup and routing
├── internal/            # Private application code
│   ├── config/         # Configuration management
│   ├── entity/         # Domain entities (Book, User, etc.)
│   ├── gateway/        # External service integrations
│   ├── http/           # HTTP handlers
│   │   └── book_handler.go
│   ├── store/          # Data access layer (repositories)
│   │   └── book_pg.go  # PostgreSQL book repository
│   └── usecase/        # Business logic layer
│       ├── book.go     # Book use cases
│       └── errors.go   # Custom error types
├── db/                 # Database migrations/scripts
├── docs/               # Documentation
│   ├── rfc.md         # Requirements and API specification
│   └── rfc-api.md     # Additional API docs
├── go.mod             # Go module definition
├── go.sum             # Go dependencies
├── README.md          # This file
└── todo.md            # Development notes and explanations
```

## 🏗 Architecture Principles

This project follows **Clean Architecture** principles:

- **Entities**: Core business objects (Book, User)
- **Usecases**: Application business rules
- **Interface Adapters**: Controllers, repositories (adapters between usecases and external concerns)
- **Frameworks & Drivers**: Database, web framework (outermost layer)

### Key Patterns Used
- **Repository Pattern**: Abstract data access
- **Dependency Injection**: Loose coupling and testability
- **Context Propagation**: Request-scoped operations with timeouts
- **Error Handling**: Custom error types with proper HTTP status codes

## 📚 Learning Resources

This project demonstrates:
- Go HTTP server without frameworks
- PostgreSQL integration with connection pooling
- SQL query building and parameterization
- JSON API design
- Clean architecture in Go
- Environment-based configuration

## 🤝 Contributing

This is a learning project, but feel free to:
- Report bugs or issues
- Suggest improvements
- Add more features following the RFC specification

## 📄 License

This project is for educational purposes.
