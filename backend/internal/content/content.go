// Package content owns the content model (drafts/posts), their platform
// targets, tags, and uploaded media metadata. The approval workflow and search
// (PR 2.2), calendar queries (PR 2.3), and scheduler dispatch (PR 2.4) build on
// this. Media blobs are stored via the media.Storage abstraction; this package
// owns their database metadata.
package content

import (
	"encoding/json"
	"time"
)

// Status is a content item's lifecycle state. The full set is declared here;
// the legal transitions between them are enforced by the workflow in PR 2.2.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusInReview  Status = "in_review"
	StatusApproved  Status = "approved"
	StatusScheduled Status = "scheduled"
	StatusPublished Status = "published"
	StatusFailed    Status = "failed"
)

// Content is a single piece of content authored in EchoBoard.
type Content struct {
	ID          string     `json:"id"`
	AuthorID    string     `json:"authorId"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	Status      Status     `json:"status"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
	Targets     []string   `json:"targets"`  // platform names (see integrations.Platform)
	Tags        []string   `json:"tags"`     // tag names
	MediaIDs    []string   `json:"mediaIds"` // associated media ids
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// Tag is a reusable label for organizing and searching content.
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Media is the database metadata for an uploaded blob. StorageKey/ThumbKey are
// internal storage paths and are never serialized to clients; HasThumbnail is
// exposed instead so the UI knows whether a thumbnail is available.
type Media struct {
	ID          string    `json:"id"`
	AuthorID    string    `json:"authorId"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	StorageKey  string    `json:"-"`
	ThumbKey    string    `json:"-"`
	CreatedAt   time.Time `json:"createdAt"`
}

// HasThumbnail reports whether a thumbnail was generated for this media.
func (m Media) HasThumbnail() bool { return m.ThumbKey != "" }

// MarshalJSON adds the derived hasThumbnail field without leaking storage keys.
func (m Media) MarshalJSON() ([]byte, error) {
	type alias Media // alias drops the custom marshaler, avoiding recursion
	return json.Marshal(struct {
		alias
		HasThumbnail bool `json:"hasThumbnail"`
	}{alias(m), m.HasThumbnail()})
}
