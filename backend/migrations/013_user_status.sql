-- 013: account status + last login for the admin users view.
ALTER TABLE users ADD COLUMN last_login_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- Backfill: the latest session activity approximates the last login for
-- existing accounts (sessions rotate on login, so this is exact for anyone
-- who logged in since rotation, a lower bound otherwise).
UPDATE users u SET last_login_at = s.last_seen
FROM (SELECT user_id, MAX(created_at) AS last_seen FROM sessions GROUP BY 1) s
WHERE s.user_id = u.id AND u.last_login_at IS NULL;
