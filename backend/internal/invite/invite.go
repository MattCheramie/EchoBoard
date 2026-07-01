// Package invite implements invite-only, time-limited provisioning: an admin
// mints an invite, and a new user redeems its token to create an account.
package invite

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/MattCheramie/echoboard/internal/user"
)

// DefaultTTL is how long a new invite stays valid if no expiry is given.
const DefaultTTL = 7 * 24 * time.Hour

// Invite is a single-use, time-limited token granting a role.
type Invite struct {
	ID         string     `json:"id"`
	Token      string     `json:"token"`
	Email      string     `json:"email,omitempty"` // optional pre-assigned email
	Role       user.Role  `json:"role"`
	CreatedBy  string     `json:"createdBy"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	RedeemedAt *time.Time `json:"redeemedAt,omitempty"`
	RedeemedBy *string    `json:"redeemedBy,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// Redeemed reports whether the invite has already been used.
func (i *Invite) Redeemed() bool { return i.RedeemedAt != nil }

// Valid reports whether the invite can still be redeemed at time now.
func (i *Invite) Valid(now time.Time) bool {
	return !i.Redeemed() && now.Before(i.ExpiresAt)
}

// newToken returns a URL-safe, cryptographically random invite token.
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
