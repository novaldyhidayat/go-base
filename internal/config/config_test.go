package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	t.Setenv("GOBASE_SERVER_PORT", "9090")
	t.Setenv("GOBASE_DATABASE_DSN", "postgres://from-env")
	t.Setenv("GOBASE_SERVER_READ_TIMEOUT", "2s")

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 8080\ndatabase:\n  dsn: postgres://from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server.Port != 9090 {
		t.Fatalf("server port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Database.DSN != "postgres://from-env" {
		t.Fatalf("database DSN = %q, want environment override", cfg.Database.DSN)
	}
	if cfg.Server.ReadTimeout.String() != "2s" {
		t.Fatalf("read timeout = %s, want 2s", cfg.Server.ReadTimeout)
	}
}
