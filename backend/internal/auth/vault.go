package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Vault encrypts and decrypts small secrets (integration tokens, API keys) for
// storage at rest, using AES-256-GCM.
type Vault struct {
	aead      cipher.AEAD
	ephemeral bool
}

// NewVault builds a Vault from a base64-encoded 32-byte key (see config
// SECRET_KEY). If secretKeyB64 is empty, a random ephemeral key is generated:
// convenient for development, but secrets will not survive a restart. Callers
// should require a real key in production (config.Validate enforces this).
func NewVault(secretKeyB64 string) (*Vault, error) {
	var key []byte
	ephemeral := false
	if secretKeyB64 == "" {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("auth: generate ephemeral key: %w", err)
		}
		ephemeral = true
	} else {
		decoded, err := base64.StdEncoding.DecodeString(secretKeyB64)
		if err != nil {
			return nil, fmt.Errorf("auth: SECRET_KEY is not valid base64: %w", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("auth: SECRET_KEY must decode to 32 bytes, got %d", len(decoded))
		}
		key = decoded
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("auth: new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("auth: new gcm: %w", err)
	}
	return &Vault{aead: aead, ephemeral: ephemeral}, nil
}

// Ephemeral reports whether the vault is using a throwaway dev key.
func (v *Vault) Ephemeral() bool { return v.ephemeral }

// Encrypt returns base64(nonce || ciphertext) for the given plaintext.
func (v *Vault) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("auth: nonce: %w", err)
	}
	ct := v.aead.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// Decrypt reverses Encrypt.
func (v *Vault) Decrypt(encoded string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("auth: decode secret: %w", err)
	}
	ns := v.aead.NonceSize()
	if len(raw) < ns {
		return nil, fmt.Errorf("auth: secret too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := v.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("auth: decrypt: %w", err)
	}
	return pt, nil
}
