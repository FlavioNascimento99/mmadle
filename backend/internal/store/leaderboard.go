package store

import (
	"context"

	"mmadle/backend/internal/domain"
)

// LeaderboardStore is the persistence port for the public leaderboard.
type LeaderboardStore interface {
	// ListLeaderboardGames returns the daily-game history of opted-in, active
	// players for one pool, oldest first, one row per (player, date). Guests
	// and non-opted-in accounts never appear.
	ListLeaderboardGames(ctx context.Context, pool domain.Pool, today string) ([]domain.LeaderboardGame, error)
	// SetLeaderboardOptIn toggles public appearance; unknown ids are ErrNotFound.
	SetLeaderboardOptIn(ctx context.Context, userID int64, optIn bool) error
	// LeaderboardOptIn reports whether the account currently appears on the
	// public board; the value reads fresh from the same store that the
	// leaderboard is derived from, so `my` is never stale.
	LeaderboardOptIn(ctx context.Context, userID int64) (bool, error)
}

// ListLeaderboardGames projects only opted-in, active accounts whose daily
// guesses precede the game date; the (user, mode, pool, game_date,
// guess_index) index covers the filter.
func (p *Postgres) ListLeaderboardGames(ctx context.Context, pool domain.Pool, today string) ([]domain.LeaderboardGame, error) {
	rows, err := p.pool.Query(ctx, `
SELECT u.id, u.username, to_char(g.game_date, 'YYYY-MM-DD'), COUNT(*), bool_or(g.correct)
FROM game_guesses g
JOIN users u ON u.id = g.user_id
WHERE u.leaderboard_opt_in AND u.is_active
  AND g.mode = 'daily' AND g.pool = $1 AND g.game_date <= $2::date
GROUP BY 1, 2, 3
ORDER BY 1 ASC, 3 ASC`, string(pool), today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.LeaderboardGame{}
	for rows.Next() {
		var g domain.LeaderboardGame
		if err := rows.Scan(&g.UserID, &g.Username, &g.Date, &g.Guesses, &g.Won); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// SetLeaderboardOptIn flips the public-leaderboard switch. Opting out
// hides the player immediately: the partial index drops them from the
// next read.
func (p *Postgres) SetLeaderboardOptIn(ctx context.Context, userID int64, optIn bool) error {
	res, err := p.pool.Exec(ctx, `UPDATE users SET leaderboard_opt_in = $2 WHERE id = $1`, userID, optIn)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// LeaderboardOptIn reads the switch directly so handlers and clients mirror
// exactly what ListLeaderboardGames exposes.
func (p *Postgres) LeaderboardOptIn(ctx context.Context, userID int64) (bool, error) {
	var optIn bool
	err := p.pool.QueryRow(ctx, `SELECT leaderboard_opt_in FROM users WHERE id = $1`, userID).Scan(&optIn)
	if err != nil {
		return false, err
	}
	return optIn, nil
}
