package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	handler "bookapi/internal/http"
	"bookapi/internal/store"
)


func main(){
	godotenv.Load(".env.local")
	addr := getEnv("APP_ADDR", ":8080")
	dsn := getEnv("DB_DSN","postgres://postgres:postgres@localhost:5432/booklibrary")

	ctx := context.Background()
	dbpool, err := pgxpool.New(ctx, dsn)

	if err != nil{
		log.Fatalf("Failed to create database pool: %v", err)
	}
	log.Println("DB connected")

	defer dbpool.Close()

	bookRepo := store.NewBookPG(dbpool)
	bookHandler := handler.NewBookHandler(bookRepo)
		userRepo := store.NewUserPG(dbpool)
	userHandler := handler.NewUserHandler(userRepo, getEnv("JWT_SECRET", "secret"))

	mux := http.NewServeMux()
	mux.HandleFunc("/users/register", userHandler.RegisterUser)
	mux.HandleFunc("/users/login", userHandler.LoginUser)
	httpMe := handler.AuthMiddleware(os.Getenv("JWT_SECRET"))(http.HandlerFunc(userHandler.GetCurrentUser))
	mux.Handle("/me", httpMe)
	
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request){
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	
	mux.HandleFunc("/books", bookHandler.List)
	mux.HandleFunc("/books/", bookHandler.GetByISBN)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})
	
	srv := &http.Server{
		Addr: addr,
		Handler: mux,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout: 15 * time.Second,
	}
	log.Printf("Starting server on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

