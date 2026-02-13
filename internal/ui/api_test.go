package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

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

func TestListUsersEmpty(t *testing.T) {
	s := testStore(t)
	h := &APIHandlers{Store: s}

	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	h.ListUsers(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	var users []store.User
	json.NewDecoder(w.Body).Decode(&users)
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestCreateAndListUsers(t *testing.T) {
	s := testStore(t)
	h := &APIHandlers{Store: s}

	// Create
	body := `{"Username":"bob","Password":"secret","Email":"bob@test.com","Name":"Bob"}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateUser(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body: %s", w.Code, w.Body.String())
	}

	// List
	req = httptest.NewRequest("GET", "/api/users", nil)
	w = httptest.NewRecorder()
	h.ListUsers(w, req)

	var users []store.User
	json.NewDecoder(w.Body).Decode(&users)
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].Username != "bob" {
		t.Errorf("username = %q, want bob", users[0].Username)
	}
}

func TestUpdateUser(t *testing.T) {
	s := testStore(t)
	h := &APIHandlers{Store: s}

	s.CreateUser(&store.User{Username: "carol", Password: "old"})

	body := `{"Password":"new","Email":"carol@new.com","Name":"Carol"}`
	req := httptest.NewRequest("PUT", "/api/users/carol", strings.NewReader(body))
	req.SetPathValue("username", "carol")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdateUser(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	got, _ := s.GetUser("carol")
	if got.Password != "new" {
		t.Errorf("password = %q, want new", got.Password)
	}
}

func TestDeleteUser(t *testing.T) {
	s := testStore(t)
	h := &APIHandlers{Store: s}

	s.CreateUser(&store.User{Username: "dave", Password: "x"})

	req := httptest.NewRequest("DELETE", "/api/users/dave", nil)
	req.SetPathValue("username", "dave")
	w := httptest.NewRecorder()
	h.DeleteUser(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d", w.Code)
	}

	_, err := s.GetUser("dave")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestCreateUserMissingUsername(t *testing.T) {
	s := testStore(t)
	h := &APIHandlers{Store: s}

	body := `{"Password":"x"}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
