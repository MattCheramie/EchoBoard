package integrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MattCheramie/echoboard/internal/auth"
	"github.com/MattCheramie/echoboard/internal/db"
	"github.com/google/uuid"
)

// ErrNotFound is returned when no connection matches the query.
var ErrNotFound = errors.New("integrations: connection not found")

// Connection status values.
const (
	StatusConnected = "connected"
	StatusExpired   = "expired"
	StatusRevoked   = "revoked"
)

// Connection is a stored link to an external platform account. It deliberately
// carries no token fields: access/refresh tokens live only in encrypted columns
// and are never serialized to API clients.
type Connection struct {
	ID         string    `json:"id"`
	Platform   Platform  `json:"platform"`
	AccountRef string    `json:"accountRef,omitempty"`
	Status     string    `json:"status"`
	Scopes     []string  `json:"scopes,omitempty"`
	ExpiresAt  time.Time `json:"expiresAt,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// Repository persists integration connections (with vault-encrypted tokens) and
// a log of received webhook events.
type Repository struct {
	db    *db.DB
	vault *auth.Vault
}

// NewRepository returns a repository backed by the database and secrets vault.
func NewRepository(database *db.DB, vault *auth.Vault) *Repository {
	return &Repository{db: database, vault: vault}
}

// Upsert creates or updates the connection for (platform, accountRef), storing
// the token encrypted at rest. It returns the redacted connection.
func (r *Repository) Upsert(ctx context.Context, platform Platform, accountRef, status string, scopes []string, tok Token) (*Connection, error) {
	accessEnc, err := r.seal(tok.AccessToken)
	if err != nil {
		return nil, err
	}
	refreshEnc, err := r.seal(tok.RefreshToken)
	if err != nil {
		return nil, err
	}
	expires := formatTime(tok.ExpiresAt)
	scopeStr := strings.Join(scopes, " ")
	now := time.Now().UTC()

	existing, err := r.getByPlatformAccount(ctx, platform, accountRef)
	switch {
	case errors.Is(err, ErrNotFound):
		c := &Connection{
			ID:         uuid.NewString(),
			Platform:   platform,
			AccountRef: accountRef,
			Status:     status,
			Scopes:     scopes,
			ExpiresAt:  tok.ExpiresAt,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		q := r.db.Rebind(`INSERT INTO integration_connections
			(id, platform, account_ref, status, scopes, access_token_enc, refresh_token_enc, expires_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if _, err := r.db.ExecContext(ctx, q, c.ID, string(platform), accountRef, status, scopeStr,
			accessEnc, refreshEnc, expires, formatTime(now), formatTime(now)); err != nil {
			return nil, fmt.Errorf("integrations: insert connection: %w", err)
		}
		return c, nil
	case err != nil:
		return nil, err
	default:
		q := r.db.Rebind(`UPDATE integration_connections
			SET status = ?, scopes = ?, access_token_enc = ?, refresh_token_enc = ?, expires_at = ?, updated_at = ?
			WHERE id = ?`)
		if _, err := r.db.ExecContext(ctx, q, status, scopeStr, accessEnc, refreshEnc, expires,
			formatTime(now), existing.ID); err != nil {
			return nil, fmt.Errorf("integrations: update connection: %w", err)
		}
		existing.Status = status
		existing.Scopes = scopes
		existing.ExpiresAt = tok.ExpiresAt
		existing.UpdatedAt = now
		return existing, nil
	}
}

// List returns all connections (redacted), newest first.
func (r *Repository) List(ctx context.Context) ([]*Connection, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, platform, account_ref, status, scopes, expires_at, created_at, updated_at
		FROM integration_connections ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("integrations: list: %w", err)
	}
	defer rows.Close()

	var out []*Connection
	for rows.Next() {
		c, err := scanConnection(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetByPlatform returns the first (oldest) connection for a platform.
func (r *Repository) GetByPlatform(ctx context.Context, platform Platform) (*Connection, error) {
	q := r.db.Rebind(`SELECT id, platform, account_ref, status, scopes, expires_at, created_at, updated_at
		FROM integration_connections WHERE platform = ? ORDER BY created_at LIMIT 1`)
	c, err := scanConnection(r.db.QueryRowContext(ctx, q, string(platform)).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

func (r *Repository) getByPlatformAccount(ctx context.Context, platform Platform, accountRef string) (*Connection, error) {
	q := r.db.Rebind(`SELECT id, platform, account_ref, status, scopes, expires_at, created_at, updated_at
		FROM integration_connections WHERE platform = ? AND account_ref = ?`)
	c, err := scanConnection(r.db.QueryRowContext(ctx, q, string(platform), accountRef).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

// Token decrypts and returns the stored token for a connection.
func (r *Repository) Token(ctx context.Context, id string) (Token, error) {
	q := r.db.Rebind(`SELECT access_token_enc, refresh_token_enc, expires_at
		FROM integration_connections WHERE id = ?`)
	var accessEnc, refreshEnc, expires string
	err := r.db.QueryRowContext(ctx, q, id).Scan(&accessEnc, &refreshEnc, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return Token{}, ErrNotFound
	}
	if err != nil {
		return Token{}, err
	}
	access, err := r.open(accessEnc)
	if err != nil {
		return Token{}, err
	}
	refresh, err := r.open(refreshEnc)
	if err != nil {
		return Token{}, err
	}
	return Token{AccessToken: access, RefreshToken: refresh, ExpiresAt: parseTime(expires)}, nil
}

// UpdateToken re-encrypts and stores a refreshed token.
func (r *Repository) UpdateToken(ctx context.Context, id string, tok Token) error {
	accessEnc, err := r.seal(tok.AccessToken)
	if err != nil {
		return err
	}
	refreshEnc, err := r.seal(tok.RefreshToken)
	if err != nil {
		return err
	}
	q := r.db.Rebind(`UPDATE integration_connections
		SET access_token_enc = ?, refresh_token_enc = ?, expires_at = ?, status = ?, updated_at = ?
		WHERE id = ?`)
	_, err = r.db.ExecContext(ctx, q, accessEnc, refreshEnc, formatTime(tok.ExpiresAt),
		StatusConnected, formatTime(time.Now().UTC()), id)
	return err
}

// Delete removes a connection by id.
func (r *Repository) Delete(ctx context.Context, id string) error {
	q := r.db.Rebind(`DELETE FROM integration_connections WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// DeleteByPlatform removes every connection for a platform, returning the count.
func (r *Repository) DeleteByPlatform(ctx context.Context, platform Platform) (int, error) {
	q := r.db.Rebind(`DELETE FROM integration_connections WHERE platform = ?`)
	res, err := r.db.ExecContext(ctx, q, string(platform))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// RecordWebhook appends a received webhook event to the log.
func (r *Repository) RecordWebhook(ctx context.Context, platform Platform, eventType string, verified bool, payload []byte) error {
	q := r.db.Rebind(`INSERT INTO webhook_events (id, platform, event_type, verified, payload, received_at)
		VALUES (?, ?, ?, ?, ?, ?)`)
	_, err := r.db.ExecContext(ctx, q, uuid.NewString(), string(platform), eventType,
		boolToText(verified), string(payload), formatTime(time.Now().UTC()))
	return err
}

// --- helpers ---

// seal encrypts a secret; an empty secret is stored as an empty column so that
// a missing refresh token round-trips as "" rather than an encrypted blank.
func (r *Repository) seal(secret string) (string, error) {
	if secret == "" {
		return "", nil
	}
	return r.vault.Encrypt([]byte(secret))
}

func (r *Repository) open(enc string) (string, error) {
	if enc == "" {
		return "", nil
	}
	b, err := r.vault.Decrypt(enc)
	if err != nil {
		return "", fmt.Errorf("integrations: decrypt token: %w", err)
	}
	return string(b), nil
}

func scanConnection(scan func(dest ...any) error) (*Connection, error) {
	var (
		c                           Connection
		platform, scopes            string
		expires, createdAt, updated string
	)
	if err := scan(&c.ID, &platform, &c.AccountRef, &c.Status, &scopes, &expires, &createdAt, &updated); err != nil {
		return nil, err
	}
	c.Platform = Platform(platform)
	if scopes != "" {
		c.Scopes = strings.Fields(scopes)
	}
	c.ExpiresAt = parseTime(expires)
	c.CreatedAt = parseTime(createdAt)
	c.UpdatedAt = parseTime(updated)
	return &c, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func boolToText(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
