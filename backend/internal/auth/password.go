// Package auth handles password hashing, sessions, authorization middleware,
// and the encrypted-at-rest secrets vault for integration tokens.
package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// MinPasswordLength is the shortest password EchoBoard will accept.
const MinPasswordLength = 8

// HashPassword returns a bcrypt hash of the given plaintext password.
func HashPassword(plaintext string) (string, error) {
	if len(plaintext) < MinPasswordLength {
		return "", fmt.Errorf("auth: password must be at least %d characters", MinPasswordLength)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword reports whether plaintext matches the stored bcrypt hash.
func CheckPassword(hash, plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)) == nil
}
