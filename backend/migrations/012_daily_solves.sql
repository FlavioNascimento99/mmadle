-- 012: anonymous-aware daily solve counts. game_guesses only records
-- signed-in players (guests play in localStorage), so the public "solved
-- today" counter needs its own table keyed by a stable per-player identity:
-- 'u:<user_id>' for accounts, 'a:<random>' from a long-lived anon cookie
-- for guests. One row per (pool, date, identity), so repeat solves dedupe
-- instead of inflating the count.
CREATE TABLE daily_solves (
    pool TEXT NOT NULL CHECK (pool IN ('all', 'men')),
    game_date DATE NOT NULL,
    identity TEXT NOT NULL CHECK (char_length(identity) BETWEEN 3 AND 90),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (pool, game_date, identity)
);

-- Backfill signed-in history so past solves keep counting.
INSERT INTO daily_solves (pool, game_date, identity)
SELECT pool, game_date, 'u:' || user_id
FROM game_guesses
WHERE mode = 'daily' AND correct
GROUP BY 1, 2, 3
ON CONFLICT DO NOTHING;
