// Package account orchestrates the user, invite, and auth packages to implement
// the two ways an EchoBoard account comes into existence: the first-run admin
// bootstrap and invite-only redemption.
package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MattCheramie/echoboard/internal/auth"
	"github.com/MattCheramie/echoboard/internal/invite"
	"github.com/MattCheramie/echoboard/internal/user"
)

// Service ties together user and invite persistence with password hashing.
type Service struct {
	users   *user.Repository
	invites *invite.Repository
}

// NewService constructs an account service.
func NewService(users *user.Repository, invites *invite.Repository) *Service {
	return &Service{users: users, invites: invites}
}

// ErrAlreadyBootstrapped is returned when CreateFirstAdmin runs on an instance
// that already has users.
var ErrAlreadyBootstrapped = errors.New("account: instance already has users")

// CreateFirstAdmin creates the master admin. It only succeeds on a fresh
// instance with no existing users, protecting against re-bootstrap.
func (s *Service) CreateFirstAdmin(ctx context.Context, email, name, password string) (*user.User, error) {
	n, err := s.users.Count(ctx)
	if err != nil {
		return nil, err
	}
	if n > 0 {
		return nil, ErrAlreadyBootstrapped
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	return s.users.Create(ctx, user.CreateInput{
		Email:        email,
		Name:         name,
		Role:         user.RoleAdmin,
		PasswordHash: hash,
	})
}

// CreateInvite mints an invite on behalf of an admin.
func (s *Service) CreateInvite(ctx context.Context, in invite.CreateInput) (*invite.Invite, error) {
	return s.invites.Create(ctx, in)
}

// ErrInvalidInvite is returned when an invite token is missing, expired, or used.
var ErrInvalidInvite = errors.New("account: invite is invalid, expired, or already used")

// RedeemInvite creates a new account from a valid invite token. The invite's
// role is applied; its optional email, if present, overrides the supplied one.
func (s *Service) RedeemInvite(ctx context.Context, token, email, name, password string) (*user.User, error) {
	inv, err := s.invites.GetByToken(ctx, token)
	if errors.Is(err, invite.ErrNotFound) {
		return nil, ErrInvalidInvite
	}
	if err != nil {
		return nil, err
	}
	if !inv.Valid(time.Now().UTC()) {
		return nil, ErrInvalidInvite
	}
	if inv.Email != "" {
		email = inv.Email
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	u, err := s.users.Create(ctx, user.CreateInput{
		Email:        email,
		Name:         name,
		Role:         inv.Role,
		PasswordHash: hash,
	})
	if err != nil {
		return nil, err
	}
	// Mark redeemed after the user exists. If this fails, the account still
	// exists but the invite stays open; surface the error to the caller.
	if err := s.invites.MarkRedeemed(ctx, token, u.ID); err != nil {
		return nil, fmt.Errorf("account: redeem: %w", err)
	}
	return u, nil
}
