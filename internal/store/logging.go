package store

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

// OIDCPaths are the paths that get logged.
var OIDCPaths = []string{
	"/.well-known/openid-configuration",
	"/jwks",
	"/auth",
	"/token",
	"/userinfo",
}

// LoggingMiddleware captures OIDC request/response pairs to the store.
func LoggingMiddleware(s Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isOIDCPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Capture request body
		var reqBody []byte
		if r.Body != nil {
			reqBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(reqBody))
		}

		// Capture response
		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     200,
			body:           &bytes.Buffer{},
		}

		next.ServeHTTP(rec, r)

		// Build header strings
		reqHeaders := headerString(r.Header)
		respHeaders := headerString(rec.Header())

		responseBody := rec.body.String()
		if r.URL.Path == "/token" && rec.statusCode == 200 {
			responseBody = decodeJWTsInResponse(responseBody)
		}

		entry := &LogEntry{
			Timestamp:       time.Now().UTC(),
			Method:          r.Method,
			Path:            r.URL.Path,
			Query:           r.URL.RawQuery,
			RequestHeaders:  reqHeaders,
			RequestBody:     string(reqBody),
			ResponseStatus:  rec.statusCode,
			ResponseHeaders: respHeaders,
			ResponseBody:    responseBody,
		}
		// Best effort logging — don't fail the request if this errors.
		s.SaveLog(entry)
	})
}

// decodeJWTsInResponse parses a token endpoint JSON response and replaces
// encoded JWT strings with their decoded header+claims for readability.
func decodeJWTsInResponse(body string) string {
	var resp map[string]any
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return body
	}

	for _, key := range []string{"access_token", "id_token"} {
		tok, ok := resp[key].(string)
		if !ok {
			continue
		}
		decoded, err := decodeJWT(tok)
		if err != nil {
			continue
		}
		resp[key] = decoded
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return body
	}
	return string(out)
}

// decodeJWT splits a JWT into its parts and returns the decoded header and claims.
func decodeJWT(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("not a JWT")
	}

	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	claims, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var h, c map[string]any
	if err := json.Unmarshal(header, &h); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(claims, &c); err != nil {
		return nil, err
	}

	return map[string]any{
		"header": h,
		"claims": c,
	}, nil
}

func isOIDCPath(path string) bool {
	for _, p := range OIDCPaths {
		if path == p {
			return true
		}
	}
	return false
}

func headerString(h http.Header) string {
	var b strings.Builder
	for k, vals := range h {
		for _, v := range vals {
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(v)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// ClearLogs deletes all entries from the request_log table.
func (s *SQLiteStore) ClearLogs() error {
	_, err := s.db.Exec("DELETE FROM request_log")
	return err
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
