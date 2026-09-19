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
	"strconv"
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
	Auth     store.AuthStore
	Selector domain.Selector
	Clock    Clock
	Logger   *slog.Logger
	// SessionTTL is the lifetime of a login session (default 30 days).
	SessionTTL time.Duration
	// SessionSecure marks the session cookie Secure; disable only for
	// plain-http local development (cookies are never sent over http when set).
	SessionSecure bool
	// AdminMetrics backs /api/admin/metrics/* (nil in tests for other routes).
	AdminMetrics store.AdminStore
	// Stats backs /api/me/stats (nil in tests for other routes).
	Stats store.StatsStore
	// AdminUsernames is the normalized allowlist auto-promoting admins.
	AdminUsernames map[string]bool
	// CloudflareToken/Account enable /api/admin/cloudflare/* (Worker secrets).
	CloudflareToken   string
	CloudflareAccount string
	// cfGraphQLURL/cfHTTPClient are seams for tests (GraphQL endpoint + client).
	cfGraphQLURL    string
	cfHTTPClient    *http.Client
	authIPLimiter   *RateLimiter
	authNameLimiter *RateLimiter
	mux             *http.ServeMux
}

// New builds the route table. auth may be nil in tests for unrelated
// handlers; auth routes then answer 501 auth_unavailable.
func New(st store.FighterStore, auth store.AuthStore, sel domain.Selector, clk Clock, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	if clk == nil {
		clk = SystemClock{Location: time.UTC}
	}
	s := &Server{
		Store: st, Auth: auth, Selector: sel, Clock: clk, Logger: logger,
		SessionTTL:      domain.DefaultSessionTTL,
		SessionSecure:   true,
		authIPLimiter:   NewRateLimiter(30, 10*time.Minute, clk.Now),
		authNameLimiter: NewRateLimiter(10, 10*time.Minute, clk.Now),
		mux:             http.NewServeMux(),
	}
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/game/today", s.handleToday)
	s.mux.HandleFunc("/api/fighters", s.handleListFighters)
	s.mux.HandleFunc("/api/fighters/search", s.handleSearch)
	s.mux.HandleFunc("/api/game/guess", s.handleGuess)
	s.mux.HandleFunc("/api/game/hints", s.handleHints)
	s.mux.HandleFunc("/api/game/stats", s.handleDailyStats)
	s.mux.HandleFunc("/api/auth/register", s.handleRegister)
	s.mux.HandleFunc("/api/auth/login", s.handleLogin)
	s.mux.HandleFunc("/api/auth/logout", s.handleLogout)
	s.mux.HandleFunc("/api/auth/me", s.handleMe)
	s.mux.HandleFunc("/api/me/guesses", s.handleMyGuesses)
	s.mux.HandleFunc("/api/me/import", s.handleImport)
	s.mux.HandleFunc("/api/me/stats", s.handleMyStats)
	s.mux.HandleFunc("/api/admin/metrics/overview", s.handleAdminOverview)
	s.mux.HandleFunc("/api/admin/cloudflare/workers", s.handleCloudflareWorkers)
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

// maxHintGuesses bounds the guesses parameter; every hint unlocks well below it.
const maxHintGuesses = 100

// targetID resolves the pool's daily target without ever exposing it to the client.
func (s *Server) targetID(ctx context.Context, pool domain.Pool) (int, time.Time, error) {
	gameDate := s.gameDate()
	ids, err := s.Store.GameFighterIDs(ctx, pool)
	if err != nil {
		return 0, gameDate, err
	}
	id, err := s.Selector.Select(gameDate, ids)
	if err != nil {
		return 0, gameDate, err
	}
	return id, gameDate, nil
}

// target loads the pool's daily target view, logging and writing a 500 on failure.
func (s *Server) target(ctx context.Context, w http.ResponseWriter, pool domain.Pool) (domain.FighterView, time.Time, bool) {
	id, gameDate, err := s.targetID(ctx, pool)
	if err != nil {
		s.Logger.Error("target selection failed", "pool", pool, "err", err)
		writeError(w, http.StatusInternalServerError, "game_unavailable", "daily game unavailable")
		return domain.FighterView{}, gameDate, false
	}
	view, err := s.Store.FighterView(ctx, id)
	if err != nil {
		s.Logger.Error("target lookup failed", "pool", pool, "err", err)
		writeError(w, http.StatusInternalServerError, "game_unavailable", "daily game unavailable")
		return domain.FighterView{}, gameDate, false
	}
	return view, gameDate, true
}

// parsePool parses an optional pool value, writing a 400 when unknown.
func parsePool(w http.ResponseWriter, raw string) (domain.Pool, bool) {
	pool, err := domain.ParsePool(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_pool", "pool must be 'all' or 'men'")
		return "", false
	}
	return pool, true
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
	pool, ok := parsePool(w, r.URL.Query().Get("pool"))
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	results, err := s.Store.SearchFighters(ctx, pool, q, 8)
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

// GET /api/fighters?pool= — the pool's roster for browsing, alphabetical.
func (s *Server) handleListFighters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	pool, ok := parsePool(w, r.URL.Query().Get("pool"))
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	roster, err := s.Store.ListFighters(ctx, pool)
	if err != nil {
		s.Logger.Error("list fighters failed", "err", err)
		writeError(w, http.StatusInternalServerError, "list_failed", "could not load fighters")
		return
	}
	if roster == nil {
		roster = []store.SearchResult{}
	}
	writeJSON(w, http.StatusOK, roster)
}

// guessRequest is validated strictly: unknown fields rejected, id must be > 0,
// pool optional (defaults to all fighters).
type guessRequest struct {
	FighterID int    `json:"fighter_id"`
	Pool      string `json:"pool"`
}

// POST /api/game/guess
func (s *Server) handleGuess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if !requireJSONContentType(w, r) {
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
	pool, ok := parsePool(w, req.Pool)
	if !ok {
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

	target, gameDate, found := s.target(ctx, w, pool)
	if !found {
		return
	}
	outcome := domain.EvaluateGuess(target, guess, gameDate)
	// Signed-in guesses are recorded server-side (best-effort); guests are
	// untouched and the response never waits on the write failing.
	s.recordGuessBestEffort(r, pool, gameDate, outcome)
	// Every solve counts towards the public counter, logged in or not
	// (guests via their anon identity); must run before writeJSON so a
	// newly issued anon cookie lands on the response.
	if outcome.Correct {
		s.recordSolveBestEffort(w, r, pool, gameDate)
	}
	writeJSON(w, http.StatusOK, outcome)
}

// GET /api/game/hints?guesses=N&pool= — hints unlocked after N guesses.
// N is client-reported: hints are a convenience, not a secret boundary.
func (s *Server) handleHints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	guesses, err := strconv.Atoi(r.URL.Query().Get("guesses"))
	if err != nil || guesses < 0 || guesses > maxHintGuesses {
		writeError(w, http.StatusBadRequest, "invalid_guesses", "guesses must be an integer between 0 and 100")
		return
	}
	pool, ok := parsePool(w, r.URL.Query().Get("pool"))
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	target, _, found := s.target(ctx, w, pool)
	if !found {
		return
	}
	writeJSON(w, http.StatusOK, domain.Hints(target, guesses))
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
// Allowed origins are trusted with credentials so the session cookie flows in
// local development (frontend :3000 -> backend :8080); in production the
// Worker serves UI and /api/* same-origin and CORS is not exercised.
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
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
