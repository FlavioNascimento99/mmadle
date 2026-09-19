-- 013: infinity (survival) mode. Rounds are stateful and server-authoritative:
-- lives live in the database so records stay meaningful, and the target is
-- never exposed before a round is solved or dead. Guest rounds carry no
-- user_id and expire after 24h; streaks for guests live in localStorage.
CREATE TABLE infinite_rounds (
    id TEXT PRIMARY KEY CHECK (char_length(id) = 32),
    user_id BIGINT REFERENCES users (id) ON DELETE CASCADE,
    pool TEXT NOT NULL CHECK (pool IN ('all', 'men')),
    target_fighter_id INTEGER NOT NULL REFERENCES fighters (id),
    lives_left INTEGER NOT NULL DEFAULT 5 CHECK (lives_left BETWEEN 0 AND 5),
    guesses INTEGER NOT NULL DEFAULT 0 CHECK (guesses >= 0),
    status TEXT NOT NULL DEFAULT 'playing' CHECK (status IN ('playing', 'solved', 'dead')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT now() + interval '24 hours'
);
CREATE INDEX infinite_rounds_user_idx ON infinite_rounds (user_id);
CREATE INDEX infinite_rounds_expires_idx ON infinite_rounds (expires_at);

-- Best/current survival runs per account and pool. A run is consecutive
-- solved rounds: a solve extends it, a death resets it. Guests have no row;
-- their best lives in the browser.
CREATE TABLE infinite_records (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    pool TEXT NOT NULL CHECK (pool IN ('all', 'men')),
    best_streak INTEGER NOT NULL DEFAULT 0 CHECK (best_streak >= 0),
    current_streak INTEGER NOT NULL DEFAULT 0 CHECK (current_streak >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, pool)
);
