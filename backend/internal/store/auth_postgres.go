package store

import (
	"context"
	"errors"
	"time"

	"mmadle/backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// CreateUser inserts a player account. Email uniqueness is enforced by the
// CITEXT unique constraint; violations map to ErrEmailTaken.
func (p *Postgres) CreateUser(ctx context.Context, email, passwordHash string, displayName *string) (User, error) {
	var u User
	err := p.pool.QueryRow(ctx, `
INSERT INTO users (email, password_hash, display_name)
VALUES ($1, $2, NULLIF($3, ''))
RETURNING id, email, password_hash, display_name, created_at`,
		email, passwordHash, nullableString(displayName)).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	return u, nil
}

func nullableString(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

// FindUserByEmail loads an account by address (CITEXT: case-insensitive).
// Missing rows map to ErrNotFound so callers can keep login timing generic.
func (p *Postgres) FindUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := p.pool.QueryRow(ctx, `
SELECT id, email, password_hash, display_name, created_at
FROM users WHERE email = $1`, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt,
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
SELECT id, email, password_hash, display_name, created_at
FROM users WHERE id = $1`, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt,
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
SELECT u.id, u.email, u.password_hash, u.display_name, u.created_at
FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > $2`, tokenHash, now).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt,
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

// gameDay formats a game timestamp as a calendar DATE string for storage.
func gameDay(t time.Time) string { return t.UTC().Format("2006-01-02") }
