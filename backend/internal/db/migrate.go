package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// ErrParseMigrationVersion is returned when a migration file name lacks an underscore separator.
var ErrParseMigrationVersion = errors.New("parse migration version")

// Migrate reads embedded SQL migration files and applies any that haven't
// been applied yet, in order. It is idempotent.
func Migrate(d *sql.DB) error {
	if _, err := d.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		prefix, _, ok := strings.Cut(name, "_")
		if !ok {
			return fmt.Errorf("%w: no underscore separator in %s", ErrParseMigrationVersion, name)
		}
		var version int
		_, err := fmt.Sscanf(prefix, "%d", &version)
		if err != nil {
			return fmt.Errorf("parse migration version from %s: %w", name, err)
		}

		var applied int
		err = d.QueryRowContext(context.Background(), "SELECT count(*) FROM schema_migrations WHERE version=?", version).Scan(&applied)
		if err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if applied > 0 {
			continue
		}

		content, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		// Apply the migration SQL and record its version atomically. If a crash
		// interrupts this, the transaction is rolled back so the migration is
		// re-applied cleanly on the next run rather than leaving a half-applied
		// state that fails (e.g. "duplicate column name").
		if err := applyMigration(d, version, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

// applyMigration runs the migration SQL and records its version in a single
// transaction, committing both or rolling back both.
func applyMigration(d *sql.DB, version int, content string) error {
	tx, err := d.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(context.Background(), content); err != nil {
		return err
	}
	if _, err := tx.ExecContext(context.Background(), "INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
		return err
	}
	return tx.Commit()
}
