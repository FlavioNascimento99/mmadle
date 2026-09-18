-- Fighter search and game views.
-- These queries are the sqlc input (see sqlc.yaml). The repository in
-- internal/store uses the same SQL with pgx directly.

-- name: SearchFighters :many
SELECT f.id, f.name, f.nickname, f.photo_url, d.name AS division, f.nationality
FROM fighters f
LEFT JOIN fighter_divisions fd ON fd.fighter_id = f.id AND fd.is_current
LEFT JOIN divisions d ON d.id = fd.division_id
WHERE f.name ILIKE '%' || $1 || '%'
   OR COALESCE(f.nickname, '') ILIKE '%' || $1 || '%'
ORDER BY f.name ASC
LIMIT $2;

-- name: GameFighterIDs :many
SELECT f.id
FROM fighters f
WHERE EXISTS (
    SELECT 1 FROM fighter_divisions fd
    WHERE fd.fighter_id = f.id AND fd.is_current
)
AND EXISTS (
    SELECT 1 FROM fights fl
    WHERE fl.fighter_a_id = f.id OR fl.fighter_b_id = f.id
)
ORDER BY f.id ASC;

-- name: FighterGameView :one
SELECT f.id, f.name, f.nickname, f.date_of_birth, f.height_cm, f.nationality,
       f.wins, f.losses, f.draws, f.no_contests, f.stance, f.photo_url,
       d.name AS division, e.name AS last_event, e.date AS last_event_date
FROM fighters f
JOIN fighter_divisions fd ON fd.fighter_id = f.id AND fd.is_current
JOIN divisions d ON d.id = fd.division_id
JOIN LATERAL (
    SELECT ev.name, ev.date
    FROM fights fl
    JOIN events ev ON ev.id = fl.event_id
    WHERE fl.fighter_a_id = f.id OR fl.fighter_b_id = f.id
    ORDER BY ev.date DESC
    LIMIT 1
) e ON TRUE
WHERE f.id = $1;
