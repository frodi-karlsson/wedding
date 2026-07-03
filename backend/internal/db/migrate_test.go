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

// TestApplyMigration_IsAtomic proves apply-SQL + record-version commit or roll
// back together. When recording the version fails, the migration's schema change
// must NOT be left behind. Under the pre-fix two-Exec code the ADD COLUMN would
// persist even though the version was never recorded, so a later full Migrate()
// re-runs 0003 and dies with "duplicate column name". With a single transaction,
// a failed record rolls back the ALTER, leaving a clean slate.
func TestApplyMigration_IsAtomic(t *testing.T) {
	d, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer d.Close()

	if _, err := d.Exec(`CREATE TABLE schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}
	if _, err := d.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY);"); err != nil {
		t.Fatalf("create t: %v", err)
	}

	// Pre-insert version 1 so applyMigration's record INSERT hits a PRIMARY KEY
	// conflict and fails, forcing a rollback of the whole transaction.
	if _, err := d.Exec("INSERT INTO schema_migrations (version) VALUES (1)"); err != nil {
		t.Fatalf("seed version: %v", err)
	}

	err = applyMigration(d, 1, "ALTER TABLE t ADD COLUMN extra TEXT NOT NULL DEFAULT '';")
	if err == nil {
		t.Fatal("applyMigration should have failed on duplicate version record")
	}

	// The ALTER must have been rolled back: column "extra" should not exist.
	var count int
	if err := d.QueryRow("SELECT count(*) FROM pragma_table_info('t') WHERE name='extra'").Scan(&count); err != nil {
		t.Fatalf("pragma_table_info: %v", err)
	}
	if count != 0 {
		t.Errorf("column 'extra' exists after rollback (count=%d); migration was not atomic", count)
	}
}

// TestMigrate_ReRunAfterCrashIsNoOp re-runs Migrate() against an already-migrated
// DB and confirms it is a no-op that does not error.
func TestMigrate_ReRunAfterCrashIsNoOp(t *testing.T) {
	d, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer d.Close()

	if err := Migrate(d); err != nil {
		t.Fatalf("first Migrate() error: %v", err)
	}
	if err := Migrate(d); err != nil {
		t.Fatalf("re-run Migrate() errored: %v", err)
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
