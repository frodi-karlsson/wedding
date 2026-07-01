package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// Open opens a SQLite database at path with WAL mode and a 5s busy timeout.
// Use ":memory:" for an in-memory database (tests).
func Open(path string) (*sql.DB, error) {
	d, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := d.Exec("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;"); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}
