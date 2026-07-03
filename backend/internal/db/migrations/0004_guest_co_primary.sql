-- Add co_primary flag to guests. A co-primary is a named peer invitee (part of a
-- "group" invite that has no single primary). Forward-only, non-destructive.
ALTER TABLE guests ADD COLUMN co_primary INTEGER NOT NULL DEFAULT 0;
