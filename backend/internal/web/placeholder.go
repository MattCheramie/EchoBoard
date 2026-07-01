//go:build !embed

package web

import (
	"errors"
	"io/fs"
)

// ErrNotEmbedded is returned by Assets in the default (non-embed) build, where
// the frontend is served separately by the Vite/SvelteKit dev server rather
// than from the binary.
var ErrNotEmbedded = errors.New("web: frontend not embedded in this build (build with -tags embed)")

// Assets returns ErrNotEmbedded in the default build. See embed.go for the
// single-binary release build.
func Assets() (fs.FS, error) {
	return nil, ErrNotEmbedded
}

// Embedded reports whether the frontend was compiled into this binary.
const Embedded = false
