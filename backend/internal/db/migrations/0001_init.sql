-- 0001_init: instance metadata key/value store.
-- Portable across SQLite and Postgres (TEXT columns only).
CREATE TABLE IF NOT EXISTS app_meta (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
