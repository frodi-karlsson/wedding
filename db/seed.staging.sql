INSERT INTO invites (name, min_plus, max_plus) VALUES ('Frodi & Carla', 0, 2);
INSERT INTO invites (name, min_plus, max_plus) VALUES ('Test Guest', 1, 3);
-- Primary guests are auto-created by the app on CreateInvite, but for a raw seed
-- we insert them manually.
INSERT INTO guests (invite_id, name, is_primary) VALUES (1, 'Frodi & Carla', 1);
INSERT INTO guests (invite_id, name, is_primary) VALUES (2, 'Test Guest', 1);
