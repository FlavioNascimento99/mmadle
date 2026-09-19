package domain

import (
	"testing"
)

func wonSet(dates ...string) map[string]bool {
	out := map[string]bool{}
	for _, d := range dates {
		out[d] = true
	}
	return out
}

func TestComputeStreaks(t *testing.T) {
	cases := []struct {
		name        string
		won         []string
		played      []string
		first       string
		today       string
		wantCurrent int
		wantMax     int
	}{
		{
			name:   "three in a row ending today",
			won:    []string{"2026-09-16", "2026-09-17", "2026-09-18"},
			played: []string{"2026-09-16", "2026-09-17", "2026-09-18"},
			first:  "2026-09-16", today: "2026-09-18",
			wantCurrent: 3, wantMax: 3,
		},
		{
			name:   "today in progress keeps yesterday's run",
			won:    []string{"2026-09-16", "2026-09-17"},
			played: []string{"2026-09-16", "2026-09-17"},
			first:  "2026-09-16", today: "2026-09-18",
			wantCurrent: 2, wantMax: 2,
		},
		{
			name:   "played but lost today ends the run",
			won:    []string{"2026-09-16", "2026-09-17"},
			played: []string{"2026-09-16", "2026-09-17", "2026-09-18"},
			first:  "2026-09-16", today: "2026-09-18",
			wantCurrent: 0, wantMax: 2,
		},
		{
			name:   "missed yesterday breaks current, max survives",
			won:    []string{"2026-09-14", "2026-09-15", "2026-09-18"},
			played: []string{"2026-09-14", "2026-09-15", "2026-09-18"},
			first:  "2026-09-14", today: "2026-09-18",
			wantCurrent: 1, wantMax: 2,
		},
		{
			name:   "nothing won yet",
			won:    []string{},
			played: []string{"2026-09-18"},
			first:  "2026-09-18", today: "2026-09-18",
			wantCurrent: 0, wantMax: 0,
		},
		{
			name:   "days before first play never count as misses",
			won:    []string{"2026-09-18"},
			played: []string{"2026-09-18"},
			first:  "2026-09-18", today: "2026-09-18",
			wantCurrent: 1, wantMax: 1,
		},
		{
			name:   "gap in the middle splits max",
			won:    []string{"2026-09-10", "2026-09-11", "2026-09-13", "2026-09-14", "2026-09-15"},
			played: []string{"2026-09-10", "2026-09-11", "2026-09-12", "2026-09-13", "2026-09-14", "2026-09-15"},
			first:  "2026-09-10", today: "2026-09-15",
			wantCurrent: 3, wantMax: 3,
		},
		{
			name:   "empty history",
			won:    []string{},
			played: []string{},
			first:  "", today: "2026-09-18",
			wantCurrent: 0, wantMax: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotCurrent, gotMax := ComputeStreaks(wonSet(tc.won...), wonSet(tc.played...), tc.first, tc.today)
			if gotCurrent != tc.wantCurrent || gotMax != tc.wantMax {
				t.Fatalf("current=%d max=%d, want %d/%d", gotCurrent, gotMax, tc.wantCurrent, tc.wantMax)
			}
		})
	}
}

func TestComputeStats(t *testing.T) {
	games := []GameSummary{
		{Date: "2026-09-16", Pool: PoolAll, Guesses: 4, Won: true},
		{Date: "2026-09-17", Pool: PoolAll, Guesses: 6, Won: false},
		{Date: "2026-09-18", Pool: PoolAll, Guesses: 2, Won: true},
		{Date: "2026-09-18", Pool: PoolMen, Guesses: 5, Won: true},
		{Date: "2099-01-01", Pool: PoolAll, Guesses: 1, Won: true}, // future: ignored
	}
	out := ComputeStats(games, "2026-09-18")

	if out.GamesTotal != 4 || out.GamesWon != 3 {
		t.Fatalf("totals = %d/%d, want 4/3", out.GamesTotal, out.GamesWon)
	}
	if out.WinRate != 0.75 {
		t.Fatalf("win rate = %v", out.WinRate)
	}
	if out.DaysPlayed != 3 {
		t.Fatalf("days = %d, want 3", out.DaysPlayed)
	}
	if out.AvgTriesDay != 17.0/3.0 {
		t.Fatalf("tries/day = %v", out.AvgTriesDay)
	}
	all := out.Pools["all"]
	if all.GamesPlayed != 3 || all.CurrentStreak != 1 || all.MaxStreak != 1 {
		t.Fatalf("all = %+v", all)
	}
	if all.Distribution[4] != 1 || all.Distribution[2] != 1 || len(all.Distribution) != 2 {
		t.Fatalf("distribution = %+v", all.Distribution)
	}
	if all.AvgTriesToWin != 3.0 {
		t.Fatalf("avg to win = %v", all.AvgTriesToWin)
	}
	men := out.Pools["men"]
	if men.GamesPlayed != 1 || !men.Won() {
		t.Fatalf("men = %+v", men)
	}
	if len(out.RecentGames) != 4 || out.RecentGames[0].Date != "2026-09-18" {
		t.Fatalf("recent = %+v", out.RecentGames)
	}

	empty := ComputeStats(nil, "2026-09-18")
	if empty.GamesTotal != 0 || empty.WinRate != 0 || len(empty.RecentGames) != 0 {
		t.Fatalf("empty = %+v", empty)
	}
	if empty.Pools["all"].Distribution == nil {
		t.Fatal("distribution must be non-nil for JSON")
	}
}

// Won reports whether the pool has any win (test helper kept next to usage).
func (s PoolStats) Won() bool { return s.GamesWon > 0 }
