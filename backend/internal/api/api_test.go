package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MattCheramie/echoboard/internal/account"
	"github.com/MattCheramie/echoboard/internal/api"
	"github.com/MattCheramie/echoboard/internal/auth"
	"github.com/MattCheramie/echoboard/internal/config"
	"github.com/MattCheramie/echoboard/internal/dbtest"
	"github.com/MattCheramie/echoboard/internal/integrations"
	"github.com/MattCheramie/echoboard/internal/invite"
	"github.com/MattCheramie/echoboard/internal/user"
)

// testSandboxWebhookSecret is the sandbox webhook secret injected into the test
// server, so webhook tests can sign requests the same way a provider would.
const testSandboxWebhookSecret = "whsec_test"

// newServer wires a full API against a temp DB and bootstraps one admin.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	d := dbtest.New(t)
	users := user.NewRepository(d)
	invites := invite.NewRepository(d)
	accounts := account.NewService(users, invites)
	sessions := auth.NewSessionStore(d, 0)
	authr := auth.NewAuthenticator(sessions, users, false)

	if _, err := accounts.CreateFirstAdmin(context.Background(), "admin@echo.test", "Admin", "supersecret"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	vault, err := auth.NewVault("")
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	registry := integrations.NewRegistry()
	registry.Register(integrations.NewSandbox())
	integSvc := integrations.NewService(integrations.Options{
		Registry: registry,
		Repo:     integrations.NewRepository(d, vault),
		Vault:    vault,
		BaseURL:  "http://localhost:8080",
		Credentials: func(integrations.Platform) integrations.ProviderCredentials {
			return integrations.ProviderCredentials{WebhookSecret: testSandboxWebhookSecret}
		},
	})

	cfg := config.Default()
	srv := httptest.NewServer(api.New(cfg, accounts, users, sessions, authr, integSvc, nil).Handler())
	t.Cleanup(srv.Close)
	return srv
}

func post(t *testing.T, client *http.Client, url, body string) *http.Response {
	t.Helper()
	resp, err := client.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func TestHealthAndVersion(t *testing.T) {
	srv := newServer(t)
	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", resp.StatusCode)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("health body = %v", body)
	}
}

func TestLoginFlow(t *testing.T) {
	srv := newServer(t)
	jar, _ := newJarClient()

	// Wrong password -> 401.
	resp := post(t, jar, srv.URL+"/api/auth/login", `{"email":"admin@echo.test","password":"wrong"}`)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad login status = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()

	// /api/auth/me before login -> 401.
	meResp, _ := jar.Get(srv.URL + "/api/auth/me")
	if meResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("me before login = %d, want 401", meResp.StatusCode)
	}
	meResp.Body.Close()

	// Correct login -> 200 and a session cookie in the jar.
	resp = post(t, jar, srv.URL+"/api/auth/login", `{"email":"admin@echo.test","password":"supersecret"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("good login status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// /api/auth/me now succeeds.
	meResp, _ = jar.Get(srv.URL + "/api/auth/me")
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("me after login = %d, want 200", meResp.StatusCode)
	}
	var me map[string]any
	json.NewDecoder(meResp.Body).Decode(&me)
	meResp.Body.Close()
	if me["email"] != "admin@echo.test" || me["role"] != "admin" {
		t.Errorf("me body = %v", me)
	}
}

func TestInviteAndRedeemFlow(t *testing.T) {
	srv := newServer(t)
	admin, _ := newJarClient()
	post(t, admin, srv.URL+"/api/auth/login", `{"email":"admin@echo.test","password":"supersecret"}`).Body.Close()

	// Admin mints an invite.
	resp := post(t, admin, srv.URL+"/api/invites", `{"role":"member"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create invite status = %d, want 201", resp.StatusCode)
	}
	var inv struct {
		Token string `json:"token"`
	}
	json.NewDecoder(resp.Body).Decode(&inv)
	resp.Body.Close()
	if inv.Token == "" {
		t.Fatal("invite token empty")
	}

	// A fresh client redeems it.
	guest, _ := newJarClient()
	body, _ := json.Marshal(map[string]string{
		"token": inv.Token, "email": "member@echo.test", "name": "Member", "password": "memberpass",
	})
	resp = post(t, guest, srv.URL+"/api/auth/redeem", string(body))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("redeem status = %d, want 201", resp.StatusCode)
	}
	resp.Body.Close()

	// The redeemed session is logged in as a member -> admin route forbidden.
	usersResp, _ := guest.Get(srv.URL + "/api/users")
	if usersResp.StatusCode != http.StatusForbidden {
		t.Fatalf("member hitting /api/users = %d, want 403", usersResp.StatusCode)
	}
	usersResp.Body.Close()
}

func TestListUsersRequiresAdmin(t *testing.T) {
	srv := newServer(t)
	// Unauthenticated -> 401.
	resp, _ := http.Get(srv.URL + "/api/users")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth /api/users = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()

	// Admin -> 200 with the bootstrap admin present.
	admin, _ := newJarClient()
	post(t, admin, srv.URL+"/api/auth/login", `{"email":"admin@echo.test","password":"supersecret"}`).Body.Close()
	resp, _ = admin.Get(srv.URL + "/api/users")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin /api/users = %d, want 200", resp.StatusCode)
	}
	var users []map[string]any
	json.NewDecoder(resp.Body).Decode(&users)
	resp.Body.Close()
	if len(users) != 1 {
		t.Errorf("user count = %d, want 1", len(users))
	}
}

// helpers -------------------------------------------------------------------

func newJarClient() (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	return &http.Client{Jar: jar}, err
}
