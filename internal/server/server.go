// Package server sets up the HTTP server, TLS, and routing.
package server

import (
	"net/http"

	"oauth-tester/internal/config"
	"oauth-tester/internal/crypto"
	"oauth-tester/internal/store"
)

// Deps holds all dependencies needed by the HTTP handlers.
type Deps struct {
	Config *config.Config
	Store  store.Store
	Keys   *crypto.KeyPair
}

// New creates a configured http.ServeMux with all routes.
func New(deps *Deps) *http.ServeMux {
	mux := http.NewServeMux()

	// OIDC endpoints (stubs for now, implemented in later tasks)
	mux.HandleFunc("GET /.well-known/openid-configuration", notImplemented)
	mux.HandleFunc("GET /jwks", notImplemented)
	mux.HandleFunc("GET /auth", notImplemented)
	mux.HandleFunc("POST /auth", notImplemented)
	mux.HandleFunc("POST /token", notImplemented)

	// Web UI pages
	mux.HandleFunc("GET /ui/users", notImplemented)
	mux.HandleFunc("GET /ui/logs", notImplemented)

	// JSON API
	mux.HandleFunc("GET /api/users", notImplemented)
	mux.HandleFunc("POST /api/users", notImplemented)
	mux.HandleFunc("PUT /api/users/{username}", notImplemented)
	mux.HandleFunc("DELETE /api/users/{username}", notImplemented)

	// Root redirect
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/ui/users", http.StatusFound)
	})

	return mux
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
