-- 0006_content: the content model (drafts/posts), their platform targets, tags,
-- and uploaded media, plus the join tables linking content to tags and media.
CREATE TABLE IF NOT EXISTS content (
    id           TEXT PRIMARY KEY,
    author_id    TEXT NOT NULL,
    title        TEXT NOT NULL DEFAULT '',
    body         TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL,
    scheduled_at TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_content_status ON content (status);
CREATE INDEX IF NOT EXISTS idx_content_author ON content (author_id);
CREATE INDEX IF NOT EXISTS idx_content_scheduled_at ON content (scheduled_at);

CREATE TABLE IF NOT EXISTS content_targets (
    content_id TEXT NOT NULL,
    platform   TEXT NOT NULL,
    PRIMARY KEY (content_id, platform)
);

CREATE TABLE IF NOT EXISTS media (
    id           TEXT PRIMARY KEY,
    author_id    TEXT NOT NULL,
    filename     TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT '',
    size         BIGINT NOT NULL DEFAULT 0,
    storage_key  TEXT NOT NULL,
    thumb_key    TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS content_media (
    content_id TEXT NOT NULL,
    media_id   TEXT NOT NULL,
    PRIMARY KEY (content_id, media_id)
);

CREATE TABLE IF NOT EXISTS tags (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_name ON tags (name);

CREATE TABLE IF NOT EXISTS content_tags (
    content_id TEXT NOT NULL,
    tag_id     TEXT NOT NULL,
    PRIMARY KEY (content_id, tag_id)
);
