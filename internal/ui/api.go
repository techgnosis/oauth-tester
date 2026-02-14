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

// --- Group endpoints ---

func (a *APIHandlers) ListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := a.Store.ListGroups()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if groups == nil {
		groups = []store.Group{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}

func (a *APIHandlers) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var body struct{ Name string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}
	if err := a.Store.CreateGroup(body.Name); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"name": body.Name})
}

func (a *APIHandlers) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := a.Store.DeleteGroup(name); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *APIHandlers) ListGroupMembers(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	members, err := a.Store.ListGroupMembers(name)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if members == nil {
		members = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

func (a *APIHandlers) AddGroupMember(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var body struct{ Username string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Username == "" {
		jsonError(w, "username is required", http.StatusBadRequest)
		return
	}
	if err := a.Store.AddUserToGroup(body.Username, name); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *APIHandlers) RemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	username := r.PathValue("username")
	if err := a.Store.RemoveUserFromGroup(username, name); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
