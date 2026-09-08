-- Invite ids are opaque short tokens the app generates (Crockford base32).
-- A raw seed has to spell them out, since SQL cannot generate them.
INSERT INTO invites (id, name, min_plus, max_plus) VALUES ('H7K2Q', 'Frodi & Carla', 0, 2);
INSERT INTO invites (id, name, min_plus, max_plus) VALUES ('X4M9B', 'Test Guest', 1, 3);
-- Primary guests are auto-created by the app on CreateInvite, but for a raw seed
-- we insert them manually.
INSERT INTO guests (invite_id, name, is_primary) VALUES ('H7K2Q', 'Frodi & Carla', 1);
INSERT INTO guests (invite_id, name, is_primary) VALUES ('X4M9B', 'Test Guest', 1);
