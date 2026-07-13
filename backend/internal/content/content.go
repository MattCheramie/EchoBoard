// Package content owns the content model, media storage, approval workflow,
// search, and calendar scheduling queries. It is delivered across Tier 2; this
// file defines the core domain model (content, platform targets, drafts).
package content

import (
	"fmt"
	"strings"
	"time"
)

// Platform identifies a destination the content can be published to. The set
// mirrors the integrations named in the README; adapters land in Tier 4.
type Platform string

const (
	PlatformFacebook  Platform = "facebook"
	PlatformInstagram Platform = "instagram"
	PlatformTikTok    Platform = "tiktok"
	PlatformYouTube   Platform = "youtube"
	PlatformX         Platform = "x"
	PlatformSnapchat  Platform = "snapchat"
	PlatformShopify   Platform = "shopify"
	PlatformSpotify   Platform = "spotify"
)

// Valid reports whether p is a known platform.
func (p Platform) Valid() bool {
	switch p {
	case PlatformFacebook, PlatformInstagram, PlatformTikTok, PlatformYouTube,
		PlatformX, PlatformSnapchat, PlatformShopify, PlatformSpotify:
		return true
	default:
		return false
	}
}

// Status is where a piece of content sits in its lifecycle. The approval
// state machine (which transitions are legal) is enforced in PR 2.2; here the
// type and its members simply describe the model.
type Status string

const (
	// StatusDraft is the initial, editable state.
	StatusDraft Status = "draft"
	// StatusInReview is awaiting reviewer approval.
	StatusInReview Status = "in_review"
	// StatusApproved has been approved and may be scheduled.
	StatusApproved Status = "approved"
	// StatusScheduled is approved and queued for a future publish time.
	StatusScheduled Status = "scheduled"
	// StatusPublished has been dispatched to its platform targets.
	StatusPublished Status = "published"
	// StatusFailed failed to publish and needs attention.
	StatusFailed Status = "failed"
)

// Valid reports whether s is a known status.
func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusInReview, StatusApproved, StatusScheduled,
		StatusPublished, StatusFailed:
		return true
	default:
		return false
	}
}

// Content is a single planned post. Body holds the master draft copy; each
// Target may override it with platform-tailored copy. Content moves through the
// Status lifecycle and can be scheduled for a future publish time.
type Content struct {
	ID          string            `json:"id"`
	AuthorID    string            `json:"authorId"`
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Status      Status            `json:"status"`
	ScheduledAt *time.Time        `json:"scheduledAt,omitempty"`
	Targets     []Target          `json:"targets"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

// Target is a platform this content will publish to. Body, when non-empty,
// overrides Content.Body for that platform (e.g. a shorter X post versus a
// longer Instagram caption); an empty Body means "use the content's body".
type Target struct {
	ID        string   `json:"id"`
	ContentID string   `json:"contentId"`
	Platform  Platform `json:"platform"`
	Body      string   `json:"body,omitempty"`
}

// EffectiveBody returns the copy to publish for this target: its override if
// set, otherwise the content's master body.
func (t Target) EffectiveBody(c *Content) string {
	if strings.TrimSpace(t.Body) != "" {
		return t.Body
	}
	return c.Body
}

// normalizeTags trims, lower-cases, and de-duplicates tags, dropping empties
// while preserving first-seen order. It returns a non-nil (possibly empty)
// slice so JSON always renders "tags" as an array.
func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := make(map[string]bool, len(tags))
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// validateContent checks the fields common to creating and updating content.
// Targets must reference known platforms and be distinct.
func validateContent(title string, targets []Target) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("content: title is required")
	}
	seen := make(map[Platform]bool, len(targets))
	for _, t := range targets {
		if !t.Platform.Valid() {
			return fmt.Errorf("content: invalid platform %q", t.Platform)
		}
		if seen[t.Platform] {
			return fmt.Errorf("content: duplicate target platform %q", t.Platform)
		}
		seen[t.Platform] = true
	}
	return nil
}
