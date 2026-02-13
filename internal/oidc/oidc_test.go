package oidc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"oauth-tester/internal/config"
	"oauth-tester/internal/crypto"
)

func TestDiscovery(t *testing.T) {
	kp, _ := crypto.GenerateKeyPair()
	h := &Handlers{
		Config: &config.Config{IssuerURL: "https://idp.example.com"},
		Keys:   kp,
	}

	req := httptest.NewRequest("GET", "/.well-known/openid-configuration", nil)
	w := httptest.NewRecorder()
	h.Discovery(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var doc map[string]any
	json.NewDecoder(w.Body).Decode(&doc)

	if doc["issuer"] != "https://idp.example.com" {
		t.Errorf("issuer = %v", doc["issuer"])
	}
	if doc["authorization_endpoint"] != "https://idp.example.com/auth" {
		t.Errorf("authorization_endpoint = %v", doc["authorization_endpoint"])
	}
	if doc["token_endpoint"] != "https://idp.example.com/token" {
		t.Errorf("token_endpoint = %v", doc["token_endpoint"])
	}
	if doc["jwks_uri"] != "https://idp.example.com/jwks" {
		t.Errorf("jwks_uri = %v", doc["jwks_uri"])
	}
}

func TestJWKSEndpoint(t *testing.T) {
	kp, _ := crypto.GenerateKeyPair()
	h := &Handlers{
		Config: &config.Config{IssuerURL: "https://idp.example.com"},
		Keys:   kp,
	}

	req := httptest.NewRequest("GET", "/jwks", nil)
	w := httptest.NewRecorder()
	h.JWKS(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var jwks map[string]any
	json.NewDecoder(w.Body).Decode(&jwks)

	keys, ok := jwks["keys"].([]any)
	if !ok || len(keys) != 1 {
		t.Fatalf("expected 1 key, got %v", jwks)
	}
}

func TestAuthGetRendersForm(t *testing.T) {
	ah := &AuthHandlers{Store: nil} // store not needed for GET

	req := httptest.NewRequest("GET", "/auth?client_id=test&redirect_uri=https://app/callback&response_type=code&scope=openid&state=xyz", nil)
	w := httptest.NewRecorder()
	ah.AuthGet(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	body := w.Body.String()
	if !contains(body, "Login") {
		t.Error("response missing Login button")
	}
	if !contains(body, "username") {
		t.Error("response missing username field")
	}
	if !contains(body, `value="xyz"`) {
		t.Error("response missing state hidden field")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && http.DetectContentType([]byte(s)) != "" && // just use strings
		len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
