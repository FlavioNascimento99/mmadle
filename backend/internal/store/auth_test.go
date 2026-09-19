package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"mmadle/backend/internal/domain"
)

// Auth integration tests against a real PostgreSQL. Run with:
//   TEST_DATABASE_URL=postgres://... go test -run TestPostgres_Auth -v ./internal/store/
// Users get a unique username per test (CITEXT unique would collide otherwise).

func authTestUser(t *testing.T, p *Postgres, ctx context.Context, tag string) User {
	t.Helper()
	// Usernames allow letters/digits/underscore: fold the timestamp's dots.
	stamp := strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	username := "t_" + tag + "_" + stamp
	u, err := p.CreateUser(ctx, username, "argon2id-test-hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.ID == 0 || u.Username != username {
		t.Fatalf("CreateUser returned %+v for %s", u, username)
	}
	return u
}

func TestPostgres_AuthUsers(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()

	u := authTestUser(t, p, ctx, "crud")

	// Duplicate username (case-insensitive via CITEXT) is a conflict.
	if _, err := p.CreateUser(ctx, u.Username, "other-hash"); err != ErrUsernameTaken {
		t.Fatalf("duplicate username err = %v, want ErrUsernameTaken", err)
	}
	// Case-insensitive duplicate too.
	upper := strings.ToUpper(u.Username)
	if _, err := p.CreateUser(ctx, upper, "other-hash"); err != ErrUsernameTaken {
		t.Fatalf("case-insensitive duplicate err = %v, want ErrUsernameTaken", err)
	}

	byName, err := p.FindUserByUsername(ctx, upper)
	if err != nil {
		t.Fatalf("FindUserByUsername (upper): %v", err)
	}
	if byName.ID != u.ID {
		t.Fatalf("FindUserByUsername id=%d want %d", byName.ID, u.ID)
	}
	byID, err := p.FindUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("FindUserByID: %v", err)
	}
	if byID.Username != u.Username {
		t.Fatalf("FindUserByID username=%q want %q", byID.Username, u.Username)
	}
	if _, err := p.FindUserByUsername(ctx, "nobody_here_ever"); err != ErrNotFound {
		t.Fatalf("missing username err = %v, want ErrNotFound", err)
	}
}

func TestPostgres_AuthSessions(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	u := authTestUser(t, p, ctx, "sess")
	now := time.Now()
	uniq := now.Format("150405.000000000")

	// Unknown token resolves to ErrNoSession (never distinguish expired).
	if _, err := p.FindSessionUser(ctx, strings.Repeat("f", 64), now); err != ErrNoSession {
		t.Fatalf("unknown session err = %v, want ErrNoSession", err)
	}
	hash := domain.HashSessionToken("test-token-1-" + uniq)
	if err := p.CreateSession(ctx, hash, u.ID, now.Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	found, err := p.FindSessionUser(ctx, hash, now)
	if err != nil {
		t.Fatalf("FindSessionUser: %v", err)
	}
	if found.ID != u.ID {
		t.Fatalf("session user id=%d want %d", found.ID, u.ID)
	}
	// Expired sessions are rejected like unknown ones.
	if _, err := p.FindSessionUser(ctx, hash, now.Add(2*time.Hour)); err != ErrNoSession {
		t.Fatalf("expired session err = %v, want ErrNoSession", err)
	}
	// Logout deletes; second delete is a no-op.
	if err := p.DeleteSession(ctx, hash); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := p.FindSessionUser(ctx, hash, now); err != ErrNoSession {
		t.Fatalf("deleted session err = %v, want ErrNoSession", err)
	}
	if err := p.DeleteSession(ctx, hash); err != nil {
		t.Fatalf("idempotent delete: %v", err)
	}

	// Rotation keeps the fresh token and drops the rest.
	old1 := domain.HashSessionToken("old-1-" + uniq)
	old2 := domain.HashSessionToken("old-2-" + uniq)
	fresh := domain.HashSessionToken("fresh-" + uniq)
	for _, h := range []string{old1, old2, fresh} {
		if err := p.CreateSession(ctx, h, u.ID, now.Add(time.Hour)); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
	}
	if err := p.DeleteUserSessions(ctx, u.ID, fresh); err != nil {
		t.Fatalf("DeleteUserSessions: %v", err)
	}
	if _, err := p.FindSessionUser(ctx, fresh, now); err != nil {
		t.Fatalf("fresh session must survive rotation: %v", err)
	}
	for _, h := range []string{old1, old2} {
		if _, err := p.FindSessionUser(ctx, h, now); err != ErrNoSession {
			t.Fatalf("rotated-out session must be gone: %v", err)
		}
	}
}

func TestPostgres_AuthGameGuesses(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	u := authTestUser(t, p, ctx, "guess")
	gameDate, _ := time.Parse("2006-01-02", "2026-09-18")

	ids, err := p.GameFighterIDs(ctx, domain.PoolAll)
	if err != nil || len(ids) < 2 {
		t.Fatalf("need seeded fighters: %v %d", err, len(ids))
	}

	// Empty history restores as empty (not null-shaped errors).
	got, err := p.ListGuesses(ctx, u.ID, domain.PoolAll, gameDate)
	if err != nil {
		t.Fatalf("ListGuesses: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("fresh user has %d guesses", len(got))
	}

	first, err := p.RecordGuess(ctx, u.ID, domain.PoolAll, gameDate, ids[0], false)
	if err != nil {
		t.Fatalf("RecordGuess: %v", err)
	}
	if first.GuessIndex != 1 || first.FighterID != ids[0] || first.Correct {
		t.Fatalf("first guess = %+v", first)
	}
	second, err := p.RecordGuess(ctx, u.ID, domain.PoolAll, gameDate, ids[1], true)
	if err != nil {
		t.Fatalf("RecordGuess: %v", err)
	}
	if second.GuessIndex != 2 || !second.Correct {
		t.Fatalf("second guess = %+v", second)
	}

	// Re-guessing the same fighter is idempotent: same index, no new row.
	dup, err := p.RecordGuess(ctx, u.ID, domain.PoolAll, gameDate, ids[0], false)
	if err != nil {
		t.Fatalf("RecordGuess dup: %v", err)
	}
	if dup.GuessIndex != 1 {
		t.Fatalf("dup guess index=%d want 1", dup.GuessIndex)
	}

	got, err = p.ListGuesses(ctx, u.ID, domain.PoolAll, gameDate)
	if err != nil {
		t.Fatalf("ListGuesses: %v", err)
	}
	if len(got) != 2 || got[0].GuessIndex != 1 || got[1].GuessIndex != 2 {
		t.Fatalf("ordered history = %+v", got)
	}

	// Pools are isolated: the same date in the men pool starts empty.
	men, err := p.ListGuesses(ctx, u.ID, domain.PoolMen, gameDate)
	if err != nil {
		t.Fatalf("ListGuesses men: %v", err)
	}
	if len(men) != 0 {
		t.Fatalf("pools must be isolated: %+v", men)
	}
}
