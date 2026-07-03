-- Switch invite IDs from INTEGER AUTOINCREMENT to opaque TEXT tokens.
--
-- !!! DESTRUCTIVE MIGRATION — DATA LOSS !!!
-- This migration DROPS the invites and guests tables and recreates them from
-- scratch. IT DELETES ALL INVITE AND RSVP DATA. It is only safe because it ran
-- before any real data existed. NEVER run this migration against a populated
-- database (production or any environment holding real RSVPs). If this file has
-- already been applied on a live DB, do NOT re-run it and do NOT reset the
-- schema_migrations record for version 2 — doing so would wipe live data.
-- Any future schema change to these tables must be written as a NON-destructive,
-- forward-only migration (e.g. ALTER TABLE), not a drop-and-recreate.
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
