package ui

import (
	"encoding/json"
	"net/http"

	"oauth-tester/internal/store"
)

type APIHandlers struct {
	Store store.Store
}

// jsonError writes a JSON error response with the given status code.
func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (a *APIHandlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.Store.ListUsers()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if users == nil {
		users = []store.User{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (a *APIHandlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	var u store.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if u.Username == "" {
		jsonError(w, "username is required", http.StatusBadRequest)
		return
	}
	if err := a.Store.CreateUser(&u); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}

func (a *APIHandlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	var u store.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	u.Username = username
	if err := a.Store.UpdateUser(&u); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func (a *APIHandlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if err := a.Store.DeleteUser(username); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
