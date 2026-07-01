// Package db provides the database handle plus SQLite (local-first) and Postgres
// (production) support behind a single database/sql interface, together with an
// embedded migration runner.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/MattCheramie/echoboard/internal/config"

	// Database drivers. Both register themselves with database/sql.
	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx"
	_ "modernc.org/sqlite"             // registers "sqlite" (pure Go, no cgo)
)

// DB wraps *sql.DB and remembers which driver backs it so callers and the
// migration runner can adapt where the SQL dialects differ.
type DB struct {
	*sql.DB
	Driver config.Driver
}

// Open connects to the configured database, verifies the connection, and
// returns a ready handle. The caller is responsible for calling Close.
func Open(cfg config.Config) (*DB, error) {
	driverName, dsn, err := resolve(cfg)
	if err != nil {
		return nil, err
	}

	sqlDB, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open %s: %w", cfg.DBDriver, err)
	}

	// SQLite is a single file; serialize writes to avoid "database is locked".
	if cfg.DBDriver == config.DriverSQLite {
		sqlDB.SetMaxOpenConns(1)
	} else {
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("db: ping %s: %w", cfg.DBDriver, err)
	}

	return &DB{DB: sqlDB, Driver: cfg.DBDriver}, nil
}

// Rebind rewrites a query written with `?` placeholders into the dialect of the
// active driver. SQLite accepts `?` as-is; Postgres needs positional `$1, $2`.
// Write all repository SQL with `?` and pass it through Rebind.
func (d *DB) Rebind(query string) string {
	return d.rebind(query)
}

func (d *DB) rebind(query string) string {
	if d.Driver != config.DriverPostgres {
		return query
	}
	var b strings.Builder
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(fmt.Sprintf("%d", n))
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

// resolve maps a Config to a database/sql driver name and DSN.
func resolve(cfg config.Config) (driverName, dsn string, err error) {
	switch cfg.DBDriver {
	case config.DriverSQLite:
		return "sqlite", sqliteDSN(cfg.DatabaseURL), nil
	case config.DriverPostgres:
		return "pgx", cfg.DatabaseURL, nil
	default:
		return "", "", fmt.Errorf("db: unsupported driver %q", cfg.DBDriver)
	}
}

// sqliteDSN turns a plain file path into a modernc.org/sqlite DSN with sane
// pragmas (foreign keys on, a busy timeout). Paths that already look like a DSN
// (contain a query string or the file: scheme) are passed through untouched.
func sqliteDSN(path string) string {
	if strings.HasPrefix(path, "file:") || strings.Contains(path, "?") {
		return path
	}
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "busy_timeout(5000)")
	return "file:" + path + "?" + q.Encode()
}
