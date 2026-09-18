-- 009: player accounts and server-side game records (see issue #1).
-- Auth lives in our own schema (not Supabase Auth) to keep mmadle players
-- isolated from czar's users in the shared Supabase project.
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email CITEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name TEXT CHECK (display_name IS NULL OR (char_length(display_name) BETWEEN 1 AND 30)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Sessions hold only the SHA-256 hash of the opaque cookie token; the raw
-- token is never stored and never logged.
CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY CHECK (char_length(token_hash) = 64),
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

-- Server-side daily-game records. Each signed-in guess is re-evaluated by the
-- server, so client-supplied outcomes are never trusted. mode/round_id are
-- forward-compatible with Infinite Mode (#3); v1 only writes mode='daily'.
CREATE TABLE game_guesses (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    mode TEXT NOT NULL DEFAULT 'daily' CHECK (mode IN ('daily', 'infinite')),
    pool TEXT NOT NULL CHECK (pool IN ('all', 'men')),
    game_date DATE CHECK (game_date IS NOT NULL OR mode = 'infinite'),
    round_id TEXT CHECK ((mode = 'daily' AND round_id IS NULL) OR mode = 'infinite'),
    fighter_id INTEGER NOT NULL REFERENCES fighters (id),
    guess_index INTEGER NOT NULL CHECK (guess_index > 0 AND guess_index <= 100),
    correct BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, mode, pool, game_date, fighter_id)
);
CREATE INDEX game_guesses_user_game_idx ON game_guesses (user_id, mode, pool, game_date, guess_index);
