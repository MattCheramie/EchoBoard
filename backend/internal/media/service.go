package media

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service ties the blob Store and the metadata Repository together and owns the
// upload pipeline: persist the bytes, thumbnail images, record metadata.
type Service struct {
	store   Store
	repo    *Repository
	maxDim  int
	nowFunc func() time.Time
}

// NewService constructs a media service. thumbMaxDim <= 0 uses
// DefaultThumbMaxDim.
func NewService(store Store, repo *Repository, thumbMaxDim int) *Service {
	if thumbMaxDim <= 0 {
		thumbMaxDim = DefaultThumbMaxDim
	}
	return &Service{store: store, repo: repo, maxDim: thumbMaxDim, nowFunc: func() time.Time { return time.Now().UTC() }}
}

// UploadInput describes a single upload. ContentID is optional; media can be
// uploaded first and attached to content later.
type UploadInput struct {
	OwnerID   string
	ContentID *string
	Filename  string
	MimeType  string
	Data      io.Reader
}

// Upload stores the asset's bytes, generates a thumbnail for images, records
// the metadata, and returns the stored Media.
func (s *Service) Upload(ctx context.Context, in UploadInput) (*Media, error) {
	if strings.TrimSpace(in.OwnerID) == "" {
		return nil, fmt.Errorf("media: ownerID is required")
	}
	if in.Data == nil {
		return nil, fmt.Errorf("media: data is required")
	}

	id := uuid.NewString()
	ext := strings.ToLower(filepath.Ext(in.Filename))
	storageKey := "originals/" + id + ext

	size, err := s.store.Put(ctx, storageKey, in.Data)
	if err != nil {
		return nil, err
	}

	m := &Media{
		ID:         id,
		OwnerID:    in.OwnerID,
		ContentID:  in.ContentID,
		Filename:   in.Filename,
		MimeType:   in.MimeType,
		Size:       size,
		StorageKey: storageKey,
		CreatedAt:  s.nowFunc(),
	}

	// Thumbnail images. Non-images (or undecodable bytes) simply get no
	// thumbnail — not an error.
	if isImage(in.MimeType) {
		if err := s.thumbnail(ctx, m); err != nil {
			// Roll back the stored original so we don't leak an orphan blob.
			_ = s.store.Delete(ctx, storageKey)
			return nil, err
		}
	}

	if err := s.repo.Insert(ctx, m); err != nil {
		_ = s.store.Delete(ctx, storageKey)
		if m.ThumbnailKey != "" {
			_ = s.store.Delete(ctx, m.ThumbnailKey)
		}
		return nil, err
	}
	return m, nil
}

// Open returns a reader for a stored asset's original bytes.
func (s *Service) Open(ctx context.Context, m *Media) (io.ReadCloser, error) {
	return s.store.Open(ctx, m.StorageKey)
}

// OpenThumbnail returns a reader for an asset's thumbnail, or ErrNotFound when
// the asset has none.
func (s *Service) OpenThumbnail(ctx context.Context, m *Media) (io.ReadCloser, error) {
	if !m.HasThumbnail() {
		return nil, ErrNotFound
	}
	return s.store.Open(ctx, m.ThumbnailKey)
}

// Delete removes an asset's metadata and its stored bytes (original and
// thumbnail).
func (s *Service) Delete(ctx context.Context, id string) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.store.Delete(ctx, m.StorageKey)
	if m.ThumbnailKey != "" {
		_ = s.store.Delete(ctx, m.ThumbnailKey)
	}
	return nil
}

// thumbnail re-reads the stored original, generates a thumbnail, stores it, and
// records the dimensions on m.
func (s *Service) thumbnail(ctx context.Context, m *Media) error {
	rc, err := s.store.Open(ctx, m.StorageKey)
	if err != nil {
		return err
	}
	defer rc.Close()

	data, srcW, srcH, ok, err := Thumbnail(rc, s.maxDim)
	if err != nil {
		return err
	}
	if !ok {
		// Declared image/* but not decodable; leave without a thumbnail.
		return nil
	}
	m.Width, m.Height = srcW, srcH

	thumbKey := "thumbnails/" + m.ID + ".png"
	if _, err := s.store.Put(ctx, thumbKey, bytes.NewReader(data)); err != nil {
		return err
	}
	m.ThumbnailKey = thumbKey
	return nil
}

func isImage(mimeType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mimeType)), "image/")
}
