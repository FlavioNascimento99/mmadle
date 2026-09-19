package httpapi

import (
	"context"
	"net/http"
	"time"

	"mmadle/backend/internal/domain"
	"mmadle/backend/internal/store"
)

// GET /api/me/stats — the signed-in player's personal statistics: games,
// win rate, streaks, tries, guess distribution and recent games, per pool.
// Pure computation lives in internal/domain (table-tested); the handler only
// loads game summaries and stamps today's game date.
func (s *Server) handleMyStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	if s.Stats == nil {
		writeError(w, http.StatusInternalServerError, "stats_unavailable", "statistics are not configured")
		return
	}
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	stored, err := s.Stats.ListUserGames(ctx, user.ID)
	if err != nil {
		s.Logger.Error("list user games failed", "user_id", user.ID)
		writeError(w, http.StatusInternalServerError, "stats_failed", "could not load statistics")
		return
	}
	summaries := make([]domain.GameSummary, 0, len(stored))
	for _, g := range stored {
		pool, err := domain.ParsePool(string(g.Pool))
		if err != nil {
			continue // unknown pool values never break the page
		}
		summaries = append(summaries, domain.GameSummary{
			Date: g.Date, Pool: pool, Guesses: g.Guesses, Won: g.Won,
		})
	}
	writeJSON(w, http.StatusOK, domain.ComputeStats(summaries, s.gameDate().UTC().Format("2006-01-02")))
}

// compile-time wiring check: Postgres serves every store port.
var (
	_ store.AuthStore  = (*store.Postgres)(nil)
	_ store.AdminStore = (*store.Postgres)(nil)
	_ store.StatsStore = (*store.Postgres)(nil)
)
