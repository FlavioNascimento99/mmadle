package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mmadle/backend/internal/domain"
)

// dailyStatsResponse is the public per-pool solvers count. Counts only —
// never fighter ids, names, or attributes, so the target cannot leak.
type dailyStatsResponse struct {
	Date    string `json:"date"`
	Pool    string `json:"pool"`
	Solvers int    `json:"solvers"`
}

// GET /api/game/stats?pool= — how many players solved today's daily game in
// one pool, logged in or not. Public (no auth): anyone can see the count but
// never who solved. Guest solves are keyed by an anonymous browser identity
// (anon cookie set on their first solve); see 012_daily_solves.sql.
func (s *Server) handleDailyStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	pool, ok := parsePool(w, r.URL.Query().Get("pool"))
	if !ok {
		return
	}
	gameDate := s.gameDate()
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	solvers, err := s.Store.CountDailySolvers(ctx, pool, gameDate)
	if err != nil {
		s.Logger.Error("daily stats failed", "pool", pool, "err", err)
		writeError(w, http.StatusInternalServerError, "stats_failed", "could not load daily stats")
		return
	}
	writeJSON(w, http.StatusOK, dailyStatsResponse{
		Date:    gameDate.UTC().Format("2006-01-02"),
		Pool:    string(pool),
		Solvers: solvers,
	})
}

// anonCookieName carries the guest solve identity: a random token with no
// account behind it, used only to dedupe repeat solves by the same browser.
// It is issued on a guest's first solve (not on every visit), HttpOnly so
// page scripts never see it.
const anonCookieName = "mmadle_anon"

// solveIdentity resolves the daily-solves identity for one request: the user
// id when signed in, otherwise the anon cookie (issuing one when missing or
// malformed). The cookie is set on w, so callers must resolve before writing
// the response body.
func (s *Server) solveIdentity(w http.ResponseWriter, r *http.Request) string {
	if user, ok := s.currentUser(r); ok {
		return fmt.Sprintf("u:%d", user.ID)
	}
	if cookie, err := r.Cookie(anonCookieName); err == nil && validAnonToken(cookie.Value) {
		return "a:" + cookie.Value
	}
	token, err := domain.GenerateSessionToken()
	if err != nil {
		s.Logger.Error("anon token generation failed")
		return ""
	}
	expires := s.Clock.Now().AddDate(1, 0, 0)
	http.SetCookie(w, &http.Cookie{
		Name:     anonCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.SessionSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
	})
	return "a:" + token
}

// validAnonToken accepts only the tokens we issue (base64url, 43 chars):
// foreign values are replaced, never trusted as identities.
func validAnonToken(v string) bool {
	if len(v) != 43 {
		return false
	}
	for _, c := range v {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// recordSolveBestEffort stores one solve without ever failing the response:
// the counter stays best-effort, gameplay never depends on the write.
func (s *Server) recordSolveBestEffort(w http.ResponseWriter, r *http.Request, pool domain.Pool, day time.Time) {
	identity := s.solveIdentity(w, r)
	if identity == "" {
		return
	}
	// Identity prefixes are internal; never log the raw value.
	kind := strings.HasPrefix(identity, "u:")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Store.RecordDailySolve(ctx, pool, day, identity); err != nil {
		s.Logger.Error("solve record failed", "pool", pool, "signed_in", kind)
	}
}
