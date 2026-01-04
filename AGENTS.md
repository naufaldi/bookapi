# AGENTS.md (bookapi)
Go REST API (`net/http`) + Postgres (`pgx/v5`). Layers: `internal/http` → `internal/usecase` → `internal/store` → `internal/entity`.

## Commands
- Run API: `JWT_SECRET=... go run ./cmd/api` (loads `.env.local` if present)
- Build: `go build ./...`
- Lint: `go vet ./...` (optional: `golangci-lint run`)
- Format: `gofmt -w .`
- Test all: `go test ./...`
- Single test (any pkg): `go test ./... -run '^TestName$' -count=1`
- Single package test: `go test ./internal/store -run '^TestName$' -count=1`

## Code Style
- Keep handlers thin; business rules in `internal/usecase`, DB in `internal/store`.
- Signatures: `ctx context.Context` first arg; pass `r.Context()` downward.
- Errors: wrap with `%w`; check with `errors.Is`; map `pgx.ErrNoRows` → `usecase.ErrNotFound`.
- HTTP: set status codes + `Content-Type: application/json`; don't leak internal errors.
- SQL: always parameterize (`$1,$2,...`); whitelist any dynamic sort/filter (see `internal/store/book_pg.go`).
- Naming: `CamelCase` exports, `lowerCamel` locals; avoid 1-letter names (except short loops).
- Imports: stdlib → third-party → local; rely on `gofmt`/`goimports` for ordering.
- Editor rules: none found (`.cursor/rules/`, `.cursorrules`, `.github/copilot-instructions.md`).