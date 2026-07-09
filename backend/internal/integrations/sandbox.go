package integrations

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

// Sandbox is a built-in, credential-free adapter. It "publishes" to an in-memory
// log and verifies webhooks with a plain HMAC secret, letting the full
// publish/webhook pipeline run end-to-end in development and tests without any
// live provider. It does not use OAuth: connecting it stores a placeholder
// connection immediately (see Service.Connect).
type Sandbox struct {
	mu        sync.Mutex
	published []ExternalRef
}

// NewSandbox returns a ready sandbox adapter.
func NewSandbox() *Sandbox { return &Sandbox{} }

func (s *Sandbox) Platform() Platform { return PlatformSandbox }

func (s *Sandbox) Capabilities() Capabilities {
	return Capabilities{Publish: true, OAuth: false, Webhooks: true}
}

func (s *Sandbox) OAuthEndpoints() (Endpoints, bool) { return Endpoints{}, false }

func (s *Sandbox) Publish(_ context.Context, _ string, c Content) (ExternalRef, error) {
	ref := ExternalRef{
		Platform:   PlatformSandbox,
		ExternalID: "sbx_" + uuid.NewString(),
		URL:        "sandbox://post/" + c.ID,
	}
	s.mu.Lock()
	s.published = append(s.published, ref)
	s.mu.Unlock()
	return ref, nil
}

func (s *Sandbox) VerifyWebhook(secret string, req WebhookRequest) error {
	return VerifyHMACSignature(secret, req.Body, req.Header.Get(SandboxSignatureHeader))
}

// ParseWebhook reads the sandbox payload {"type":"...","externalId":"..."}.
func (s *Sandbox) ParseWebhook(req WebhookRequest) ([]InboundEvent, error) {
	var p struct {
		Type       string `json:"type"`
		ExternalID string `json:"externalId"`
	}
	// A malformed body still yields one raw event; the sandbox is permissive.
	_ = json.Unmarshal(req.Body, &p)
	return []InboundEvent{{Type: p.Type, ExternalID: p.ExternalID, Raw: json.RawMessage(req.Body)}}, nil
}

// Published returns a copy of the refs this sandbox has published, for tests.
func (s *Sandbox) Published() []ExternalRef {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ExternalRef, len(s.published))
	copy(out, s.published)
	return out
}
