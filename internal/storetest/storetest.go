// Package storetest provides test helpers for creating temporary stores.
package storetest

import (
	"os"
	"testing"

	"oauth-tester/internal/store"
)

// OpenStore creates a temporary SQLite store for use in tests.
// The database file and connection are cleaned up automatically.
func OpenStore(t *testing.T) store.Store {
	t.Helper()
	f, err := os.CreateTemp("", "oauth-tester-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	s, err := store.Open(f.Name())
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
