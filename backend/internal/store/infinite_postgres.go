package store

import (
	"context"
	"errors"
	"time"

	"mmadle/backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

// roundColumns projects infinite_rounds rows in scan order.
const roundColumns = `id, user_id, pool, target_fighter_id, lives_left, guesses, status, expires_at`

func scanRound(row pgx.Row) (InfiniteRound, error) {
	var r InfiniteRound
	var pool string
	err := row.Scan(&r.ID, &r.UserID, &pool, &r.TargetID, &r.LivesLeft, &r.Guesses, &r.Status, &r.ExpiresAt)
	if err != nil {
		return InfiniteRound{}, err
	}
	r.Pool = domain.Pool(pool)
	return r, nil
}

// CreateRound opens a round with full lives. The id is opaque (never the
// target); userID is nil for guest rounds.
func (p *Postgres) CreateRound(ctx context.Context, id string, userID *int64, pool domain.Pool, targetID int, expiresAt time.Time) error {
	_, err := p.pool.Exec(ctx, `
INSERT INTO infinite_rounds (id, user_id, pool, target_fighter_id, expires_at)
VALUES ($1, $2, $3, $4, $5)`, id, userID, string(pool), targetID, expiresAt)
	return err
}

// FindRound loads a round in any status for finished-round answers;
// unknown or expired ids map to ErrNoRound.
func (p *Postgres) FindRound(ctx context.Context, id string, now time.Time) (InfiniteRound, error) {
	r, err := scanRound(p.pool.QueryRow(ctx, `
SELECT `+roundColumns+` FROM infinite_rounds WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InfiniteRound{}, ErrNoRound
		}
		return InfiniteRound{}, err
	}
	if !r.ExpiresAt.After(now) {
		return InfiniteRound{}, ErrNoRound
	}
	return r, nil
}

// ApplyRoundGuess folds one evaluated guess into a live round in a single
// UPDATE, so concurrent guesses cannot overspend lives. Finished rounds map
// to ErrRoundOver, unknown/expired ones to ErrNoRound.
func (p *Postgres) ApplyRoundGuess(ctx context.Context, id string, correct bool, now time.Time) (InfiniteRound, error) {
	r, err := scanRound(p.pool.QueryRow(ctx, `
UPDATE infinite_rounds SET
    guesses = guesses + 1,
    lives_left = CASE WHEN $2 THEN lives_left ELSE lives_left - 1 END,
    status = CASE WHEN $2 THEN 'solved'
                  WHEN lives_left - 1 <= 0 THEN 'dead'
                  ELSE 'playing' END
WHERE id = $1 AND status = 'playing' AND expires_at > $3
RETURNING `+roundColumns, id, correct, now))
	if err == nil {
		return r, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return InfiniteRound{}, err
	}
	var status string
	var expires time.Time
	if err := p.pool.QueryRow(ctx, `SELECT status, expires_at FROM infinite_rounds WHERE id = $1`, id).Scan(&status, &expires); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InfiniteRound{}, ErrNoRound
		}
		return InfiniteRound{}, err
	}
	if !expires.After(now) {
		return InfiniteRound{}, ErrNoRound
	}
	return InfiniteRound{}, ErrRoundOver
}

// FetchRecord reads an account's run; accounts without history get zeros
// (never null-shaped errors for the UI).
func (p *Postgres) FetchRecord(ctx context.Context, userID int64, pool domain.Pool) (InfiniteRecord, error) {
	var rec InfiniteRecord
	err := p.pool.QueryRow(ctx, `
SELECT best_streak, current_streak FROM infinite_records
WHERE user_id = $1 AND pool = $2`, userID, string(pool)).Scan(&rec.BestStreak, &rec.CurrentStreak)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InfiniteRecord{}, nil
		}
		return InfiniteRecord{}, err
	}
	return rec, nil
}

// NoteSolve extends the run: current + 1, best raised when surpassed.
func (p *Postgres) NoteSolve(ctx context.Context, userID int64, pool domain.Pool) (InfiniteRecord, error) {
	var rec InfiniteRecord
	err := p.pool.QueryRow(ctx, `
INSERT INTO infinite_records (user_id, pool, best_streak, current_streak)
VALUES ($1, $2, 1, 1)
ON CONFLICT (user_id, pool) DO UPDATE SET
    current_streak = infinite_records.current_streak + 1,
    best_streak = GREATEST(infinite_records.best_streak, infinite_records.current_streak + 1),
    updated_at = now()
RETURNING best_streak, current_streak`, userID, string(pool)).Scan(&rec.BestStreak, &rec.CurrentStreak)
	if err != nil {
		return InfiniteRecord{}, err
	}
	return rec, nil
}

// NoteDeath resets the current run; the best survives.
func (p *Postgres) NoteDeath(ctx context.Context, userID int64, pool domain.Pool) error {
	_, err := p.pool.Exec(ctx, `
INSERT INTO infinite_records (user_id, pool, best_streak, current_streak)
VALUES ($1, $2, 0, 0)
ON CONFLICT (user_id, pool) DO UPDATE SET current_streak = 0, updated_at = now()`,
		userID, string(pool))
	return err
}
