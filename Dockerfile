FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for go modules
RUN apk add --no-cache git

# Copy go mod and sum files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the API binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api ./cmd/api

# Build the migrate binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o migrate ./cmd/migrate

# Final stage - minimal image
FROM alpine:3.19

# Install ca-certificates for HTTPS connections
RUN apk add --no-cache ca-certificates postgresql-client

# Create non-root user
RUN addgroup -g 1000 app && adduser -u 1000 -G app -s /bin/sh -D app

# Copy binaries from builder
COPY --from=builder /app/api /app/
COPY --from=builder /app/migrate /app/

# Copy DB migrations for "migrate" command
COPY --from=builder /app/db/migrations /app/db/migrations

# Create logs directory
RUN mkdir -p /app/logs && chown -R app:app /app

USER app

EXPOSE 8080

WORKDIR /app

CMD ["/app/api"]
