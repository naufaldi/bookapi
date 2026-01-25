---
name: phase-2-remaining-no-epic6
overview: "Complete the remaining RFC Phase-2 work (excluding Epic 6): enforce /v1 routing, fix PATCH /me/profile registration, harden Postgres data layer (timeouts + migrations + indexes), and register catalog routes. Keep observability simple: logs + request_id correlation + health/readiness checks for Docker+Caddy VPS operation."
todos:
  - id: v1-routing
    content: Enforce /v1-only routing and fix PATCH /v1/me/profile registration in cmd/api/main.go
    status: completed
  - id: db-timeouts
    content: Add config-driven context timeouts to all PostgresRepo methods across internal/*/postgres_repo.go
    status: completed
  - id: migrations-goose
    content: Add goose-based migration management (cmd/migrate) for db/migrations and document usage for VPS
    status: completed
  - id: missing-indexes
    content: Add migration for missing indexes (users(email), sessions(user_id), readinglist(user_id) after verifying actual table name)
    status: completed
  - id: catalog-routes
    content: Register catalog routes in cmd/api/main.go for GET /v1/catalog/search and GET /v1/catalog/books/{isbn}
    status: completed
  - id: ops-runbook
    content: "Update operator documentation for Docker+Caddy: log viewing, request_id correlation, health/readiness checks"
    status: completed
isProject: false
---

# Phase 2 Remaining Tasks (No Epic 6)

## 1. User Stories

- US-1: As an **API Developer**, I want all endpoints under **`/v1`** so that client integrations are consistent and future versioning is easy. Priority: High
- US-2: As a **Reader**, I want the catalog endpoints available so that book lookups/search work without code changes. Priority: Medium
- US-3: As an **Operator**, I want **DB queries to time out** so that slow queries don’t wedge the API under load. Priority: High
- US-4: As an **Operator**, I want **repeatable DB migrations** so that deployments to a VPS are safe and deterministic. Priority: High
- US-5: As an **Operator**, I want **simple observability** (request_id + access logs + health/readiness) so that I can quickly detect and trace errors in Docker+Caddy deployments. Priority: High

## 2. Acceptance Criteria

- AC-1: Given a request to any API endpoint, When the request path does not start with `/v1`, Then the server responds with 404 (or routes are absent) and only `/v1/*` is served.
- AC-2: Given a request to `PATCH /v1/me/profile`, When the route is invoked with valid auth, Then the request is handled by the profile handler (not 404/misrouted).
- AC-3: Given a request to `GET /v1/catalog/books/{isbn}`, When the route is hit, Then the existing catalog handler is executed (route is registered in `cmd/api/main.go`).
- AC-4: Given a request to `GET /v1/catalog/search?...`, When the route is hit, Then the search handler is executed and returns a consistent response envelope.
- AC-5: Given any Postgres repository call, When the DB is slow/hung beyond the configured timeout, Then the call fails with `context deadline exceeded` (wrapped) and the handler returns a JSON error including `meta.request_id`.
- AC-6: Given the repo is deployed to a VPS, When the operator runs migrations via a single command, Then all migrations in `db/migrations/` apply in order and failures are clearly reported.
- AC-7: Given the system is running in Docker behind Caddy, When an operator calls `/healthz` and `/readyz`, Then liveness returns 200 and readiness returns 200 only when DB connectivity is OK.
- AC-8: Given any error response, When returned to the client, Then it includes `X-Request-Id` header and `meta.request_id` in the JSON envelope (for log correlation).
- AC-9: Given the `/v1` routing change, When running unit tests that exercise routing, Then `PATCH /v1/me/profile` is reachable and old non-`/v1` paths are not served (v1-only).
- AC-10: Given a Postgres repo method, When the configured DB query timeout is exceeded, Then the returned error wraps `context.DeadlineExceeded` and surfaces as a JSON error including `meta.request_id`.
- AC-11: Given the codebase after changes, When running `go test ./...`, `go vet ./...`, and `go build ./...`, Then all commands succeed.

## 3. Edge Cases

### Input validation

- Missing/invalid path params for catalog ISBN.
- Invalid query parameters on catalog search (empty query, very large query).

### Error handling

- DB timeout/hang: ensure request returns promptly with JSON error (and logs contain request_id).
- Migration failure mid-run: partial application should stop with clear error.
- Catalog OpenLibrary upstream failure (if catalog uses read-through caching): ensure errors surface with request_id.

### Boundary conditions

- Empty catalog results.
- Large catalog search result sets (ensure pagination defaults and limits are respected if implemented).

### Integration

- VPS deployment: Caddy reverse_proxy should preserve/forward `X-Request-Id`.
- Docker healthcheck should target `/healthz` (liveness) and optionally `/readyz` (readiness) for dependent services.

### Performance

- DB query timeouts must be short enough to protect the API, but configurable (avoid hard-coding).
- Index creation should use `CONCURRENTLY` where appropriate (note: requires non-transactional migrations; handled via goose settings or separate migration strategy).

## 4. Technical Architecture

### Architecture Decisions

- **/v1-only routing**: enforce a clean break so the API surface is consistent and future versioning is straightforward.
- **Repo-level DB timeouts**: apply `context.WithTimeout` inside each `PostgresRepo` method using a config value, ensuring all queries are protected.
- **Goose for migrations**: standardize applying `db/migrations/*.sql` via a dedicated CLI entry point (safe for VPS + Docker).
- **Simple observability**: rely on existing request_id + access logs + recovery, plus health/readiness endpoints; no metrics stack per selected scope.

### Design Patterns

- Middleware pipeline (existing `internal/httpx/*` middlewares).
- Clean Architecture module boundaries (`internal/<feature>/service.go`, `postgres_repo.go`, `http_handler.go`).
- Dependency Injection in `cmd/api/main.go` for repos/services/handlers.
- “Ports & adapters” for repo interfaces and DB implementations.

### Technical Diagram (routing + operator checks)

```mermaid
flowchart TD
  Client[Client] --> Caddy[Caddy]
  Caddy --> Api[API_/v1]
  Api --> Health[GET_/healthz]
  Api --> Ready[GET_/readyz]
  Api --> V1Routes[/v1_routes]
  V1Routes --> Profile[PATCH_/v1/me/profile]
  V1Routes --> Catalog[GET_/v1/catalog/...]
  Api --> DB[(Postgres)]
  Ready --> DB
```

## Implementation Steps

### Epic 2: API Contract & v1 (complete remaining)

- Update routing in [`cmd/api/main.go`](cmd/api/main.go) to register **only** `/v1/*` endpoints.
- Fix the routing bug so `PATCH /v1/me/profile` is correctly registered (ensure pattern/method matches the handler).
- Add/verify consistent response envelope usage for the endpoints touched by routing changes.

### Epic 3: Data Layer Hardening

- Add a configurable DB query timeout to the app config (existing `Config` struct in [`cmd/api/main.go`](cmd/api/main.go)), and thread it into repo constructors.
- Update all `postgres_repo.go` implementations to wrap DB calls with `context.WithTimeout(ctx, cfg.DBQueryTimeout)`.
  - Targets likely include: [`internal/book/postgres_repo.go`](internal/book/postgres_repo.go), [`internal/user/postgres_repo.go`](internal/user/postgres_repo.go), [`internal/session/postgres_repo.go`](internal/session/postgres_repo.go), [`internal/catalog/postgres_repo.go`](internal/catalog/postgres_repo.go), [`internal/ingest/postgres_repo.go`](internal/ingest/postgres_repo.go), [`internal/readinglist/postgres_repo.go`](internal/readinglist/postgres_repo.go), [`internal/rating/postgres_repo.go`](internal/rating/postgres_repo.go), [`internal/profile/service.go`](internal/profile/service.go) (where it hits repos).
- Introduce goose-based migration management:
  - Add a migration CLI (e.g. [`cmd/migrate/main.go`](cmd/migrate/main.go)) that runs `goose up/down/status` against the configured DB.
  - Ensure it points to `db/migrations/`.
- Add a new migration to create missing indexes:
  - `users(email)`
  - `sessions(user_id)`
  - reading list table’s `user_id` (verify actual table name in `db/schema.sql` during implementation)

### Epic 4: Catalog routes registration

- Register catalog routes in [`cmd/api/main.go`](cmd/api/main.go):
  - `GET /v1/catalog/search`
  - `GET /v1/catalog/books/{isbn}`
- Ensure handlers return the standard envelope and include request_id in meta.

### Tests (add/update where necessary)

- Add/update unit tests to prevent regressions from the `/v1` routing change:
  - Confirm `PATCH /v1/me/profile` is registered and does not 404.
  - Confirm old non-`/v1` paths are not served (v1-only choice).
- Add/update unit tests for repo timeouts:
  - Force a context timeout and assert `errors.Is(err, context.DeadlineExceeded)`.
- Add/update tests for catalog route registration:
  - Confirm `GET /v1/catalog/books/{isbn}` and `GET /v1/catalog/search` are reachable and hit the expected handlers.

### Observability (simple operator readiness)

- Verify `X-Request-Id` is preserved end-to-end when behind Caddy (document recommended `header_up X-Request-Id {http.request.id}` usage).
- Verify `/healthz` and `/readyz` are present and documented for Docker healthchecks.
- Add/extend an operator runbook section (likely [`README.md`](README.md) or [`docs/rfc/phase-2.md`](docs/rfc/phase-2.md) depending on project convention) with:
  - Docker log commands
  - Request-id correlation workflow
  - Basic “find 4xx/5xx” and “find panics” commands

## Verification Checklist

- [ ] All user stories addressed
- [ ] All acceptance criteria met
- [ ] Edge cases handled
- [ ] Code follows project style (thin handlers, logic in services)
- [ ] Functions reused where possible
- [ ] Routing verified: only `/v1/*` endpoints are served
- [ ] `/healthz` and `/readyz` verified behind Docker+Caddy
- [ ] Tests updated/added for routing changes and repo timeouts
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go build ./...` passes