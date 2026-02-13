// Package oidc implements the OIDC discovery, authorization, and token endpoints.
package oidc

import (
	"encoding/json"
	"net/http"

	"oauth-tester/internal/config"
	"oauth-tester/internal/crypto"
)

type Handlers struct {
	Config *config.Config
	Keys   *crypto.KeyPair
}

// Discovery returns the OpenID Connect discovery document.
func (h *Handlers) Discovery(w http.ResponseWriter, r *http.Request) {
	issuer := h.Config.IssuerURL

	doc := map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/auth",
		"token_endpoint":                        issuer + "/token",
		"jwks_uri":                              issuer + "/jwks",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "email", "profile"},
		"grant_types_supported":                 []string{"authorization_code"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

// JWKS returns the JSON Web Key Set containing the public signing key.
func (h *Handlers) JWKS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.Keys.JWKS())
}
