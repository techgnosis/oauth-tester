// Package crypto handles RSA key pair generation and JWT signing.
package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"math/big"
)

// KeyPair holds an RSA key pair generated at startup.
type KeyPair struct {
	Private *rsa.PrivateKey
	Public  *rsa.PublicKey
	KID     string // key ID derived from the public key
}

// GenerateKeyPair creates a new 2048-bit RSA key pair with a deterministic key ID.
func GenerateKeyPair() (*KeyPair, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate RSA key: %w", err)
	}

	pub := &priv.PublicKey
	kid, err := computeKID(pub)
	if err != nil {
		return nil, fmt.Errorf("compute key ID: %w", err)
	}

	return &KeyPair{
		Private: priv,
		Public:  pub,
		KID:     kid,
	}, nil
}

// JWKS returns the JSON Web Key Set representation of the public key.
func (kp *KeyPair) JWKS() map[string]any {
	pub := kp.Public
	return map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": kp.KID,
				"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
			},
		},
	}
}

// computeKID generates a stable key ID by hashing the DER-encoded public key.
func computeKID(pub *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(hash[:8]), nil
}
