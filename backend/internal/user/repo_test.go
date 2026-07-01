package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MattCheramie/echoboard/internal/dbtest"
	"github.com/MattCheramie/echoboard/internal/user"
)

func TestCreateAndFetch(t *testing.T) {
	repo := user.NewRepository(dbtest.New(t))
	ctx := context.Background()

	u, err := repo.Create(ctx, user.CreateInput{
		Email:        "Admin@Example.com",
		Name:         "Admin",
		Role:         user.RoleAdmin,
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.Email != "admin@example.com" {
		t.Errorf("email not normalized: %q", u.Email)
	}

	got, err := repo.GetByEmail(ctx, "admin@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.ID != u.ID || got.Role != user.RoleAdmin {
		t.Errorf("round-trip mismatch: %+v", got)
	}

	if _, err := repo.GetByID(ctx, "missing"); !errors.Is(err, user.ErrNotFound) {
		t.Errorf("GetByID(missing) err = %v, want ErrNotFound", err)
	}
}

func TestDuplicateEmailRejected(t *testing.T) {
	repo := user.NewRepository(dbtest.New(t))
	ctx := context.Background()
	in := user.CreateInput{Email: "a@b.com", Name: "A", Role: user.RoleMember, PasswordHash: "h"}
	if _, err := repo.Create(ctx, in); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if _, err := repo.Create(ctx, in); err == nil {
		t.Error("second Create with duplicate email should fail")
	}
}

func TestCountAndValidation(t *testing.T) {
	repo := user.NewRepository(dbtest.New(t))
	ctx := context.Background()

	if n, _ := repo.Count(ctx); n != 0 {
		t.Errorf("fresh Count = %d, want 0", n)
	}
	if _, err := repo.Create(ctx, user.CreateInput{Email: "bad", Name: "x", Role: user.RoleMember, PasswordHash: "h"}); err == nil {
		t.Error("invalid email should be rejected")
	}
	if _, err := repo.Create(ctx, user.CreateInput{Email: "ok@x.com", Name: "x", Role: "root", PasswordHash: "h"}); err == nil {
		t.Error("invalid role should be rejected")
	}
}
