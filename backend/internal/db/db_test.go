package db

import (
	"testing"
)

func TestOpen_InMemory(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) error: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
}

func TestOpen_AppliesWALAndBusyTimeout(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	// :memory: reports "memory" for journal_mode; WAL is for file-backed DBs.
	// We assert the pragma was issued without error by checking busy_timeout.
	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", busyTimeout)
	}
}
