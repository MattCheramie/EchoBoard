// Package integrations is the framework external platform adapters plug into:
// a normalized Adapter interface, a registry that maps a Platform to its
// adapter, an OAuth2 connect/refresh flow whose tokens are encrypted at rest via
// the secrets vault, and webhook signature verification. The concrete platform
// adapters (Meta, TikTok, YouTube, X, Snapchat, Shopify, Spotify) are added in
// Tier 4 PRs 4.2–4.5; this PR ships the framework plus a built-in sandbox
// adapter so the whole publish/webhook pipeline is exercisable without any live
// provider credentials.
package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"sync"
	"time"
)

// Platform identifies an external integration.
type Platform string

const (
	// PlatformSandbox is the built-in, credential-free adapter used for local
	// development and end-to-end tests.
	PlatformSandbox Platform = "sandbox"

	// The real platforms, implemented in Tier 4 PRs 4.2–4.5. Declared here so
	// the connect route can validate a platform name before an adapter exists.
	PlatformMeta      Platform = "meta"
	PlatformInstagram Platform = "instagram"
	PlatformTikTok    Platform = "tiktok"
	PlatformYouTube   Platform = "youtube"
	PlatformX         Platform = "x"
	PlatformSnapchat  Platform = "snapchat"
	PlatformShopify   Platform = "shopify"
	PlatformSpotify   Platform = "spotify"
)

// Framework errors.
var (
	ErrUnknownPlatform = errors.New("integrations: unknown platform")
	ErrNotConnected    = errors.New("integrations: platform is not connected")
	ErrNotConfigured   = errors.New("integrations: platform is not configured (missing operator credentials)")
	ErrInvalidState    = errors.New("integrations: invalid or expired oauth state")
	ErrBadSignature    = errors.New("integrations: webhook signature verification failed")
	ErrNoOAuth         = errors.New("integrations: adapter does not use oauth")
)

// Capabilities describes what an adapter supports.
type Capabilities struct {
	Publish  bool `json:"publish"`
	OAuth    bool `json:"oauth"`
	Webhooks bool `json:"webhooks"`
}

// Endpoints is a provider's OAuth2 authorize/token configuration.
type Endpoints struct {
	AuthURL  string
	TokenURL string
	Scopes   []string
}

// Content is the platform-agnostic payload an adapter publishes. The Tier 2
// content model maps its records onto this before dispatch.
type Content struct {
	ID        string
	Body      string
	MediaURLs []string
}

// ExternalRef identifies published content on the provider side.
type ExternalRef struct {
	Platform   Platform `json:"platform"`
	ExternalID string   `json:"externalId"`
	URL        string   `json:"url,omitempty"`
}

// WebhookRequest is a received webhook, handed to an adapter for verification
// and parsing.
type WebhookRequest struct {
	Header http.Header
	Body   []byte
}

// InboundEvent is a normalized webhook event (comment, DM, mention, …). Routing
// events into the interactions inbox lands in Tier 3 / PR 4.2; this PR records
// them.
type InboundEvent struct {
	Type       string          `json:"type"`
	ExternalID string          `json:"externalId,omitempty"`
	Raw        json.RawMessage `json:"raw,omitempty"`
}

// Adapter is the interface every platform integration implements. Adapters are
// stateless: the framework owns token storage/refresh and passes a valid access
// token into Publish, so an adapter never touches the vault.
type Adapter interface {
	Platform() Platform
	Capabilities() Capabilities
	// OAuthEndpoints returns the provider's OAuth2 endpoints and default scopes;
	// ok is false for adapters that do not use OAuth (e.g. the sandbox).
	OAuthEndpoints() (ep Endpoints, ok bool)
	// Publish sends content using a valid access token and returns the ref.
	Publish(ctx context.Context, accessToken string, c Content) (ExternalRef, error)
	// VerifyWebhook validates the signature of an inbound webhook request.
	VerifyWebhook(secret string, req WebhookRequest) error
	// ParseWebhook normalizes a (already verified) webhook body into events.
	ParseWebhook(req WebhookRequest) ([]InboundEvent, error)
}

// Registry maps a Platform to its Adapter. It is safe for concurrent use.
type Registry struct {
	mu       sync.RWMutex
	adapters map[Platform]Adapter
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{adapters: make(map[Platform]Adapter)}
}

// Register adds (or replaces) an adapter for its platform.
func (r *Registry) Register(a Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[a.Platform()] = a
}

// Get returns the adapter for a platform, or ok=false.
func (r *Registry) Get(p Platform) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[p]
	return a, ok
}

// Platforms returns the registered platforms in a stable, sorted order.
func (r *Registry) Platforms() []Platform {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Platform, 0, len(r.adapters))
	for p := range r.adapters {
		out = append(out, p)
	}
	sortPlatforms(out)
	return out
}

func sortPlatforms(ps []Platform) {
	sort.Slice(ps, func(i, j int) bool { return ps[i] < ps[j] })
}

// Token is a provider credential. It is stored encrypted (see Repository) and
// never serialized to API clients.
type Token struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// Expired reports whether the token is expired at now, allowing a refresh skew.
// A zero ExpiresAt means "unknown/never expires" and is treated as not expired.
func (t Token) Expired(now time.Time, skew time.Duration) bool {
	if t.ExpiresAt.IsZero() {
		return false
	}
	return now.Add(skew).After(t.ExpiresAt)
}
