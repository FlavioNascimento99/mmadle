package domain

import (
	"sort"
	"time"
)

// Player statistics (personal stats page). All functions here are pure:
// dates are YYYY-MM-DD game-date strings from the server clock, so streaks
// follow the game timezone without the domain knowing about clocks.
//
// A "game" is one (pool, date) tuple with ≥1 stored guess. A game is "won"
// when any of its guesses is correct. Guests have no stats: their guesses
// never leave the browser.

// GameSummary is one played game in one pool on one date.
type GameSummary struct {
	Date    string // YYYY-MM-DD
	Pool    Pool
	Guesses int
	Won     bool
}

// PoolStats aggregates one pool's history.
type PoolStats struct {
	GamesPlayed   int         `json:"games_played"`
	GamesWon      int         `json:"games_won"`
	WinRate       float64     `json:"win_rate"`
	CurrentStreak int         `json:"current_streak"`
	MaxStreak     int         `json:"max_streak"`
	AvgTries      float64     `json:"avg_tries"`
	AvgTriesToWin float64     `json:"avg_tries_to_win"`
	Distribution  map[int]int `json:"distribution"`
	DaysPlayed    int         `json:"days_played"`
}

// RecentGame is one row of the recent-games list (newest first).
type RecentGame struct {
	Date    string `json:"date"`
	Pool    Pool   `json:"pool"`
	Guesses int    `json:"guesses"`
	Won     bool   `json:"won"`
}

// UserStats is the GET /api/me/stats payload.
type UserStats struct {
	Pools       map[string]PoolStats `json:"pools"`
	GamesTotal  int                  `json:"games_total"`
	GamesWon    int                  `json:"games_won"`
	WinRate     float64              `json:"win_rate"`
	DaysPlayed  int                  `json:"days_played"`
	AvgTriesDay float64              `json:"avg_tries_per_day"`
	RecentGames []RecentGame         `json:"recent_games"`
}

// ComputeStats folds game summaries into per-pool and overall stats.
// today is the server game date (YYYY-MM-DD); games after it are ignored.
func ComputeStats(games []GameSummary, today string) UserStats {
	out := UserStats{
		Pools:       map[string]PoolStats{},
		RecentGames: []RecentGame{},
	}
	byPool := map[Pool][]GameSummary{}
	daySet := map[string]bool{}
	totalGuesses := 0
	ordered := make([]GameSummary, 0, len(games))
	for _, g := range games {
		if g.Date == "" || g.Date > today {
			continue
		}
		ordered = append(ordered, g)
		byPool[g.Pool] = append(byPool[g.Pool], g)
		daySet[g.Date] = true
		totalGuesses += g.Guesses
		out.GamesTotal++
		if g.Won {
			out.GamesWon++
		}
	}
	for _, pool := range []Pool{PoolAll, PoolMen} {
		out.Pools[string(pool)] = summarizePool(byPool[pool], today)
	}
	if out.GamesTotal > 0 {
		out.WinRate = float64(out.GamesWon) / float64(out.GamesTotal)
	}
	out.DaysPlayed = len(daySet)
	if out.DaysPlayed > 0 {
		out.AvgTriesDay = float64(totalGuesses) / float64(out.DaysPlayed)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Date != ordered[j].Date {
			return ordered[i].Date > ordered[j].Date
		}
		return ordered[i].Pool < ordered[j].Pool
	})
	for i, g := range ordered {
		if i >= 10 {
			break
		}
		out.RecentGames = append(out.RecentGames, RecentGame{
			Date: g.Date, Pool: g.Pool, Guesses: g.Guesses, Won: g.Won,
		})
	}
	if out.RecentGames == nil {
		out.RecentGames = []RecentGame{}
	}
	return out
}

// summarizePool aggregates one pool: win rate, streaks, averages and the
// guess distribution (won games only, keyed by guesses needed).
func summarizePool(games []GameSummary, today string) PoolStats {
	st := PoolStats{Distribution: map[int]int{}}
	if len(games) == 0 {
		return st
	}
	daySet := map[string]bool{}
	wonSet := map[string]bool{}
	first := games[0].Date
	totalGuesses, winGuesses := 0, 0
	for _, g := range games {
		if g.Date < first {
			first = g.Date
		}
		daySet[g.Date] = true
		totalGuesses += g.Guesses
		st.GamesPlayed++
		if g.Won {
			st.GamesWon++
			wonSet[g.Date] = true
			winGuesses += g.Guesses
			st.Distribution[g.Guesses]++
		}
	}
	st.WinRate = float64(st.GamesWon) / float64(st.GamesPlayed)
	st.AvgTries = float64(totalGuesses) / float64(st.GamesPlayed)
	if st.GamesWon > 0 {
		st.AvgTriesToWin = float64(winGuesses) / float64(st.GamesWon)
	}
	st.DaysPlayed = len(daySet)
	st.CurrentStreak, st.MaxStreak = ComputeStreaks(wonSet, daySet, first, today)
	return st
}

// ComputeStreaks derives streaks from won/played dates. An unfinished today
// never breaks the run (anchors on yesterday); a played-but-lost today ends
// it. Dates before firstPlayed (before the player ever played) never count
// as misses.
func ComputeStreaks(wonSet, playedSet map[string]bool, firstPlayed, today string) (current, max int) {
	if firstPlayed == "" || firstPlayed > today {
		return 0, 0
	}
	if !wonSet[today] {
		if playedSet[today] {
			// Lost today: the run ends here (max below still counts history).
			return 0, maxStreak(wonSet, firstPlayed, today)
		}
		today = prevDate(today)
	}
	for d := today; d >= firstPlayed; d = prevDate(d) {
		if !wonSet[d] {
			break
		}
		current++
	}
	return current, maxStreak(wonSet, firstPlayed, today)
}

// maxStreak is the longest run of consecutive won dates in [first, last].
func maxStreak(wonSet map[string]bool, first, last string) (max int) {
	run := 0
	for d := first; d <= last; d = nextDate(d) {
		if d == "" {
			break
		}
		if wonSet[d] {
			run++
			if run > max {
				max = run
			}
		} else {
			run = 0
		}
	}
	return max
}

// prevDate / nextDate step one calendar day. Inputs are validated game dates;
// a parse failure yields "" which safely terminates the loops above.
func prevDate(day string) string {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return ""
	}
	return t.AddDate(0, 0, -1).Format("2006-01-02")
}

func nextDate(day string) string {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return ""
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}
