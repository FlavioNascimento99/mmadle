package domain

import "testing"

func TestGameScore(t *testing.T) {
	cases := []struct {
		tries int
		won   bool
		want  int
	}{
		{1, true, 11},
		{2, true, 10},
		{6, true, 6},
		{11, true, 1},
		{30, true, 1},
		{0, true, 11}, // degenerate tries clamp to 1
		{2, false, 0},
	}
	for _, tc := range cases {
		if got := gameScore(tc.won, tc.tries); got != tc.want {
			t.Fatalf("gameScore(won=%v, tries=%d) = %d, want %d", tc.won, tc.tries, got, tc.want)
		}
	}
}

// wantEntry asserts one ranked entry field-by-field (zero values are
// deliberately ignored so tests only pin what matters).
func wantEntry(t *testing.T, e LeaderboardEntry, score, games, wins, streak, maxStreak int) {
	t.Helper()
	if e.Score != score {
		t.Fatalf("%s score=%d want %d", e.Username, e.Score, score)
	}
	if e.Games != games {
		t.Fatalf("%s games=%d want %d", e.Username, e.Games, games)
	}
	if e.Wins != wins {
		t.Fatalf("%s wins=%d want %d", e.Username, e.Wins, wins)
	}
	if e.CurrentStreak != streak {
		t.Fatalf("%s current streak=%d want %d", e.Username, e.CurrentStreak, streak)
	}
	if e.MaxStreak != maxStreak {
		t.Fatalf("%s max streak=%d want %d", e.Username, e.MaxStreak, maxStreak)
	}
}

func TestRankLeaderboard(t *testing.T) {
	t.Run("fewer tries scores more, misses zero", func(t *testing.T) {
		entries := RankLeaderboard([]LeaderboardGame{
			{UserID: 1, Username: "alpha", Date: "2026-09-10", Guesses: 1, Won: true},
			{UserID: 2, Username: "beta", Date: "2026-09-10", Guesses: 6, Won: true},
			{UserID: 3, Username: "zeta", Date: "2026-09-10", Guesses: 2, Won: false},
		}, "2026-09-10")
		wantEntry(t, entries[0], 11, 1, 1, 1, 1)
		if entries[0].Username != "alpha" || entries[0].Rank != 1 {
			t.Fatalf("first = %+v", entries[0])
		}
		wantEntry(t, entries[1], 6, 1, 1, 1, 1)
		if entries[1].Username != "beta" {
			t.Fatalf("second = %+v", entries[1])
		}
		wantEntry(t, entries[2], 0, 1, 0, 0, 0)
		if entries[2].Username != "zeta" {
			t.Fatalf("third = %+v", entries[2])
		}
	})

	t.Run("slow wins share the score floor", func(t *testing.T) {
		entries := RankLeaderboard([]LeaderboardGame{
			{UserID: 1, Username: "slow", Date: "2026-09-10", Guesses: 11, Won: true},
			{UserID: 2, Username: "slower", Date: "2026-09-10", Guesses: 30, Won: true},
		}, "2026-09-10")
		wantEntry(t, entries[0], 1, 1, 1, 1, 1)
		wantEntry(t, entries[1], 1, 1, 1, 1, 1)
	})

	t.Run("scores accumulate across days and streak folds in", func(t *testing.T) {
		entries := RankLeaderboard([]LeaderboardGame{
			{UserID: 2, Username: "dabbler", Date: "2026-09-16", Guesses: 3, Won: true},
			{UserID: 2, Username: "dabbler", Date: "2026-09-17", Guesses: 2, Won: false},
			{UserID: 1, Username: "streaker", Date: "2026-09-16", Guesses: 2, Won: true},
			{UserID: 1, Username: "streaker", Date: "2026-09-17", Guesses: 3, Won: true},
		}, "2026-09-17")
		wantEntry(t, entries[0], 19, 2, 2, 2, 2)
		if entries[0].Username != "streaker" {
			t.Fatalf("first = %+v", entries[0])
		}
		wantEntry(t, entries[1], 9, 2, 1, 0, 1)
		if entries[1].Username != "dabbler" {
			t.Fatalf("second = %+v", entries[1])
		}
	})

	t.Run("future dates and zero-guess rows are ignored", func(t *testing.T) {
		entries := RankLeaderboard([]LeaderboardGame{
			{UserID: 1, Username: "timey", Date: "2999-01-01", Guesses: 1, Won: true},
			{UserID: 2, Username: "ghost", Date: "2026-09-10", Guesses: 0, Won: false},
		}, "2026-09-18")
		if len(entries) != 0 {
			t.Fatalf("entries = %+v, want none", entries)
		}
	})

	t.Run("no players still encodes an empty array", func(t *testing.T) {
		entries := RankLeaderboard(nil, "2026-09-18")
		if len(entries) != 0 || entries == nil {
			t.Fatalf("entries = %+v, want empty non-nil slice", entries)
		}
	})
}
