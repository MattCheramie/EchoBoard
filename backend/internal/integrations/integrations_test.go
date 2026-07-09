package integrations_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/echoboard/internal/auth"
	"github.com/MattCheramie/echoboard/internal/db"
	"github.com/MattCheramie/echoboard/internal/dbtest"
	"github.com/MattCheramie/echoboard/internal/integrations"
)

func testVault(t *testing.T) *auth.Vault {
	t.Helper()
	v, err := auth.NewVault("")
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	return v
}

func TestRegistry(t *testing.T) {
	reg := integrations.NewRegistry()
	sb := integrations.NewSandbox()
	reg.Register(sb)

	got, ok := reg.Get(integrations.PlatformSandbox)
	if !ok || got != sb {
		t.Fatalf("Get(sandbox) = %v, %v; want the registered adapter", got, ok)
	}
	if _, ok := reg.Get(integrations.PlatformMeta); ok {
		t.Errorf("Get(meta) ok = true; want false (not registered)")
	}
	ps := reg.Platforms()
	if len(ps) != 1 || ps[0] != integrations.PlatformSandbox {
		t.Errorf("Platforms() = %v; want [sandbox]", ps)
	}
}

func TestWebhookHMACVerify(t *testing.T) {
	const secret = "whsec"
	body := []byte(`{"type":"comment"}`)
	sig := integrations.SignHMAC(secret, body)

	if err := integrations.VerifyHMACSignature(secret, body, sig); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	// Tampered body must be rejected.
	if err := integrations.VerifyHMACSignature(secret, []byte(`{"type":"spam"}`), sig); err == nil {
		t.Error("tampered body accepted; want rejection")
	}
	// Wrong secret must be rejected.
	if err := integrations.VerifyHMACSignature("other", body, sig); err == nil {
		t.Error("wrong secret accepted; want rejection")
	}
	// Empty secret means "not configured".
	if err := integrations.VerifyHMACSignature("", body, sig); err == nil {
		t.Error("empty secret accepted; want ErrNotConfigured")
	}
}

func TestConnectionVaultRoundTrip(t *testing.T) {
	d := dbtest.New(t)
	vault := testVault(t)
	repo := integrations.NewRepository(d, vault)
	ctx := context.Background()

	const accessTok = "super-secret-access-token"
	tok := integrations.Token{
		AccessToken:  accessTok,
		RefreshToken: "refresh-xyz",
		ExpiresAt:    time.Now().UTC().Add(time.Hour).Truncate(time.Second),
	}
	conn, err := repo.Upsert(ctx, integrations.PlatformSandbox, "", integrations.StatusConnected, []string{"read", "write"}, tok)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Token round-trips through the vault.
	got, err := repo.Token(ctx, conn.ID)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if got.AccessToken != accessTok || got.RefreshToken != "refresh-xyz" {
		t.Errorf("token round-trip = %+v; want access/refresh preserved", got)
	}

	// The token must be encrypted at rest: the raw column must not equal the plaintext.
	var rawEnc string
	q := d.Rebind(`SELECT access_token_enc FROM integration_connections WHERE id = ?`)
	if err := d.QueryRowContext(ctx, q, conn.ID).Scan(&rawEnc); err != nil {
		t.Fatalf("read raw column: %v", err)
	}
	if rawEnc == "" || strings.Contains(rawEnc, accessTok) {
		t.Errorf("access token stored in plaintext (raw=%q)", rawEnc)
	}

	// The token must never appear in the API JSON representation.
	blob, _ := json.Marshal(conn)
	if strings.Contains(string(blob), accessTok) || strings.Contains(string(blob), "refresh-xyz") {
		t.Errorf("connection JSON leaks token material: %s", blob)
	}

	// A second Upsert updates in place (unique platform+account) rather than duplicating.
	if _, err := repo.Upsert(ctx, integrations.PlatformSandbox, "", integrations.StatusConnected, nil, tok); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("connection count = %d; want 1 (upsert should update in place)", len(list))
	}
}

func TestSandboxConnectAndPublish(t *testing.T) {
	d := dbtest.New(t)
	sb := integrations.NewSandbox()
	reg := integrations.NewRegistry()
	reg.Register(sb)
	svc := integrations.NewService(integrations.Options{
		Registry: reg, Repo: integrations.NewRepository(d, testVault(t)), Vault: testVault(t), BaseURL: "http://localhost",
	})
	ctx := context.Background()

	// The sandbox is non-OAuth: connecting stores a connection immediately.
	res, err := svc.BeginConnect(ctx, integrations.PlatformSandbox)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if !res.Connected || res.AuthURL != "" {
		t.Fatalf("sandbox connect = %+v; want Connected without an AuthURL", res)
	}

	ref, err := svc.Publish(ctx, integrations.PlatformSandbox, integrations.Content{ID: "c1", Body: "hello"})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if ref.Platform != integrations.PlatformSandbox || ref.ExternalID == "" {
		t.Errorf("publish ref = %+v; want sandbox ref with an external id", ref)
	}
	if pub := sb.Published(); len(pub) != 1 {
		t.Errorf("sandbox published count = %d; want 1", len(pub))
	}
}

func TestPublishNotConnected(t *testing.T) {
	d := dbtest.New(t)
	reg := integrations.NewRegistry()
	reg.Register(integrations.NewSandbox())
	svc := integrations.NewService(integrations.Options{
		Registry: reg, Repo: integrations.NewRepository(d, testVault(t)), Vault: testVault(t),
	})
	_, err := svc.Publish(context.Background(), integrations.PlatformSandbox, integrations.Content{ID: "c1"})
	if err != integrations.ErrNotConnected {
		t.Errorf("publish without connect = %v; want ErrNotConnected", err)
	}
}

func TestUnknownPlatform(t *testing.T) {
	d := dbtest.New(t)
	reg := integrations.NewRegistry() // nothing registered
	svc := integrations.NewService(integrations.Options{
		Registry: reg, Repo: integrations.NewRepository(d, testVault(t)), Vault: testVault(t),
	})
	if _, err := svc.BeginConnect(context.Background(), integrations.PlatformMeta); err != integrations.ErrUnknownPlatform {
		t.Errorf("connect unknown = %v; want ErrUnknownPlatform", err)
	}
}

// fakeAdapter is an OAuth-using adapter whose token endpoint is a test server.
type fakeAdapter struct {
	platform integrations.Platform
	ep       integrations.Endpoints
	mu       sync.Mutex
	lastTok  string
}

func (f *fakeAdapter) Platform() integrations.Platform { return f.platform }
func (f *fakeAdapter) Capabilities() integrations.Capabilities {
	return integrations.Capabilities{Publish: true, OAuth: true, Webhooks: true}
}
func (f *fakeAdapter) OAuthEndpoints() (integrations.Endpoints, bool) { return f.ep, true }
func (f *fakeAdapter) Publish(_ context.Context, accessToken string, c integrations.Content) (integrations.ExternalRef, error) {
	f.mu.Lock()
	f.lastTok = accessToken
	f.mu.Unlock()
	return integrations.ExternalRef{Platform: f.platform, ExternalID: "ext_" + c.ID}, nil
}
func (f *fakeAdapter) VerifyWebhook(secret string, req integrations.WebhookRequest) error {
	return integrations.VerifyHMACSignature(secret, req.Body, req.Header.Get("X-Sig"))
}
func (f *fakeAdapter) ParseWebhook(integrations.WebhookRequest) ([]integrations.InboundEvent, error) {
	return []integrations.InboundEvent{{Type: "test"}}, nil
}

func (f *fakeAdapter) lastToken() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastTok
}

// tokenServer records the last received form and returns a canned token.
func tokenServer(t *testing.T, access, refresh string, gotForm *url.Values) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if gotForm != nil {
			*gotForm = r.Form
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": access, "refresh_token": refresh, "expires_in": 3600, "token_type": "bearer",
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newFakeService(t *testing.T, d *db.DB, fa *fakeAdapter) *integrations.Service {
	t.Helper()
	reg := integrations.NewRegistry()
	reg.Register(fa)
	return integrations.NewService(integrations.Options{
		Registry: reg,
		Repo:     integrations.NewRepository(d, testVault(t)),
		Vault:    testVault(t),
		BaseURL:  "http://localhost",
		Credentials: func(integrations.Platform) integrations.ProviderCredentials {
			return integrations.ProviderCredentials{ClientID: "id", ClientSecret: "secret"}
		},
	})
}

func TestOAuthConnectFlow(t *testing.T) {
	d := dbtest.New(t)
	var form url.Values
	ts := tokenServer(t, "access-1", "refresh-1", &form)
	fa := &fakeAdapter{platform: integrations.PlatformMeta, ep: integrations.Endpoints{
		AuthURL: "https://provider.example/auth", TokenURL: ts.URL, Scopes: []string{"pages"},
	}}
	svc := newFakeService(t, d, fa)
	ctx := context.Background()

	// BeginConnect returns an authorize URL carrying a signed state.
	res, err := svc.BeginConnect(ctx, integrations.PlatformMeta)
	if err != nil {
		t.Fatalf("begin connect: %v", err)
	}
	if res.AuthURL == "" {
		t.Fatal("expected an auth URL for an OAuth adapter")
	}
	u, _ := url.Parse(res.AuthURL)
	state := u.Query().Get("state")
	if state == "" {
		t.Fatal("auth URL missing state")
	}

	// Completing with a bad state is rejected.
	if _, err := svc.CompleteConnect(ctx, integrations.PlatformMeta, "code", "tampered"); err != integrations.ErrInvalidState {
		t.Errorf("bad state = %v; want ErrInvalidState", err)
	}

	// Completing with the real state exchanges the code and stores the connection.
	conn, err := svc.CompleteConnect(ctx, integrations.PlatformMeta, "authcode", state)
	if err != nil {
		t.Fatalf("complete connect: %v", err)
	}
	if conn.Status != integrations.StatusConnected {
		t.Errorf("status = %q; want connected", conn.Status)
	}
	if form.Get("grant_type") != "authorization_code" || form.Get("code") != "authcode" {
		t.Errorf("token exchange form = %v; want authorization_code/authcode", form)
	}

	// Publishing uses the freshly stored access token.
	if _, err := svc.Publish(ctx, integrations.PlatformMeta, integrations.Content{ID: "x"}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if fa.lastToken() != "access-1" {
		t.Errorf("publish used token %q; want access-1", fa.lastToken())
	}
}

func TestOAuthRefreshOnRead(t *testing.T) {
	d := dbtest.New(t)
	var form url.Values
	ts := tokenServer(t, "access-2", "", &form) // provider omits refresh on refresh
	fa := &fakeAdapter{platform: integrations.PlatformMeta, ep: integrations.Endpoints{
		AuthURL: "https://provider.example/auth", TokenURL: ts.URL,
	}}
	vault := testVault(t)
	repo := integrations.NewRepository(d, vault)
	reg := integrations.NewRegistry()
	reg.Register(fa)
	svc := integrations.NewService(integrations.Options{
		Registry: reg, Repo: repo, Vault: vault, BaseURL: "http://localhost",
		Credentials: func(integrations.Platform) integrations.ProviderCredentials {
			return integrations.ProviderCredentials{ClientID: "id", ClientSecret: "secret"}
		},
	})
	ctx := context.Background()

	// Store an already-expired token with a refresh token.
	_, err := repo.Upsert(ctx, integrations.PlatformMeta, "", integrations.StatusConnected, nil, integrations.Token{
		AccessToken: "stale", RefreshToken: "r-1", ExpiresAt: time.Now().UTC().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Publishing must refresh first, then use the new token.
	if _, err := svc.Publish(ctx, integrations.PlatformMeta, integrations.Content{ID: "y"}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if form.Get("grant_type") != "refresh_token" || form.Get("refresh_token") != "r-1" {
		t.Errorf("refresh form = %v; want refresh_token/r-1", form)
	}
	if fa.lastToken() != "access-2" {
		t.Errorf("publish used %q; want refreshed access-2", fa.lastToken())
	}
	// The refreshed token is persisted, and the old refresh token is retained.
	stored, _ := repo.Token(ctx, mustConnID(t, svc, ctx))
	if stored.AccessToken != "access-2" || stored.RefreshToken != "r-1" {
		t.Errorf("stored token = %+v; want access-2 with retained refresh r-1", stored)
	}
}

func mustConnID(t *testing.T, svc *integrations.Service, ctx context.Context) string {
	t.Helper()
	conns, err := svc.Connections(ctx)
	if err != nil || len(conns) == 0 {
		t.Fatalf("connections: %v (n=%d)", err, len(conns))
	}
	return conns[0].ID
}
