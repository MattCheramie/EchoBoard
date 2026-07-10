// Package media provides the blob-storage abstraction for uploaded media
// (images, video) plus thumbnail generation. Storage is an interface so the
// local-disk backend used today can be swapped for an S3-compatible backend
// later without touching callers; the DB metadata for media lives in the
// content package.
package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotFound is returned when a key does not exist.
var ErrNotFound = errors.New("media: not found")

// Storage persists and retrieves media blobs by an opaque key.
type Storage interface {
	// Put stores r under key, returning the number of bytes written.
	Put(ctx context.Context, key string, r io.Reader) (int64, error)
	// Open returns a reader for key, or ErrNotFound.
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes key. Removing a missing key is not an error.
	Delete(ctx context.Context, key string) error
}

// LocalStorage stores blobs as files under a root directory.
type LocalStorage struct {
	root string
}

// NewLocalStorage creates the root directory if needed and returns a store.
func NewLocalStorage(root string) (*LocalStorage, error) {
	if root == "" {
		return nil, fmt.Errorf("media: local storage root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("media: create root %q: %w", root, err)
	}
	return &LocalStorage{root: root}, nil
}

// safePath resolves key to a path inside root, rejecting traversal and
// separators so a key can never escape the media directory.
func (s *LocalStorage) safePath(key string) (string, error) {
	if key == "" || key != filepath.Base(key) || strings.Contains(key, "..") {
		return "", fmt.Errorf("media: invalid key %q", key)
	}
	return filepath.Join(s.root, key), nil
}

func (s *LocalStorage) Put(_ context.Context, key string, r io.Reader) (int64, error) {
	path, err := s.safePath(key)
	if err != nil {
		return 0, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, fmt.Errorf("media: create %q: %w", key, err)
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return n, fmt.Errorf("media: write %q: %w", key, err)
	}
	return n, nil
}

func (s *LocalStorage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := s.safePath(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("media: open %q: %w", key, err)
	}
	return f, nil
}

func (s *LocalStorage) Delete(_ context.Context, key string) error {
	path, err := s.safePath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("media: delete %q: %w", key, err)
	}
	return nil
}
