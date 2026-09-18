// Package store is the persistence boundary: parameterized SQL over pgx,
// mapping rows to domain views. HTTP and game logic never touch SQL.
package store

import (
	"context"
	"fmt"

	"mmadle/backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FighterStore is the persistence port consumed by the HTTP layer.
// A small interface keeps handlers unit-testable with fakes.
type FighterStore interface {
	GameFighterIDs(ctx context.Context, pool domain.Pool) ([]int, error)
	FighterView(ctx context.Context, id int) (domain.FighterView, error)
	SearchFighters(ctx context.Context, pool domain.Pool, q string, limit int) ([]SearchResult, error)
	ListFighters(ctx context.Context, pool domain.Pool) ([]SearchResult, error)
	Ping(ctx context.Context) error
}

// SearchResult is the minimal autocomplete payload: no game attributes.
type SearchResult struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Nickname    *string `json:"nickname"`
	PhotoURL    *string `json:"photo_url"`
	PhotoCredit *string `json:"photo_credit"`
	Division    *string `json:"division"`
	Nationality string  `json:"nationality"`
}

// searchResultSelect projects the SearchResult columns, joining the current
// division (NULL when a fighter has none).
const searchResultSelect = `
SELECT f.id, f.name, f.nickname, f.photo_url, f.photo_credit, d.name AS division, f.nationality
FROM fighters f
LEFT JOIN fighter_divisions fd ON fd.fighter_id = f.id AND fd.is_current
LEFT JOIN divisions d ON d.id = fd.division_id`

// inPool matches the current division's gender against the pool bound to $1;
// the "all" pool matches every fighter, including those without a division.
const inPool = `($1::text = 'all' OR d.gender = $1::text)`

// Postgres is the pgx-backed implementation of FighterStore.
type Postgres struct {
	pool *pgxpool.Pool
}

// NewPostgres opens a connection pool. DATABASE_URL comes from the environment.
func NewPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

// Close releases pool resources.
func (p *Postgres) Close() { p.pool.Close() }

// Ping checks database connectivity.
func (p *Postgres) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }

// GameFighterIDs returns eligible daily-target ids in stable order:
// fighters with a current division in the pool AND at least one recorded
// fight. Incomplete fighters can never become the target.
func (p *Postgres) GameFighterIDs(ctx context.Context, pool domain.Pool) ([]int, error) {
	rows, err := p.pool.Query(ctx, `
SELECT f.id
FROM fighters f
WHERE EXISTS (
    SELECT 1 FROM fighter_divisions fd
    JOIN divisions d ON d.id = fd.division_id
    WHERE fd.fighter_id = f.id AND fd.is_current AND `+inPool+`
)
AND EXISTS (
    SELECT 1 FROM fights fl
    WHERE fl.fighter_a_id = f.id OR fl.fighter_b_id = f.id
)
ORDER BY f.id ASC`, string(pool))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// FighterView loads the full game view for one fighter, deriving the current
// division (is_current flag) and the latest event (max event date over the
// fighter's fights). All values are parameterized; no string concatenation.
func (p *Postgres) FighterView(ctx context.Context, id int) (domain.FighterView, error) {
	var v domain.FighterView
	err := p.pool.QueryRow(ctx, `
SELECT f.id, f.name, f.nickname, f.date_of_birth, f.height_cm, f.nationality,
       f.wins, f.losses, f.draws, f.no_contests, f.stance, f.photo_url, f.photo_credit,
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
WHERE f.id = $1`, id).Scan(
		&v.ID, &v.Name, &v.Nickname, &v.DateOfBirth, &v.HeightCm, &v.Nationality,
		&v.Wins, &v.Losses, &v.Draws, &v.NoContests, &v.Stance, &v.PhotoURL, &v.PhotoCredit,
		&v.Division, &v.LastEvent, &v.LastEventDate,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return v, ErrNotFound
		}
		return v, err
	}
	return v, nil
}

// SearchFighters performs a case-insensitive partial-name search backed by the
// pg_trgm GIN index (see migration 006). The LIKE pattern is built inside SQL
// with a bound parameter, so user input can never alter query structure.
func (p *Postgres) SearchFighters(ctx context.Context, pool domain.Pool, q string, limit int) ([]SearchResult, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	rows, err := p.pool.Query(ctx, searchResultSelect+`
WHERE `+inPool+`
  AND (f.name ILIKE '%' || $2 || '%' OR COALESCE(f.nickname, '') ILIKE '%' || $2 || '%')
ORDER BY f.name ASC
LIMIT $3`, string(pool), q, limit)
	if err != nil {
		return nil, err
	}
	return scanSearchResults(rows)
}

// ListFighters returns the pool's roster alphabetically, in the same minimal
// shape as search so browsing reveals nothing about the daily target.
func (p *Postgres) ListFighters(ctx context.Context, pool domain.Pool) ([]SearchResult, error) {
	rows, err := p.pool.Query(ctx, searchResultSelect+`
WHERE `+inPool+`
ORDER BY f.name ASC`, string(pool))
	if err != nil {
		return nil, err
	}
	return scanSearchResults(rows)
}

func scanSearchResults(rows pgx.Rows) ([]SearchResult, error) {
	defer rows.Close()
	out := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Name, &r.Nickname, &r.PhotoURL, &r.PhotoCredit, &r.Division, &r.Nationality); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
