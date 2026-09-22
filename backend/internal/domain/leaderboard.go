package domain

import "sort"

// Public leaderboard. Pure computation, mirrors stats.go: ranking derives
// from per-player game summaries and is table-tested; no hidden state.

// LeaderboardGame is one played daily game by an opted-in player, pre-folded
// by SQL to one row per (player, date).
type LeaderboardGame struct {
	UserID   int64
	Username string
	Date     string // YYYY-MM-DD
	Guesses  int
	Won      bool
}

// LeaderboardEntry is one ranked player. UserID never leaves the backend.
type LeaderboardEntry struct {
	UserID        int64   `json:"-"`
	Rank          int     `json:"rank"`
	Username      string  `json:"username"`
	Score         int     `json:"score"`
	Games         int     `json:"games"`
	Wins          int     `json:"wins"`
	WinRate       float64 `json:"win_rate"`
	CurrentStreak int     `json:"current_streak"`
	MaxStreak     int     `json:"max_streak"`
	AvgTries      float64 `json:"avg_tries"`
}

// gameScore awards fewer-tries wins more points. The daily game has no fixed
// guess cap, so 12 is a generous ceiling: a 1st-guess solve is 11 points, a
// 6-guess solve 6, and any win in 11+ guesses earns the floor of 1. Misses
// score 0.
func gameScore(won bool, tries int) int {
	if !won {
		return 0
	}
	if tries < 1 {
		tries = 1
	}
	if score := 12 - tries; score > 1 {
		return score
	}
	return 1
}

// RankLeaderboard folds per-player games into sorted rankings. Players with
// no playable rows never appear. today is the server game date (YYYY-MM-DD);
// games after it are ignored. Order is total: score desc, current streak
// desc, wins desc, average tries asc, then username asc.
func RankLeaderboard(games []LeaderboardGame, today string) []LeaderboardEntry {
	byUser := map[int64][]LeaderboardGame{}
	names := map[int64]string{}
	for _, g := range games {
		if g.Date == "" || g.Date > today || g.Guesses < 1 {
			continue
		}
		byUser[g.UserID] = append(byUser[g.UserID], g)
		names[g.UserID] = g.Username
	}
	entries := make([]LeaderboardEntry, 0, len(byUser))
	for userID, gs := range byUser {
		entries = append(entries, summarizeLeaderboardUser(userID, names[userID], gs, today))
	}
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.CurrentStreak != b.CurrentStreak {
			return a.CurrentStreak > b.CurrentStreak
		}
		if a.Wins != b.Wins {
			return a.Wins > b.Wins
		}
		if a.AvgTries != b.AvgTries {
			return a.AvgTries < b.AvgTries
		}
		return a.Username < b.Username
	})
	for i := range entries {
		entries[i].Rank = i + 1
	}
	if entries == nil {
		return []LeaderboardEntry{}
	}
	return entries
}

// summarizeLeaderboardUser folds one player's games: aggregate counts, win
// rate, averages and streaks (reusing the stats streak logic).
func summarizeLeaderboardUser(userID int64, username string, games []LeaderboardGame, today string) LeaderboardEntry {
	e := LeaderboardEntry{UserID: userID, Username: username}
	daySet := map[string]bool{}
	wonSet := map[string]bool{}
	first := games[0].Date
	totalGuesses := 0
	for _, g := range games {
		if g.Date < first {
			first = g.Date
		}
		daySet[g.Date] = true
		totalGuesses += g.Guesses
		e.Games++
		if g.Won {
			e.Wins++
			wonSet[g.Date] = true
			e.Score += gameScore(true, g.Guesses)
		}
	}
	if e.Games > 0 {
		e.WinRate = float64(e.Wins) / float64(e.Games)
		e.AvgTries = float64(totalGuesses) / float64(e.Games)
	}
	e.CurrentStreak, e.MaxStreak = ComputeStreaks(wonSet, daySet, first, today)
	return e
}
