package store

import (
	"context"
	"errors"
	"time"

	"mmadle/backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// CreateUser inserts a player account. Username uniqueness is enforced by the
// CITEXT unique constraint; violations map to ErrUsernameTaken.
func (p *Postgres) CreateUser(ctx context.Context, username, passwordHash string) (User, error) {
	var u User
	err := p.pool.QueryRow(ctx, `
INSERT INTO users (username, password_hash)
VALUES ($1, $2)
RETURNING id, username, password_hash, role, is_active, created_at, last_login_at`,
		username, passwordHash).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.LastLoginAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrUsernameTaken
		}
		return User{}, err
	}
	return u, nil
}

// FindUserByUsername loads an account by username (CITEXT: case-insensitive).
// Missing rows map to ErrNotFound so callers can keep login timing generic.
func (p *Postgres) FindUserByUsername(ctx context.Context, username string) (User, error) {
	var u User
	err := p.pool.QueryRow(ctx, `
SELECT id, username, password_hash, role, is_active, created_at, last_login_at
FROM users WHERE username = $1`, username).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.LastLoginAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return u, nil
}

// FindUserByID loads an account for session resolution.
func (p *Postgres) FindUserByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := p.pool.QueryRow(ctx, `
SELECT id, username, password_hash, role, is_active, created_at, last_login_at
FROM users WHERE id = $1`, id).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.LastLoginAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return u, nil
}

// CreateSession stores the SHA-256 hash of an opaque token. The raw token is
// never persisted and must never be logged.
func (p *Postgres) CreateSession(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error {
	_, err := p.pool.Exec(ctx, `
INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt)
	return err
}

// FindSessionUser resolves a session token hash to its account, enforcing
// expiry against the caller's clock. Unknown or expired tokens map to
// ErrNoSession (never distinguish the two to callers).
func (p *Postgres) FindSessionUser(ctx context.Context, tokenHash string, now time.Time) (User, error) {
	var u User
	err := p.pool.QueryRow(ctx, `
SELECT u.id, u.username, u.password_hash, u.role, u.is_active, u.created_at, u.last_login_at
FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > $2`, tokenHash, now).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.LastLoginAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNoSession
		}
		return User{}, err
	}
	return u, nil
}

// DeleteSession removes one session; unknown tokens are a no-op so logout is
// idempotent.
func (p *Postgres) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

// DeleteUserSessions removes all of a user's sessions except one (session
// rotation on login keeps the fresh token and drops the rest; empty
// exceptTokenHash drops all).
func (p *Postgres) DeleteUserSessions(ctx context.Context, userID int64, exceptTokenHash string) error {
	_, err := p.pool.Exec(ctx, `
DELETE FROM sessions WHERE user_id = $1 AND token_hash <> $2`, userID, exceptTokenHash)
	return err
}

// RecordGuess persists one daily guess. Re-guessing the same fighter returns
// the existing row (idempotent); the outcome itself is re-evaluated by the
// caller and never stored.
func (p *Postgres) RecordGuess(ctx context.Context, userID int64, pool domain.Pool, gameDate time.Time, fighterID int, correct bool) (GameGuess, error) {
	date := gameDay(gameDate)
	var g GameGuess
	err := p.pool.QueryRow(ctx, `
INSERT INTO game_guesses (user_id, mode, pool, game_date, fighter_id, guess_index, correct)
SELECT $1, 'daily', $2, $3, $4,
       COALESCE((SELECT max(guess_index) FROM game_guesses
                 WHERE user_id = $1 AND mode = 'daily' AND pool = $2 AND game_date = $3), 0) + 1,
       $5
ON CONFLICT (user_id, mode, pool, game_date, fighter_id) DO NOTHING
RETURNING fighter_id, guess_index, correct, created_at`, userID, string(pool), date, fighterID, correct).Scan(
		&g.FighterID, &g.GuessIndex, &g.Correct, &g.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Conflict path: return the existing row.
			err = p.pool.QueryRow(ctx, `
SELECT fighter_id, guess_index, correct, created_at FROM game_guesses
WHERE user_id = $1 AND mode = 'daily' AND pool = $2 AND game_date = $3 AND fighter_id = $4`,
				userID, string(pool), date, fighterID).Scan(
				&g.FighterID, &g.GuessIndex, &g.Correct, &g.CreatedAt,
			)
		}
		if err != nil {
			return GameGuess{}, err
		}
	}
	return g, nil
}

// ListGuesses returns a day's guesses in play order for restore-on-login.
func (p *Postgres) ListGuesses(ctx context.Context, userID int64, pool domain.Pool, gameDate time.Time) ([]GameGuess, error) {
	rows, err := p.pool.Query(ctx, `
SELECT fighter_id, guess_index, correct, created_at FROM game_guesses
WHERE user_id = $1 AND mode = 'daily' AND pool = $2 AND game_date = $3
ORDER BY guess_index ASC`, userID, string(pool), gameDay(gameDate))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []GameGuess{}
	for rows.Next() {
		var g GameGuess
		if err := rows.Scan(&g.FighterID, &g.GuessIndex, &g.Correct, &g.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// SetUserRole changes an account's role. Callers pass 'player' or 'admin';
// the CHECK constraint fails closed on anything else.
func (p *Postgres) SetUserRole(ctx context.Context, userID int64, role string) error {
	_, err := p.pool.Exec(ctx, `UPDATE users SET role = $2 WHERE id = $1`, userID, role)
	return err
}

// TouchLastLogin stamps a successful login. Best-effort admin info: callers
// log failures but never fail the login over it.
func (p *Postgres) TouchLastLogin(ctx context.Context, userID int64, now time.Time) error {
	_, err := p.pool.Exec(ctx, `UPDATE users SET last_login_at = $2 WHERE id = $1`, userID, now)
	return err
}

// SetUserActive (de)activates an account. Deactivating drops every session
// so the lockout takes effect immediately, including the admin's own
// presented session (callers guard self-deactivation first). Unknown ids
// are ErrNotFound.
func (p *Postgres) SetUserActive(ctx context.Context, userID int64, active bool) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	res, err := tx.Exec(ctx, `UPDATE users SET is_active = $2 WHERE id = $1`, userID, active)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	if !active {
		if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// gameDay formats a game timestamp as a calendar DATE string for storage.
func gameDay(t time.Time) string { return t.UTC().Format("2006-01-02") }
