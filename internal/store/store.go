// Package store provides SQLite-backed persistent storage for users,
// authorization codes, and OIDC request/response logs.
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store is the interface for all data access. Designed to be extended with
// SCIM operations later.
type Store interface {
	UserStore
	AuthCodeStore
	LogStore
	Close() error
}

type UserStore interface {
	CreateUser(u *User) error
	GetUser(username string) (*User, error)
	ListUsers() ([]User, error)
	UpdateUser(u *User) error
	DeleteUser(username string) error
}

type AuthCodeStore interface {
	SaveAuthCode(code *AuthCode) error
	ConsumeAuthCode(code string) (*AuthCode, error)
}

type LogStore interface {
	SaveLog(entry *LogEntry) error
	ListLogs(limit int) ([]LogEntry, error)
	ClearLogs() error
}

type User struct {
	ID       int64
	Username string
	Password string
	Email    string
	Name     string
}

type AuthCode struct {
	Code        string
	Username    string
	RedirectURI string
	Scope       string
	Nonce       string
	ClientID    string
	CreatedAt   time.Time
}

type LogEntry struct {
	ID              int64
	Timestamp       time.Time
	Method          string
	Path            string
	Query           string
	RequestHeaders  string
	RequestBody     string
	ResponseStatus  int
	ResponseHeaders string
	ResponseBody    string
}

// SQLiteStore implements Store using modernc.org/sqlite.
type SQLiteStore struct {
	db *sql.DB
}

// Open creates or opens the SQLite database and runs migrations.
func Open(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for better concurrent access.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS request_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL DEFAULT (datetime('now')),
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			query TEXT NOT NULL DEFAULT '',
			request_headers TEXT NOT NULL DEFAULT '',
			request_body TEXT NOT NULL DEFAULT '',
			response_status INTEGER NOT NULL DEFAULT 0,
			response_headers TEXT NOT NULL DEFAULT '',
			response_body TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS auth_codes (
			code TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			redirect_uri TEXT NOT NULL DEFAULT '',
			scope TEXT NOT NULL DEFAULT '',
			nonce TEXT NOT NULL DEFAULT '',
			client_id TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("exec migration: %w", err)
		}
	}

	// Add columns if they don't exist (for existing databases).
	alterMigrations := []string{
		"ALTER TABLE auth_codes ADD COLUMN nonce TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE auth_codes ADD COLUMN client_id TEXT NOT NULL DEFAULT ''",
	}
	for _, m := range alterMigrations {
		s.db.Exec(m) // ignore errors (column already exists)
	}

	return nil
}

// --- User operations ---

func (s *SQLiteStore) CreateUser(u *User) error {
	res, err := s.db.Exec(
		"INSERT INTO users (username, password, email, name) VALUES (?, ?, ?, ?)",
		u.Username, u.Password, u.Email, u.Name,
	)
	if err != nil {
		return err
	}
	u.ID, _ = res.LastInsertId()
	return nil
}

func (s *SQLiteStore) GetUser(username string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT id, username, password, email, name FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Email, &u.Name)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *SQLiteStore) ListUsers() ([]User, error) {
	rows, err := s.db.Query("SELECT id, username, password, email, name FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Password, &u.Email, &u.Name); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *SQLiteStore) UpdateUser(u *User) error {
	_, err := s.db.Exec(
		"UPDATE users SET password = ?, email = ?, name = ? WHERE username = ?",
		u.Password, u.Email, u.Name, u.Username,
	)
	return err
}

func (s *SQLiteStore) DeleteUser(username string) error {
	_, err := s.db.Exec("DELETE FROM users WHERE username = ?", username)
	return err
}

// --- Auth code operations ---

func (s *SQLiteStore) SaveAuthCode(code *AuthCode) error {
	_, err := s.db.Exec(
		"INSERT INTO auth_codes (code, username, redirect_uri, scope, nonce, client_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		code.Code, code.Username, code.RedirectURI, code.Scope, code.Nonce, code.ClientID, code.CreatedAt,
	)
	return err
}

// ConsumeAuthCode retrieves and deletes an authorization code atomically.
func (s *SQLiteStore) ConsumeAuthCode(code string) (*AuthCode, error) {
	ac := &AuthCode{}
	err := s.db.QueryRow(
		"SELECT code, username, redirect_uri, scope, nonce, client_id, created_at FROM auth_codes WHERE code = ?",
		code,
	).Scan(&ac.Code, &ac.Username, &ac.RedirectURI, &ac.Scope, &ac.Nonce, &ac.ClientID, &ac.CreatedAt)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.Exec("DELETE FROM auth_codes WHERE code = ?", code)
	return ac, nil
}

// --- Log operations ---

func (s *SQLiteStore) SaveLog(entry *LogEntry) error {
	_, err := s.db.Exec(
		`INSERT INTO request_log (timestamp, method, path, query, request_headers, request_body, response_status, response_headers, response_body)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.Timestamp, entry.Method, entry.Path, entry.Query,
		entry.RequestHeaders, entry.RequestBody,
		entry.ResponseStatus, entry.ResponseHeaders, entry.ResponseBody,
	)
	return err
}

func (s *SQLiteStore) ListLogs(limit int) ([]LogEntry, error) {
	rows, err := s.db.Query(
		"SELECT id, timestamp, method, path, query, request_headers, request_body, response_status, response_headers, response_body FROM request_log ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(
			&e.ID, &e.Timestamp, &e.Method, &e.Path, &e.Query,
			&e.RequestHeaders, &e.RequestBody,
			&e.ResponseStatus, &e.ResponseHeaders, &e.ResponseBody,
		); err != nil {
			return nil, err
		}
		logs = append(logs, e)
	}
	return logs, rows.Err()
}
