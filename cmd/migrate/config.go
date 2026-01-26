package main

import (
	"os"

	"github.com/joho/godotenv"
)

func loadEnvFiles() {
	// Do not override environment provided by the runtime (e.g. Docker).
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.local")
}

func migrationsDir() string {
	if v := os.Getenv("MIGRATIONS_DIR"); v != "" {
		return v
	}
	return "db/migrations"
}
