package oidc

import (
	"encoding/json"
	"net/http"
	"strings"

	"oauth-tester/internal/crypto"
)

// UserInfoHandlers serves the OpenID Connect UserInfo endpoint.
type UserInfoHandlers struct {
	Keys *crypto.KeyPair
}

// UserInfo returns claims for the authenticated user based on the Bearer token.
func (h *UserInfoHandlers) UserInfo(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return
	}
	token := strings.TrimPrefix(auth, "Bearer ")

	claims, err := h.Keys.VerifyIDToken(token)
	if err != nil {
		http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	resp := map[string]any{
		"sub":            claims.Subject,
		"email":          claims.Email,
		"email_verified": claims.EmailVerified,
		"name":           claims.Name,
		"groups":         claims.Groups,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
