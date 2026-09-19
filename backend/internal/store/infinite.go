package store

import (
	"context"
	"errors"
	"time"

	"mmadle/backend/internal/domain"
)

// Infinite persistence boundary: stateful survival rounds plus per-account
// best/current streaks. The target fighter id never leaves the store except
// through the guess handler, which only reveals it for finished rounds.

var (
	// ErrNoRound is returned for unknown, finished-but-mismatched, or expired
	// round ids. Callers never distinguish the cases.
	ErrNoRound = errors.New("round not found")
	// ErrRoundOver is returned when guessing into a solved or dead round.
	ErrRoundOver = errors.New("round already finished")
)

// InfiniteRound is one survival round. UserID is nil for guest rounds.
type InfiniteRound struct {
	ID        string
	UserID    *int64
	Pool      domain.Pool
	TargetID  int
	LivesLeft int
	Guesses   int
	Status    string
	ExpiresAt time.Time
}

// InfiniteRecord is an account's survival run: consecutive solved rounds.
// A solve extends Current (and Best when surpassed); a death resets Current.
type InfiniteRecord struct {
	BestStreak    int `json:"best_streak"`
	CurrentStreak int `json:"current_streak"`
}

// InfiniteStore is the persistence port for infinity mode.
type InfiniteStore interface {
	CreateRound(ctx context.Context, id string, userID *int64, pool domain.Pool, targetID int, expiresAt time.Time) error
	// FindRound loads a round in any status; unknown or expired ids map to
	// ErrNoRound. Handlers answer finished rounds without re-evaluating.
	FindRound(ctx context.Context, id string, now time.Time) (InfiniteRound, error)
	// ApplyRoundGuess folds one evaluated guess into a live round atomically
	// (single UPDATE): correct solves, wrong costs a life, zero kills.
	// Finished rounds map to ErrRoundOver, unknown/expired to ErrNoRound.
	ApplyRoundGuess(ctx context.Context, id string, correct bool, now time.Time) (InfiniteRound, error)
	FetchRecord(ctx context.Context, userID int64, pool domain.Pool) (InfiniteRecord, error)
	NoteSolve(ctx context.Context, userID int64, pool domain.Pool) (InfiniteRecord, error)
	NoteDeath(ctx context.Context, userID int64, pool domain.Pool) error
}
