package crypto

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}
	if kp.Private == nil {
		t.Fatal("private key is nil")
	}
	if kp.Public == nil {
		t.Fatal("public key is nil")
	}
	if kp.KID == "" {
		t.Fatal("KID is empty")
	}
	if kp.Private.N.BitLen() != 2048 {
		t.Errorf("key size = %d, want 2048", kp.Private.N.BitLen())
	}
}

func TestJWKS(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	jwks := kp.JWKS()
	keys, ok := jwks["keys"].([]map[string]any)
	if !ok || len(keys) != 1 {
		t.Fatalf("expected 1 key in JWKS, got %v", jwks)
	}

	key := keys[0]
	if key["kty"] != "RSA" {
		t.Errorf("kty = %v, want RSA", key["kty"])
	}
	if key["alg"] != "RS256" {
		t.Errorf("alg = %v, want RS256", key["alg"])
	}
	if key["use"] != "sig" {
		t.Errorf("use = %v, want sig", key["use"])
	}
	if key["kid"] != kp.KID {
		t.Errorf("kid = %v, want %v", key["kid"], kp.KID)
	}
	if key["n"] == "" {
		t.Error("n (modulus) is empty")
	}
	if key["e"] == "" {
		t.Error("e (exponent) is empty")
	}
}

func TestSignIDToken(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	claims := &Claims{
		Issuer:   "https://example.com",
		Subject:  "testuser",
		Audience: "client123",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	token, err := kp.SignIDToken(claims)
	if err != nil {
		t.Fatalf("SignIDToken() error: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want 3", len(parts))
	}

	// Verify header
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode header: %v", err)
	}
	var header map[string]string
	json.Unmarshal(headerJSON, &header)
	if header["alg"] != "RS256" {
		t.Errorf("header alg = %v, want RS256", header["alg"])
	}
	if header["kid"] != kp.KID {
		t.Errorf("header kid = %v, want %v", header["kid"], kp.KID)
	}

	// Verify claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode claims: %v", err)
	}
	var decoded Claims
	json.Unmarshal(claimsJSON, &decoded)
	if decoded.Issuer != "https://example.com" {
		t.Errorf("iss = %v, want https://example.com", decoded.Issuer)
	}
	if decoded.Subject != "testuser" {
		t.Errorf("sub = %v, want testuser", decoded.Subject)
	}
	if decoded.Iat == 0 {
		t.Error("iat is 0")
	}
	if decoded.Exp == 0 {
		t.Error("exp is 0")
	}

	// Verify signature
	sigInput := parts[0] + "." + parts[1]
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	hash := sha256.Sum256([]byte(sigInput))
	err = rsa.VerifyPKCS1v15(kp.Public, crypto.SHA256, hash[:], sigBytes)
	if err != nil {
		t.Errorf("signature verification failed: %v", err)
	}
}
