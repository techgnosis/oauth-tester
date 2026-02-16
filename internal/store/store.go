// Package store provides SQLite-backed persistent storage for users,
// authorization codes, and OIDC request/response logs.
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store is the interface for all data access.
type Store interface {
	UserStore
	GroupStore
	AuthCodeStore
	LogStore
	ScimConfigStore
	Close() error
}

type UserStore interface {
	CreateUser(u *User) error
	GetUser(username string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	ListUsers() ([]User, error)
	UpdateUser(u *User) error
	DeleteUser(username string) error
}

type GroupStore interface {
	CreateGroup(name string) error
	ListGroups() ([]Group, error)
	DeleteGroup(name string) error
	AddUserToGroup(username, groupName string) error
	RemoveUserFromGroup(username, groupName string) error
	ListGroupMembers(groupName string) ([]string, error)
	ListUserGroups(username string) ([]string, error)
}

type AuthCodeStore interface {
	SaveAuthCode(code *AuthCode) error
	ConsumeAuthCode(code string) (*AuthCode, error)
}

// LogQuery specifies filters for listing log entries. Zero values are ignored.
type LogQuery struct {
	Source     string // Filter by log source ("oidc" or "scim")
	Method     string // Filter by HTTP method (e.g. "POST")
	Path       string // Filter by request path (e.g. "/token")
	Status     int    // Filter by response status code (e.g. 400)
	Limit      int    // Max results to return (default 50)
	MinutesAgo int    // Only return entries from the last N minutes (0 = no time filter)
}

type LogStore interface {
	SaveLog(entry *LogEntry) error
	ListLogs(limit int) ([]LogEntry, error)
	QueryLogs(q LogQuery) ([]LogEntry, error)
	ClearLogs() error
}

type ScimConfig struct {
	HoustonURL string `json:"houston_url"`
	AuthCode   string `json:"auth_code"`
}

type ScimConfigStore interface {
	GetScimConfig() (*ScimConfig, error)
	SetScimConfig(cfg *ScimConfig) error
}

type User struct {
	ID       int64
	Username string
	Password string
	Email    string
	Name     string
}

type Group struct {
	ID          int64
	Name        string
	MemberCount int
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
	Source          string // "oidc" or "scim"
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

	// SQLite only supports one writer at a time; limit to one connection
	// so that per-connection PRAGMAs (like foreign_keys) stay in effect.
	db.SetMaxOpenConns(1)

	// Enable WAL mode for better concurrent access.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	// Enable foreign keys for cascade deletes.
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
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
			source TEXT NOT NULL DEFAULT 'oidc',
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
		`CREATE TABLE IF NOT EXISTS groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS user_groups (
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			group_id INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
			UNIQUE(user_id, group_id)
		)`,
		`CREATE TABLE IF NOT EXISTS scim_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
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
		"ALTER TABLE request_log ADD COLUMN source TEXT NOT NULL DEFAULT 'oidc'",
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

func (s *SQLiteStore) GetUserByEmail(email string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT id, username, password, email, name FROM users WHERE email = ?",
		email,
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

// --- Group operations ---

func (s *SQLiteStore) CreateGroup(name string) error {
	_, err := s.db.Exec("INSERT INTO groups (name) VALUES (?)", name)
	return err
}

func (s *SQLiteStore) ListGroups() ([]Group, error) {
	rows, err := s.db.Query(`
		SELECT g.id, g.name, COUNT(ug.user_id) as member_count
		FROM groups g
		LEFT JOIN user_groups ug ON g.id = ug.group_id
		GROUP BY g.id, g.name
		ORDER BY g.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.MemberCount); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (s *SQLiteStore) DeleteGroup(name string) error {
	_, err := s.db.Exec("DELETE FROM groups WHERE name = ?", name)
	return err
}

func (s *SQLiteStore) AddUserToGroup(username, groupName string) error {
	_, err := s.db.Exec(`
		INSERT INTO user_groups (user_id, group_id)
		SELECT u.id, g.id FROM users u, groups g
		WHERE u.username = ? AND g.name = ?`,
		username, groupName)
	return err
}

func (s *SQLiteStore) RemoveUserFromGroup(username, groupName string) error {
	_, err := s.db.Exec(`
		DELETE FROM user_groups
		WHERE user_id = (SELECT id FROM users WHERE username = ?)
		AND group_id = (SELECT id FROM groups WHERE name = ?)`,
		username, groupName)
	return err
}

func (s *SQLiteStore) ListGroupMembers(groupName string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT u.username FROM users u
		JOIN user_groups ug ON u.id = ug.user_id
		JOIN groups g ON g.id = ug.group_id
		WHERE g.name = ?
		ORDER BY u.username`, groupName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, err
		}
		members = append(members, username)
	}
	return members, rows.Err()
}

func (s *SQLiteStore) ListUserGroups(username string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT g.name FROM groups g
		JOIN user_groups ug ON g.id = ug.group_id
		JOIN users u ON u.id = ug.user_id
		WHERE u.username = ?
		ORDER BY g.name`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		groups = append(groups, name)
	}
	return groups, rows.Err()
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
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	ac := &AuthCode{}
	err = tx.QueryRow(
		"SELECT code, username, redirect_uri, scope, nonce, client_id, created_at FROM auth_codes WHERE code = ?",
		code,
	).Scan(&ac.Code, &ac.Username, &ac.RedirectURI, &ac.Scope, &ac.Nonce, &ac.ClientID, &ac.CreatedAt)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec("DELETE FROM auth_codes WHERE code = ?", code); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ac, nil
}

// --- Log operations ---

func (s *SQLiteStore) SaveLog(entry *LogEntry) error {
	source := entry.Source
	if source == "" {
		source = "oidc"
	}
	_, err := s.db.Exec(
		`INSERT INTO request_log (timestamp, source, method, path, query, request_headers, request_body, response_status, response_headers, response_body)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.Timestamp, source, entry.Method, entry.Path, entry.Query,
		entry.RequestHeaders, entry.RequestBody,
		entry.ResponseStatus, entry.ResponseHeaders, entry.ResponseBody,
	)
	return err
}

func (s *SQLiteStore) ListLogs(limit int) ([]LogEntry, error) {
	rows, err := s.db.Query(
		"SELECT id, timestamp, source, method, path, query, request_headers, request_body, response_status, response_headers, response_body FROM request_log ORDER BY id DESC LIMIT ?",
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
			&e.ID, &e.Timestamp, &e.Source, &e.Method, &e.Path, &e.Query,
			&e.RequestHeaders, &e.RequestBody,
			&e.ResponseStatus, &e.ResponseHeaders, &e.ResponseBody,
		); err != nil {
			return nil, err
		}
		logs = append(logs, e)
	}
	return logs, rows.Err()
}

// --- SCIM config operations ---

func (s *SQLiteStore) GetScimConfig() (*ScimConfig, error) {
	cfg := &ScimConfig{}
	rows, err := s.db.Query("SELECT key, value FROM scim_config WHERE key IN ('houston_url', 'auth_code')")
	if err != nil {
		return cfg, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return cfg, err
		}
		switch k {
		case "houston_url":
			cfg.HoustonURL = v
		case "auth_code":
			cfg.AuthCode = v
		}
	}
	return cfg, rows.Err()
}

func (s *SQLiteStore) SetScimConfig(cfg *ScimConfig) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, kv := range [][2]string{
		{"houston_url", cfg.HoustonURL},
		{"auth_code", cfg.AuthCode},
	} {
		if _, err := tx.Exec(
			"INSERT INTO scim_config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
			kv[0], kv[1],
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) QueryLogs(q LogQuery) ([]LogEntry, error) {
	query := "SELECT id, timestamp, source, method, path, query, request_headers, request_body, response_status, response_headers, response_body FROM request_log WHERE 1=1"
	var args []any

	if q.Source != "" {
		query += " AND source = ?"
		args = append(args, q.Source)
	}
	if q.Method != "" {
		query += " AND method = ?"
		args = append(args, q.Method)
	}
	if q.Path != "" {
		query += " AND path = ?"
		args = append(args, q.Path)
	}
	if q.Status != 0 {
		query += " AND response_status = ?"
		args = append(args, q.Status)
	}
	if q.MinutesAgo > 0 {
		query += " AND timestamp >= datetime('now', ?)"
		args = append(args, fmt.Sprintf("-%d minutes", q.MinutesAgo))
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	query += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(
			&e.ID, &e.Timestamp, &e.Source, &e.Method, &e.Path, &e.Query,
			&e.RequestHeaders, &e.RequestBody,
			&e.ResponseStatus, &e.ResponseHeaders, &e.ResponseBody,
		); err != nil {
			return nil, err
		}
		logs = append(logs, e)
	}
	return logs, rows.Err()
}
