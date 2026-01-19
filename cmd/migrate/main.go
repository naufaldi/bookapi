package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

func main() {
	var (
		command = flag.String("command", "up", "Migration command: up, down, status, create")
		name    = flag.String("name", "", "Name for 'create' command")
	)
	flag.Parse()

	_ = godotenv.Load(".env.local")

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/booklibrary"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	goose.SetBaseFS(nil)
	goose.SetDialect("postgres")

	migrationsDir := "db/migrations"

	switch *command {
	case "up":
		if err := goose.Up(db, migrationsDir); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("Migrations applied successfully")
	case "down":
		if err := goose.Down(db, migrationsDir); err != nil {
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
		fmt.Println("Migrations rolled back successfully")
	case "status":
		if err := goose.Status(db, migrationsDir); err != nil {
			log.Fatalf("Failed to check migration status: %v", err)
		}
	case "create":
		if *name == "" {
			log.Fatal("Name is required for 'create' command")
		}
		if err := goose.Create(nil, migrationsDir, *name, "sql"); err != nil {
			log.Fatalf("Failed to create migration: %v", err)
		}
		fmt.Printf("Migration created: %s\n", *name)
	default:
		log.Fatalf("Unknown command: %s. Use: up, down, status, create", *command)
	}
}
