package store

import "context"

// Admin product-metrics boundary: aggregate reads over users and the
// server-side game records. Cloudflare infrastructure metrics live behind a
// separate endpoint (GraphQL proxy); game semantics only exist here because
// Cloudflare sees HTTP requests, not guesses, wins, or fighters.

// DayCount is one point of a per-day series (date as YYYY-MM-DD).
type DayCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// PoolSplit breaks games down by pool.
type PoolSplit struct {
	Pool    string `json:"pool"`
	Games   int    `json:"games"`
	Won     int    `json:"won"`
	Guesses int    `json:"guesses"`
}

// TopFighter ranks guessed fighters by volume.
type TopFighter struct {
	FighterID int    `json:"fighter_id"`
	Name      string `json:"name"`
	Guesses   int    `json:"guesses"`
}

// MetricsOverview is the admin dashboard payload. A "game" is one
// (player, pool, date) tuple; a game is "won" when any of its guesses is
// correct. Guests are invisible here: their guesses never leave the browser.
type MetricsOverview struct {
	Days            int          `json:"days"`
	Since           string       `json:"since"`
	SignupsTotal    int          `json:"signups_total"`
	SignupsByDay    []DayCount   `json:"signups_by_day"`
	PlayersByDay    []DayCount   `json:"players_by_day"`
	GuessesByDay    []DayCount   `json:"guesses_by_day"`
	GamesTotal      int          `json:"games_total"`
	GamesWon        int          `json:"games_won"`
	WinRate         float64      `json:"win_rate"`
	AvgGuessesToWin float64      `json:"avg_guesses_to_win"`
	ByPool          []PoolSplit  `json:"by_pool"`
	TopFighters     []TopFighter `json:"top_fighters"`
}

// AdminStore is the persistence port for the metrics interface.
type AdminStore interface {
	MetricsOverview(ctx context.Context, days int) (MetricsOverview, error)
}
