-- 010: admin role for the metrics interface. Admins are bootstrapped from
-- the ADMIN_USERNAMES env allowlist (matched on login/register); the column
-- itself is the source of truth afterwards, so removing an address from the
-- list does not demote anyone (demote with an explicit UPDATE).
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'player'
    CHECK (role IN ('player', 'admin'));
