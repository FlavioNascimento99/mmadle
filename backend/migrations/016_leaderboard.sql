-- 016: public leaderboard opt-in. Defaults to off: only players who
-- explicitly opt in have their username, score and streak appear publicly.
-- The partial index keeps the ranking query on a narrow, opt-in user set.
ALTER TABLE users ADD COLUMN leaderboard_opt_in BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX users_leaderboard_idx ON users (id) WHERE leaderboard_opt_in AND is_active;