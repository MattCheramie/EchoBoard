package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/MattCheramie/echoboard/internal/db"
)

// DefaultSessionTTL is how long a session stays valid after creation.
const DefaultSessionTTL = 30 * 24 * time.Hour

// ErrNoSession is returned when a token is unknown or expired.
var ErrNoSession = errors.New("auth: no valid session")

// Session is a server-side login session.
type Session struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// SessionStore persists sessions in the database.
type SessionStore struct {
	db  *db.DB
	ttl time.Duration
}

// NewSessionStore returns a store with the given TTL (0 uses DefaultSessionTTL).
func NewSessionStore(database *db.DB, ttl time.Duration) *SessionStore {
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	return &SessionStore{db: database, ttl: ttl}
}

// Create issues a new session for userID with a crypto-random token.
func (s *SessionStore) Create(ctx context.Context, userID string) (*Session, error) {
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	sess := &Session{
		Token:     token,
		UserID:    userID,
		ExpiresAt: now.Add(s.ttl),
		CreatedAt: now,
	}
	q := s.db.Rebind(`INSERT INTO sessions (token, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`)
	if _, err := s.db.ExecContext(ctx, q, sess.Token, sess.UserID,
		sess.ExpiresAt.Format(time.RFC3339), sess.CreatedAt.Format(time.RFC3339)); err != nil {
		return nil, fmt.Errorf("auth: create session: %w", err)
	}
	return sess, nil
}

// Get returns the session for token, or ErrNoSession if it is missing or
// expired. Expired sessions are opportunistically deleted.
func (s *SessionStore) Get(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, ErrNoSession
	}
	q := s.db.Rebind(`SELECT token, user_id, expires_at, created_at FROM sessions WHERE token = ?`)
	var (
		sess                 Session
		expiresAt, createdAt string
	)
	err := s.db.QueryRowContext(ctx, q, token).Scan(&sess.Token, &sess.UserID, &expiresAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoSession
	}
	if err != nil {
		return nil, fmt.Errorf("auth: get session: %w", err)
	}
	sess.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	sess.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if !time.Now().UTC().Before(sess.ExpiresAt) {
		_ = s.Delete(ctx, token)
		return nil, ErrNoSession
	}
	return &sess, nil
}

// Delete removes a session (logout). Deleting a missing token is not an error.
func (s *SessionStore) Delete(ctx context.Context, token string) error {
	q := s.db.Rebind(`DELETE FROM sessions WHERE token = ?`)
	if _, err := s.db.ExecContext(ctx, q, token); err != nil {
		return fmt.Errorf("auth: delete session: %w", err)
	}
	return nil
}

// DeleteExpired purges expired sessions; intended for periodic cleanup.
func (s *SessionStore) DeleteExpired(ctx context.Context) error {
	q := s.db.Rebind(`DELETE FROM sessions WHERE expires_at < ?`)
	if _, err := s.db.ExecContext(ctx, q, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("auth: delete expired sessions: %w", err)
	}
	return nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
