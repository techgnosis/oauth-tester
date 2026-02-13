package store

import (
	"bytes"
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

		entry := &LogEntry{
			Timestamp:       time.Now().UTC(),
			Method:          r.Method,
			Path:            r.URL.Path,
			Query:           r.URL.RawQuery,
			RequestHeaders:  reqHeaders,
			RequestBody:     string(reqBody),
			ResponseStatus:  rec.statusCode,
			ResponseHeaders: respHeaders,
			ResponseBody:    rec.body.String(),
		}
		// Best effort logging — don't fail the request if this errors.
		s.SaveLog(entry)
	})
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
