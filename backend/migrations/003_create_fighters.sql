-- 003: fighters (canonical data only; age/last_event are derived, never stored)
CREATE TABLE fighters (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    nickname TEXT,
    date_of_birth DATE NOT NULL,
    height_cm INTEGER NOT NULL CHECK (height_cm BETWEEN 140 AND 230),
    nationality TEXT NOT NULL,
    wins INTEGER NOT NULL DEFAULT 0 CHECK (wins >= 0),
    losses INTEGER NOT NULL DEFAULT 0 CHECK (losses >= 0),
    draws INTEGER NOT NULL DEFAULT 0 CHECK (draws >= 0),
    no_contests INTEGER NOT NULL DEFAULT 0 CHECK (no_contests >= 0),
    stance TEXT,
    photo_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
