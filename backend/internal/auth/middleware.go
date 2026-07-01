package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/MattCheramie/echoboard/internal/user"
)

// CookieName is the session cookie used by browser clients.
const CookieName = "echoboard_session"

type contextKey struct{ name string }

var userContextKey = contextKey{"currentUser"}

// Authenticator resolves the current user from a request's session token and
// provides HTTP middleware for optional and required authentication.
type Authenticator struct {
	sessions *SessionStore
	users    *user.Repository
	secure   bool // set Secure flag on cookies (production)
}

// NewAuthenticator wires the session store and user repository. secure controls
// the Secure cookie attribute (true in production/HTTPS).
func NewAuthenticator(sessions *SessionStore, users *user.Repository, secure bool) *Authenticator {
	return &Authenticator{sessions: sessions, users: users, secure: secure}
}

// tokenFrom extracts a session token from the cookie or an Authorization:
// Bearer header (the latter for native/API clients).
func tokenFrom(r *http.Request) string {
	if c, err := r.Cookie(CookieName); err == nil && c.Value != "" {
		return c.Value
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

// WithUser attaches the current user to the request context when a valid
// session is present. It never rejects; use RequireAuth to enforce.
func (a *Authenticator) WithUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u := a.resolve(r); u != nil {
			r = r.WithContext(context.WithValue(r.Context(), userContextKey, u))
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAuth rejects requests without a valid session with 401.
func (a *Authenticator) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := CurrentUser(r.Context()); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole rejects requests whose user does not hold the given role with 403.
// It should be chained after WithUser (and typically RequireAuth).
func (a *Authenticator) RequireRole(role user.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := CurrentUser(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if u.Role != role {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// resolve loads the user for the request's session token, or nil.
func (a *Authenticator) resolve(r *http.Request) *user.User {
	sess, err := a.sessions.Get(r.Context(), tokenFrom(r))
	if err != nil {
		return nil
	}
	u, err := a.users.GetByID(r.Context(), sess.UserID)
	if err != nil {
		return nil
	}
	return u
}

// SetSessionCookie writes the session cookie on w.
func (a *Authenticator) SetSessionCookie(w http.ResponseWriter, sess *Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt,
	})
}

// ClearSessionCookie expires the session cookie on w (logout).
func (a *Authenticator) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// CurrentUser returns the authenticated user attached to ctx, if any.
func CurrentUser(ctx context.Context) (*user.User, bool) {
	u, ok := ctx.Value(userContextKey).(*user.User)
	return u, ok
}
