package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MattCheramie/echoboard/internal/db"
)

// ErrNotFound is returned when no media matches the query.
var ErrNotFound = errors.New("media: not found")

// Repository persists media metadata (the bytes live in a Store).
type Repository struct {
	db *db.DB
}

// NewRepository returns a media repository backed by the given database.
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// Insert stores a media metadata row.
func (r *Repository) Insert(ctx context.Context, m *Media) error {
	q := r.db.Rebind(`INSERT INTO media
		(id, owner_id, content_id, filename, mime_type, size, width, height, storage_key, thumbnail_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	_, err := r.db.ExecContext(ctx, q, m.ID, m.OwnerID, nullStr(m.ContentID), m.Filename, m.MimeType,
		m.Size, m.Width, m.Height, m.StorageKey, m.ThumbnailKey, m.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("media: insert: %w", err)
	}
	return nil
}

// GetByID returns the media row with the given id, or ErrNotFound.
func (r *Repository) GetByID(ctx context.Context, id string) (*Media, error) {
	q := r.db.Rebind(`SELECT id, owner_id, content_id, filename, mime_type, size, width, height, storage_key, thumbnail_key, created_at
		FROM media WHERE id = ?`)
	m, err := scanMedia(r.db.QueryRowContext(ctx, q, id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("media: get: %w", err)
	}
	return m, nil
}

// ListByContent returns the media attached to a content id, oldest first.
func (r *Repository) ListByContent(ctx context.Context, contentID string) ([]*Media, error) {
	q := r.db.Rebind(`SELECT id, owner_id, content_id, filename, mime_type, size, width, height, storage_key, thumbnail_key, created_at
		FROM media WHERE content_id = ? ORDER BY created_at`)
	rows, err := r.db.QueryContext(ctx, q, contentID)
	if err != nil {
		return nil, fmt.Errorf("media: list: %w", err)
	}
	defer rows.Close()

	var out []*Media
	for rows.Next() {
		m, err := scanMedia(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Attach links a media asset to a content record.
func (r *Repository) Attach(ctx context.Context, mediaID, contentID string) error {
	q := r.db.Rebind(`UPDATE media SET content_id = ? WHERE id = ?`)
	res, err := r.db.ExecContext(ctx, q, contentID, mediaID)
	if err != nil {
		return fmt.Errorf("media: attach: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a media metadata row.
func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, r.db.Rebind(`DELETE FROM media WHERE id = ?`), id)
	if err != nil {
		return fmt.Errorf("media: delete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanMedia(scan func(dest ...any) error) (*Media, error) {
	var (
		m         Media
		contentID sql.NullString
		createdAt string
	)
	if err := scan(&m.ID, &m.OwnerID, &contentID, &m.Filename, &m.MimeType, &m.Size,
		&m.Width, &m.Height, &m.StorageKey, &m.ThumbnailKey, &createdAt); err != nil {
		return nil, err
	}
	if contentID.Valid {
		v := contentID.String
		m.ContentID = &v
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &m, nil
}

func nullStr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
