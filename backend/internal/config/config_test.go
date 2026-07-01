package config

import "testing"

func TestDefaults(t *testing.T) {
	c := Default()
	if c.Port != 8080 || c.DBDriver != DriverSQLite {
		t.Fatalf("unexpected defaults: %+v", c)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("default config should validate: %v", err)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("APP_ENV", "production")
	t.Setenv("DB_DRIVER", "POSTGRES")
	t.Setenv("DATABASE_URL", "postgres://localhost/echo")
	t.Setenv("SECRET_KEY", "abc")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Port != 9090 {
		t.Errorf("Port = %d, want 9090", c.Port)
	}
	if c.DBDriver != DriverPostgres {
		t.Errorf("DBDriver = %q, want postgres (case-insensitive)", c.DBDriver)
	}
	if !c.IsProduction() {
		t.Error("IsProduction() = false, want true")
	}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"ok", func(*Config) {}, false},
		{"bad port", func(c *Config) { c.Port = 0 }, true},
		{"unknown driver", func(c *Config) { c.DBDriver = "mysql" }, true},
		{"empty url", func(c *Config) { c.DatabaseURL = "" }, true},
		{"prod needs secret", func(c *Config) { c.AppEnv = "production"; c.SecretKey = "" }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Default()
			c.SecretKey = "dev"
			tt.mutate(&c)
			err := c.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
