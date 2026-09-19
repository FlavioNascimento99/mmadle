package store

import (
	"context"
	"time"
)

// ListUsers returns one newest-first slice of the account listing, with
// per-player lifetime game totals. A "game" is one (pool, date) tuple with
// ≥1 daily guess; it is "won" when any of its guesses is correct (same
// semantics as MetricsOverview). The username filter is a case-insensitive
// partial match (CITEXT column); empty matches everything.
func (p *Postgres) ListUsers(ctx context.Context, q string, limit, offset int) (UsersPage, error) {
	out := UsersPage{Users: []AdminUser{}}
	err := p.pool.QueryRow(ctx, `
SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active)
FROM users WHERE ($1 = '' OR username ILIKE '%' || $1 || '%')`, q).Scan(&out.Total, &out.Active)
	if err != nil {
		return UsersPage{}, err
	}
	rows, err := p.pool.Query(ctx, `
SELECT u.id, u.username, u.role, u.is_active, u.created_at, u.last_login_at,
       COALESCE(g.games, 0), COALESCE(g.won, 0)
FROM users u
LEFT JOIN (
    SELECT user_id, COUNT(*) AS games, COUNT(*) FILTER (WHERE won) AS won
    FROM (SELECT user_id, pool, game_date, bool_or(correct) AS won
          FROM game_guesses WHERE mode = 'daily' GROUP BY 1, 2, 3) t
    GROUP BY 1
) g ON g.user_id = u.id
WHERE ($1 = '' OR u.username ILIKE '%' || $1 || '%')
ORDER BY u.id DESC LIMIT $2 OFFSET $3`, q, limit, offset)
	if err != nil {
		return UsersPage{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var a AdminUser
		var created, lastLogin *time.Time
		var lastLoginStr *string
		if err := rows.Scan(&a.ID, &a.Username, &a.Role, &a.IsActive, &created, &lastLogin, &a.Games, &a.GamesWon); err != nil {
			return UsersPage{}, err
		}
		if created != nil {
			a.CreatedAt = created.UTC().Format(time.RFC3339)
		}
		if lastLogin != nil {
			s := lastLogin.UTC().Format(time.RFC3339)
			lastLoginStr = &s
		}
		a.LastLoginAt = lastLoginStr
		out.Users = append(out.Users, a)
	}
	return out, rows.Err()
}
