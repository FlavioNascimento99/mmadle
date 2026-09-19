package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"mmadle/backend/internal/domain"
)

// Admin metrics integration tests against a real PostgreSQL. Run with:
//   TEST_DATABASE_URL=postgres://... go test -run TestPostgres_Admin -v ./internal/store/

func TestPostgres_SetUserRole(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	username := "m_role_" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	u, err := p.CreateUser(ctx, username, "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.Role != "player" {
		t.Fatalf("new users must default to player, got %q", u.Role)
	}
	if err := p.SetUserRole(ctx, u.ID, "admin"); err != nil {
		t.Fatalf("SetUserRole: %v", err)
	}
	got, err := p.FindUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("FindUserByID: %v", err)
	}
	if got.Role != "admin" {
		t.Fatalf("role = %q, want admin", got.Role)
	}
	byName, err := p.FindUserByUsername(ctx, username)
	if err != nil {
		t.Fatalf("FindUserByUsername: %v", err)
	}
	if byName.Role != "admin" {
		t.Fatalf("session-visible role = %q, want admin", byName.Role)
	}
}

func TestPostgres_MetricsOverview(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	stamp := strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	n := 0
	mkUser := func() User {
		n++
		username := "m_metrics_" + stamp + "_" + string(rune('a'+n))
		u, err := p.CreateUser(ctx, username, "hash")
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		return u
	}
	u1 := mkUser()
	u2 := mkUser()
	gameDate, _ := time.Parse("2006-01-02", "2026-09-18")

	ids, err := p.GameFighterIDs(ctx, domain.PoolAll)
	if err != nil || len(ids) < 3 {
		t.Fatalf("need seeded fighters: %v %d", err, len(ids))
	}

	// u1 plays all-pool: two misses then a solve (won in 3).
	for i, id := range []int{ids[0], ids[1], ids[2]} {
		if _, err := p.RecordGuess(ctx, u1.ID, domain.PoolAll, gameDate, id, i == 2); err != nil {
			t.Fatalf("RecordGuess: %v", err)
		}
	}
	// u2 plays men-pool: one miss (unfinished game).
	menIDs, err := p.GameFighterIDs(ctx, domain.PoolMen)
	if err != nil || len(menIDs) == 0 {
		t.Fatalf("need men fighters: %v", err)
	}
	if _, err := p.RecordGuess(ctx, u2.ID, domain.PoolMen, gameDate, menIDs[0], false); err != nil {
		t.Fatalf("RecordGuess: %v", err)
	}

	out, err := p.MetricsOverview(ctx, 30)
	if err != nil {
		t.Fatalf("MetricsOverview: %v", err)
	}
	if out.Days != 30 || out.GamesTotal < 2 {
		t.Fatalf("overview games = %+v", out)
	}
	if out.GamesWon < 1 {
		t.Fatalf("at least one won game expected: %+v", out)
	}
	if out.WinRate <= 0 || out.WinRate > 1 {
		t.Fatalf("win rate out of range: %v", out.WinRate)
	}
	if out.AvgGuessesToWin < 1 {
		t.Fatalf("avg guesses to win must be positive: %v", out.AvgGuessesToWin)
	}
	pools := map[string]PoolSplit{}
	for _, s := range out.ByPool {
		pools[s.Pool] = s
	}
	if pools["all"].Games < 1 || pools["men"].Games < 1 {
		t.Fatalf("pool split = %+v", out.ByPool)
	}
	if len(out.TopFighters) == 0 || out.TopFighters[0].Guesses < 1 {
		t.Fatalf("top fighters = %+v", out.TopFighters)
	}
	if out.SignupsTotal < 2 {
		t.Fatalf("signups = %d", out.SignupsTotal)
	}
	if out.GuessesByDay == nil || out.PlayersByDay == nil || out.SignupsByDay == nil {
		t.Fatal("series must be non-nil for JSON ([])")
	}
}
