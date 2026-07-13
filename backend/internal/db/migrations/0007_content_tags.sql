-- 0007_content_tags: free-form tags and a JSON metadata bag for content.
-- Tags live in a child table (one row per tag, unique per content). Metadata is
-- a JSON object stored inline. Both SQLite and Postgres accept ADD COLUMN with a
-- NOT NULL default.
ALTER TABLE content ADD COLUMN metadata TEXT NOT NULL DEFAULT '{}';

CREATE TABLE IF NOT EXISTS content_tags (
    content_id TEXT NOT NULL,
    tag        TEXT NOT NULL,
    PRIMARY KEY (content_id, tag)
);

CREATE INDEX IF NOT EXISTS idx_content_tags_tag ON content_tags (tag);
