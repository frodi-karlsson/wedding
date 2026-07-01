package db

import (
	"testing"
)

func TestMigrate_CreatesTables(t *testing.T) {
	d, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer d.Close()

	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	var count int
	for _, table := range []string{"schema_migrations", "invites", "guests"} {
		err := d.QueryRow(
			"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&count)
		if err != nil {
			t.Fatalf("query %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("table %s: count=%d, want 1", table, count)
		}
	}
}

func TestMigrate_IsIdempotent(t *testing.T) {
	d, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer d.Close()

	if err := Migrate(d); err != nil {
		t.Fatalf("first Migrate() error: %v", err)
	}
	if err := Migrate(d); err != nil {
		t.Fatalf("second Migrate() error: %v", err)
	}

	var version int
	err = d.QueryRow("SELECT version FROM schema_migrations WHERE version=1").Scan(&version)
	if err != nil {
		t.Fatalf("query migration version: %v", err)
	}
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}
}
