package db

import (
	"io/fs"
	"strings"
	"testing"

	dbembed "github.com/memohai/memoh/db"
	"github.com/memohai/memoh/internal/config"
)

func TestRunMigrateUnknownCommand(t *testing.T) {
	cfg := config.PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "memoh",
		Password: "secret",
		Database: "memoh",
		SSLMode:  "disable",
	}
	err := RunMigrate(nil, cfg, nil, "invalid", nil)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestEmbeddedMigrationsHaveUniqueDirectionPerVersion(t *testing.T) {
	migrationsFS, err := fs.Sub(dbembed.MigrationsFS, "migrations")
	if err != nil {
		t.Fatalf("sub migrations fs: %v", err)
	}

	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	type migrationKey struct {
		version   string
		direction string
	}

	seen := make(map[migrationKey]string)
	pairs := make(map[string]map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		direction := ""
		switch {
		case strings.HasSuffix(name, ".up.sql"):
			direction = "up"
		case strings.HasSuffix(name, ".down.sql"):
			direction = "down"
		default:
			t.Fatalf("unexpected migration filename: %s", name)
		}

		parts := strings.SplitN(name, "_", 2)
		if len(parts) != 2 || parts[0] == "" {
			t.Fatalf("invalid migration filename: %s", name)
		}

		key := migrationKey{
			version:   parts[0],
			direction: direction,
		}
		if previous, ok := seen[key]; ok {
			t.Fatalf("duplicate migration version %s (%s): %s and %s", key.version, key.direction, previous, name)
		}
		seen[key] = name

		if pairs[key.version] == nil {
			pairs[key.version] = make(map[string]bool)
		}
		pairs[key.version][key.direction] = true
	}

	for version, directions := range pairs {
		if !directions["up"] || !directions["down"] {
			t.Fatalf("migration version %s must include both up and down files", version)
		}
	}
}
