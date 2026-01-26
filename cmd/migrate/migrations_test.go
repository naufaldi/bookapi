package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestCollectMigrations_ParsesMigrationsDir(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// this file lives in cmd/migrate/, so repo root is ../..
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	dir := filepath.Join(repoRoot, "db", "migrations")

	if _, err := goose.CollectMigrations(dir, 0, goose.MaxVersion); err != nil {
		t.Fatalf("expected migrations to parse, got error: %v", err)
	}
}
