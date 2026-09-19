package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"mmadle/backend/internal/domain"
)

// Personal-stats integration tests against a real PostgreSQL. Run with:
//   TEST_DATABASE_URL=postgres://... go test -run TestPostgres_UserGames -v ./internal/store/

func TestPostgres_ListUserGames(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	username := "t_stats_" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	u, err := p.CreateUser(ctx, username, "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	empty, err := p.ListUserGames(ctx, u.ID)
	if err != nil {
		t.Fatalf("ListUserGames: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("fresh user games = %+v", empty)
	}

	ids, err := p.GameFighterIDs(ctx, domain.PoolAll)
	if err != nil || len(ids) < 2 {
		t.Fatalf("need seeded fighters: %v", err)
	}
	day1, _ := time.Parse("2006-01-02", "2026-09-16")
	day2, _ := time.Parse("2006-01-02", "2026-09-17")
	if _, err := p.RecordGuess(ctx, u.ID, domain.PoolAll, day1, ids[0], false); err != nil {
		t.Fatal(err)
	}
	if _, err := p.RecordGuess(ctx, u.ID, domain.PoolAll, day1, ids[1], true); err != nil {
		t.Fatal(err)
	}
	menIDs, err := p.GameFighterIDs(ctx, domain.PoolMen)
	if err != nil || len(menIDs) == 0 {
		t.Fatalf("need men fighters: %v", err)
	}
	if _, err := p.RecordGuess(ctx, u.ID, domain.PoolMen, day2, menIDs[0], false); err != nil {
		t.Fatal(err)
	}

	games, err := p.ListUserGames(ctx, u.ID)
	if err != nil {
		t.Fatalf("ListUserGames: %v", err)
	}
	if len(games) != 2 {
		t.Fatalf("games = %+v, want 2 (pool, date) tuples", games)
	}
	if games[0].Date != "2026-09-16" || games[0].Guesses != 2 || !games[0].Won {
		t.Fatalf("day1 = %+v", games[0])
	}
	if games[1].Pool != domain.PoolMen || games[1].Guesses != 1 || games[1].Won {
		t.Fatalf("day2 = %+v", games[1])
	}
}
