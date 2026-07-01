package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies every embedded migration that has not yet been recorded, in
// filename order, each inside its own transaction. It is safe to call on every
// startup: already-applied migrations are skipped.
func (d *DB) Migrate(ctx context.Context) error {
	if err := d.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	applied, err := d.appliedVersions(ctx)
	if err != nil {
		return err
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("db: read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		version := strings.TrimSuffix(name, ".sql")
		if applied[version] {
			continue
		}
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("db: read migration %s: %w", name, err)
		}
		if err := d.applyMigration(ctx, version, string(body)); err != nil {
			return fmt.Errorf("db: apply migration %s: %w", name, err)
		}
	}
	return nil
}

func (d *DB) ensureMigrationsTable(ctx context.Context) error {
	const ddl = `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`
	if _, err := d.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("db: create schema_migrations: %w", err)
	}
	return nil
}

func (d *DB) appliedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := d.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("db: read applied migrations: %w", err)
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func (d *DB) applyMigration(ctx context.Context, version, body string) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after a successful commit

	for _, stmt := range splitStatements(body) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("statement %q: %w", truncate(stmt, 60), err)
		}
	}

	insert := d.rebind(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`)
	if _, err := tx.ExecContext(ctx, insert, version, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	return tx.Commit()
}

// splitStatements breaks a migration file into individual statements on
// semicolons that terminate a line. Migration SQL is kept simple enough that
// this is sufficient (no semicolons inside string literals or bodies).
func splitStatements(body string) []string {
	parts := strings.Split(body, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(stripComments(p)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func stripComments(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
