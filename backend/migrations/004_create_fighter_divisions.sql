-- 004: fighter <-> division many-to-many.
-- DECISION (documented in README): a fighter may compete in several divisions
-- over their career. The game displays the fighter's MOST RECENT recorded UFC
-- division, flagged by is_current = TRUE. Exactly one current division per
-- fighter is enforced by a partial unique index.
CREATE TABLE fighter_divisions (
    fighter_id INTEGER NOT NULL REFERENCES fighters(id) ON DELETE CASCADE,
    division_id INTEGER NOT NULL REFERENCES divisions(id) ON DELETE RESTRICT,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (fighter_id, division_id)
);

CREATE UNIQUE INDEX uq_fighter_current_division
    ON fighter_divisions (fighter_id)
    WHERE is_current;
