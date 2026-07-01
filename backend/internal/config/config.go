package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Driver identifies a supported database backend.
type Driver string

const (
	DriverSQLite   Driver = "sqlite"
	DriverPostgres Driver = "postgres"
)

// Config holds all runtime configuration for the EchoBoard server. Values are
// sourced from environment variables (see .env.example); the caller may layer
// command-line flags on top before calling Validate.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port int
	// AppEnv is "development" or "production".
	AppEnv string
	// DBDriver selects the database backend.
	DBDriver Driver
	// DatabaseURL is the SQLite file path or the Postgres DSN.
	DatabaseURL string
	// SecretKey is the base64-encoded 32-byte key used to encrypt secrets at
	// rest (integration tokens). Required outside development.
	SecretKey string
	// PublicAPIBaseURL is the base URL the frontend uses to reach the API.
	PublicAPIBaseURL string
}

// Default returns a Config populated with development-friendly defaults.
func Default() Config {
	return Config{
		Port:             8080,
		AppEnv:           "development",
		DBDriver:         DriverSQLite,
		DatabaseURL:      "./echoboard.db",
		SecretKey:        "",
		PublicAPIBaseURL: "http://localhost:8080",
	}
}

// Load builds a Config from Default() overlaid with any environment variables
// that are set. It does not validate; call Validate after any flag overrides.
func Load() (Config, error) {
	c := Default()

	if v, ok := os.LookupEnv("PORT"); ok {
		p, err := strconv.Atoi(v)
		if err != nil {
			return c, fmt.Errorf("config: invalid PORT %q: %w", v, err)
		}
		c.Port = p
	}
	if v, ok := os.LookupEnv("APP_ENV"); ok {
		c.AppEnv = v
	}
	if v, ok := os.LookupEnv("DB_DRIVER"); ok {
		c.DBDriver = Driver(strings.ToLower(v))
	}
	if v, ok := os.LookupEnv("DATABASE_URL"); ok {
		c.DatabaseURL = v
	}
	if v, ok := os.LookupEnv("SECRET_KEY"); ok {
		c.SecretKey = v
	}
	if v, ok := os.LookupEnv("PUBLIC_API_BASE_URL"); ok {
		c.PublicAPIBaseURL = v
	}
	return c, nil
}

// IsProduction reports whether the server is running in production mode.
func (c Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

// Validate checks that the configuration is internally consistent and usable.
func (c Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("config: port %d out of range", c.Port)
	}
	switch c.DBDriver {
	case DriverSQLite, DriverPostgres:
	default:
		return fmt.Errorf("config: unknown DB_DRIVER %q (want sqlite or postgres)", c.DBDriver)
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("config: DATABASE_URL is required")
	}
	// The at-rest encryption key is mandatory in production; in development we
	// allow it to be empty (the vault falls back to a dev-only ephemeral key).
	if c.IsProduction() && c.SecretKey == "" {
		return fmt.Errorf("config: SECRET_KEY is required in production")
	}
	return nil
}
