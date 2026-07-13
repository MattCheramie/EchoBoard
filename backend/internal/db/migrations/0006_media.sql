-- 0006_media: uploaded media metadata. The bytes live in the media Store
-- (local filesystem by default); this table records what and where. content_id
-- is nullable so media can be uploaded before being attached to content.
CREATE TABLE IF NOT EXISTS media (
    id            TEXT PRIMARY KEY,
    owner_id      TEXT NOT NULL,
    content_id    TEXT,
    filename      TEXT NOT NULL,
    mime_type     TEXT NOT NULL,
    size          INTEGER NOT NULL DEFAULT 0,
    width         INTEGER NOT NULL DEFAULT 0,
    height        INTEGER NOT NULL DEFAULT 0,
    storage_key   TEXT NOT NULL,
    thumbnail_key TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_media_owner ON media (owner_id);
CREATE INDEX IF NOT EXISTS idx_media_content ON media (content_id);
