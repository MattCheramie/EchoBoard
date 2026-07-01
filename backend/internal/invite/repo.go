package invite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MattCheramie/echoboard/internal/db"
	"github.com/MattCheramie/echoboard/internal/user"
	"github.com/google/uuid"
)

// ErrNotFound is returned when no invite matches the token.
var ErrNotFound = errors.New("invite: not found")

// Repository persists invites.
type Repository struct {
	db *db.DB
}

// NewRepository returns an invite repository backed by the given database.
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// CreateInput describes a new invite. Email is optional; ExpiresAt defaults to
// DefaultTTL from now when zero.
type CreateInput struct {
	Email     string
	Role      user.Role
	CreatedBy string
	ExpiresAt time.Time
}

// Create mints and stores a new invite with a random token.
func (r *Repository) Create(ctx context.Context, in CreateInput) (*Invite, error) {
	if !in.Role.Valid() {
		return nil, fmt.Errorf("invite: invalid role %q", in.Role)
	}
	if in.CreatedBy == "" {
		return nil, fmt.Errorf("invite: createdBy is required")
	}
	token, err := newToken()
	if err != nil {
		return nil, fmt.Errorf("invite: token: %w", err)
	}
	now := time.Now().UTC()
	exp := in.ExpiresAt
	if exp.IsZero() {
		exp = now.Add(DefaultTTL)
	}
	inv := &Invite{
		ID:        uuid.NewString(),
		Token:     token,
		Email:     user.NormalizeEmail(in.Email),
		Role:      in.Role,
		CreatedBy: in.CreatedBy,
		ExpiresAt: exp.UTC(),
		CreatedAt: now,
	}
	q := r.db.Rebind(`INSERT INTO invites (id, token, email, role, created_by, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	_, err = r.db.ExecContext(ctx, q, inv.ID, inv.Token, inv.Email, string(inv.Role),
		inv.CreatedBy, inv.ExpiresAt.Format(time.RFC3339), inv.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("invite: create: %w", err)
	}
	return inv, nil
}

// GetByToken returns the invite with the given token, or ErrNotFound.
func (r *Repository) GetByToken(ctx context.Context, token string) (*Invite, error) {
	q := r.db.Rebind(`SELECT id, token, email, role, created_by, expires_at, redeemed_at, redeemed_by, created_at
		FROM invites WHERE token = ?`)
	inv, err := scanInvite(r.db.QueryRowContext(ctx, q, token).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return inv, err
}

// MarkRedeemed records that userID redeemed the invite. It fails if the invite
// was already redeemed (guarded by the WHERE clause) to prevent reuse races.
func (r *Repository) MarkRedeemed(ctx context.Context, token, userID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	q := r.db.Rebind(`UPDATE invites SET redeemed_at = ?, redeemed_by = ?
		WHERE token = ? AND redeemed_at IS NULL`)
	res, err := r.db.ExecContext(ctx, q, now, userID, token)
	if err != nil {
		return fmt.Errorf("invite: mark redeemed: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("invite: already redeemed or missing")
	}
	return nil
}

// ListActive returns invites that have not yet been redeemed, newest first.
func (r *Repository) ListActive(ctx context.Context) ([]*Invite, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, token, email, role, created_by, expires_at, redeemed_at, redeemed_by, created_at
		FROM invites WHERE redeemed_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("invite: list: %w", err)
	}
	defer rows.Close()

	var out []*Invite
	for rows.Next() {
		inv, err := scanInvite(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func scanInvite(scan func(dest ...any) error) (*Invite, error) {
	var (
		inv                    Invite
		role                   string
		email                  sql.NullString
		expiresAt, createdAt   string
		redeemedAt, redeemedBy sql.NullString
	)
	if err := scan(&inv.ID, &inv.Token, &email, &role, &inv.CreatedBy,
		&expiresAt, &redeemedAt, &redeemedBy, &createdAt); err != nil {
		return nil, err
	}
	inv.Email = email.String
	inv.Role = user.Role(role)
	inv.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	inv.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if redeemedAt.Valid {
		t, _ := time.Parse(time.RFC3339, redeemedAt.String)
		inv.RedeemedAt = &t
	}
	if redeemedBy.Valid {
		inv.RedeemedBy = &redeemedBy.String
	}
	return &inv, nil
}
