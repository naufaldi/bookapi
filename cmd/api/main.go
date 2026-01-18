package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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

	_ "bookapi/docs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Config struct {
	AppAddr        string
	DBDSN          string
	JWTSecret      string
	AllowedOrigins []string
	MaxRequestSize int64

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

	return Config{
		AppAddr:        getEnv("APP_ADDR", ":8080"),
		DBDSN:          getEnv("DB_DSN", "postgres://postgres:postgres@localhost:5432/booklibrary"),
		JWTSecret:      mustGetEnv("JWT_SECRET"),
		AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"), ","),
		MaxRequestSize: maxRequestSize,

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
	bookRepo := book.NewPostgresRepo(dbPool)
	bookService := book.NewService(bookRepo)
	bookHandler := book.NewHTTPHandler(bookService)

	userRepo := user.NewPostgresRepo(dbPool)
	userService := user.NewService(userRepo)
	userHandler := user.NewHTTPHandler(userService)

	sessionRepo := session.NewPostgresRepo(dbPool)
	blacklistRepo := session.NewBlacklistPostgresRepo(dbPool)
	sessionService := session.NewService(sessionRepo, blacklistRepo)
	sessionHandler := session.NewHTTPHandler(sessionService)

	authService := auth.NewService(cfg.JWTSecret, userService, sessionService)
	authHandler := auth.NewHTTPHandler(authService)

	ratingRepo := rating.NewPostgresRepo(dbPool)
	ratingService := rating.NewService(ratingRepo)
	ratingHandler := rating.NewHTTPHandler(ratingService)

	readingListRepo := readinglist.NewPostgresRepo(dbPool)
	readingListService := readinglist.NewService(readingListRepo)
	readingListHandler := readinglist.NewHTTPHandler(readingListService)

	profileService := profile.NewService(userService, ratingService, readingListService)
	profileHandler := profile.NewHTTPHandler(profileService)

	// Ingest & Catalog
	olClient := openlibrary.NewClient("BookAPI/1.0", cfg.IngestRPS, cfg.IngestMaxRetries)
	catalogRepo := catalog.NewPostgresRepo(dbPool)
	ingestRepo := ingest.NewPostgresRepo(dbPool)
	ingestService := ingest.NewService(olClient, catalogRepo, ingestRepo, ingest.Config{
		BooksMax:      cfg.IngestBooksMax,
		AuthorsMax:    cfg.IngestAuthorsMax,
		Subjects:      cfg.IngestSubjects,
		BatchSize:     cfg.IngestBooksBatchSize,
		FreshnessDays: cfg.IngestFreshDays,
	})
	ingestHandler := ingest.NewHTTPHandler(ingestService, cfg.InternalJobsSecret)

	// 2. Middlewares & Routing
	authMid := httpx.AuthMiddleware(cfg.JWTSecret, blacklistRepo)

	mux := http.NewServeMux()

	// Infrastructure & Public
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		if err := dbPool.Ping(ctx); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})

	// Books
	mux.HandleFunc("GET /books", bookHandler.List)
	mux.HandleFunc("GET /books/{isbn}", bookHandler.GetByISBN)
	mux.HandleFunc("GET /books/{isbn}/rating", ratingHandler.GetRating)
	mux.Handle("POST /books/{isbn}/rating", authMid(http.HandlerFunc(ratingHandler.CreateRating)))

	// Auth & Users
	mux.HandleFunc("POST /users/register", userHandler.RegisterUser)
	mux.HandleFunc("POST /users/login", authHandler.Login)
	mux.HandleFunc("POST /auth/refresh", authHandler.RefreshToken)
	mux.Handle("POST /auth/logout", authMid(http.HandlerFunc(authHandler.Logout)))

	// Me
	mux.Handle("GET /me", authMid(http.HandlerFunc(userHandler.GetCurrentUser)))
	mux.Handle("GET /me/profile", authMid(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			profileHandler.UpdateProfile(w, r)
			return
		}
		profileHandler.GetOwnProfile(w, r)
	})))
	mux.Handle("GET /me/sessions", authMid(http.HandlerFunc(sessionHandler.ListSessions)))
	mux.Handle("DELETE /me/sessions/{id}", authMid(http.HandlerFunc(sessionHandler.DeleteSession)))

	// Users & Reading Lists
	mux.HandleFunc("GET /users/{id}/profile", profileHandler.GetPublicProfile)
	mux.Handle("POST /users/readinglist", authMid(http.HandlerFunc(readingListHandler.AddOrUpdate)))
	mux.HandleFunc("GET /users/{id}/{status}", readingListHandler.ListByStatus)

	// Internal Jobs
	mux.HandleFunc("POST /internal/jobs/ingest", ingestHandler.Ingest)

	// Global Middlewares
	var handler http.Handler = mux
	handler = httpx.SecurityHeadersMiddleware(handler)
	handler = httpx.RequestSizeLimitMiddleware(cfg.MaxRequestSize)(handler)
	handler = httpx.CORSMiddleware(cfg.AllowedOrigins)(handler)

	log.Printf("Starting server on %s", cfg.AppAddr)
	httpServer := &http.Server{
		Addr:         cfg.AppAddr,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
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
