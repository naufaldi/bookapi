package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	apphttp "bookapi/internal/http"
	"bookapi/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env.local")

	serverAddress := getEnv("APP_ADDR", ":8080")
	databaseDSN := getEnv("DB_DSN", "postgres://postgres:postgres@localhost:5432/booklibrary")
	jwtSecret := mustGetEnv("JWT_SECRET")

	dbPool := mustOpenDB(databaseDSN)
	defer dbPool.Close()

	bookRepository := store.NewBookPG(dbPool)
	userRepository := store.NewUserPG(dbPool)
	readingListRepository := store.NewReadingListPG(dbPool)

	bookHandler := apphttp.NewBookHandler(bookRepository)
	userHandler := apphttp.NewUserHandler(userRepository, jwtSecret)
	readingListHandler := apphttp.NewReadingListHandler(readingListRepository)

	router := http.NewServeMux()

	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	router.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		if err := dbPool.Ping(ctx); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	router.HandleFunc("/books", bookHandler.List)
	router.HandleFunc("/books/", bookHandler.GetByISBN)

	router.HandleFunc("/users/register", userHandler.RegisterUser)
	router.HandleFunc("/users/login", userHandler.LoginUser)

	protectedMe := apphttp.AuthMiddleware(jwtSecret)(http.HandlerFunc(userHandler.GetCurrentUser))
	router.Handle("/me", protectedMe)

	readingListMux := http.NewServeMux()
	readingListMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			readingListHandler.AddOrUpdateReadingListItem(w, r)
		case http.MethodGet:
			readingListHandler.ListReadingListByStatus(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	protectedReadingLists := apphttp.AuthMiddleware(jwtSecret)(readingListMux)
	router.Handle("/users/", protectedReadingLists)

	httpServer := &http.Server{
		Addr:         serverAddress,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting server on %s", serverAddress)
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
		log.Fatalf("cannot ping database (%s): %v", redactDSN(dsn), err)
	}
	log.Println("database connection OK")
	return pool
}

func redactDSN(dsn string) string {
	const marker = "://"
	start := strings.Index(dsn, marker)
	if start < 0 {
		return dsn
	}
	start += len(marker)
	end := strings.Index(dsn[start:], "@")
	if end < 0 {
		return dsn
	}
	return dsn[:start] + "***" + dsn[start+end:]
}
