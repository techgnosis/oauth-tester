package store

import (
	"os"
	"testing"
	"time"
)

func testDB(t *testing.T) *SQLiteStore {
	t.Helper()
	f, err := os.CreateTemp("", "oauth-tester-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	s, err := Open(f.Name())
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestUserCRUD(t *testing.T) {
	s := testDB(t)

	// Create
	u := &User{Username: "alice", Password: "pass123", Email: "alice@test.com", Name: "Alice"}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.ID == 0 {
		t.Error("expected non-zero ID after create")
	}

	// Get
	got, err := s.GetUser("alice")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.Password != "pass123" {
		t.Errorf("password = %q, want %q", got.Password, "pass123")
	}
	if got.Email != "alice@test.com" {
		t.Errorf("email = %q, want %q", got.Email, "alice@test.com")
	}

	// List
	users, err := s.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("ListUsers count = %d, want 1", len(users))
	}

	// Update
	u.Password = "newpass"
	u.Email = "alice2@test.com"
	if err := s.UpdateUser(u); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	got, _ = s.GetUser("alice")
	if got.Password != "newpass" {
		t.Errorf("after update password = %q, want %q", got.Password, "newpass")
	}

	// Delete
	if err := s.DeleteUser("alice"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	_, err = s.GetUser("alice")
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestDuplicateUsername(t *testing.T) {
	s := testDB(t)
	s.CreateUser(&User{Username: "bob", Password: "x"})
	err := s.CreateUser(&User{Username: "bob", Password: "y"})
	if err == nil {
		t.Error("expected error for duplicate username")
	}
}

func TestAuthCode(t *testing.T) {
	s := testDB(t)

	ac := &AuthCode{
		Code:        "testcode123",
		Username:    "alice",
		RedirectURI: "https://app.example.com/callback",
		Scope:       "openid",
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.SaveAuthCode(ac); err != nil {
		t.Fatalf("SaveAuthCode: %v", err)
	}

	// Consume
	got, err := s.ConsumeAuthCode("testcode123")
	if err != nil {
		t.Fatalf("ConsumeAuthCode: %v", err)
	}
	if got.Username != "alice" {
		t.Errorf("username = %q, want alice", got.Username)
	}
	if got.RedirectURI != "https://app.example.com/callback" {
		t.Errorf("redirect_uri = %q", got.RedirectURI)
	}

	// Second consume should fail (one-time use)
	_, err = s.ConsumeAuthCode("testcode123")
	if err == nil {
		t.Error("expected error on second consume")
	}
}

func TestLogs(t *testing.T) {
	s := testDB(t)

	entry := &LogEntry{
		Timestamp:       time.Now().UTC(),
		Method:          "GET",
		Path:            "/.well-known/openid-configuration",
		ResponseStatus:  200,
		ResponseBody:    `{"issuer":"https://example.com"}`,
	}
	if err := s.SaveLog(entry); err != nil {
		t.Fatalf("SaveLog: %v", err)
	}

	logs, err := s.ListLogs(10)
	if err != nil {
		t.Fatalf("ListLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("log count = %d, want 1", len(logs))
	}
	if logs[0].Method != "GET" {
		t.Errorf("method = %q, want GET", logs[0].Method)
	}

	// Clear
	if err := s.ClearLogs(); err != nil {
		t.Fatalf("ClearLogs: %v", err)
	}
	logs, _ = s.ListLogs(10)
	if len(logs) != 0 {
		t.Errorf("after clear, log count = %d, want 0", len(logs))
	}
}
