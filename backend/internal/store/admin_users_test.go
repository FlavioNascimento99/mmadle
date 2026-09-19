package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"mmadle/backend/internal/domain"
)

// User-status integration tests against a real PostgreSQL (migration 013).
// Skipped without TEST_DATABASE_URL; CI runs them against a migrated DB.

func TestPostgres_UserStatusRoundTrip(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	stamp := strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	u, err := p.CreateUser(ctx, "u_status_"+stamp, "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if !u.IsActive {
		t.Fatal("new accounts must be active")
	}
	if u.LastLoginAt != nil {
		t.Fatal("new accounts must have no last login yet")
	}

	now := time.Now().UTC().Truncate(time.Second)
	if err := p.TouchLastLogin(ctx, u.ID, now); err != nil {
		t.Fatalf("TouchLastLogin: %v", err)
	}
	got, err := p.FindUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("FindUserByID: %v", err)
	}
	if got.LastLoginAt == nil || !got.LastLoginAt.UTC().Equal(now) {
		t.Fatalf("last login = %v, want %v", got.LastLoginAt, now)
	}

	if err := p.SetUserActive(ctx, u.ID, false); err != nil {
		t.Fatalf("SetUserActive: %v", err)
	}
	got, err = p.FindUserByUsername(ctx, u.Username)
	if err != nil {
		t.Fatalf("FindUserByUsername: %v", err)
	}
	if got.IsActive {
		t.Fatal("account must be inactive after deactivation")
	}
	if err := p.SetUserActive(ctx, u.ID, true); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if err := p.SetUserActive(ctx, 0, false); err == nil {
		t.Fatal("unknown id must fail")
	}
}

func TestPostgres_SetUserActiveDropsSessions(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	stamp := strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	u, err := p.CreateUser(ctx, "u_sessdrop_"+stamp, "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	exp := time.Now().Add(time.Hour)
	tok1 := strings.Repeat("a1", 32)
	tok2 := strings.Repeat("b2", 32)
	if err := p.CreateSession(ctx, tok1, u.ID, exp); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := p.CreateSession(ctx, tok2, u.ID, exp); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if _, err := p.FindSessionUser(ctx, tok1, time.Now()); err != nil {
		t.Fatalf("session must resolve before deactivation: %v", err)
	}
	if err := p.SetUserActive(ctx, u.ID, false); err != nil {
		t.Fatalf("SetUserActive: %v", err)
	}
	if _, err := p.FindSessionUser(ctx, tok1, time.Now()); err == nil {
		t.Fatal("sessions must not resolve after deactivation")
	}
	if _, err := p.FindSessionUser(ctx, tok2, time.Now()); err == nil {
		t.Fatal("sessions must not resolve after deactivation")
	}
}

func TestPostgres_ListUsers(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	stamp := strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	mk := func(tag string) User {
		u, err := p.CreateUser(ctx, "u_list_"+stamp+"_"+tag, "hash")
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		return u
	}
	keep := mk("keep")
	off := mk("off")
	if err := p.SetUserActive(ctx, off.ID, false); err != nil {
		t.Fatalf("SetUserActive: %v", err)
	}

	ids, err := p.GameFighterIDs(ctx, domain.PoolAll)
	if err != nil || len(ids) == 0 {
		t.Fatalf("need seeded fighters: %v", err)
	}
	gameDate, _ := time.Parse("2006-01-02", "2026-09-18")
	// keep: one solved game (2 guesses) + one open game (1 miss).
	if _, err := p.RecordGuess(ctx, keep.ID, domain.PoolAll, gameDate, ids[0], false); err != nil {
		t.Fatalf("RecordGuess: %v", err)
	}
	if _, err := p.RecordGuess(ctx, keep.ID, domain.PoolAll, gameDate, ids[1], true); err != nil {
		t.Fatalf("RecordGuess: %v", err)
	}

	page, err := p.ListUsers(ctx, "", 100, 0)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if page.Total < 2 || page.Active < 1 {
		t.Fatalf("totals = %+v", page)
	}
	byName := map[string]AdminUser{}
	for _, a := range page.Users {
		byName[a.Username] = a
	}
	k, ok := byName[keep.Username]
	if !ok {
		t.Fatalf("missing %s in %+v", keep.Username, page.Users)
	}
	if k.Games != 1 || k.GamesWon != 1 {
		t.Fatalf("keep aggregates = %+v, want 1 game won", k)
	}
	if k.CreatedAt == "" || k.LastLoginAt != nil {
		t.Fatalf("keep timestamps = %+v", k)
	}
	o, ok := byName[off.Username]
	if !ok || o.IsActive {
		t.Fatalf("off row = %+v", o)
	}

	filtered, err := p.ListUsers(ctx, "u_list_"+stamp+"_off", 100, 0)
	if err != nil {
		t.Fatalf("ListUsers q: %v", err)
	}
	if filtered.Total != 1 || len(filtered.Users) != 1 || filtered.Users[0].Username != off.Username {
		t.Fatalf("filtered = %+v", filtered)
	}

	first, err := p.ListUsers(ctx, "", 1, 0)
	if err != nil {
		t.Fatalf("ListUsers limit: %v", err)
	}
	if len(first.Users) != 1 {
		t.Fatalf("limit 1 must return one row: %+v", first)
	}
	second, err := p.ListUsers(ctx, "", 1, 1)
	if err != nil {
		t.Fatalf("ListUsers offset: %v", err)
	}
	if len(second.Users) != 1 || second.Users[0].ID == first.Users[0].ID {
		t.Fatalf("offset 1 must return the next row: %+v vs %+v", first, second)
	}
}
