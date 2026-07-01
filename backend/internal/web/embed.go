//go:build embed

package web

import (
	"embed"
	"io/fs"
)

// buildFS holds the compiled SvelteKit output. Populated by `make embed`
// (Tier 6), which copies frontend/build into backend/internal/web/build
// before building with `-tags embed`.
//
//go:embed all:build
var buildFS embed.FS

// Assets is the frontend file system served by the API when the binary is
// built for single-binary release (`-tags embed`).
func Assets() (fs.FS, error) {
	return fs.Sub(buildFS, "build")
}

// Embedded reports whether the frontend was compiled into this binary.
const Embedded = true
