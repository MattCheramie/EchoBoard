package db

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/echoboard/internal/config"
)

// OpenTest returns a migrated SQLite database backed by a temp file, closed
// automatically when the test ends. Shared by tests across the backend.
func OpenTest(t *testing.T) *DB {
	t.Helper()
	cfg := config.Default()
	cfg.DatabaseURL = filepath.Join(t.TempDir(), "test.db")
	d, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	if err := d.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return d
}

func TestOpenAndMigrate(t *testing.T) {
	d := OpenTest(t)

	// app_meta from 0001 should exist and be usable.
	_, err := d.Exec(`INSERT INTO app_meta (key, value, updated_at) VALUES ('k', 'v', '2026-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert into app_meta: %v", err)
	}
	var v string
	if err := d.QueryRow(`SELECT value FROM app_meta WHERE key = 'k'`).Scan(&v); err != nil {
		t.Fatalf("select: %v", err)
	}
	if v != "v" {
		t.Errorf("value = %q, want v", v)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	d := OpenTest(t)
	// Running again should be a no-op, not an error.
	if err := d.Migrate(context.Background()); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if n != 1 {
		t.Errorf("applied migrations = %d, want 1", n)
	}
}

func TestRebind(t *testing.T) {
	sqlite := &DB{Driver: config.DriverSQLite}
	if got := sqlite.Rebind(`SELECT * FROM t WHERE a = ? AND b = ?`); got != `SELECT * FROM t WHERE a = ? AND b = ?` {
		t.Errorf("sqlite rebind changed query: %q", got)
	}
	pg := &DB{Driver: config.DriverPostgres}
	want := `SELECT * FROM t WHERE a = $1 AND b = $2`
	if got := pg.Rebind(`SELECT * FROM t WHERE a = ? AND b = ?`); got != want {
		t.Errorf("pg rebind = %q, want %q", got, want)
	}
}
