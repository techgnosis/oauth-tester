package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

// Claims represents the JWT claims for an ID token.
type Claims struct {
	Issuer   string   `json:"iss"`
	Subject  string   `json:"sub"`
	Audience string   `json:"aud"`
	Exp      int64    `json:"exp"`
	Iat      int64    `json:"iat"`
	Nonce    string   `json:"nonce,omitempty"`
	Email    string   `json:"email,omitempty"`
	Name     string   `json:"name,omitempty"`
	Groups   []string `json:"groups"`
}

// SignIDToken creates a signed JWT ID token.
func (kp *KeyPair) SignIDToken(claims *Claims) (string, error) {
	now := time.Now().UTC()
	claims.Iat = now.Unix()
	if claims.Exp == 0 {
		claims.Exp = now.Add(1 * time.Hour).Unix()
	}

	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kp.KID,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerB64 + "." + claimsB64

	hash := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, kp.Private, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}

	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return strings.Join([]string{headerB64, claimsB64, sigB64}, "."), nil
}
