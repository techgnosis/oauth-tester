// Package server sets up the HTTP server, TLS, and routing.
package server

import (
	"net/http"

	"oauth-tester/internal/config"
	"oauth-tester/internal/crypto"
	"oauth-tester/internal/oidc"
	"oauth-tester/internal/store"
	"oauth-tester/internal/ui"
)

// Deps holds all dependencies needed by the HTTP handlers.
type Deps struct {
	Config *config.Config
	Store  store.Store
	Keys   *crypto.KeyPair
}

// New creates a configured http.Handler with all routes.
func New(deps *Deps) http.Handler {
	mux := http.NewServeMux()

	oidcHandlers := &oidc.DiscoveryHandlers{
		Config: deps.Config,
		Keys:   deps.Keys,
	}

	authHandlers := &oidc.AuthHandlers{
		Store: deps.Store,
	}

	tokenHandlers := &oidc.TokenHandlers{
		Config: deps.Config,
		Store:  deps.Store,
		Keys:   deps.Keys,
	}

	userInfoHandlers := &oidc.UserInfoHandlers{
		Keys: deps.Keys,
	}

	apiHandlers := &ui.APIHandlers{
		Store: deps.Store,
	}

	pageHandlers := &ui.PageHandlers{
		Store: deps.Store,
	}

	// OIDC endpoints
	mux.HandleFunc("GET /.well-known/openid-configuration", oidcHandlers.Discovery)
	mux.HandleFunc("GET /jwks", oidcHandlers.JWKS)
	mux.HandleFunc("GET /auth", authHandlers.AuthGet)
	mux.HandleFunc("POST /auth", authHandlers.AuthPost)
	mux.HandleFunc("POST /token", tokenHandlers.Token)
	mux.HandleFunc("GET /userinfo", userInfoHandlers.UserInfo)

	// Web UI pages
	mux.HandleFunc("GET /ui/users", pageHandlers.UsersPage)
	mux.HandleFunc("GET /ui/groups", pageHandlers.GroupsPage)
	mux.HandleFunc("GET /ui/logs/oidc", pageHandlers.OIDCLogsPage)
	mux.HandleFunc("GET /ui/logs/scim", pageHandlers.SCIMLogsPage)
	mux.HandleFunc("GET /ui/scim", pageHandlers.ScimPage)

	// JSON API - users
	mux.HandleFunc("GET /api/users", apiHandlers.ListUsers)
	mux.HandleFunc("POST /api/users", apiHandlers.CreateUser)
	mux.HandleFunc("PUT /api/users/{username}", apiHandlers.UpdateUser)
	mux.HandleFunc("DELETE /api/users/{username}", apiHandlers.DeleteUser)

	// JSON API - groups
	mux.HandleFunc("GET /api/groups", apiHandlers.ListGroups)
	mux.HandleFunc("POST /api/groups", apiHandlers.CreateGroup)
	mux.HandleFunc("DELETE /api/groups/{name}", apiHandlers.DeleteGroup)
	mux.HandleFunc("GET /api/groups/{name}/members", apiHandlers.ListGroupMembers)
	mux.HandleFunc("POST /api/groups/{name}/members", apiHandlers.AddGroupMember)
	mux.HandleFunc("DELETE /api/groups/{name}/members/{username}", apiHandlers.RemoveGroupMember)

	// JSON API - SCIM
	mux.HandleFunc("GET /api/scim/config", apiHandlers.GetScimConfig)
	mux.HandleFunc("PUT /api/scim/config", apiHandlers.SetScimConfig)
	mux.HandleFunc("POST /api/scim/push", apiHandlers.ScimPush)

	// JSON API - logs
	mux.HandleFunc("GET /api/logs", apiHandlers.ListLogs)
	mux.HandleFunc("DELETE /api/logs", apiHandlers.ClearLogs)

	// Root redirect
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/ui/users", http.StatusFound)
	})

	// Wrap with logging middleware for OIDC paths
	return store.LoggingMiddleware(deps.Store, mux)
}
