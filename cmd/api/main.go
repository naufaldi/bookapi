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
	"bookapi/internal/httpx"
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

func main() {
	_ = godotenv.Load(".env.local")

	serverAddress := getEnv("APP_ADDR", ":8080")
	databaseDSN := getEnv("DB_DSN", "postgres://postgres:postgres@localhost:5432/booklibrary")
	jwtSecret := mustGetEnv("JWT_SECRET")

	dbPool := mustOpenDB(databaseDSN)
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

	authService := auth.NewService(jwtSecret, userService, sessionService)
	authHandler := auth.NewHTTPHandler(authService)

	ratingRepo := rating.NewPostgresRepo(dbPool)
	ratingService := rating.NewService(ratingRepo)
	ratingHandler := rating.NewHTTPHandler(ratingService)

	readingListRepo := readinglist.NewPostgresRepo(dbPool)
	readingListService := readinglist.NewService(readingListRepo)
	readingListHandler := readinglist.NewHTTPHandler(readingListService)

	profileService := profile.NewService(userService, ratingService, readingListService)
	profileHandler := profile.NewHTTPHandler(profileService)

	// 2. Middlewares & Routing
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:5173"}
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = strings.Split(origins, ",")
	}
	maxRequestSize := int64(1 * 1024 * 1024)
	if sizeMB := os.Getenv("MAX_REQUEST_SIZE_MB"); sizeMB != "" {
		if size, err := strconv.ParseInt(sizeMB, 10, 64); err == nil {
			maxRequestSize = size * 1024 * 1024
		}
	}

	authMid := httpx.AuthMiddleware(jwtSecret, blacklistRepo)

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

	// Global Middlewares
	var handler http.Handler = mux
	handler = httpx.SecurityHeadersMiddleware(handler)
	handler = httpx.RequestSizeLimitMiddleware(maxRequestSize)(handler)
	handler = httpx.CORSMiddleware(allowedOrigins)(handler)

	log.Printf("Starting server on %s", serverAddress)
	httpServer := &http.Server{
		Addr:         serverAddress,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
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
