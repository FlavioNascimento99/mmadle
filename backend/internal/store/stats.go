package store

import (
	"context"

	"mmadle/backend/internal/domain"
)

// UserGame is one played game: a (pool, date) tuple with its guess count and
// whether any guess solved it.
type UserGame struct {
	Date    string
	Pool    domain.Pool
	Guesses int
	Won     bool
}

// StatsStore is the persistence port for personal statistics.
type StatsStore interface {
	ListUserGames(ctx context.Context, userID int64) ([]UserGame, error)
}

// ListUserGames returns every daily game the player touched, oldest first,
// with guess counts and solved flags. Outcomes stay server-side derived:
// bool_or(correct) over the stored guesses.
func (p *Postgres) ListUserGames(ctx context.Context, userID int64) ([]UserGame, error) {
	rows, err := p.pool.Query(ctx, `
SELECT to_char(game_date, 'YYYY-MM-DD'), pool, COUNT(*), bool_or(correct)
FROM game_guesses
WHERE user_id = $1 AND mode = 'daily'
GROUP BY 1, 2 ORDER BY 1 ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UserGame{}
	for rows.Next() {
		var g UserGame
		var pool string
		if err := rows.Scan(&g.Date, &pool, &g.Guesses, &g.Won); err != nil {
			return nil, err
		}
		g.Pool = domain.Pool(pool)
		out = append(out, g)
	}
	return out, rows.Err()
}
