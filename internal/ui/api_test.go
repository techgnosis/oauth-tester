package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

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

func TestListLogsFiltered(t *testing.T) {
	s := testStore(t)
	h := &APIHandlers{Store: s}

	now := time.Now().UTC()
	s.SaveLog(&store.LogEntry{Timestamp: now, Method: "GET", Path: "/auth", ResponseStatus: 302})
	s.SaveLog(&store.LogEntry{Timestamp: now, Method: "POST", Path: "/token", ResponseStatus: 200, ResponseBody: `{"access_token":"ok"}`})
	s.SaveLog(&store.LogEntry{Timestamp: now, Method: "POST", Path: "/token", ResponseStatus: 400, ResponseBody: `{"error":"bad"}`})

	tests := []struct {
		name  string
		query string
		want  int
	}{
		{"no filter", "/api/logs", 3},
		{"method=POST", "/api/logs?method=POST", 2},
		{"path=/token", "/api/logs?path=/token", 2},
		{"status=400", "/api/logs?status=400", 1},
		{"limit=1", "/api/logs?limit=1", 1},
		{"combined", "/api/logs?method=POST&path=/token&status=400", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.query, nil)
			w := httptest.NewRecorder()
			h.ListLogs(w, req)

			if w.Code != 200 {
				t.Fatalf("status = %d", w.Code)
			}

			var logs []store.LogEntry
			json.NewDecoder(w.Body).Decode(&logs)
			if len(logs) != tt.want {
				t.Errorf("got %d logs, want %d", len(logs), tt.want)
			}
		})
	}
}

func TestClearLogs(t *testing.T) {
	s := testStore(t)
	h := &APIHandlers{Store: s}

	s.SaveLog(&store.LogEntry{Timestamp: time.Now().UTC(), Method: "GET", Path: "/auth", ResponseStatus: 200})

	req := httptest.NewRequest("DELETE", "/api/logs", nil)
	w := httptest.NewRecorder()
	h.ClearLogs(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}

	// Verify cleared
	req = httptest.NewRequest("GET", "/api/logs", nil)
	w = httptest.NewRecorder()
	h.ListLogs(w, req)

	var logs []store.LogEntry
	json.NewDecoder(w.Body).Decode(&logs)
	if len(logs) != 0 {
		t.Errorf("expected 0 logs after clear, got %d", len(logs))
	}
}
