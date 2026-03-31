package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var databaseEnvKeyOrder = []string{
	"DATABASE_HOST",
	"DATABASE_PORT",
	"DATABASE_USERNAME",
	"DATABASE_PASSWORD",
	"DATABASE_NAME",
	"DATABASE_SCHEMA",
}

func TestLoadDatabaseConfig_Success(t *testing.T) {
	clearDatabaseEnv(t)
	withWorkingDir(t, writeEnvFile(t, requiredDatabaseEnvValues()))

	cfg, err := LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Fatalf("host mismatch: got %q want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 5432 {
		t.Fatalf("port mismatch: got %d want %d", cfg.Port, 5432)
	}
	if cfg.Username != "postgres" {
		t.Fatalf("username mismatch: got %q want %q", cfg.Username, "postgres")
	}
	if cfg.Password != "postgres" {
		t.Fatalf("password mismatch: got %q want %q", cfg.Password, "postgres")
	}
	if cfg.Database != "mthesis" {
		t.Fatalf("database mismatch: got %q want %q", cfg.Database, "mthesis")
	}
	if cfg.Schema != "public" {
		t.Fatalf("schema mismatch: got %q want %q", cfg.Schema, "public")
	}
}

func TestLoadDatabaseConfig_MissingVariable(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "missing host", key: "DATABASE_HOST"},
		{name: "missing port", key: "DATABASE_PORT"},
		{name: "missing username", key: "DATABASE_USERNAME"},
		{name: "missing password", key: "DATABASE_PASSWORD"},
		{name: "missing database", key: "DATABASE_NAME"},
		{name: "missing schema", key: "DATABASE_SCHEMA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearDatabaseEnv(t)

			values := requiredDatabaseEnvValues()
			delete(values, tt.key)
			withWorkingDir(t, writeEnvFile(t, values))

			_, err := LoadDatabaseConfig()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("error does not include key %q: %v", tt.key, err)
			}
		})
	}
}

func TestLoadDatabaseConfig_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{name: "not a number", port: "abc"},
		{name: "out of range", port: "70000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearDatabaseEnv(t)

			values := requiredDatabaseEnvValues()
			values["DATABASE_PORT"] = tt.port
			withWorkingDir(t, writeEnvFile(t, values))

			_, err := LoadDatabaseConfig()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !strings.Contains(err.Error(), "invalid DATABASE_PORT") {
				t.Fatalf("unexpected error message: %v", err)
			}
		})
	}
}

func requiredDatabaseEnvValues() map[string]string {
	return map[string]string{
		"DATABASE_HOST":     "localhost",
		"DATABASE_PORT":     "5432",
		"DATABASE_USERNAME": "postgres",
		"DATABASE_PASSWORD": "postgres",
		"DATABASE_NAME":     "mthesis",
		"DATABASE_SCHEMA":   "public",
	}
}

func clearDatabaseEnv(t *testing.T) {
	t.Helper()

	for _, key := range databaseEnvKeyOrder {
		t.Setenv(key, "")
	}
}

func writeEnvFile(t *testing.T, values map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	lines := make([]string, 0, len(databaseEnvKeyOrder))
	for _, key := range databaseEnvKeyOrder {
		value, ok := values[key]
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}

	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o644); err != nil {
		t.Fatalf("write .env file: %v", err)
	}

	return dir
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get current directory: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change directory to %q: %v", dir, err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
}
