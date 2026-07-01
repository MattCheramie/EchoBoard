// Package dbtest provides a migrated, temp-file SQLite database for use in
// tests across the backend. It is only imported from _test.go files, so it does
// not add weight to the production binary.
package dbtest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/echoboard/internal/config"
	"github.com/MattCheramie/echoboard/internal/db"
)

// New returns a fully-migrated SQLite database backed by a temp file that is
// closed automatically when the test ends.
func New(t testing.TB) *db.DB {
	t.Helper()
	cfg := config.Default()
	cfg.DatabaseURL = filepath.Join(t.TempDir(), "test.db")
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatalf("dbtest: open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	if err := d.Migrate(context.Background()); err != nil {
		t.Fatalf("dbtest: migrate: %v", err)
	}
	return d
}
