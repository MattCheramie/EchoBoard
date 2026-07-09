package integrations

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/echoboard/internal/auth"
)

const (
	stateTTL    = 10 * time.Minute
	refreshSkew = 60 * time.Second
)

// ProviderCredentials are the operator-supplied secrets that make a platform
// live. They cannot ship in the repo; the deploy docs (PR 6.3) document the
// per-platform app registration and env vars that populate them.
type ProviderCredentials struct {
	ClientID      string
	ClientSecret  string
	WebhookSecret string
	RedirectURI   string // optional override of the default callback URL
}

// Options configures a Service.
type Options struct {
	Registry   *Registry
	Repo       *Repository
	Vault      *auth.Vault
	HTTPClient *http.Client
	// BaseURL is the public origin used to build default OAuth redirect URIs
	// (e.g. https://echoboard.example.com). Usually config.PublicAPIBaseURL.
	BaseURL string
	// Credentials resolves operator credentials for a platform. Defaults to
	// reading INTEGRATION_<PLATFORM>_* environment variables.
	Credentials func(Platform) ProviderCredentials
}

// Service is the application-facing integration façade: it lists platforms,
// drives the OAuth connect/refresh flow (tokens encrypted via the vault),
// disconnects, verifies + records webhooks, and publishes content through the
// right adapter.
type Service struct {
	reg     *Registry
	repo    *Repository
	vault   *auth.Vault
	hc      *http.Client
	baseURL string
	credsFn func(Platform) ProviderCredentials
}

// NewService builds a Service from Options.
func NewService(opts Options) *Service {
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	credsFn := opts.Credentials
	if credsFn == nil {
		credsFn = envCredentials
	}
	return &Service{
		reg:     opts.Registry,
		repo:    opts.Repo,
		vault:   opts.Vault,
		hc:      hc,
		baseURL: strings.TrimRight(opts.BaseURL, "/"),
		credsFn: credsFn,
	}
}

// PlatformStatus describes one platform for the API's list view.
type PlatformStatus struct {
	Platform     Platform     `json:"platform"`
	Capabilities Capabilities `json:"capabilities"`
	Configured   bool         `json:"configured"`
	Connected    bool         `json:"connected"`
	Connection   *Connection  `json:"connection,omitempty"`
}

// ConnectResult is returned by BeginConnect: either an OAuth URL to redirect the
// user to, or (for non-OAuth adapters like the sandbox) an immediate connection.
type ConnectResult struct {
	AuthURL    string      `json:"authUrl,omitempty"`
	Connected  bool        `json:"connected,omitempty"`
	Connection *Connection `json:"connection,omitempty"`
}

// Available lists every registered platform with its capabilities, whether it
// is configured (operator credentials present, or non-OAuth), and whether it is
// currently connected.
func (s *Service) Available(ctx context.Context) ([]PlatformStatus, error) {
	var out []PlatformStatus
	for _, p := range s.reg.Platforms() {
		adapter, _ := s.reg.Get(p)
		conn, err := s.repo.GetByPlatform(ctx, p)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		_, usesOAuth := adapter.OAuthEndpoints()
		creds := s.credsFn(p)
		configured := !usesOAuth || (creds.ClientID != "" && creds.ClientSecret != "")
		st := PlatformStatus{
			Platform:     p,
			Capabilities: adapter.Capabilities(),
			Configured:   configured,
			Connected:    conn != nil,
			Connection:   conn,
		}
		out = append(out, st)
	}
	return out, nil
}

// Connections returns all stored connections (redacted).
func (s *Service) Connections(ctx context.Context) ([]*Connection, error) {
	return s.repo.List(ctx)
}

// BeginConnect starts connecting a platform. For OAuth adapters it returns the
// provider authorization URL; for non-OAuth adapters it stores and returns a
// connection immediately.
func (s *Service) BeginConnect(ctx context.Context, platform Platform) (ConnectResult, error) {
	adapter, ok := s.reg.Get(platform)
	if !ok {
		return ConnectResult{}, ErrUnknownPlatform
	}
	ep, usesOAuth := adapter.OAuthEndpoints()
	if !usesOAuth {
		conn, err := s.repo.Upsert(ctx, platform, "", StatusConnected, nil, Token{AccessToken: "connected"})
		if err != nil {
			return ConnectResult{}, err
		}
		return ConnectResult{Connected: true, Connection: conn}, nil
	}
	creds := s.credsFn(platform)
	if creds.ClientID == "" || creds.ClientSecret == "" {
		return ConnectResult{}, ErrNotConfigured
	}
	state, err := s.signState(platform)
	if err != nil {
		return ConnectResult{}, err
	}
	return ConnectResult{AuthURL: authorizeURL(ep, creds.ClientID, s.redirectURI(platform, creds), state)}, nil
}

// CompleteConnect finishes the OAuth flow: it validates the state, exchanges the
// code for a token, and stores the (encrypted) connection.
func (s *Service) CompleteConnect(ctx context.Context, platform Platform, code, state string) (*Connection, error) {
	if err := s.validateState(platform, state); err != nil {
		return nil, err
	}
	adapter, ok := s.reg.Get(platform)
	if !ok {
		return nil, ErrUnknownPlatform
	}
	ep, ok := adapter.OAuthEndpoints()
	if !ok {
		return nil, ErrNoOAuth
	}
	creds := s.credsFn(platform)
	if creds.ClientID == "" || creds.ClientSecret == "" {
		return nil, ErrNotConfigured
	}
	tok, err := exchangeCode(ctx, s.hc, ep, creds.ClientID, creds.ClientSecret, s.redirectURI(platform, creds), code)
	if err != nil {
		return nil, err
	}
	return s.repo.Upsert(ctx, platform, "", StatusConnected, ep.Scopes, tok)
}

// Disconnect removes all connections for a platform.
func (s *Service) Disconnect(ctx context.Context, platform Platform) error {
	if _, ok := s.reg.Get(platform); !ok {
		return ErrUnknownPlatform
	}
	_, err := s.repo.DeleteByPlatform(ctx, platform)
	return err
}

// HandleWebhook verifies an inbound webhook's signature and, if valid, records
// it. Routing events into the interactions inbox lands in Tier 3 / PR 4.2.
func (s *Service) HandleWebhook(ctx context.Context, platform Platform, req WebhookRequest) error {
	adapter, ok := s.reg.Get(platform)
	if !ok {
		return ErrUnknownPlatform
	}
	creds := s.credsFn(platform)
	if err := adapter.VerifyWebhook(creds.WebhookSecret, req); err != nil {
		// Do not persist unverified payloads (spoofing / storage-DoS surface).
		return err
	}
	events, err := adapter.ParseWebhook(req)
	if err != nil {
		return err
	}
	eventType := ""
	if len(events) > 0 {
		eventType = events[0].Type
	}
	return s.repo.RecordWebhook(ctx, platform, eventType, true, req.Body)
}

// Publish dispatches content to a connected platform, refreshing the access
// token first if it is near expiry. The Tier 2 scheduler calls this.
func (s *Service) Publish(ctx context.Context, platform Platform, c Content) (ExternalRef, error) {
	adapter, ok := s.reg.Get(platform)
	if !ok {
		return ExternalRef{}, ErrUnknownPlatform
	}
	conn, err := s.repo.GetByPlatform(ctx, platform)
	if errors.Is(err, ErrNotFound) {
		return ExternalRef{}, ErrNotConnected
	}
	if err != nil {
		return ExternalRef{}, err
	}
	access, err := s.accessToken(ctx, adapter, conn)
	if err != nil {
		return ExternalRef{}, err
	}
	return adapter.Publish(ctx, access, c)
}

// accessToken returns a usable access token for a connection, refreshing it
// (refresh-on-read, with a skew margin) when it is near expiry.
func (s *Service) accessToken(ctx context.Context, adapter Adapter, conn *Connection) (string, error) {
	tok, err := s.repo.Token(ctx, conn.ID)
	if err != nil {
		return "", err
	}
	if tok.Expired(time.Now().UTC(), refreshSkew) && tok.RefreshToken != "" {
		ep, ok := adapter.OAuthEndpoints()
		if ok {
			creds := s.credsFn(conn.Platform)
			newTok, rerr := refreshAccessToken(ctx, s.hc, ep, creds.ClientID, creds.ClientSecret, tok.RefreshToken)
			if rerr != nil {
				return "", fmt.Errorf("integrations: refresh %s: %w", conn.Platform, rerr)
			}
			if newTok.RefreshToken == "" {
				newTok.RefreshToken = tok.RefreshToken // some providers omit it on refresh
			}
			if err := s.repo.UpdateToken(ctx, conn.ID, newTok); err != nil {
				return "", err
			}
			tok = newTok
		}
	}
	return tok.AccessToken, nil
}

// redirectURI is the OAuth callback URL for a platform.
func (s *Service) redirectURI(platform Platform, creds ProviderCredentials) string {
	if creds.RedirectURI != "" {
		return creds.RedirectURI
	}
	return fmt.Sprintf("%s/api/integrations/%s/callback", s.baseURL, platform)
}

// signState produces an opaque, tamper-proof OAuth state bound to the platform
// and an expiry, sealed with the vault (no server-side state storage needed).
func (s *Service) signState(platform Platform) (string, error) {
	payload := fmt.Sprintf("%s|%d", platform, time.Now().UTC().Add(stateTTL).Unix())
	return s.vault.Encrypt([]byte(payload))
}

func (s *Service) validateState(platform Platform, state string) error {
	raw, err := s.vault.Decrypt(state)
	if err != nil {
		return ErrInvalidState
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 || parts[0] != string(platform) {
		return ErrInvalidState
	}
	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().UTC().Unix() > expiry {
		return ErrInvalidState
	}
	return nil
}

// envCredentials reads INTEGRATION_<PLATFORM>_{CLIENT_ID,CLIENT_SECRET,
// WEBHOOK_SECRET,REDIRECT_URI} from the environment.
func envCredentials(p Platform) ProviderCredentials {
	up := strings.ToUpper(string(p))
	return ProviderCredentials{
		ClientID:      os.Getenv("INTEGRATION_" + up + "_CLIENT_ID"),
		ClientSecret:  os.Getenv("INTEGRATION_" + up + "_CLIENT_SECRET"),
		WebhookSecret: os.Getenv("INTEGRATION_" + up + "_WEBHOOK_SECRET"),
		RedirectURI:   os.Getenv("INTEGRATION_" + up + "_REDIRECT_URI"),
	}
}
