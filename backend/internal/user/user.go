// Package user defines the user model, roles, and repository, plus the
// admin-bootstrap and invite-only provisioning flows.
package user

import (
	"fmt"
	"strings"
	"time"
)

// Role is a user's authorization level.
type Role string

const (
	// RoleAdmin can manage users, invites, and all content.
	RoleAdmin Role = "admin"
	// RoleMember is a standard team member.
	RoleMember Role = "member"
)

// Valid reports whether r is a known role.
func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

// User is an EchoBoard account. PasswordHash is never serialized to clients.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         Role      `json:"role"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// NormalizeEmail lower-cases and trims an email for consistent storage/lookup.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validateForCreate checks the fields required to persist a new user.
func validateForCreate(email, name string, role Role) error {
	if NormalizeEmail(email) == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("user: invalid email %q", email)
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("user: name is required")
	}
	if !role.Valid() {
		return fmt.Errorf("user: invalid role %q", role)
	}
	return nil
}
