// @title Personal Book Tracking API
// @version 1.0
// @description A REST API for tracking books you've read, want to read, or are currently reading.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @host book-api.faldi.xyz
// @basePath /
// @schemes http https
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"bookapi/internal/auth"
	"bookapi/internal/book"
	"bookapi/internal/catalog"
	"bookapi/internal/httpx"
	"bookapi/internal/ingest"
	"bookapi/internal/platform/openlibrary"
	"bookapi/internal/profile"
	"bookapi/internal/rating"
	"bookapi/internal/readinglist"
	"bookapi/internal/session"
	"bookapi/internal/user"

	docs "bookapi/docs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Config struct {
	AppAddr        string
	DBDSN          string
	JWTSecret      string
	AllowedOrigins []string
	MaxRequestSize int64
	DBQueryTimeout time.Duration

	// Ingest
	IngestEnabled        bool
	IngestBooksMax       int
	IngestAuthorsMax     int
	IngestSubjects       []string
	IngestBooksBatchSize int
	IngestRPS            int
	IngestMaxRetries     int
	IngestFreshDays      int
	InternalJobsSecret   string
}

func (c Config) Validate() {
	if c.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	if c.IngestEnabled && c.InternalJobsSecret == "" {
		log.Fatal("INTERNAL_JOBS_SECRET is required when ingestion is enabled")
	}
}

func loadConfig() Config {
	_ = godotenv.Load(".env.local")

	maxRequestSize := int64(1 * 1024 * 1024)
	if sizeMB := os.Getenv("MAX_REQUEST_SIZE_MB"); sizeMB != "" {
		if size, err := strconv.ParseInt(sizeMB, 10, 64); err == nil {
			maxRequestSize = size * 1024 * 1024
		}
	}

	subjects := []string{"fiction", "history", "science"}
	if s := os.Getenv("INGEST_SUBJECTS"); s != "" {
		subjects = strings.Split(s, ",")
	}

	dbQueryTimeout := 5 * time.Second
	if timeoutStr := os.Getenv("DB_QUERY_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			dbQueryTimeout = timeout
		}
	}

	return Config{
		AppAddr:        getEnv("APP_ADDR", ":8080"),
		DBDSN:          getEnv("DB_DSN", "postgres://postgres:postgres@localhost:5432/booklibrary"),
		JWTSecret:      mustGetEnv("JWT_SECRET"),
		AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"), ","),
		MaxRequestSize: maxRequestSize,
		DBQueryTimeout: dbQueryTimeout,

		IngestEnabled:        getEnv("INGEST_ENABLED", "false") == "true",
		IngestBooksMax:       getEnvInt("INGEST_BOOKS_MAX", 100),
		IngestAuthorsMax:     getEnvInt("INGEST_AUTHORS_MAX", 100),
		IngestSubjects:       subjects,
		IngestBooksBatchSize: getEnvInt("INGEST_BOOKS_BATCH_SIZE", 50),
		IngestRPS:            getEnvInt("INGEST_RPS", 1),
		IngestMaxRetries:     getEnvInt("INGEST_MAX_RETRIES", 3),
		IngestFreshDays:      getEnvInt("INGEST_FRESH_DAYS", 7),
		InternalJobsSecret:   getEnv("INTERNAL_JOBS_SECRET", ""),
	}
}

func main() {
	cfg := loadConfig()
	cfg.Validate()

	dbPool := mustOpenDB(cfg.DBDSN)
	defer dbPool.Close()

	// 1. Setup Modules (Repositories & Services)
	bookRepo := book.NewPostgresRepo(dbPool, cfg.DBQueryTimeout)
	bookService := book.NewService(bookRepo)
	bookHandler := book.NewHTTPHandler(bookService)

	userRepo := user.NewPostgresRepo(dbPool, cfg.DBQueryTimeout)
	userService := user.NewService(userRepo)
	userHandler := user.NewHTTPHandler(userService)

	sessionRepo := session.NewPostgresRepo(dbPool, cfg.DBQueryTimeout)
	blacklistRepo := session.NewBlacklistPostgresRepo(dbPool, cfg.DBQueryTimeout)
	sessionService := session.NewService(sessionRepo, blacklistRepo)
	sessionHandler := session.NewHTTPHandler(sessionService)

	authService := auth.NewService(cfg.JWTSecret, userService, sessionService)
	authHandler := auth.NewHTTPHandler(authService)

	ratingRepo := rating.NewPostgresRepo(dbPool, cfg.DBQueryTimeout)
	ratingService := rating.NewService(ratingRepo)
	ratingHandler := rating.NewHTTPHandler(ratingService)

	readingListRepo := readinglist.NewPostgresRepo(dbPool, cfg.DBQueryTimeout)
	readingListService := readinglist.NewService(readingListRepo)
	readingListHandler := readinglist.NewHTTPHandler(readingListService)

	profileService := profile.NewService(userService, ratingService, readingListService)
	profileHandler := profile.NewHTTPHandler(profileService)

	// Ingest & Catalog
	olClient := openlibrary.NewClient("BookAPI/1.0", cfg.IngestRPS, cfg.IngestMaxRetries)
	catalogRepo := catalog.NewPostgresRepo(dbPool, cfg.DBQueryTimeout)
	catalogService := catalog.NewService(catalogRepo)

	ingestRepo := ingest.NewPostgresRepo(dbPool, cfg.DBQueryTimeout)
	ingestService := ingest.NewService(olClient, catalogRepo, bookRepo, ingestRepo, ingest.Config{
		BooksMax:      cfg.IngestBooksMax,
		AuthorsMax:    cfg.IngestAuthorsMax,
		Subjects:      cfg.IngestSubjects,
		BatchSize:     cfg.IngestBooksBatchSize,
		FreshnessDays: cfg.IngestFreshDays,
	})
	ingestHandler := ingest.NewHTTPHandler(ingestService, cfg.InternalJobsSecret)

	// 2. Middlewares & Routing
	authMid := httpx.AuthMiddleware(cfg.JWTSecret, blacklistRepo)
	rateLimiter := httpx.NewRateLimitMiddleware(5.0, 10) // 5 req/sec, burst of 10

	mux := http.NewServeMux()

	// Swagger UI needs a correct scheme/host/basePath to avoid mixed-content failures
	// when the API is behind an HTTPS-terminating reverse proxy.
	mux.HandleFunc("GET /swagger/openapi.json", swaggerDocHandler("/v1"))

	// Infrastructure & Public (not versioned)
	mux.HandleFunc("/swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/openapi.json")))
	mux.HandleFunc("GET /healthz", healthzHandler)
	mux.HandleFunc("GET /readyz", readyzHandler(dbPool))
	mux.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))

	// v1 API Router
	v1 := http.NewServeMux()

	// Books
	v1.HandleFunc("GET /books", bookHandler.List)
	v1.HandleFunc("GET /books/{isbn}", bookHandler.GetByISBN)
	v1.HandleFunc("GET /books/{isbn}/rating", ratingHandler.GetRating)
	v1.Handle("POST /books/{isbn}/rating", authMid(http.HandlerFunc(ratingHandler.CreateRating)))

	// Auth & Users (rate limited)
	v1.Handle("POST /users/register", rateLimiter.Middleware(http.HandlerFunc(userHandler.RegisterUser)))
	v1.Handle("POST /users/login", rateLimiter.Middleware(http.HandlerFunc(authHandler.Login)))
	v1.Handle("POST /auth/refresh", rateLimiter.Middleware(http.HandlerFunc(authHandler.RefreshToken)))
	v1.Handle("POST /auth/logout", authMid(http.HandlerFunc(authHandler.Logout)))

	// Me
	v1.Handle("GET /me", authMid(http.HandlerFunc(userHandler.GetCurrentUser)))
	v1.Handle("GET /me/profile", authMid(http.HandlerFunc(profileHandler.GetOwnProfile)))
	v1.Handle("PATCH /me/profile", authMid(http.HandlerFunc(profileHandler.UpdateProfile)))
	v1.Handle("GET /me/sessions", authMid(http.HandlerFunc(sessionHandler.ListSessions)))
	v1.Handle("DELETE /me/sessions/{id}", authMid(http.HandlerFunc(sessionHandler.DeleteSession)))

	// Users & Reading Lists
	v1.HandleFunc("GET /users/{id}/profile", profileHandler.GetPublicProfile)
	v1.Handle("POST /users/readinglist", authMid(http.HandlerFunc(readingListHandler.AddOrUpdate)))
	v1.HandleFunc("GET /users/{id}/{status}", readingListHandler.ListByStatus)

	// Catalog
	catalogHandler := catalog.NewHTTPHandler(catalogService)
	v1.HandleFunc("GET /catalog/search", catalogHandler.Search)
	v1.HandleFunc("GET /catalog/books/{isbn}", catalogHandler.GetByISBN)

	// Internal Jobs (rate limited)
	v1.Handle("POST /internal/jobs/ingest", rateLimiter.Middleware(http.HandlerFunc(ingestHandler.Ingest)))

	// Mount v1 router
	mux.Handle("/v1/", http.StripPrefix("/v1", v1))

	// Global Middlewares (order matters: outermost first)
	var handler http.Handler = mux
	handler = httpx.RequestIDMiddleware(handler)
	handler = httpx.RecoveryMiddleware(handler)
	handler = httpx.AccessLogMiddleware(handler)
	handler = httpx.SecurityHeadersMiddleware(handler)
	handler = httpx.RequestSizeLimitMiddleware(cfg.MaxRequestSize)(handler)
	handler = httpx.CORSMiddleware(cfg.AllowedOrigins)(handler)

	log.Printf("Starting server on %s", cfg.AppAddr)
	httpServer := &http.Server{
		Addr:              cfg.AppAddr,
		Handler:           handler,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	// Graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	log.Println("Server started. Press Ctrl+C to shutdown.")
	<-shutdown
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server stopped gracefully")
	}
}

func swaggerDocHandler(basePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		if xf := r.Header.Get("X-Forwarded-Proto"); xf != "" {
			if p := strings.TrimSpace(strings.Split(xf, ",")[0]); p != "" {
				scheme = p
			}
		}

		host := r.Host
		if xfh := r.Header.Get("X-Forwarded-Host"); xfh != "" {
			if h := strings.TrimSpace(strings.Split(xfh, ",")[0]); h != "" {
				host = h
			}
		}

		// Copy the generated spec and override per-request values.
		spec := *docs.SwaggerInfo
		spec.Host = host
		spec.Schemes = []string{scheme}
		spec.BasePath = basePath

		doc := spec.ReadDoc()

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(doc))
	}
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustGetEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	log.Fatalf("missing required environment variable: %s", key)
	return ""
}

// healthzHandler handles GET /healthz
// @Summary Health check
// @Description Simple health check endpoint
// @Tags infrastructure
// @Accept json
// @Produce text/plain
// @Success 200 {string} string "ok"
// @Router /healthz [get]
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// readyzHandler handles GET /readyz
// @Summary Readiness check
// @Description Check if the service is ready (database connectivity)
// @Tags infrastructure
// @Accept json
// @Produce text/plain
// @Success 200 {string} string "ready"
// @Failure 503 {string} string "db not ready"
// @Router /readyz [get]
func readyzHandler(dbPool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		if err := dbPool.Ping(ctx); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	}
}

func mustOpenDB(dsn string) *pgxpool.Pool {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("cannot create db pool: %v", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		log.Fatalf("cannot ping database: %v", err)
	}
	log.Println("database connection OK")
	return pool
}
