package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/MattCheramie/echoboard/internal/integrations"
)

func TestIntegrationsListAndConnectSandbox(t *testing.T) {
	srv := newServer(t)
	admin, _ := newJarClient()
	post(t, admin, srv.URL+"/api/auth/login", `{"email":"admin@echo.test","password":"supersecret"}`).Body.Close()

	// Initially the sandbox is registered but not connected.
	resp, err := admin.Get(srv.URL + "/api/integrations")
	if err != nil {
		t.Fatal(err)
	}
	var list []integrations.PlatformStatus
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 1 || list[0].Platform != integrations.PlatformSandbox || list[0].Connected {
		t.Fatalf("integrations list = %+v; want [sandbox, not connected]", list)
	}

	// Connect the sandbox (non-OAuth: immediate).
	resp = post(t, admin, srv.URL+"/api/integrations/sandbox/connect", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("connect sandbox status = %d, want 200", resp.StatusCode)
	}
	var res integrations.ConnectResult
	json.NewDecoder(resp.Body).Decode(&res)
	resp.Body.Close()
	if !res.Connected {
		t.Fatalf("connect result = %+v; want Connected", res)
	}

	// Now the list reflects the connection.
	resp, _ = admin.Get(srv.URL + "/api/integrations")
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if !list[0].Connected {
		t.Errorf("sandbox not marked connected after connect")
	}

	// Disconnect returns 204.
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/integrations/sandbox/disconnect", nil)
	dresp, _ := admin.Do(req)
	if dresp.StatusCode != http.StatusNoContent {
		t.Errorf("disconnect status = %d, want 204", dresp.StatusCode)
	}
	dresp.Body.Close()
}

func TestWebhookSignatureVerification(t *testing.T) {
	srv := newServer(t)
	body := `{"type":"comment","externalId":"c-1"}`
	sig := integrations.SignHMAC(testSandboxWebhookSecret, []byte(body))

	// Valid signature -> 200.
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/webhooks/sandbox", strings.NewReader(body))
	req.Header.Set(integrations.SandboxSignatureHeader, sig)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("valid webhook status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// Tampered body (signature no longer matches) -> 401.
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/webhooks/sandbox", strings.NewReader(`{"type":"spam"}`))
	req.Header.Set(integrations.SandboxSignatureHeader, sig)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("tampered webhook status = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()

	// Unknown platform -> 404.
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/webhooks/nope", strings.NewReader(body))
	req.Header.Set(integrations.SandboxSignatureHeader, sig)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown-platform webhook status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestIntegrationsRequireAuthAndAdmin(t *testing.T) {
	srv := newServer(t)

	// Unauthenticated list -> 401.
	resp, _ := http.Get(srv.URL + "/api/integrations")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth list = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()

	// A member (non-admin) cannot connect integrations -> 403.
	admin, _ := newJarClient()
	post(t, admin, srv.URL+"/api/auth/login", `{"email":"admin@echo.test","password":"supersecret"}`).Body.Close()
	inviteResp := post(t, admin, srv.URL+"/api/invites", `{"role":"member"}`)
	var inv struct {
		Token string `json:"token"`
	}
	json.NewDecoder(inviteResp.Body).Decode(&inv)
	inviteResp.Body.Close()

	member, _ := newJarClient()
	body, _ := json.Marshal(map[string]string{"token": inv.Token, "email": "m@echo.test", "name": "M", "password": "memberpass"})
	post(t, member, srv.URL+"/api/auth/redeem", string(body)).Body.Close()

	resp = post(t, member, srv.URL+"/api/integrations/sandbox/connect", "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("member connect = %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()
}
