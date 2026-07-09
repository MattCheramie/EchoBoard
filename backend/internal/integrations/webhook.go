package integrations

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SandboxSignatureHeader carries the HMAC signature on sandbox webhooks.
const SandboxSignatureHeader = "X-Echoboard-Signature"

// SignHMAC returns the "sha256=<hex>" signature of body under secret. Adapters
// and tests use it to produce the header a provider would send.
func SignHMAC(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifyHMACSignature checks a "sha256=<hex>" signature over body using a
// constant-time comparison. An empty secret means the provider is not
// configured and verification cannot succeed.
func VerifyHMACSignature(secret string, body []byte, provided string) error {
	if secret == "" {
		return ErrNotConfigured
	}
	got, err := hex.DecodeString(strings.TrimPrefix(provided, "sha256="))
	if err != nil {
		return ErrBadSignature
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	want := mac.Sum(nil)
	if !hmac.Equal(want, got) {
		return ErrBadSignature
	}
	return nil
}
