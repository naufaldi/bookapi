package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSQLMigrations_HaveGooseDirectives(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	migrationsDir := filepath.Join(repoRoot, "db", "migrations")

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", migrationsDir, err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(migrationsDir, e.Name()))
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", e.Name(), err)
		}
		s := string(b)
		if !strings.Contains(s, "-- +goose Up") {
			t.Fatalf("%s missing '-- +goose Up'", e.Name())
		}
		if !strings.Contains(s, "-- +goose Down") {
			t.Fatalf("%s missing '-- +goose Down'", e.Name())
		}
	}
}
