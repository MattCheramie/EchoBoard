package api

import (
	"errors"
	"io"
	"net/http"

	"github.com/MattCheramie/echoboard/internal/integrations"
)

// maxWebhookBytes caps how much of a webhook body we read, to bound memory on
// unauthenticated requests.
const maxWebhookBytes = 1 << 20 // 1 MiB

// handleListIntegrations returns every registered platform with its capabilities
// and connection status (tokens are never included).
func (a *API) handleListIntegrations(w http.ResponseWriter, r *http.Request) {
	list, err := a.integrations.Available(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list integrations")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleConnectIntegration starts connecting a platform: it returns an OAuth
// authorization URL, or (for non-OAuth adapters) a completed connection.
func (a *API) handleConnectIntegration(w http.ResponseWriter, r *http.Request) {
	platform := integrations.Platform(r.PathValue("platform"))
	res, err := a.integrations.BeginConnect(r.Context(), platform)
	switch {
	case errors.Is(err, integrations.ErrUnknownPlatform):
		writeError(w, http.StatusNotFound, "unknown platform")
	case errors.Is(err, integrations.ErrNotConfigured):
		writeError(w, http.StatusBadRequest, "platform is not configured; set its operator credentials")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "could not start connection")
	default:
		writeJSON(w, http.StatusOK, res)
	}
}

// handleIntegrationCallback completes the OAuth flow after the provider redirects
// the admin's browser back with a code and state.
func (a *API) handleIntegrationCallback(w http.ResponseWriter, r *http.Request) {
	platform := integrations.Platform(r.PathValue("platform"))
	q := r.URL.Query()
	if provErr := q.Get("error"); provErr != "" {
		writeError(w, http.StatusBadRequest, "authorization was denied: "+provErr)
		return
	}
	conn, err := a.integrations.CompleteConnect(r.Context(), platform, q.Get("code"), q.Get("state"))
	switch {
	case errors.Is(err, integrations.ErrInvalidState):
		writeError(w, http.StatusBadRequest, "invalid or expired authorization state")
	case errors.Is(err, integrations.ErrUnknownPlatform):
		writeError(w, http.StatusNotFound, "unknown platform")
	case errors.Is(err, integrations.ErrNotConfigured):
		writeError(w, http.StatusBadRequest, "platform is not configured; set its operator credentials")
	case err != nil:
		writeError(w, http.StatusBadGateway, "token exchange with the provider failed")
	default:
		writeJSON(w, http.StatusOK, conn)
	}
}

// handleDisconnectIntegration removes a platform's connection(s).
func (a *API) handleDisconnectIntegration(w http.ResponseWriter, r *http.Request) {
	platform := integrations.Platform(r.PathValue("platform"))
	err := a.integrations.Disconnect(r.Context(), platform)
	switch {
	case errors.Is(err, integrations.ErrUnknownPlatform):
		writeError(w, http.StatusNotFound, "unknown platform")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "could not disconnect")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleWebhook receives an inbound provider webhook. It is public: the request
// is authenticated by its per-provider signature, verified inside the service.
func (a *API) handleWebhook(w http.ResponseWriter, r *http.Request) {
	platform := integrations.Platform(r.PathValue("platform"))
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read request body")
		return
	}
	req := integrations.WebhookRequest{Header: r.Header, Body: body}
	err = a.integrations.HandleWebhook(r.Context(), platform, req)
	switch {
	case errors.Is(err, integrations.ErrUnknownPlatform):
		writeError(w, http.StatusNotFound, "unknown platform")
	case errors.Is(err, integrations.ErrBadSignature), errors.Is(err, integrations.ErrNotConfigured):
		writeError(w, http.StatusUnauthorized, "invalid webhook signature")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "webhook processing failed")
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
	}
}
