package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// MetricsOverview aggregates signups and daily-game activity over the last
// days calendar days (game_date >= today - days + 1). All queries are
// read-only; empty tables yield zeros and empty (non-nil) series.
func (p *Postgres) MetricsOverview(ctx context.Context, days int) (MetricsOverview, error) {
	if days < 1 {
		days = 1
	}
	if days > 90 {
		days = 90
	}
	since := time.Now().UTC().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	out := MetricsOverview{
		Days: days, Since: since,
		SignupsByDay: []DayCount{}, PlayersByDay: []DayCount{}, GuessesByDay: []DayCount{},
		ByPool: []PoolSplit{}, TopFighters: []TopFighter{},
	}

	if err := p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&out.SignupsTotal); err != nil {
		return MetricsOverview{}, err
	}
	signups, err := daySeries(ctx, p, `SELECT to_char(created_at, 'YYYY-MM-DD'), COUNT(*)
FROM users WHERE created_at >= $1::date GROUP BY 1 ORDER BY 1`, since)
	if err != nil {
		return MetricsOverview{}, err
	}
	out.SignupsByDay = signups

	activity, err := p.pool.Query(ctx, `SELECT to_char(game_date, 'YYYY-MM-DD'),
       COUNT(*), COUNT(DISTINCT user_id)
FROM game_guesses WHERE mode = 'daily' AND game_date >= $1::date
GROUP BY 1 ORDER BY 1`, since)
	if err != nil {
		return MetricsOverview{}, err
	}
	guessesByDay, playersByDay, err := scanTwoSeries(activity)
	if err != nil {
		return MetricsOverview{}, err
	}
	out.GuessesByDay, out.PlayersByDay = guessesByDay, playersByDay

	var avgToWin *float64
	err = p.pool.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE won), AVG(n) FILTER (WHERE won)
FROM (SELECT user_id, pool, game_date, bool_or(correct) AS won, COUNT(*) AS n
      FROM game_guesses WHERE mode = 'daily' AND game_date >= $1::date
      GROUP BY 1, 2, 3) g`, since).Scan(&out.GamesTotal, &out.GamesWon, &avgToWin)
	if err != nil {
		return MetricsOverview{}, err
	}
	if out.GamesTotal > 0 {
		out.WinRate = float64(out.GamesWon) / float64(out.GamesTotal)
	}
	if avgToWin != nil {
		out.AvgGuessesToWin = *avgToWin
	}

	rows, err := p.pool.Query(ctx, `SELECT pool, COUNT(*), COUNT(*) FILTER (WHERE won), SUM(n)
FROM (SELECT user_id, pool, game_date, bool_or(correct) AS won, COUNT(*) AS n
      FROM game_guesses WHERE mode = 'daily' AND game_date >= $1::date
      GROUP BY 1, 2, 3) g GROUP BY 1 ORDER BY 1`, since)
	if err != nil {
		return MetricsOverview{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var s PoolSplit
		var guesses int64
		if err := rows.Scan(&s.Pool, &s.Games, &s.Won, &guesses); err != nil {
			return MetricsOverview{}, err
		}
		s.Guesses = int(guesses)
		out.ByPool = append(out.ByPool, s)
	}
	if err := rows.Err(); err != nil {
		return MetricsOverview{}, err
	}

	topRows, err := p.pool.Query(ctx, `SELECT f.id, f.name, COUNT(*)
FROM game_guesses g JOIN fighters f ON f.id = g.fighter_id
WHERE g.mode = 'daily' AND g.game_date >= $1::date
GROUP BY 1, 2 ORDER BY 3 DESC, 2 ASC LIMIT 10`, since)
	if err != nil {
		return MetricsOverview{}, err
	}
	defer topRows.Close()
	for topRows.Next() {
		var t TopFighter
		if err := topRows.Scan(&t.FighterID, &t.Name, &t.Guesses); err != nil {
			return MetricsOverview{}, err
		}
		out.TopFighters = append(out.TopFighters, t)
	}
	return out, topRows.Err()
}

// daySeries scans a (date, count) result into a series.
func daySeries(ctx context.Context, p *Postgres, sql, since string) ([]DayCount, error) {
	rows, err := p.pool.Query(ctx, sql, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DayCount{}
	for rows.Next() {
		var d DayCount
		if err := rows.Scan(&d.Date, &d.Count); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// scanTwoSeries splits a (date, count_a, count_b) result into two series.
func scanTwoSeries(rows pgx.Rows) ([]DayCount, []DayCount, error) {
	defer rows.Close()
	a, b := []DayCount{}, []DayCount{}
	for rows.Next() {
		var date string
		var ca, cb int
		if err := rows.Scan(&date, &ca, &cb); err != nil {
			return nil, nil, err
		}
		a = append(a, DayCount{Date: date, Count: ca})
		b = append(b, DayCount{Date: date, Count: cb})
	}
	return a, b, rows.Err()
}
