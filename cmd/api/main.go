package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	apphttp "bookapi/internal/http"
	"bookapi/internal/store"
	"bookapi/internal/usecase"

	_ "bookapi/docs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Book API
// @version 1.0
// @description Simple Book Tracking API with authentication
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Enter the token with the `Bearer ` prefix

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
	sessionRepository := store.NewSessionPG(dbPool)
	blacklistRepository := store.NewBlacklistPG(dbPool)

	ratingRepository := store.NewRatingPG(dbPool)
	ratingHandler := apphttp.NewRatingHandler(ratingRepository)

	bookHandler := apphttp.NewBookHandler(bookRepository)
	userHandler := apphttp.NewUserHandler(userRepository, sessionRepository, jwtSecret)
	authHandler := apphttp.NewAuthHandler(jwtSecret, sessionRepository, blacklistRepository, userRepository)
	sessionHandler := apphttp.NewSessionHandler(sessionRepository)
	readingListHandler := apphttp.NewReadingListHandler(readingListRepository)

	profileUsecase := usecase.NewProfileUsecase(userRepository, ratingRepository, readingListRepository)
	profileHandler := apphttp.NewProfileHandler(profileUsecase)

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

	router := http.NewServeMux()
	booksSubRouter := http.NewServeMux()

	// Swagger UI endpoint
	router.HandleFunc("/swagger/", httpSwagger.WrapHandler)

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

	// Public routes - register before protected routes to avoid conflicts
	router.HandleFunc("/books", bookHandler.List)

	router.HandleFunc("/users/register", userHandler.RegisterUser)
	router.HandleFunc("/users/login", userHandler.LoginUser)
	
	protectedLogout := apphttp.AuthMiddleware(jwtSecret, blacklistRepository)(http.HandlerFunc(authHandler.LogoutHandler))
	router.Handle("/auth/logout", protectedLogout)
	router.HandleFunc("/auth/refresh", authHandler.RefreshTokenHandler)

	// Protected route - GET /me
	protectedMe := apphttp.AuthMiddleware(jwtSecret, blacklistRepository)(http.HandlerFunc(userHandler.GetCurrentUser))
	router.Handle("/me", protectedMe)

	// Profile routes
	router.Handle("/me/profile", apphttp.AuthMiddleware(jwtSecret, blacklistRepository)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			profileHandler.UpdateProfile(w, r)
			return
		}
		profileHandler.GetOwnProfile(w, r)
	})))

	// Reading list sub-router for protected /users/* routes
	readingListMux := http.NewServeMux()
	readingListMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		
		// /users/{id}/profile
		if len(parts) == 3 && parts[2] == "profile" {
			if r.Method == http.MethodGet {
				profileHandler.GetPublicProfile(w, r)
				return
			}
		}

		// Reading list routes
		if len(parts) == 2 {
			list := strings.ToUpper(parts[1])
			if list == "WISHLIST" || list == "READING" || list == "FINISHED" {
				// Reading list routes need auth
				apphttp.AuthMiddleware(jwtSecret, blacklistRepository)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.Method {
					case http.MethodPost:
						readingListHandler.AddOrUpdateReadingListItem(w, r)
					case http.MethodGet:
						readingListHandler.ListReadingListByStatus(w, r)
					default:
						w.WriteHeader(http.StatusMethodNotAllowed)
					}
				})).ServeHTTP(w, r)
				return
			}
		}
		
		http.NotFound(w, r)
	})
	router.Handle("/users/", readingListMux)

	// Session management routes
	sessionMux := http.NewServeMux()
	sessionMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) == 3 && parts[0] == "me" && parts[1] == "sessions" {
			if r.Method == http.MethodDelete {
				sessionHandler.DeleteSessionHandler(w, r)
				return
			}
		}
		if r.Method == http.MethodGet {
			sessionHandler.ListSessionsHandler(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	protectedSessions := apphttp.AuthMiddleware(jwtSecret, blacklistRepository)(sessionMux)
	router.Handle("/me/sessions/", protectedSessions)

	// Books sub-router for /books/* routes (includes /books/{isbn}/rating)
	booksSubRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(r.URL.Path, "/")
		if strings.Count(path, "/") == 2 && strings.HasSuffix(path, "/rating") {
			switch r.Method {
			case http.MethodPost:
				apphttp.AuthMiddleware(jwtSecret, blacklistRepository)(http.HandlerFunc(ratingHandler.CreateRating)).ServeHTTP(w, r)
			case http.MethodGet:
				ratingHandler.GetRating(w, r)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		}
		switch r.Method {
		case http.MethodGet:
			bookHandler.GetByISBN(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	router.Handle("/books/", booksSubRouter)

	handler := apphttp.SecurityHeadersMiddleware(router)
	handler = apphttp.RequestSizeLimitMiddleware(maxRequestSize)(handler)
	handler = apphttp.CORSMiddleware(allowedOrigins)(handler)

	httpServer := &http.Server{
		Addr:         serverAddress,
		Handler:      handler,
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
