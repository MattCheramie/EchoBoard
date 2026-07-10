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

// ErrNotFound is returned when no row matches the query.
var ErrNotFound = errors.New("content: not found")

// execer is satisfied by both *sql.DB and *sql.Tx, so association helpers can
// run inside or outside a transaction.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Repository persists content, tags, media metadata, and their associations.
type Repository struct {
	db *db.DB
}

// NewRepository returns a content repository.
func NewRepository(database *db.DB) *Repository { return &Repository{db: database} }

// CreateInput carries the fields to create content.
type CreateInput struct {
	AuthorID    string
	Title       string
	Body        string
	Status      Status
	ScheduledAt *time.Time
	Targets     []string
	Tags        []string
	MediaIDs    []string
}

// CreateContent inserts content and its associations atomically.
func (r *Repository) CreateContent(ctx context.Context, in CreateInput) (*Content, error) {
	if in.AuthorID == "" {
		return nil, fmt.Errorf("content: author is required")
	}
	status := in.Status
	if status == "" {
		status = StatusDraft
	}
	id := uuid.NewString()
	now := time.Now().UTC()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after commit

	q := r.db.Rebind(`INSERT INTO content (id, author_id, title, body, status, scheduled_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if _, err := tx.ExecContext(ctx, q, id, in.AuthorID, in.Title, in.Body, string(status),
		formatTime(in.ScheduledAt), fmtTime(now), fmtTime(now)); err != nil {
		return nil, fmt.Errorf("content: insert: %w", err)
	}
	if err := r.replaceTargets(ctx, tx, id, in.Targets); err != nil {
		return nil, err
	}
	if err := r.replaceTags(ctx, tx, id, in.Tags); err != nil {
		return nil, err
	}
	if err := r.replaceMedia(ctx, tx, id, in.MediaIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetContent(ctx, id)
}

// UpdateInput is a partial update: nil scalar pointers and nil slices are left
// unchanged. ScheduledAt is applied only when SetSchedule is true (a nil
// ScheduledAt then clears the schedule).
type UpdateInput struct {
	Title       *string
	Body        *string
	Status      *Status
	ScheduledAt *time.Time
	SetSchedule bool
	Targets     *[]string
	Tags        *[]string
	MediaIDs    *[]string
}

// UpdateContent applies a partial update and returns the reloaded content.
func (r *Repository) UpdateContent(ctx context.Context, id string, in UpdateInput) (*Content, error) {
	if _, err := r.getContentRow(ctx, r.db, id); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after commit

	sets := []string{"updated_at = ?"}
	args := []any{fmtTime(time.Now().UTC())}
	if in.Title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *in.Title)
	}
	if in.Body != nil {
		sets = append(sets, "body = ?")
		args = append(args, *in.Body)
	}
	if in.Status != nil {
		sets = append(sets, "status = ?")
		args = append(args, string(*in.Status))
	}
	if in.SetSchedule {
		sets = append(sets, "scheduled_at = ?")
		args = append(args, formatTime(in.ScheduledAt))
	}
	args = append(args, id)
	q := r.db.Rebind("UPDATE content SET " + join(sets, ", ") + " WHERE id = ?")
	if _, err := tx.ExecContext(ctx, q, args...); err != nil {
		return nil, fmt.Errorf("content: update: %w", err)
	}

	if in.Targets != nil {
		if err := r.replaceTargets(ctx, tx, id, *in.Targets); err != nil {
			return nil, err
		}
	}
	if in.Tags != nil {
		if err := r.replaceTags(ctx, tx, id, *in.Tags); err != nil {
			return nil, err
		}
	}
	if in.MediaIDs != nil {
		if err := r.replaceMedia(ctx, tx, id, *in.MediaIDs); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetContent(ctx, id)
}

// GetContent returns content with its targets, tags, and media ids loaded.
func (r *Repository) GetContent(ctx context.Context, id string) (*Content, error) {
	c, err := r.getContentRow(ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, c)
}

// ListContent returns all content (associations loaded), newest first.
func (r *Repository) ListContent(ctx context.Context) ([]*Content, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, author_id, title, body, status, scheduled_at, created_at, updated_at
		FROM content ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("content: list: %w", err)
	}
	// Drain the cursor fully before hydrating: SQLite runs with a single
	// connection, so issuing the per-item association queries while this cursor
	// is still open would deadlock.
	var out []*Content
	for rows.Next() {
		c, err := scanContent(rows.Scan)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for _, c := range out {
		if _, err := r.hydrate(ctx, c); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// DeleteContent removes content and all of its association rows.
func (r *Repository) DeleteContent(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after commit
	for _, stmt := range []string{
		`DELETE FROM content_targets WHERE content_id = ?`,
		`DELETE FROM content_tags WHERE content_id = ?`,
		`DELETE FROM content_media WHERE content_id = ?`,
		`DELETE FROM content WHERE id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, r.db.Rebind(stmt), id); err != nil {
			return fmt.Errorf("content: delete: %w", err)
		}
	}
	return tx.Commit()
}

// --- tags ---

// GetOrCreateTag returns the tag with the given name, creating it if needed.
func (r *Repository) GetOrCreateTag(ctx context.Context, name string) (*Tag, error) {
	return r.getOrCreateTag(ctx, r.db, name)
}

func (r *Repository) getOrCreateTag(ctx context.Context, e execer, name string) (*Tag, error) {
	var id string
	q := r.db.Rebind(`SELECT id FROM tags WHERE name = ?`)
	err := e.QueryRowContext(ctx, q, name).Scan(&id)
	if err == nil {
		return &Tag{ID: id, Name: name}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	id = uuid.NewString()
	ins := r.db.Rebind(`INSERT INTO tags (id, name) VALUES (?, ?)`)
	if _, err := e.ExecContext(ctx, ins, id, name); err != nil {
		return nil, fmt.Errorf("content: create tag: %w", err)
	}
	return &Tag{ID: id, Name: name}, nil
}

// ListTags returns all tags ordered by name.
func (r *Repository) ListTags(ctx context.Context) ([]*Tag, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM tags ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("content: list tags: %w", err)
	}
	defer rows.Close()
	var out []*Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

// --- media metadata ---

// CreateMedia inserts media metadata.
func (r *Repository) CreateMedia(ctx context.Context, m *Media) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	q := r.db.Rebind(`INSERT INTO media (id, author_id, filename, content_type, size, storage_key, thumb_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	_, err := r.db.ExecContext(ctx, q, m.ID, m.AuthorID, m.Filename, m.ContentType, m.Size,
		m.StorageKey, m.ThumbKey, fmtTime(m.CreatedAt))
	if err != nil {
		return fmt.Errorf("content: insert media: %w", err)
	}
	return nil
}

// GetMedia returns media metadata by id.
func (r *Repository) GetMedia(ctx context.Context, id string) (*Media, error) {
	q := r.db.Rebind(`SELECT id, author_id, filename, content_type, size, storage_key, thumb_key, created_at
		FROM media WHERE id = ?`)
	var (
		m       Media
		created string
	)
	err := r.db.QueryRowContext(ctx, q, id).Scan(&m.ID, &m.AuthorID, &m.Filename, &m.ContentType,
		&m.Size, &m.StorageKey, &m.ThumbKey, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	m.CreatedAt = parseTime(created)
	return &m, nil
}

// DeleteMedia removes media metadata (the caller deletes the blob).
func (r *Repository) DeleteMedia(ctx context.Context, id string) error {
	q := r.db.Rebind(`DELETE FROM media WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// --- association helpers ---

func (r *Repository) replaceTargets(ctx context.Context, e execer, contentID string, targets []string) error {
	if _, err := e.ExecContext(ctx, r.db.Rebind(`DELETE FROM content_targets WHERE content_id = ?`), contentID); err != nil {
		return err
	}
	for _, p := range dedupe(targets) {
		q := r.db.Rebind(`INSERT INTO content_targets (content_id, platform) VALUES (?, ?)`)
		if _, err := e.ExecContext(ctx, q, contentID, p); err != nil {
			return fmt.Errorf("content: set target: %w", err)
		}
	}
	return nil
}

func (r *Repository) replaceTags(ctx context.Context, e execer, contentID string, names []string) error {
	if _, err := e.ExecContext(ctx, r.db.Rebind(`DELETE FROM content_tags WHERE content_id = ?`), contentID); err != nil {
		return err
	}
	for _, name := range dedupe(names) {
		tag, err := r.getOrCreateTag(ctx, e, name)
		if err != nil {
			return err
		}
		q := r.db.Rebind(`INSERT INTO content_tags (content_id, tag_id) VALUES (?, ?)`)
		if _, err := e.ExecContext(ctx, q, contentID, tag.ID); err != nil {
			return fmt.Errorf("content: set tag: %w", err)
		}
	}
	return nil
}

func (r *Repository) replaceMedia(ctx context.Context, e execer, contentID string, mediaIDs []string) error {
	if _, err := e.ExecContext(ctx, r.db.Rebind(`DELETE FROM content_media WHERE content_id = ?`), contentID); err != nil {
		return err
	}
	for _, mid := range dedupe(mediaIDs) {
		q := r.db.Rebind(`INSERT INTO content_media (content_id, media_id) VALUES (?, ?)`)
		if _, err := e.ExecContext(ctx, q, contentID, mid); err != nil {
			return fmt.Errorf("content: set media: %w", err)
		}
	}
	return nil
}

func (r *Repository) hydrate(ctx context.Context, c *Content) (*Content, error) {
	var err error
	if c.Targets, err = r.scanStrings(ctx, `SELECT platform FROM content_targets WHERE content_id = ? ORDER BY platform`, c.ID); err != nil {
		return nil, err
	}
	if c.Tags, err = r.scanStrings(ctx, `SELECT t.name FROM content_tags ct JOIN tags t ON t.id = ct.tag_id WHERE ct.content_id = ? ORDER BY t.name`, c.ID); err != nil {
		return nil, err
	}
	if c.MediaIDs, err = r.scanStrings(ctx, `SELECT media_id FROM content_media WHERE content_id = ?`, c.ID); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *Repository) scanStrings(ctx context.Context, query, arg string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, r.db.Rebind(query), arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) getContentRow(ctx context.Context, e execer, id string) (*Content, error) {
	q := r.db.Rebind(`SELECT id, author_id, title, body, status, scheduled_at, created_at, updated_at
		FROM content WHERE id = ?`)
	c, err := scanContent(e.QueryRowContext(ctx, q, id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

func scanContent(scan func(dest ...any) error) (*Content, error) {
	var (
		c                    Content
		status, scheduled    string
		createdAt, updatedAt string
	)
	if err := scan(&c.ID, &c.AuthorID, &c.Title, &c.Body, &status, &scheduled, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	c.Status = Status(status)
	if scheduled != "" {
		t := parseTime(scheduled)
		c.ScheduledAt = &t
	}
	c.CreatedAt = parseTime(createdAt)
	c.UpdatedAt = parseTime(updatedAt)
	c.Targets = []string{}
	c.Tags = []string{}
	c.MediaIDs = []string{}
	return &c, nil
}
