package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"mmadle/backend/internal/domain"
	"mmadle/backend/internal/store"
)

// fakeStatsStore returns canned per-user games for handler tests.
type fakeStatsStore struct{ games []store.UserGame }

func (f *fakeStatsStore) ListUserGames(ctx context.Context, userID int64) ([]store.UserGame, error) {
	return append([]store.UserGame{}, f.games...), nil
}

func statsTestServer(games []store.UserGame) (*Server, *fakeAuthStore) {
	srv, auth := authTestServer()
	srv.Stats = &fakeStatsStore{games: games}
	return srv, auth
}

func TestMyStatsRequiresAuth(t *testing.T) {
	srv, _ := statsTestServer(nil)
	if rec := getAuth(srv, "/api/me/stats"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("guest status=%d want 401", rec.Code)
	}
}

func TestMyStatsEmpty(t *testing.T) {
	srv, _ := statsTestServer(nil)
	rec := postAuth(srv, "/api/auth/register", `{"username":"fresh_stats","password":"a-correct-horse-battery9"}`)
	cookie := rec.Result().Cookies()[0]

	got := getAuth(srv, "/api/me/stats", cookie)
	if got.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", got.Code, got.Body.String())
	}
	var out domain.UserStats
	if err := json.Unmarshal(got.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.GamesTotal != 0 || out.WinRate != 0 || len(out.RecentGames) != 0 {
		t.Fatalf("fresh stats = %+v", out)
	}
}

func TestMyStatsAggregates(t *testing.T) {
	srv, _ := statsTestServer([]store.UserGame{
		{Date: "2026-09-16", Pool: domain.PoolAll, Guesses: 4, Won: true},
		{Date: "2026-09-17", Pool: domain.PoolAll, Guesses: 6, Won: true},
		{Date: "2026-09-18", Pool: domain.PoolMen, Guesses: 2, Won: true},
	})
	rec := postAuth(srv, "/api/auth/register", `{"username":"stat_pro","password":"a-correct-horse-battery9"}`)
	cookie := rec.Result().Cookies()[0]

	got := getAuth(srv, "/api/me/stats", cookie)
	if got.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", got.Code, got.Body.String())
	}
	var out domain.UserStats
	if err := json.Unmarshal(got.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	// testServer's clock is fixed at 2026-09-18: all three days won in a row.
	if out.GamesTotal != 3 || out.GamesWon != 3 || out.WinRate != 1 {
		t.Fatalf("totals = %+v", out)
	}
	if out.DaysPlayed != 3 || out.AvgTriesDay != 4 {
		t.Fatalf("days=%d tries/day=%v", out.DaysPlayed, out.AvgTriesDay)
	}
	all := out.Pools["all"]
	if all.CurrentStreak != 2 || all.MaxStreak != 2 || all.AvgTriesToWin != 5 {
		t.Fatalf("all = %+v", all)
	}
	if len(out.RecentGames) != 3 || out.RecentGames[0].Date != "2026-09-18" {
		t.Fatalf("recent = %+v", out.RecentGames)
	}
	// Never leak the target or anything beyond aggregates + own guesses.
	for _, leak := range []string{"Target Fighter", "password"} {
		if strings.Contains(got.Body.String(), leak) {
			t.Fatalf("stats must not contain %q", leak)
		}
	}
}
