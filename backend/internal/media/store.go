package media

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Store is a blob store keyed by opaque string keys. The default
// implementation writes to the local filesystem; production deployments can
// swap in an object-store backend without touching callers.
type Store interface {
	// Put writes r under key and returns the number of bytes stored.
	Put(ctx context.Context, key string, r io.Reader) (int64, error)
	// Open returns a reader for the blob at key. The caller must Close it.
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the blob at key. Removing a missing key is not an error.
	Delete(ctx context.Context, key string) error
}

// FSStore stores blobs as files under a root directory. Keys are treated as
// slash-separated relative paths; traversal outside the root is rejected.
type FSStore struct {
	root string
}

// NewFSStore returns a filesystem-backed store rooted at dir, creating it if
// necessary.
func NewFSStore(dir string) (*FSStore, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("media: store dir is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("media: create store dir: %w", err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("media: resolve store dir: %w", err)
	}
	return &FSStore{root: abs}, nil
}

func (s *FSStore) Put(_ context.Context, key string, r io.Reader) (int64, error) {
	path, err := s.resolve(key)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, fmt.Errorf("media: put: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("media: put: %w", err)
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return n, fmt.Errorf("media: put: %w", err)
	}
	return n, nil
}

func (s *FSStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("media: open: %w", err)
	}
	return f, nil
}

func (s *FSStore) Delete(_ context.Context, key string) error {
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("media: delete: %w", err)
	}
	return nil
}

// resolve maps a store key to an absolute path under the root, rejecting any
// key that would escape it (path traversal).
func (s *FSStore) resolve(key string) (string, error) {
	clean := filepath.Clean("/" + filepath.FromSlash(key))
	path := filepath.Join(s.root, clean)
	if path != s.root && !strings.HasPrefix(path, s.root+string(os.PathSeparator)) {
		return "", fmt.Errorf("media: invalid key %q", key)
	}
	return path, nil
}
