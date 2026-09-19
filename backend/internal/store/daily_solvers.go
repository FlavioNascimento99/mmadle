package store

import (
	"context"
	"time"

	"mmadle/backend/internal/domain"
)

// CountDailySolvers counts distinct players who solved the daily game for
// one pool + game date. Signed-in solves use their user identity; guest
// solves use an anonymous browser identity (see 012_daily_solves.sql), so
// every solve counts whether the player is logged in or not.
func (p *Postgres) CountDailySolvers(ctx context.Context, pool domain.Pool, gameDate time.Time) (int, error) {
	var n int
	err := p.pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM daily_solves
WHERE pool = $1 AND game_date = $2::date`,
		string(pool), gameDate.UTC().Format("2006-01-02")).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// RecordDailySolve stores one solve idempotently: the (pool, date, identity)
// primary key dedupes repeat solves by the same player.
func (p *Postgres) RecordDailySolve(ctx context.Context, pool domain.Pool, gameDate time.Time, identity string) error {
	_, err := p.pool.Exec(ctx, `
INSERT INTO daily_solves (pool, game_date, identity)
VALUES ($1, $2::date, $3)
ON CONFLICT DO NOTHING`,
		string(pool), gameDate.UTC().Format("2006-01-02"), identity)
	return err
}
