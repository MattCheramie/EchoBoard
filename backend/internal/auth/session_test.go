package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MattCheramie/echoboard/internal/auth"
	"github.com/MattCheramie/echoboard/internal/dbtest"
)

func TestSessionLifecycle(t *testing.T) {
	d := dbtest.New(t)
	store := auth.NewSessionStore(d, 0)
	ctx := context.Background()

	sess, err := store.Create(ctx, "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := store.Get(ctx, sess.Token)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.UserID != "user-1" {
		t.Errorf("UserID = %q, want user-1", got.UserID)
	}

	if err := store.Delete(ctx, sess.Token); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(ctx, sess.Token); !errors.Is(err, auth.ErrNoSession) {
		t.Errorf("Get after delete err = %v, want ErrNoSession", err)
	}
}

func TestSessionExpiry(t *testing.T) {
	d := dbtest.New(t)
	store := auth.NewSessionStore(d, 0)
	ctx := context.Background()

	sess, err := store.Create(ctx, "user-2")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Force expiry in the past.
	if _, err := d.Exec(d.Rebind(`UPDATE sessions SET expires_at = ? WHERE token = ?`),
		"2000-01-01T00:00:00Z", sess.Token); err != nil {
		t.Fatalf("expire: %v", err)
	}
	if _, err := store.Get(ctx, sess.Token); !errors.Is(err, auth.ErrNoSession) {
		t.Errorf("expired Get err = %v, want ErrNoSession", err)
	}
}

func TestGetEmptyToken(t *testing.T) {
	store := auth.NewSessionStore(dbtest.New(t), 0)
	if _, err := store.Get(context.Background(), ""); !errors.Is(err, auth.ErrNoSession) {
		t.Errorf("empty token err = %v, want ErrNoSession", err)
	}
}
