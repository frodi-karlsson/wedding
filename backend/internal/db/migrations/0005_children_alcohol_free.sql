-- Preset the children invited in production as alcohol-free, so their RSVP
-- form opens with the box ticked. Data only, forward-only. Matches exact names
-- as stored in production on 2026-11-09. In other environments it is a no-op.
UPDATE guests SET alcohol_free = 1, updated_at = CURRENT_TIMESTAMP
WHERE name IN ('Rán Karlsdóttir', 'Lára Björk Hall', 'Stella Kejmar', 'Viggo Kejmar');
