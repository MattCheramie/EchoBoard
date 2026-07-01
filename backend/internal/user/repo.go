package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MattCheramie/echoboard/internal/db"
	"github.com/google/uuid"
)

// ErrNotFound is returned when no user matches the query.
var ErrNotFound = errors.New("user: not found")

// Repository persists users.
type Repository struct {
	db *db.DB
}

// NewRepository returns a user repository backed by the given database.
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// CreateInput carries the fields needed to create a user. PasswordHash must
// already be hashed (see auth.HashPassword).
type CreateInput struct {
	Email        string
	Name         string
	Role         Role
	PasswordHash string
}

// Create inserts a new user and returns it. Email is normalized and must be
// unique; a duplicate returns an error.
func (r *Repository) Create(ctx context.Context, in CreateInput) (*User, error) {
	if err := validateForCreate(in.Email, in.Name, in.Role); err != nil {
		return nil, err
	}
	if in.PasswordHash == "" {
		return nil, fmt.Errorf("user: password hash is required")
	}
	now := time.Now().UTC()
	u := &User{
		ID:           uuid.NewString(),
		Email:        NormalizeEmail(in.Email),
		Name:         in.Name,
		Role:         in.Role,
		PasswordHash: in.PasswordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	q := r.db.Rebind(`INSERT INTO users (id, email, name, role, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	_, err := r.db.ExecContext(ctx, q, u.ID, u.Email, u.Name, string(u.Role), u.PasswordHash,
		u.CreatedAt.Format(time.RFC3339), u.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("user: create: %w", err)
	}
	return u, nil
}

// GetByID returns the user with the given id, or ErrNotFound.
func (r *Repository) GetByID(ctx context.Context, id string) (*User, error) {
	q := r.db.Rebind(`SELECT id, email, name, role, password_hash, created_at, updated_at
		FROM users WHERE id = ?`)
	return r.scanOne(r.db.QueryRowContext(ctx, q, id))
}

// GetByEmail returns the user with the given (normalized) email, or ErrNotFound.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	q := r.db.Rebind(`SELECT id, email, name, role, password_hash, created_at, updated_at
		FROM users WHERE email = ?`)
	return r.scanOne(r.db.QueryRowContext(ctx, q, NormalizeEmail(email)))
}

// Count returns the number of users. Used by the admin-bootstrap flow to detect
// a fresh instance.
func (r *Repository) Count(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("user: count: %w", err)
	}
	return n, nil
}

// List returns all users ordered by creation time.
func (r *Repository) List(ctx context.Context) ([]*User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, email, name, role, password_hash, created_at, updated_at
		FROM users ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("user: list: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u, err := scanUser(rows.Scan)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// scanner matches the Scan signature of both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func (r *Repository) scanOne(row scanner) (*User, error) {
	u, err := scanUser(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func scanUser(scan func(dest ...any) error) (*User, error) {
	var (
		u                  User
		role               string
		createdAt, updated string
	)
	if err := scan(&u.ID, &u.Email, &u.Name, &role, &u.PasswordHash, &createdAt, &updated); err != nil {
		return nil, err
	}
	u.Role = Role(role)
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &u, nil
}
