-- 0003_invites: single-use, time-limited provisioning tokens.
CREATE TABLE IF NOT EXISTS invites (
    id          TEXT PRIMARY KEY,
    token       TEXT NOT NULL UNIQUE,
    email       TEXT,
    role        TEXT NOT NULL,
    created_by  TEXT NOT NULL,
    expires_at  TEXT NOT NULL,
    redeemed_at TEXT,
    redeemed_by TEXT,
    created_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_invites_redeemed_at ON invites (redeemed_at);
