package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"mmadle/backend/internal/domain"
)

// Leaderboard integration tests against a real PostgreSQL. Run with:
//   TEST_DATABASE_URL=postgres://... go test -run TestPostgres_Leaderboard -v ./internal/store/

// cleanupLeaderboardUser removes a test account and its guesses so tests are
// self-contained; the leaderboard reads all opted-in players at once, so
// leftovers from an earlier run would corrupt assertions.
func cleanupLeaderboardUser(t *testing.T, p *Postgres, userID int64) {
	t.Helper()
	_, _ = p.pool.Exec(context.Background(), `DELETE FROM game_guesses WHERE user_id = $1`, userID)
	_, _ = p.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
}

func TestPostgres_Leaderboard(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	uniq := strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	a, err := p.CreateUser(ctx, "t_lb_alpha_"+uniq, "hash")
	if err != nil {
		t.Fatalf("CreateUser alpha: %v", err)
	}
	b, err := p.CreateUser(ctx, "t_lb_beta_"+uniq, "hash")
	if err != nil {
		t.Fatalf("CreateUser beta: %v", err)
	}
	t.Cleanup(func() { cleanupLeaderboardUser(t, p, a.ID) })
	t.Cleanup(func() { cleanupLeaderboardUser(t, p, b.ID) })

	// Opt in is off by default: nobody ranks until they opt in.
	empty, err := p.ListLeaderboardGames(ctx, domain.PoolAll, "2026-09-18")
	if err != nil {
		t.Fatalf("ListLeaderboardGames: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("fresh leaderboard = %+v, want none", empty)
	}
	if err := p.SetLeaderboardOptIn(ctx, a.ID, true); err != nil {
		t.Fatalf("SetLeaderboardOptIn: %v", err)
	}
	if err := p.SetLeaderboardOptIn(ctx, b.ID, true); err != nil {
		t.Fatalf("SetLeaderboardOptIn: %v", err)
	}
	if got, err := p.LeaderboardOptIn(ctx, a.ID); err != nil || !got {
		t.Fatalf("LeaderboardOptIn = %v, %v want true", got, err)
	}
	if err := p.SetLeaderboardOptIn(ctx, 999999, true); err != ErrNotFound {
		t.Fatalf("unknown opt-in err=%v want ErrNotFound", err)
	}

	ids, err := p.GameFighterIDs(ctx, domain.PoolAll)
	if err != nil || len(ids) < 2 {
		t.Fatalf("need seeded fighters: %v", err)
	}
	day1, _ := time.Parse("2006-01-02", "2026-09-16")
	day2, _ := time.Parse("2006-01-02", "2026-09-17")
	if _, err := p.RecordGuess(ctx, a.ID, domain.PoolAll, day1, ids[0], false); err != nil {
		t.Fatal(err)
	}
	if _, err := p.RecordGuess(ctx, a.ID, domain.PoolAll, day1, ids[1], true); err != nil {
		t.Fatal(err)
	}
	if _, err := p.RecordGuess(ctx, a.ID, domain.PoolAll, day2, ids[0], true); err != nil {
		t.Fatal(err)
	}
	if _, err := p.RecordGuess(ctx, b.ID, domain.PoolAll, day2, ids[1], false); err != nil {
		t.Fatal(err)
	}

	rows, err := p.ListLeaderboardGames(ctx, domain.PoolAll, "2026-09-18")
	if err != nil {
		t.Fatalf("ListLeaderboardGames: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %+v, want 3 (alpha day1+day2, beta day2)", rows)
	}
	if rows[0].Username != a.Username || rows[0].Guesses != 2 || !rows[0].Won {
		t.Fatalf("row0 = %+v", rows[0])
	}
	if rows[1].Username != a.Username || rows[1].Guesses != 1 || !rows[1].Won {
		t.Fatalf("row1 = %+v", rows[1])
	}
	if rows[2].Username != b.Username || rows[2].Guesses != 1 || rows[2].Won {
		t.Fatalf("row2 = %+v", rows[2])
	}

	// A past clip date excludes guesses that came later.
	clipped, err := p.ListLeaderboardGames(ctx, domain.PoolAll, "2026-09-16")
	if err != nil {
		t.Fatal(err)
	}
	if len(clipped) != 1 {
		t.Fatalf("clipped rows = %+v, want alpha day1 only", clipped)
	}

	// Opting out drops the player from the next read.
	if err := p.SetLeaderboardOptIn(ctx, a.ID, false); err != nil {
		t.Fatal(err)
	}
	rowsOut, err := p.ListLeaderboardGames(ctx, domain.PoolAll, "2026-09-18")
	if err != nil {
		t.Fatal(err)
	}
	if len(rowsOut) != 1 || rowsOut[0].UserID != b.ID {
		t.Fatalf("after opt-out rows = %+v, want beta only", rowsOut)
	}
}

func TestPostgres_LeaderboardFiltersPool(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	uniq := strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	u, err := p.CreateUser(ctx, "t_lb_pool_"+uniq, "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupLeaderboardUser(t, p, u.ID) })
	if err := p.SetLeaderboardOptIn(ctx, u.ID, true); err != nil {
		t.Fatal(err)
	}
	men, err := p.GameFighterIDs(ctx, domain.PoolMen)
	if err != nil || len(men) == 0 {
		t.Fatalf("need men fighters: %v", err)
	}
	day, _ := time.Parse("2006-01-02", "2026-09-16")
	if _, err := p.RecordGuess(ctx, u.ID, domain.PoolAll, day, men[0], false); err != nil {
		t.Fatal(err)
	}
	all, err := p.ListLeaderboardGames(ctx, domain.PoolAll, "2026-09-18")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].UserID != u.ID {
		t.Fatalf("all pool rows = %+v, want the test user only", all)
	}
	only, err := p.ListLeaderboardGames(ctx, domain.PoolMen, "2026-09-18")
	if err != nil {
		t.Fatal(err)
	}
	if len(only) != 0 {
		t.Fatalf("men pool must exclude all-only guesses: %+v", only)
	}
}
