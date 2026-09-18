-- 005: events + fights. Last UFC event per fighter is DERIVED from fights.
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    date DATE NOT NULL,
    location TEXT NOT NULL
);

CREATE TABLE fights (
    id SERIAL PRIMARY KEY,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    fighter_a_id INTEGER NOT NULL REFERENCES fighters(id) ON DELETE CASCADE,
    fighter_b_id INTEGER NOT NULL REFERENCES fighters(id) ON DELETE CASCADE,
    winner_id INTEGER REFERENCES fighters(id) ON DELETE SET NULL,
    method TEXT,
    round INTEGER CHECK (round BETWEEN 1 AND 5),
    time TEXT,
    CHECK (fighter_a_id <> fighter_b_id)
);

CREATE INDEX idx_fights_event_id ON fights (event_id);
CREATE INDEX idx_fights_fighter_a ON fights (fighter_a_id);
CREATE INDEX idx_fights_fighter_b ON fights (fighter_b_id);
