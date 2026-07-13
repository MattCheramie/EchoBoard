package content

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MattCheramie/echoboard/internal/db"
	"github.com/google/uuid"
)

// ErrNotFound is returned when no content matches the query.
var ErrNotFound = errors.New("content: not found")

// Repository persists content and its platform targets.
type Repository struct {
	db *db.DB
}

// NewRepository returns a content repository backed by the given database.
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// CreateInput carries the fields needed to create a piece of content. Status
// defaults to StatusDraft when empty.
type CreateInput struct {
	AuthorID    string
	Title       string
	Body        string
	Status      Status
	ScheduledAt *time.Time
	Targets     []Target
}

// Create inserts content and its targets in a single transaction and returns
// the stored record.
func (r *Repository) Create(ctx context.Context, in CreateInput) (*Content, error) {
	if in.AuthorID == "" {
		return nil, fmt.Errorf("content: authorID is required")
	}
	if err := validateContent(in.Title, in.Targets); err != nil {
		return nil, err
	}
	status := in.Status
	if status == "" {
		status = StatusDraft
	}
	if !status.Valid() {
		return nil, fmt.Errorf("content: invalid status %q", status)
	}

	now := time.Now().UTC()
	c := &Content{
		ID:          uuid.NewString(),
		AuthorID:    in.AuthorID,
		Title:       in.Title,
		Body:        in.Body,
		Status:      status,
		ScheduledAt: normalizeTime(in.ScheduledAt),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	c.Targets = make([]Target, len(in.Targets))
	for i, t := range in.Targets {
		c.Targets[i] = Target{
			ID:        uuid.NewString(),
			ContentID: c.ID,
			Platform:  t.Platform,
			Body:      t.Body,
		}
	}

	err := r.inTx(ctx, func(tx *sql.Tx) error {
		q := r.db.Rebind(`INSERT INTO content (id, author_id, title, body, status, scheduled_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
		if _, err := tx.ExecContext(ctx, q, c.ID, c.AuthorID, c.Title, c.Body, string(c.Status),
			nullTime(c.ScheduledAt), c.CreatedAt.Format(time.RFC3339), c.UpdatedAt.Format(time.RFC3339)); err != nil {
			return err
		}
		return r.insertTargets(ctx, tx, c.Targets)
	})
	if err != nil {
		return nil, fmt.Errorf("content: create: %w", err)
	}
	return c, nil
}

// GetByID returns the content with the given id (including targets), or
// ErrNotFound.
func (r *Repository) GetByID(ctx context.Context, id string) (*Content, error) {
	q := r.db.Rebind(`SELECT id, author_id, title, body, status, scheduled_at, created_at, updated_at
		FROM content WHERE id = ?`)
	c, err := scanContent(r.db.QueryRowContext(ctx, q, id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("content: get: %w", err)
	}
	if c.Targets, err = r.loadTargets(ctx, c.ID); err != nil {
		return nil, err
	}
	return c, nil
}

// ListByAuthor returns an author's content, newest first, with targets loaded.
func (r *Repository) ListByAuthor(ctx context.Context, authorID string) ([]*Content, error) {
	q := r.db.Rebind(`SELECT id, author_id, title, body, status, scheduled_at, created_at, updated_at
		FROM content WHERE author_id = ? ORDER BY created_at DESC`)
	rows, err := r.db.QueryContext(ctx, q, authorID)
	if err != nil {
		return nil, fmt.Errorf("content: list: %w", err)
	}
	return r.collect(ctx, rows)
}

// UpdateInput carries the mutable fields of a content record. Targets fully
// replace the existing set.
type UpdateInput struct {
	Title       string
	Body        string
	Status      Status
	ScheduledAt *time.Time
	Targets     []Target
}

// Update rewrites the mutable fields of a content record and replaces its
// targets, in a single transaction.
func (r *Repository) Update(ctx context.Context, id string, in UpdateInput) (*Content, error) {
	if err := validateContent(in.Title, in.Targets); err != nil {
		return nil, err
	}
	if !in.Status.Valid() {
		return nil, fmt.Errorf("content: invalid status %q", in.Status)
	}

	targets := make([]Target, len(in.Targets))
	for i, t := range in.Targets {
		targets[i] = Target{
			ID:        uuid.NewString(),
			ContentID: id,
			Platform:  t.Platform,
			Body:      t.Body,
		}
	}
	updatedAt := time.Now().UTC()

	err := r.inTx(ctx, func(tx *sql.Tx) error {
		q := r.db.Rebind(`UPDATE content SET title = ?, body = ?, status = ?, scheduled_at = ?, updated_at = ?
			WHERE id = ?`)
		res, err := tx.ExecContext(ctx, q, in.Title, in.Body, string(in.Status),
			nullTime(normalizeTime(in.ScheduledAt)), updatedAt.Format(time.RFC3339), id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
		del := r.db.Rebind(`DELETE FROM content_targets WHERE content_id = ?`)
		if _, err := tx.ExecContext(ctx, del, id); err != nil {
			return err
		}
		return r.insertTargets(ctx, tx, targets)
	})
	if errors.Is(err, ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("content: update: %w", err)
	}
	return r.GetByID(ctx, id)
}

// SetStatus updates only the lifecycle status of a content record. Transition
// legality is enforced by the workflow service in PR 2.2.
func (r *Repository) SetStatus(ctx context.Context, id string, status Status) error {
	if !status.Valid() {
		return fmt.Errorf("content: invalid status %q", status)
	}
	q := r.db.Rebind(`UPDATE content SET status = ?, updated_at = ? WHERE id = ?`)
	res, err := r.db.ExecContext(ctx, q, string(status), time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("content: set status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a content record and its targets.
func (r *Repository) Delete(ctx context.Context, id string) error {
	err := r.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, r.db.Rebind(`DELETE FROM content_targets WHERE content_id = ?`), id); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, r.db.Rebind(`DELETE FROM content WHERE id = ?`), id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
		return nil
	})
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("content: delete: %w", err)
	}
	return nil
}

// --- helpers ---

func (r *Repository) insertTargets(ctx context.Context, tx *sql.Tx, targets []Target) error {
	q := r.db.Rebind(`INSERT INTO content_targets (id, content_id, platform, body) VALUES (?, ?, ?, ?)`)
	for _, t := range targets {
		if _, err := tx.ExecContext(ctx, q, t.ID, t.ContentID, string(t.Platform), t.Body); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) loadTargets(ctx context.Context, contentID string) ([]Target, error) {
	q := r.db.Rebind(`SELECT id, content_id, platform, body FROM content_targets
		WHERE content_id = ? ORDER BY platform`)
	rows, err := r.db.QueryContext(ctx, q, contentID)
	if err != nil {
		return nil, fmt.Errorf("content: load targets: %w", err)
	}
	defer rows.Close()

	var targets []Target
	for rows.Next() {
		var t Target
		var platform string
		if err := rows.Scan(&t.ID, &t.ContentID, &platform, &t.Body); err != nil {
			return nil, err
		}
		t.Platform = Platform(platform)
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

func (r *Repository) collect(ctx context.Context, rows *sql.Rows) ([]*Content, error) {
	defer rows.Close()
	var out []*Content
	for rows.Next() {
		c, err := scanContent(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, c := range out {
		targets, err := r.loadTargets(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		c.Targets = targets
	}
	return out, nil
}

func (r *Repository) inTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after a successful commit
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func scanContent(scan func(dest ...any) error) (*Content, error) {
	var (
		c                    Content
		status               string
		scheduledAt          sql.NullString
		createdAt, updatedAt string
	)
	if err := scan(&c.ID, &c.AuthorID, &c.Title, &c.Body, &status, &scheduledAt, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	c.Status = Status(status)
	if scheduledAt.Valid && scheduledAt.String != "" {
		if t, err := time.Parse(time.RFC3339, scheduledAt.String); err == nil {
			c.ScheduledAt = &t
		}
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &c, nil
}

// normalizeTime returns a UTC copy of t, or nil.
func normalizeTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

// nullTime formats t for storage, or returns nil so a NULL is written.
func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
