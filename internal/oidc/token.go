package oidc

import (
	"encoding/json"
	"net/http"
	"time"

	"oauth-tester/internal/crypto"
	"oauth-tester/internal/store"
)

type TokenHandlers struct {
	Config struct {
		IssuerURL    string
		ClientID     string
		ClientSecret string
	}
	Store store.Store
	Keys  *crypto.KeyPair
}

func (t *TokenHandlers) Token(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	grantType := r.FormValue("grant_type")
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")

	if grantType != "authorization_code" {
		tokenError(w, "unsupported_grant_type", "only authorization_code is supported")
		return
	}

	if code == "" {
		tokenError(w, "invalid_request", "code is required")
		return
	}

	// Consume the auth code (one-time use)
	ac, err := t.Store.ConsumeAuthCode(code)
	if err != nil {
		tokenError(w, "invalid_grant", "invalid or expired authorization code")
		return
	}

	// Check expiry (5 minute TTL)
	if time.Since(ac.CreatedAt) > 5*time.Minute {
		tokenError(w, "invalid_grant", "authorization code expired")
		return
	}

	// Validate redirect_uri matches
	if redirectURI != "" && redirectURI != ac.RedirectURI {
		tokenError(w, "invalid_grant", "redirect_uri mismatch")
		return
	}

	// Validate client credentials (support both client_secret_post and client_secret_basic)
	clientSecret := r.FormValue("client_secret")
	if basicUser, basicPass, ok := r.BasicAuth(); ok {
		if clientID == "" {
			clientID = basicUser
		}
		if clientSecret == "" {
			clientSecret = basicPass
		}
	}
	if t.Config.ClientSecret != "" {
		if clientSecret != t.Config.ClientSecret {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error":             "invalid_client",
				"error_description": "invalid client_secret",
			})
			return
		}
	}
	if clientID != "" && clientID != ac.ClientID {
		tokenError(w, "invalid_client", "client_id mismatch")
		return
	}

	// Look up user for claims
	user, err := t.Store.GetUser(ac.Username)
	if err != nil {
		tokenError(w, "server_error", "user not found")
		return
	}

	// Use client_id from the original auth request if available, fall back to config
	audience := ac.ClientID
	if audience == "" {
		audience = t.Config.ClientID
	}

	// Generate ID token
	claims := &crypto.Claims{
		Issuer:        t.Config.IssuerURL,
		Subject:       user.Username,
		Audience:      []string{audience},
		Nonce:         ac.Nonce,
		Email:         user.Email,
		EmailVerified: true,
		Name:          user.Name,
		Groups:        []string{"users"},
	}

	idToken, err := t.Keys.SignIDToken(claims)
	if err != nil {
		tokenError(w, "server_error", "failed to sign token")
		return
	}

	resp := map[string]any{
		"access_token": idToken,
		"token_type":   "bearer",
		"id_token":     idToken,
		"expires_in":   3600,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func tokenError(w http.ResponseWriter, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": desc,
	})
}
