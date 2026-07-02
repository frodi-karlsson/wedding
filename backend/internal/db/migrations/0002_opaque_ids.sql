-- Switch invite IDs from INTEGER AUTOINCREMENT to opaque TEXT tokens.
-- Breaking schema change. Existing data is dropped (no RSVP data exists yet).
DROP TABLE IF EXISTS guests;
DROP TABLE IF EXISTS invites;

CREATE TABLE invites (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    min_plus INTEGER NOT NULL DEFAULT 0,
    max_plus INTEGER NOT NULL DEFAULT 0,
    submitted INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE guests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    invite_id TEXT NOT NULL,
    name TEXT NOT NULL,
    dietary_preference TEXT NOT NULL DEFAULT '',
    alcohol_free INTEGER NOT NULL DEFAULT 0,
    is_primary INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (invite_id) REFERENCES invites(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_guests_invite_id ON guests(invite_id);
