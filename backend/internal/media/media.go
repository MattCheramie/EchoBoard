// Package media handles uploaded media: a storage abstraction (local
// filesystem today, pluggable later), image thumbnailing, and a repository
// recording each asset's metadata. Content attaches media by id.
package media

import "time"

// Media is a stored asset. StorageKey and ThumbnailKey are opaque keys into the
// Store; they are never absolute filesystem paths exposed to clients.
type Media struct {
	ID           string    `json:"id"`
	OwnerID      string    `json:"ownerId"`
	ContentID    *string   `json:"contentId,omitempty"`
	Filename     string    `json:"filename"`
	MimeType     string    `json:"mimeType"`
	Size         int64     `json:"size"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	StorageKey   string    `json:"-"`
	ThumbnailKey string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

// HasThumbnail reports whether a generated thumbnail exists for this asset.
func (m *Media) HasThumbnail() bool { return m.ThumbnailKey != "" }
