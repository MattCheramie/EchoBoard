package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/MattCheramie/echoboard/internal/account"
	"github.com/MattCheramie/echoboard/internal/auth"
	"github.com/MattCheramie/echoboard/internal/invite"
	"github.com/MattCheramie/echoboard/internal/user"
)

// currentUserID returns the authenticated user's id from the request context.
func currentUserID(r *http.Request) (string, bool) {
	u, ok := auth.CurrentUser(r.Context())
	if !ok {
		return "", false
	}
	return u.ID, true
}

func (a *API) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version": Version,
		"env":     a.cfg.AppEnv,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	u, err := a.users.GetByEmail(r.Context(), req.Email)
	if err != nil || !auth.CheckPassword(u.PasswordHash, req.Password) {
		// Same response whether the user is missing or the password is wrong.
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	sess, err := a.sessions.Create(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	a.auth.SetSessionCookie(w, sess)
	writeJSON(w, http.StatusOK, u)
}

func (a *API) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName); err == nil {
		_ = a.sessions.Delete(r.Context(), c.Value)
	}
	a.auth.ClearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleMe(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.CurrentUser(r.Context())
	writeJSON(w, http.StatusOK, u)
}

type redeemRequest struct {
	Token    string `json:"token"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (a *API) handleRedeem(w http.ResponseWriter, r *http.Request) {
	var req redeemRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	u, err := a.accounts.RedeemInvite(r.Context(), req.Token, req.Email, req.Name, req.Password)
	if errors.Is(err, account.ErrInvalidInvite) {
		writeError(w, http.StatusBadRequest, "invite is invalid, expired, or already used")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Auto-login the new user.
	if sess, serr := a.sessions.Create(r.Context(), u.ID); serr == nil {
		a.auth.SetSessionCookie(w, sess)
	}
	writeJSON(w, http.StatusCreated, u)
}

type createInviteRequest struct {
	Email    string `json:"email"`
	Role     string `json:"role"`
	TTLHours int    `json:"ttlHours"`
}

func (a *API) handleCreateInvite(w http.ResponseWriter, r *http.Request) {
	current, _ := auth.CurrentUser(r.Context())
	var req createInviteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	role := user.Role(req.Role)
	if role == "" {
		role = user.RoleMember
	}
	in := invite.CreateInput{Email: req.Email, Role: role, CreatedBy: current.ID}
	if req.TTLHours > 0 {
		in.ExpiresAt = time.Now().UTC().Add(time.Duration(req.TTLHours) * time.Hour)
	}
	inv, err := a.accounts.CreateInvite(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, inv)
}

func (a *API) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.users.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list users")
		return
	}
	writeJSON(w, http.StatusOK, users)
}
