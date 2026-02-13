package oidc

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"oauth-tester/internal/store"
)

var loginTmpl = template.Must(template.New("login").Parse(`<!DOCTYPE html>
<html>
<head>
<title>Login - oauth-tester</title>
<style>
  body { font-family: system-ui, sans-serif; margin: 0; padding: 0; background: #f5f5f5; }
  nav { background: #1a1a2e; padding: 12px 24px; display: flex; gap: 16px; }
  nav a { color: #e0e0e0; text-decoration: none; font-size: 14px; }
  nav a:hover { color: #fff; }
  .container { max-width: 400px; margin: 80px auto; padding: 32px; background: #fff; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
  h1 { margin: 0 0 24px; font-size: 24px; }
  label { display: block; margin-bottom: 4px; font-size: 14px; font-weight: 500; }
  input { width: 100%; padding: 8px; margin-bottom: 16px; border: 1px solid #ccc; border-radius: 4px; box-sizing: border-box; }
  button { width: 100%; padding: 10px; background: #1a1a2e; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; }
  button:hover { background: #16213e; }
  .error { color: #d32f2f; margin-bottom: 16px; padding: 8px; background: #fdecea; border-radius: 4px; }
</style>
</head>
<body>
<nav>
  <a href="/ui/users">Users</a>
  <a href="/ui/logs">Logs</a>
</nav>
<div class="container">
  <h1>Login</h1>
  {{if .Error}}<div class="error">{{.Error}}</div>{{end}}
  <form method="POST" action="/auth">
    <input type="hidden" name="client_id" value="{{.ClientID}}">
    <input type="hidden" name="redirect_uri" value="{{.RedirectURI}}">
    <input type="hidden" name="response_type" value="{{.ResponseType}}">
    <input type="hidden" name="scope" value="{{.Scope}}">
    <input type="hidden" name="state" value="{{.State}}">
    <input type="hidden" name="nonce" value="{{.Nonce}}">
    <label for="username">Username</label>
    <input type="text" id="username" name="username" required autofocus>
    <label for="password">Password</label>
    <input type="password" id="password" name="password" required>
    <button type="submit">Login</button>
  </form>
</div>
</body>
</html>`))

type loginData struct {
	ClientID     string
	RedirectURI  string
	ResponseType string
	Scope        string
	State        string
	Nonce        string
	Error        string
}

// AuthHandlers needs the store for user lookup and auth code storage.
type AuthHandlers struct {
	Store store.Store
}

// AuthGet renders the login page.
func (a *AuthHandlers) AuthGet(w http.ResponseWriter, r *http.Request) {
	data := loginData{
		ClientID:     r.URL.Query().Get("client_id"),
		RedirectURI:  r.URL.Query().Get("redirect_uri"),
		ResponseType: r.URL.Query().Get("response_type"),
		Scope:        r.URL.Query().Get("scope"),
		State:        r.URL.Query().Get("state"),
		Nonce:        r.URL.Query().Get("nonce"),
	}
	w.Header().Set("Content-Type", "text/html")
	loginTmpl.Execute(w, data)
}

// AuthPost handles login form submission.
func (a *AuthHandlers) AuthPost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	username := r.FormValue("username")
	password := r.FormValue("password")
	redirectURI := r.FormValue("redirect_uri")
	state := r.FormValue("state")
	scope := r.FormValue("scope")

	clientID := r.FormValue("client_id")
	nonce := r.FormValue("nonce")

	data := loginData{
		ClientID:     clientID,
		RedirectURI:  redirectURI,
		ResponseType: r.FormValue("response_type"),
		Scope:        scope,
		State:        state,
		Nonce:        nonce,
	}

	// Validate credentials
	user, err := a.Store.GetUser(username)
	if err != nil || user.Password != password {
		data.Error = "Invalid username or password"
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusUnauthorized)
		loginTmpl.Execute(w, data)
		return
	}

	// Generate authorization code
	code, err := generateCode()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	err = a.Store.SaveAuthCode(&store.AuthCode{
		Code:        code,
		Username:    username,
		RedirectURI: redirectURI,
		Scope:       scope,
		Nonce:       nonce,
		ClientID:    clientID,
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Redirect back to the client
	location := fmt.Sprintf("%s?code=%s&state=%s", redirectURI, code, state)
	http.Redirect(w, r, location, http.StatusFound)
}

func generateCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
