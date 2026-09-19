-- 011: usernames replace emails as player identity. The username column keeps
-- the CITEXT type so uniqueness stays case-insensitive. Existing rows keep
-- their values and keep working (lookup is an exact normalized match); only
-- new registrations must satisfy the username rules enforced in the backend.
-- display_name is dropped: the username is the single shown identity.
ALTER TABLE users RENAME COLUMN email TO username;
ALTER TABLE users DROP COLUMN display_name;
