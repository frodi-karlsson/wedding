-- Add free-text message field to invites (per-submission note from guest).
ALTER TABLE invites ADD COLUMN message TEXT NOT NULL DEFAULT '';
