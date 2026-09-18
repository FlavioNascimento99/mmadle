package store

import (
	"context"
	"errors"
	"time"

	"mmadle/backend/internal/domain"
)

// Auth persistence boundary: player accounts, opaque sessions, and
// server-side daily-game records. Password hashes and session token hashes
// are opaque strings here; hashing rules live in internal/domain.

var (
	// ErrEmailTaken is returned when registering an address that already exists.
	ErrEmailTaken = errors.New("email already registered")
	// ErrNoSession is returned for unknown or expired session tokens.
	ErrNoSession = errors.New("session not found")
)

// User is a player account. PasswordHash is the argon2id PHC string and must
// never leave the backend (no JSON tags on purpose).
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	DisplayName  *string
	CreatedAt    time.Time
}

// GameGuess is one persisted daily guess. Outcomes are never stored: they are
// re-evaluated from the fighter views on read, so attribute changes and the
// "never trust the client" rule both hold.
type GameGuess struct {
	FighterID  int
	GuessIndex int
	Correct    bool
	CreatedAt  time.Time
}

// AuthStore is the persistence port for accounts and game records.
type AuthStore interface {
	CreateUser(ctx context.Context, email, passwordHash string, displayName *string) (User, error)
	FindUserByEmail(ctx context.Context, email string) (User, error)
	FindUserByID(ctx context.Context, id int64) (User, error)
	CreateSession(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error
	FindSessionUser(ctx context.Context, tokenHash string, now time.Time) (User, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteUserSessions(ctx context.Context, userID int64, exceptTokenHash string) error
	RecordGuess(ctx context.Context, userID int64, pool domain.Pool, gameDate time.Time, fighterID int, correct bool) (GameGuess, error)
	ListGuesses(ctx context.Context, userID int64, pool domain.Pool, gameDate time.Time) ([]GameGuess, error)
}
