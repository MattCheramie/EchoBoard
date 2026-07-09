-- 0005_integrations: external platform connections and received webhook events.
-- OAuth access/refresh tokens are stored encrypted at rest (AES-256-GCM via the
-- secrets vault) so the plaintext never touches these columns.
CREATE TABLE IF NOT EXISTS integration_connections (
    id                TEXT PRIMARY KEY,
    platform          TEXT NOT NULL,
    account_ref       TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL,
    scopes            TEXT NOT NULL DEFAULT '',
    access_token_enc  TEXT NOT NULL DEFAULT '',
    refresh_token_enc TEXT NOT NULL DEFAULT '',
    expires_at        TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_integration_connections_platform_account
    ON integration_connections (platform, account_ref);

CREATE TABLE IF NOT EXISTS webhook_events (
    id          TEXT PRIMARY KEY,
    platform    TEXT NOT NULL,
    event_type  TEXT NOT NULL DEFAULT '',
    verified    TEXT NOT NULL,
    payload     TEXT NOT NULL,
    received_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_webhook_events_platform ON webhook_events (platform, received_at);
