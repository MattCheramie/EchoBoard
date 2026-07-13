-- 0005_content: content records and their per-platform targets. Portable
-- across SQLite and Postgres. Referential integrity between the two tables is
-- maintained by the content repository (targets are written/removed with their
-- parent inside one transaction), matching the FK-free style of earlier tables.
CREATE TABLE IF NOT EXISTS content (
    id           TEXT PRIMARY KEY,
    author_id    TEXT NOT NULL,
    title        TEXT NOT NULL,
    body         TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL,
    scheduled_at TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_content_author ON content (author_id);
CREATE INDEX IF NOT EXISTS idx_content_status ON content (status);
CREATE INDEX IF NOT EXISTS idx_content_scheduled ON content (scheduled_at);

CREATE TABLE IF NOT EXISTS content_targets (
    id         TEXT PRIMARY KEY,
    content_id TEXT NOT NULL,
    platform   TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    UNIQUE (content_id, platform)
);

CREATE INDEX IF NOT EXISTS idx_content_targets_content ON content_targets (content_id);
