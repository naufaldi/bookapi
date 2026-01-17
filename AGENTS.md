# AGENTS.md (bookapi)
Go REST API (`net/http`) + Postgres (`pgx/v5`). Architecture: Feature-based Clean Architecture (Screaming Architecture).

## Project Structure
- `cmd/api/`: Entry point and DI wiring.
- `internal/<feature>/`: Cohesive modules (e.g., `book/`, `user/`).
  - `<feature>.go`: Domain entities.
  - `service.go`: Business logic.
  - `ports.go`: Interfaces (Repository).
  - `http_handler.go`: HTTP entry point.
  - `postgres_repo.go`: DB implementation.
- `internal/httpx/`: Shared HTTP kit (JSON, middleware, validator).
- `internal/platform/`: Infrastructure (crypto, etc.).

## Commands
- Run API: `JWT_SECRET=... go run ./cmd/api` (loads `.env.local` if present)
- Build: `go build ./...`
- Lint: `go vet ./...` (optional: `golangci-lint run`)
- Format: `gofmt -w .`
- Test all: `go test ./...`
- Single module test: `go test ./internal/book -run '^TestName$' -count=1`

## Code Style
- Keep handlers thin; business rules in `internal/<feature>/service.go`.
- Signatures: `ctx context.Context` first arg; pass `r.Context()` downward.
- Errors: wrap with `%w`; check with `errors.Is`.
- HTTP: Use `internal/httpx` for responses and validation.
- SQL: always parameterize (`$1,$2,...`); implementation stays in `postgres_repo.go`.
- Naming: `CamelCase` exports, `lowerCamel` locals.
- Routing: Use modern `net/http` patterns in `main.go` (e.g., `GET /path/{id}`).
- Editor rules: none found (`.cursor/rules/`, `.cursorrules`, `.github/copilot-instructions.md`).