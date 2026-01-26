package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrationsDir_EnvOverride(t *testing.T) {
	os.Setenv("MIGRATIONS_DIR", "/custom/migrations")
	t.Cleanup(func() { _ = os.Unsetenv("MIGRATIONS_DIR") })

	if got := migrationsDir(); got != "/custom/migrations" {
		t.Fatalf("expected MIGRATIONS_DIR override, got %q", got)
	}
}

func TestMigrationsDir_Default(t *testing.T) {
	_ = os.Unsetenv("MIGRATIONS_DIR")

	if got := migrationsDir(); got != "db/migrations" {
		t.Fatalf("expected default migrations dir, got %q", got)
	}
}

func TestLoadEnvFiles_DoesNotOverrideExistingEnv(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, ".env")

	if err := os.WriteFile(p, []byte("DB_DSN=from_file\n"), 0644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	os.Setenv("DB_DSN", "from_env")
	t.Cleanup(func() { _ = os.Unsetenv("DB_DSN") })

	cwd, _ := os.Getwd()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	loadEnvFiles()

	if got := os.Getenv("DB_DSN"); got != "from_env" {
		t.Fatalf("expected existing env to win, got %q", got)
	}
}
