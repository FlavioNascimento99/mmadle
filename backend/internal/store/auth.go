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
	// ErrUsernameTaken is returned when registering a username that already exists.
	ErrUsernameTaken = errors.New("username already registered")
	// ErrNoSession is returned for unknown or expired session tokens.
	ErrNoSession = errors.New("session not found")
)

// User is a player account. PasswordHash is the argon2id PHC string and must
// never leave the backend (no JSON tags on purpose). Role gates the admin
// metrics interface; 'player' is the default. LastLoginAt is NULL until the
// first login (migration 013 backfills from sessions); IsActive false locks
// the account out (sessions are dropped on deactivation).
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         string
	IsActive     bool
	CreatedAt    time.Time
	LastLoginAt  *time.Time
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
	CreateUser(ctx context.Context, username, passwordHash string) (User, error)
	FindUserByUsername(ctx context.Context, username string) (User, error)
	FindUserByID(ctx context.Context, id int64) (User, error)
	CreateSession(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error
	FindSessionUser(ctx context.Context, tokenHash string, now time.Time) (User, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteUserSessions(ctx context.Context, userID int64, exceptTokenHash string) error
	RecordGuess(ctx context.Context, userID int64, pool domain.Pool, gameDate time.Time, fighterID int, correct bool) (GameGuess, error)
	ListGuesses(ctx context.Context, userID int64, pool domain.Pool, gameDate time.Time) ([]GameGuess, error)
	// SetUserRole changes an account's role (promotion/demotion path).
	SetUserRole(ctx context.Context, userID int64, role string) error
	// TouchLastLogin stamps a successful login; best-effort admin info.
	TouchLastLogin(ctx context.Context, userID int64, now time.Time) error
	// SetUserActive (de)activates an account. Deactivating drops every
	// session so the lockout is immediate; unknown ids are ErrNotFound.
	SetUserActive(ctx context.Context, userID int64, active bool) error
}
