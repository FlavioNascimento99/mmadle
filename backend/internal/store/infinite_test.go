package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"mmadle/backend/internal/domain"
)

// Infinity integration tests against a real PostgreSQL. Run with:
//   TEST_DATABASE_URL=postgres://... go test -run TestPostgres_Infinite -v ./internal/store/

func infiniteUser(t *testing.T, p *Postgres, ctx context.Context) User {
	t.Helper()
	username := "t_inf_" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	u, err := p.CreateUser(ctx, username, "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return u
}

func infiniteTarget(t *testing.T, p *Postgres, ctx context.Context) int {
	t.Helper()
	ids, err := p.GameFighterIDs(ctx, domain.PoolAll)
	if err != nil || len(ids) == 0 {
		t.Fatalf("need seeded fighters: %v", err)
	}
	return ids[0]
}

func TestPostgres_InfiniteRoundLifecycle(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	now := time.Now()
	u := infiniteUser(t, p, ctx)
	target := infiniteTarget(t, p, ctx)

	id, err := domain.NewRoundID()
	if err != nil {
		t.Fatal(err)
	}
	if err := p.CreateRound(ctx, id, &u.ID, domain.PoolAll, target, now.Add(24*time.Hour)); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	round, err := p.FindRound(ctx, id, now)
	if err != nil {
		t.Fatalf("FindRound: %v", err)
	}
	if round.LivesLeft != domain.MaxLives || round.Status != "playing" || round.TargetID != target {
		t.Fatalf("fresh round = %+v", round)
	}
	if round.UserID == nil || *round.UserID != u.ID {
		t.Fatalf("round user = %+v", round.UserID)
	}

	// Two misses cost two lives; the third state stays alive.
	for want := domain.MaxLives - 1; want >= domain.MaxLives-2; want-- {
		round, err = p.ApplyRoundGuess(ctx, id, false, now)
		if err != nil {
			t.Fatalf("ApplyRoundGuess: %v", err)
		}
		if round.LivesLeft != want || round.Status != "playing" {
			t.Fatalf("after miss = %+v, want lives=%d playing", round, want)
		}
	}

	// A solve ends the round with lives untouched; replaying is refused.
	round, err = p.ApplyRoundGuess(ctx, id, true, now)
	if err != nil {
		t.Fatalf("ApplyRoundGuess solve: %v", err)
	}
	if round.Status != "solved" || round.LivesLeft != domain.MaxLives-2 {
		t.Fatalf("solved = %+v", round)
	}
	if _, err := p.ApplyRoundGuess(ctx, id, false, now); err != ErrRoundOver {
		t.Fatalf("replay err = %v, want ErrRoundOver", err)
	}

	// Unknown and expired rounds are indistinguishable.
	if _, err := p.FindRound(ctx, strings.Repeat("a", 32), now); err != ErrNoRound {
		t.Fatalf("unknown err = %v, want ErrNoRound", err)
	}
	oldID, err := domain.NewRoundID()
	if err != nil {
		t.Fatal(err)
	}
	if err := p.CreateRound(ctx, oldID, nil, domain.PoolAll, target, now.Add(-time.Minute)); err != nil {
		t.Fatalf("CreateRound expired: %v", err)
	}
	if _, err := p.FindRound(ctx, oldID, now); err != ErrNoRound {
		t.Fatalf("expired err = %v, want ErrNoRound", err)
	}
	if _, err := p.ApplyRoundGuess(ctx, oldID, true, now); err != ErrNoRound {
		t.Fatalf("expired guess err = %v, want ErrNoRound", err)
	}
}

func TestPostgres_InfiniteDeath(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	now := time.Now()
	target := infiniteTarget(t, p, ctx)

	id, err := domain.NewRoundID()
	if err != nil {
		t.Fatal(err)
	}
	if err := p.CreateRound(ctx, id, nil, domain.PoolMen, target, now.Add(24*time.Hour)); err != nil {
		t.Fatalf("CreateRound guest: %v", err)
	}
	var round InfiniteRound
	for i := 0; i < domain.MaxLives; i++ {
		round, err = p.ApplyRoundGuess(ctx, id, false, now)
		if err != nil {
			t.Fatalf("miss %d: %v", i, err)
		}
	}
	if round.Status != "dead" || round.LivesLeft != 0 {
		t.Fatalf("after %d misses = %+v, want dead at 0", domain.MaxLives, round)
	}
	if round.UserID != nil {
		t.Fatalf("guest round must keep nil user: %+v", round)
	}
}

func TestPostgres_InfiniteRecords(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	u := infiniteUser(t, p, ctx)

	zero, err := p.FetchRecord(ctx, u.ID, domain.PoolAll)
	if err != nil {
		t.Fatalf("FetchRecord: %v", err)
	}
	if zero.BestStreak != 0 || zero.CurrentStreak != 0 {
		t.Fatalf("fresh record = %+v", zero)
	}

	first, err := p.NoteSolve(ctx, u.ID, domain.PoolAll)
	if err != nil {
		t.Fatalf("NoteSolve: %v", err)
	}
	if first.BestStreak != 1 || first.CurrentStreak != 1 {
		t.Fatalf("first solve = %+v", first)
	}
	second, err := p.NoteSolve(ctx, u.ID, domain.PoolAll)
	if err != nil {
		t.Fatalf("NoteSolve: %v", err)
	}
	if second.BestStreak != 2 || second.CurrentStreak != 2 {
		t.Fatalf("second solve = %+v", second)
	}
	if err := p.NoteDeath(ctx, u.ID, domain.PoolAll); err != nil {
		t.Fatalf("NoteDeath: %v", err)
	}
	after, err := p.FetchRecord(ctx, u.ID, domain.PoolAll)
	if err != nil {
		t.Fatalf("FetchRecord: %v", err)
	}
	if after.BestStreak != 2 || after.CurrentStreak != 0 {
		t.Fatalf("after death = %+v, want best kept, current reset", after)
	}
	// Pools are isolated.
	men, err := p.FetchRecord(ctx, u.ID, domain.PoolMen)
	if err != nil {
		t.Fatalf("FetchRecord men: %v", err)
	}
	if men.BestStreak != 0 {
		t.Fatalf("men record = %+v, want zeros", men)
	}
}
