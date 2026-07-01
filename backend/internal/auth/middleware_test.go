package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MattCheramie/echoboard/internal/auth"
	"github.com/MattCheramie/echoboard/internal/dbtest"
	"github.com/MattCheramie/echoboard/internal/user"
)

// authFixture builds an authenticator with one member user and a live session.
func authFixture(t *testing.T, role user.Role) (*auth.Authenticator, *auth.Session) {
	t.Helper()
	d := dbtest.New(t)
	users := user.NewRepository(d)
	sessions := auth.NewSessionStore(d, 0)
	u, err := users.Create(context.Background(), user.CreateInput{
		Email: "u@echo.test", Name: "U", Role: role, PasswordHash: "h",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	sess, err := sessions.Create(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return auth.NewAuthenticator(sessions, users, false), sess
}

func ok(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

func TestRequireAuth(t *testing.T) {
	a, sess := authFixture(t, user.RoleMember)
	handler := a.WithUser(a.RequireAuth(http.HandlerFunc(ok)))

	// No cookie -> 401.
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("no session: code = %d, want 401", rr.Code)
	}

	// Valid cookie -> 200.
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: sess.Token})
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("valid session: code = %d, want 200", rr.Code)
	}
}

func TestBearerToken(t *testing.T) {
	a, sess := authFixture(t, user.RoleMember)
	handler := a.WithUser(a.RequireAuth(http.HandlerFunc(ok)))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("bearer auth: code = %d, want 200", rr.Code)
	}
}

func TestRequireRole(t *testing.T) {
	a, sess := authFixture(t, user.RoleMember)
	handler := a.WithUser(a.RequireRole(user.RoleAdmin)(http.HandlerFunc(ok)))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: sess.Token})
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("member hitting admin route: code = %d, want 403", rr.Code)
	}
}
