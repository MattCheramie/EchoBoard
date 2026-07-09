package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// oauthResponse is the standard OAuth2 token endpoint payload.
type oauthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// authorizeURL builds the provider authorization URL the user is redirected to.
func authorizeURL(ep Endpoints, clientID, redirectURI, state string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	if len(ep.Scopes) > 0 {
		q.Set("scope", strings.Join(ep.Scopes, " "))
	}
	sep := "?"
	if strings.Contains(ep.AuthURL, "?") {
		sep = "&"
	}
	return ep.AuthURL + sep + q.Encode()
}

// exchangeCode swaps an authorization code for a token at the provider.
func exchangeCode(ctx context.Context, hc *http.Client, ep Endpoints, clientID, clientSecret, redirectURI, code string) (Token, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	return postToken(ctx, hc, ep.TokenURL, form)
}

// refreshAccessToken exchanges a refresh token for a fresh access token.
func refreshAccessToken(ctx context.Context, hc *http.Client, ep Endpoints, clientID, clientSecret, refreshToken string) (Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	return postToken(ctx, hc, ep.TokenURL, form)
}

func postToken(ctx context.Context, hc *http.Client, tokenURL string, form url.Values) (Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := hc.Do(req)
	if err != nil {
		return Token{}, fmt.Errorf("integrations: token request: %w", err)
	}
	defer resp.Body.Close()

	var out oauthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Token{}, fmt.Errorf("integrations: decode token response: %w", err)
	}
	if out.Error != "" {
		return Token{}, fmt.Errorf("integrations: token endpoint error: %s %s", out.Error, out.ErrorDesc)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || out.AccessToken == "" {
		return Token{}, fmt.Errorf("integrations: token endpoint returned status %d", resp.StatusCode)
	}

	tok := Token{AccessToken: out.AccessToken, RefreshToken: out.RefreshToken}
	if out.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().UTC().Add(time.Duration(out.ExpiresIn) * time.Second)
	}
	return tok, nil
}
