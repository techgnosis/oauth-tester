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

	oidcHandlers := &oidc.Handlers{
		Config: deps.Config,
		Keys:   deps.Keys,
	}

	authHandlers := &oidc.AuthHandlers{
		Store: deps.Store,
	}

	tokenHandlers := &oidc.TokenHandlers{
		Store: deps.Store,
		Keys:  deps.Keys,
	}
	tokenHandlers.Config.IssuerURL = deps.Config.IssuerURL
	tokenHandlers.Config.ClientID = deps.Config.ClientID

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

	// Web UI pages
	mux.HandleFunc("GET /ui/users", pageHandlers.UsersPage)
	mux.HandleFunc("GET /ui/logs", pageHandlers.LogsPage)

	// JSON API - users
	mux.HandleFunc("GET /api/users", apiHandlers.ListUsers)
	mux.HandleFunc("POST /api/users", apiHandlers.CreateUser)
	mux.HandleFunc("PUT /api/users/{username}", apiHandlers.UpdateUser)
	mux.HandleFunc("DELETE /api/users/{username}", apiHandlers.DeleteUser)

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

func notImplemented(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
