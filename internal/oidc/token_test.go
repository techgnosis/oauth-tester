package oidc

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"oauth-tester/internal/crypto"
	"oauth-tester/internal/store"
)

func testStore(t *testing.T) store.Store {
	t.Helper()
	f, err := os.CreateTemp("", "oauth-tester-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	s, err := store.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestTokenEndpoint(t *testing.T) {
	s := testStore(t)
	kp, _ := crypto.GenerateKeyPair()

	// Create a user and auth code
	s.CreateUser(&store.User{Username: "alice", Password: "pass", Email: "alice@test.com", Name: "Alice"})
	s.SaveAuthCode(&store.AuthCode{
		Code:        "validcode",
		Username:    "alice",
		RedirectURI: "https://app/callback",
		Scope:       "openid",
		CreatedAt:   time.Now().UTC(),
	})

	th := &TokenHandlers{
		Store: s,
		Keys:  kp,
	}
	th.Config.IssuerURL = "https://idp.example.com"
	th.Config.ClientID = "testclient"

	body := "grant_type=authorization_code&code=validcode&redirect_uri=https://app/callback&client_id=testclient"
	req := httptest.NewRequest("POST", "/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	th.Token(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["token_type"] != "bearer" {
		t.Errorf("token_type = %v, want bearer", resp["token_type"])
	}
	if resp["id_token"] == nil || resp["id_token"] == "" {
		t.Error("id_token is empty")
	}
	if resp["access_token"] == nil || resp["access_token"] == "" {
		t.Error("access_token is empty")
	}

	// Verify the JWT has 3 parts
	idToken, _ := resp["id_token"].(string)
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		t.Errorf("id_token has %d parts, want 3", len(parts))
	}
}

func TestTokenEndpointInvalidCode(t *testing.T) {
	s := testStore(t)
	kp, _ := crypto.GenerateKeyPair()

	th := &TokenHandlers{
		Store: s,
		Keys:  kp,
	}
	th.Config.IssuerURL = "https://idp.example.com"
	th.Config.ClientID = "testclient"

	body := "grant_type=authorization_code&code=badcode&client_id=testclient"
	req := httptest.NewRequest("POST", "/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	th.Token(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestTokenEndpointUnsupportedGrant(t *testing.T) {
	s := testStore(t)
	kp, _ := crypto.GenerateKeyPair()

	th := &TokenHandlers{
		Store: s,
		Keys:  kp,
	}

	body := "grant_type=client_credentials"
	req := httptest.NewRequest("POST", "/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	th.Token(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
