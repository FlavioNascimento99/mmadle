package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mmadle/backend/internal/domain"
	"mmadle/backend/internal/store"
)

// Public leaderboard (opt-in only). Anyone can read the ranking, but only
// players who explicitly opted in have their username, score and streak
// exposed; guests and opted-out accounts never appear.

// leaderboardMyEntry is the signed-in player's own row plus their opt-in
// state, so the UI can highlight and toggle their position.
type leaderboardMyEntry struct {
	OptIn bool                     `json:"opt_in"`
	Entry *domain.LeaderboardEntry `json:"entry"`
}

// leaderboardResponse is the GET /api/leaderboard payload. my is null for
// guests; entries past the page window still rank the player's row.
type leaderboardResponse struct {
	Pool     string                    `json:"pool"`
	Total    int                       `json:"total"`
	Rankings []domain.LeaderboardEntry `json:"rankings"`
	My       *leaderboardMyEntry       `json:"my"`
}

// parseLeaderboardPage parses ?limit= (default 50, 1..100) and ?offset=
// (default 0, >= 0).
func parseLeaderboardPage(w http.ResponseWriter, r *http.Request) (limit, offset int, ok bool) {
	limit = 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			writeError(w, http.StatusBadRequest, "invalid_limit", "limit must be an integer between 1 and 100")
			return 0, 0, false
		}
		limit = n
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "invalid_offset", "offset must be a non-negative integer")
			return 0, 0, false
		}
		offset = n
	}
	return limit, offset, true
}

// GET /api/leaderboard?pool=&limit=&offset= — the ranking for one pool,
// public. Signed-in visitors get their own entry and opt-in state in `my`,
// even when it falls outside the requested page.
func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	if s.Leaderboard == nil {
		writeError(w, http.StatusInternalServerError, "leaderboard_unavailable", "leaderboard is not configured")
		return
	}
	pool, ok := parsePool(w, r.URL.Query().Get("pool"))
	if !ok {
		return
	}
	limit, offset, ok := parseLeaderboardPage(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	rows, err := s.Leaderboard.ListLeaderboardGames(ctx, pool, s.gameDate().UTC().Format("2006-01-02"))
	if err != nil {
		s.Logger.Error("leaderboard load failed", "pool", pool, "err", err)
		writeError(w, http.StatusInternalServerError, "leaderboard_failed", "could not load the leaderboard")
		return
	}
	ranked := domain.RankLeaderboard(rows, s.gameDate().UTC().Format("2006-01-02"))

	start := offset
	if start > len(ranked) {
		start = len(ranked)
	}
	end := start + limit
	if end > len(ranked) {
		end = len(ranked)
	}

	resp := leaderboardResponse{
		Pool:     string(pool),
		Total:    len(ranked),
		Rankings: ranked[start:end],
	}
	if resp.Rankings == nil {
		resp.Rankings = []domain.LeaderboardEntry{}
	}

	user, signedIn := s.currentUser(r)
	if signedIn {
		optIn, err := s.Leaderboard.LeaderboardOptIn(ctx, user.ID)
		if err != nil {
			// A stale badge is better than failing a public page; the entry
			// itself is computed from source rows so it stays accurate.
			s.Logger.Error("leaderboard opt-in read failed", "user_id", user.ID, "err", err)
			optIn = false
		}
		my := &leaderboardMyEntry{OptIn: optIn}
		if optIn {
			for i := range ranked {
				if ranked[i].UserID == user.ID {
					e := ranked[i]
					my.Entry = &e
					break
				}
			}
		}
		resp.My = my
	}
	writeJSON(w, http.StatusOK, resp)
}

type leaderboardOptInRequest struct {
	OptIn bool `json:"opt_in"`
}

// POST /api/me/leaderboard — toggle the public-leaderboard switch for the
// signed-in account. Opting out hides the entry immediately.
func (s *Server) handleLeaderboardOptIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if !requireJSONContentType(w, r) {
		return
	}
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if s.Leaderboard == nil {
		writeError(w, http.StatusInternalServerError, "leaderboard_unavailable", "leaderboard is not configured")
		return
	}
	var req leaderboardOptInRequest
	if !decodeStrict(w, r, &req) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	if err := s.Leaderboard.SetLeaderboardOptIn(ctx, user.ID, req.OptIn); err != nil {
		s.Logger.Error("leaderboard opt-in update failed", "user_id", user.ID)
		writeError(w, http.StatusInternalServerError, "leaderboard_failed", "could not update leaderboard settings")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": user.ID, "leaderboard_opt_in": req.OptIn})
}

// compile-time wiring check: Postgres serves the leaderboard port.
var _ store.LeaderboardStore = (*store.Postgres)(nil)
