package content

import (
	"bytes"
	"context"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/MattCheramie/echoboard/internal/media"
	"github.com/google/uuid"
)

// thumbnailMaxDim is the bounding box (px) for generated thumbnails.
const thumbnailMaxDim = 256

// Service is the content façade the API depends on: it wraps the repository and
// the media blob store, owning media upload (store blob + thumbnail + metadata)
// and retrieval, and delegating content/tag persistence to the repository.
type Service struct {
	repo    *Repository
	storage media.Storage
}

// NewService ties the repository to a media blob store.
func NewService(repo *Repository, storage media.Storage) *Service {
	return &Service{repo: repo, storage: storage}
}

// --- content / tags (delegated) ---

func (s *Service) CreateContent(ctx context.Context, in CreateInput) (*Content, error) {
	return s.repo.CreateContent(ctx, in)
}
func (s *Service) GetContent(ctx context.Context, id string) (*Content, error) {
	return s.repo.GetContent(ctx, id)
}
func (s *Service) ListContent(ctx context.Context) ([]*Content, error) {
	return s.repo.ListContent(ctx)
}
func (s *Service) UpdateContent(ctx context.Context, id string, in UpdateInput) (*Content, error) {
	return s.repo.UpdateContent(ctx, id, in)
}
func (s *Service) DeleteContent(ctx context.Context, id string) error {
	return s.repo.DeleteContent(ctx, id)
}
func (s *Service) ListTags(ctx context.Context) ([]*Tag, error) { return s.repo.ListTags(ctx) }
func (s *Service) CreateTag(ctx context.Context, name string) (*Tag, error) {
	return s.repo.GetOrCreateTag(ctx, strings.TrimSpace(name))
}

// --- media ---

// UploadMedia stores the blob, generates a thumbnail when the input is an image
// (best-effort), and records the metadata. On a metadata failure the stored
// blobs are cleaned up so nothing is orphaned.
func (s *Service) UploadMedia(ctx context.Context, authorID, filename, contentType string, r io.Reader) (*Media, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	key := id + extFor(filename, contentType)
	if _, err := s.storage.Put(ctx, key, bytes.NewReader(data)); err != nil {
		return nil, err
	}

	m := &Media{
		ID:          id,
		AuthorID:    authorID,
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(data)),
		StorageKey:  key,
	}

	// Best-effort thumbnail for images; non-images are simply skipped.
	if thumb, terr := media.Thumbnail(bytes.NewReader(data), thumbnailMaxDim); terr == nil {
		thumbKey := id + "_thumb.jpg"
		if _, perr := s.storage.Put(ctx, thumbKey, bytes.NewReader(thumb)); perr == nil {
			m.ThumbKey = thumbKey
		}
	}

	if err := s.repo.CreateMedia(ctx, m); err != nil {
		_ = s.storage.Delete(ctx, key)
		if m.ThumbKey != "" {
			_ = s.storage.Delete(ctx, m.ThumbKey)
		}
		return nil, err
	}
	return m, nil
}

// OpenMedia returns the metadata plus a reader for the original blob, or the
// thumbnail when thumb is true and one exists. The caller closes the reader.
func (s *Service) OpenMedia(ctx context.Context, id string, thumb bool) (*Media, io.ReadCloser, string, error) {
	m, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		return nil, nil, "", err
	}
	key, ct := m.StorageKey, m.ContentType
	if thumb && m.ThumbKey != "" {
		key, ct = m.ThumbKey, media.ThumbnailContentType
	}
	rc, err := s.storage.Open(ctx, key)
	if err != nil {
		return nil, nil, "", err
	}
	return m, rc, ct, nil
}

// DeleteMedia removes the metadata and the underlying blobs.
func (s *Service) DeleteMedia(ctx context.Context, id string) error {
	m, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		return err
	}
	_ = s.storage.Delete(ctx, m.StorageKey)
	if m.ThumbKey != "" {
		_ = s.storage.Delete(ctx, m.ThumbKey)
	}
	return s.repo.DeleteMedia(ctx, id)
}

// extFor picks a file extension from the filename, falling back to the MIME type.
func extFor(filename, contentType string) string {
	if ext := filepath.Ext(filename); ext != "" && len(ext) <= 6 {
		return strings.ToLower(ext)
	}
	if exts, err := mime.ExtensionsByType(contentType); err == nil && len(exts) > 0 {
		return exts[0]
	}
	return ""
}
