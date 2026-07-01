package account_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MattCheramie/echoboard/internal/account"
	"github.com/MattCheramie/echoboard/internal/auth"
	"github.com/MattCheramie/echoboard/internal/dbtest"
	"github.com/MattCheramie/echoboard/internal/invite"
	"github.com/MattCheramie/echoboard/internal/user"
)

func newService(t *testing.T) (*account.Service, *user.Repository) {
	t.Helper()
	d := dbtest.New(t)
	users := user.NewRepository(d)
	invites := invite.NewRepository(d)
	return account.NewService(users, invites), users
}

func TestCreateFirstAdmin(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	admin, err := svc.CreateFirstAdmin(ctx, "admin@echo.test", "Admin", "supersecret")
	if err != nil {
		t.Fatalf("CreateFirstAdmin: %v", err)
	}
	if admin.Role != user.RoleAdmin {
		t.Errorf("role = %q, want admin", admin.Role)
	}
	if !auth.CheckPassword(admin.PasswordHash, "supersecret") {
		t.Error("stored hash does not verify against the password")
	}

	// Second attempt must be rejected — instance already bootstrapped.
	if _, err := svc.CreateFirstAdmin(ctx, "other@echo.test", "Other", "supersecret"); !errors.Is(err, account.ErrAlreadyBootstrapped) {
		t.Errorf("second bootstrap err = %v, want ErrAlreadyBootstrapped", err)
	}
}

func TestRedeemInvite(t *testing.T) {
	svc, users := newService(t)
	ctx := context.Background()

	admin, err := svc.CreateFirstAdmin(ctx, "admin@echo.test", "Admin", "supersecret")
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	inv, err := svc.CreateInvite(ctx, invite.CreateInput{Role: user.RoleMember, CreatedBy: admin.ID})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	member, err := svc.RedeemInvite(ctx, inv.Token, "member@echo.test", "Member", "anotherpass")
	if err != nil {
		t.Fatalf("RedeemInvite: %v", err)
	}
	if member.Role != user.RoleMember {
		t.Errorf("role = %q, want member", member.Role)
	}
	if n, _ := users.Count(ctx); n != 2 {
		t.Errorf("user count = %d, want 2", n)
	}

	// Reusing the same token must fail.
	if _, err := svc.RedeemInvite(ctx, inv.Token, "again@echo.test", "Again", "anotherpass"); !errors.Is(err, account.ErrInvalidInvite) {
		t.Errorf("reuse err = %v, want ErrInvalidInvite", err)
	}
}

func TestRedeemExpiredInvite(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	admin, _ := svc.CreateFirstAdmin(ctx, "admin@echo.test", "Admin", "supersecret")

	inv, err := svc.CreateInvite(ctx, invite.CreateInput{
		Role:      user.RoleMember,
		CreatedBy: admin.ID,
		ExpiresAt: time.Now().UTC().Add(-time.Hour), // already expired
	})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	if _, err := svc.RedeemInvite(ctx, inv.Token, "m@echo.test", "M", "anotherpass"); !errors.Is(err, account.ErrInvalidInvite) {
		t.Errorf("expired redeem err = %v, want ErrInvalidInvite", err)
	}
}
