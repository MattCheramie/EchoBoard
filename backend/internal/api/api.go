// Package api hosts the REST router, middleware chain, health/version
// endpoints, authentication routes, and the WebSocket hub for realtime updates.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/MattCheramie/echoboard/internal/account"
	"github.com/MattCheramie/echoboard/internal/auth"
	"github.com/MattCheramie/echoboard/internal/config"
	"github.com/MattCheramie/echoboard/internal/integrations"
	"github.com/MattCheramie/echoboard/internal/user"
)

// Version is the API/server version, overridable at build time.
var Version = "0.0.0-dev"

// API bundles the dependencies the HTTP handlers need.
type API struct {
	cfg          config.Config
	accounts     *account.Service
	users        *user.Repository
	sessions     *auth.SessionStore
	auth         *auth.Authenticator
	integrations *integrations.Service
	hub          *Hub
	log          *slog.Logger
}

// New constructs the API.
func New(cfg config.Config, accounts *account.Service, users *user.Repository,
	sessions *auth.SessionStore, authr *auth.Authenticator, integ *integrations.Service, log *slog.Logger) *API {
	if log == nil {
		log = slog.Default()
	}
	return &API{
		cfg:          cfg,
		accounts:     accounts,
		users:        users,
		sessions:     sessions,
		auth:         authr,
		integrations: integ,
		hub:          NewHub(),
		log:          log,
	}
}

// Handler builds the fully-wired HTTP handler: routes wrapped with user
// resolution, panic recovery, and request logging.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()

	// Public.
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("GET /api/version", a.handleVersion)
	mux.HandleFunc("POST /api/auth/login", a.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", a.handleLogout)
	mux.HandleFunc("POST /api/auth/redeem", a.handleRedeem)

	// Public webhook receiver — authenticated by per-provider signature, not a
	// session, so it must sit outside the auth-gated routes.
	mux.HandleFunc("POST /api/webhooks/{platform}", a.handleWebhook)

	// Authenticated.
	mux.Handle("GET /api/auth/me", a.auth.RequireAuth(http.HandlerFunc(a.handleMe)))
	mux.Handle("GET /ws", a.auth.RequireAuth(http.HandlerFunc(a.handleWS)))
	mux.Handle("GET /api/integrations", a.auth.RequireAuth(http.HandlerFunc(a.handleListIntegrations)))

	// Admin-only.
	admin := a.auth.RequireRole(user.RoleAdmin)
	mux.Handle("POST /api/invites", a.auth.RequireAuth(admin(http.HandlerFunc(a.handleCreateInvite))))
	mux.Handle("GET /api/users", a.auth.RequireAuth(admin(http.HandlerFunc(a.handleListUsers))))
	mux.Handle("POST /api/integrations/{platform}/connect", a.auth.RequireAuth(admin(http.HandlerFunc(a.handleConnectIntegration))))
	mux.Handle("GET /api/integrations/{platform}/callback", a.auth.RequireAuth(admin(http.HandlerFunc(a.handleIntegrationCallback))))
	mux.Handle("POST /api/integrations/{platform}/disconnect", a.auth.RequireAuth(admin(http.HandlerFunc(a.handleDisconnectIntegration))))

	return a.recover(a.logRequests(a.auth.WithUser(mux)))
}

// Hub exposes the realtime hub (e.g. for other subsystems to broadcast).
func (a *API) Hub() *Hub { return a.hub }

// --- middleware ---

func (a *API) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.log.Debug("request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (a *API) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				a.log.Error("panic", "err", rec, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// --- JSON helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// errorResponse is the standard error envelope: {"error": {"message": "..."}}.
type errorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	var e errorResponse
	e.Error.Message = msg
	writeJSON(w, status, e)
}

// decodeJSON reads a JSON body into dst, rejecting unknown fields.
func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
