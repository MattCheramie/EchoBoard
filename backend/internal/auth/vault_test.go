package auth_test

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/MattCheramie/echoboard/internal/auth"
)

func testKey(t *testing.T) string {
	t.Helper()
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(k)
}

func TestVaultRoundTrip(t *testing.T) {
	v, err := auth.NewVault(testKey(t))
	if err != nil {
		t.Fatalf("NewVault: %v", err)
	}
	secret := []byte("meta-integration-token-123")
	enc, err := v.Encrypt(secret)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc == string(secret) {
		t.Fatal("ciphertext equals plaintext")
	}
	dec, err := v.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(dec, secret) {
		t.Errorf("round-trip mismatch: %q", dec)
	}
}

func TestVaultWrongKeyFails(t *testing.T) {
	v1, _ := auth.NewVault(testKey(t))
	v2, _ := auth.NewVault(testKey(t))
	enc, _ := v1.Encrypt([]byte("secret"))
	if _, err := v2.Decrypt(enc); err == nil {
		t.Error("decrypt with a different key should fail")
	}
}

func TestVaultEphemeralAndBadKey(t *testing.T) {
	v, err := auth.NewVault("")
	if err != nil {
		t.Fatalf("ephemeral NewVault: %v", err)
	}
	if !v.Ephemeral() {
		t.Error("expected ephemeral vault for empty key")
	}
	if _, err := auth.NewVault("not-base64!!!"); err == nil {
		t.Error("invalid base64 key should error")
	}
	if _, err := auth.NewVault(base64.StdEncoding.EncodeToString([]byte("short"))); err == nil {
		t.Error("wrong-length key should error")
	}
}
