// Package httpapi is the transport boundary: JSON + routing only.
// Comparison and selection rules live in internal/domain; SQL lives in
// internal/store. Handlers wire the two together and validate input.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"mmadle/backend/internal/domain"
	"mmadle/backend/internal/store"
)

// Clock abstracts the current date so the game date is injectable in tests
// and deterministic per request (one date per evaluation).
type Clock interface {
	Now() time.Time
}

// SystemClock uses real time in the configured location.
type SystemClock struct{ Location *time.Location }

func (c SystemClock) Now() time.Time {
	if c.Location == nil {
		return time.Now().UTC()
	}
	return time.Now().In(c.Location)
}

// Server wires routes to the store, selector, and clock.
type Server struct {
	Store    store.FighterStore
	Selector domain.Selector
	Clock    Clock
	Logger   *slog.Logger
	mux      *http.ServeMux
}

// New builds the route table.
func New(st store.FighterStore, sel domain.Selector, clk Clock, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{Store: st, Selector: sel, Clock: clk, Logger: logger, mux: http.NewServeMux()}
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/game/today", s.handleToday)
	s.mux.HandleFunc("/api/fighters/search", s.handleSearch)
	s.mux.HandleFunc("/api/game/guess", s.handleGuess)
	return s
}

// Handler applies CORS + JSON middleware around the mux.
func (s *Server) Handler(allowedOrigins string) http.Handler {
	return corsMiddleware(allowedOrigins, jsonMiddleware(s.mux))
}

// gameDate returns the current game date truncated to the day.
func (s *Server) gameDate() time.Time {
	now := s.Clock.Now()
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}

// targetID resolves today's target without ever exposing it to the client.
func (s *Server) targetID(ctx context.Context) (int, time.Time, error) {
	gameDate := s.gameDate()
	ids, err := s.Store.GameFighterIDs(ctx)
	if err != nil {
		return 0, gameDate, err
	}
	id, err := s.Selector.Select(gameDate, ids)
	if err != nil {
		return 0, gameDate, err
	}
	return id, gameDate, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"error": code, "message": message})
}

// GET /api/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.Store.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "db_unavailable", "database unreachable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /api/game/today — game metadata ONLY. Never includes the target fighter.
func (s *Server) handleToday(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"date":   s.gameDate().UTC().Format("2006-01-02"),
		"status": "playing",
	})
}

// GET /api/fighters/search?q=
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "missing_query", "query parameter 'q' is required")
		return
	}
	if len(q) > 100 {
		writeError(w, http.StatusBadRequest, "query_too_long", "query must be at most 100 characters")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	results, err := s.Store.SearchFighters(ctx, q, 8)
	if err != nil {
		s.Logger.Error("search failed", "err", err)
		writeError(w, http.StatusInternalServerError, "search_failed", "search failed")
		return
	}
	if results == nil {
		results = []store.SearchResult{}
	}
	writeJSON(w, http.StatusOK, results)
}

// guessRequest is validated strictly: unknown fields rejected, id must be > 0.
type guessRequest struct {
	FighterID int `json:"fighter_id"`
}

// POST /api/game/guess
func (s *Server) handleGuess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var req guessRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "body must be JSON like {\"fighter_id\": 123}")
		return
	}
	if req.FighterID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_fighter_id", "fighter_id must be a positive integer")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	// Never trust the frontend id beyond lookup: unknown ids are 404.
	guess, err := s.Store.FighterView(ctx, req.FighterID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "fighter_not_found", "unknown fighter id")
			return
		}
		s.Logger.Error("guess lookup failed", "err", err)
		writeError(w, http.StatusInternalServerError, "guess_failed", "could not evaluate guess")
		return
	}

	targetID, gameDate, err := s.targetID(ctx)
	if err != nil {
		s.Logger.Error("target selection failed", "err", err)
		writeError(w, http.StatusInternalServerError, "game_unavailable", "daily game unavailable")
		return
	}
	target, err := s.Store.FighterView(ctx, targetID)
	if err != nil {
		s.Logger.Error("target lookup failed", "err", err)
		writeError(w, http.StatusInternalServerError, "game_unavailable", "daily game unavailable")
		return
	}

	writeJSON(w, http.StatusOK, domain.EvaluateGuess(target, guess, gameDate))
}

// --- middleware ---

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware applies an explicit allow-list (exact origin match, no
// wildcard reflection). Empty ALLOWED_ORIGINS disables cross-origin access.
func corsMiddleware(allowedOrigins string, next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range strings.Split(allowedOrigins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			allowed[o] = true
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
