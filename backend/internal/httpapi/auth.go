package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"mmadle/backend/internal/domain"
	"mmadle/backend/internal/store"
)

// Authentication (issue #1). Sessions are opaque random tokens carried in an
// HttpOnly; Secure; SameSite=Lax cookie. Only the SHA-256 hash is stored; the
// raw token is never persisted and never logged. Guests keep working exactly
// as before: auth is optional everywhere except /api/me/*.

// sessionCookieName is the session cookie. SameSite=Lax plus a required
// JSON Content-Type on state-changing routes is the CSRF defense (simple
// cross-site forms cannot set a JSON content type without a preflight).
const sessionCookieName = "mmadle_session"

// dummyPasswordHash absorbs login timing when the username does not exist, so
// unknown addresses take ~the same time as a real password check. Generated
// once at startup; on failure it is replaced by a per-boot random hash.
var dummyPasswordHash string

func init() {
	hash, err := domain.HashPassword("mmadle-dummy-login-timing-absorber-01")
	if err != nil {
		// Argon2 with a random salt cannot fail in practice; fall back to a
		// syntactically valid hash so logins fail closed instead of 500ing.
		dummyPasswordHash = "$argon2id$v=19$m=65536,t=3,p=4$ZHVtbXlzYWx0ZHVtbXk$ZHVtbXloYXNoZHVtbXloYXNoZHVtbXloYXNo"
		return
	}
	dummyPasswordHash = hash
}

// userResponse is the public account shape. The password hash never leaves
// the backend: it is mapped explicitly, never serialized from store.User.
type userResponse struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
	// LeaderboardOptIn mirrors users.leaderboard_opt_in so the UI can show
	// the toggle without a separate round trip after auth.
	LeaderboardOptIn bool `json:"leaderboard_opt_in"`
}

func toUserResponse(u store.User) userResponse {
	return userResponse{
		ID: u.ID, Username: u.Username, Role: u.Role,
		CreatedAt:        u.CreatedAt.UTC().Format(time.RFC3339),
		LeaderboardOptIn: u.LeaderboardOptIn,
	}
}

// requireJSONContentType rejects state-changing requests without a JSON
// content type (CSRF defense for cookie-authenticated POSTs).
func requireJSONContentType(w http.ResponseWriter, r *http.Request) bool {
	if ct := r.Header.Get("Content-Type"); !strings.Contains(strings.ToLower(ct), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return false
	}
	return true
}

// clientIP prefers proxy headers (the Worker/container sits behind proxies)
// and falls back to the connection address. Used only for rate-limit keys.
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if first, _, _ := strings.Cut(fwd, ","); strings.TrimSpace(first) != "" {
			return strings.TrimSpace(first)
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func decodeStrict(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "body must be JSON for this endpoint")
		return false
	}
	return true
}

// currentUser resolves the session cookie to its account. ok=false means
// guest (no cookie, unknown or expired token) or a deactivated account:
// callers treat that as unauthenticated, never as an error.
func (s *Server) currentUser(r *http.Request) (user store.User, ok bool) {
	if s.Auth == nil {
		return store.User{}, false
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return store.User{}, false
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	u, err := s.Auth.FindSessionUser(ctx, domain.HashSessionToken(cookie.Value), s.Clock.Now())
	if err != nil || !u.IsActive {
		return store.User{}, false
	}
	return u, true
}

// requireUser enforces authentication for /api/me/*.
func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	u, ok := s.currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return store.User{}, false
	}
	return u, true
}

// setSessionCookie issues the opaque token. MaxAge mirrors Expires for
// clients that only honor one of the two.
func (s *Server) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.SessionSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
	})
}

// clearSessionCookie expires the cookie so browsers drop it on logout.
func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.SessionSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
	})
}

// issueSession creates a session row, rotates out older ones, and sets the
// cookie. The raw token only exists in this function and the cookie.
func (s *Server) issueSession(ctx context.Context, w http.ResponseWriter, userID int64) bool {
	token, err := domain.GenerateSessionToken()
	if err != nil {
		s.Logger.Error("session token generation failed")
		writeError(w, http.StatusInternalServerError, "auth_failed", "could not sign in")
		return false
	}
	expires := s.Clock.Now().Add(s.SessionTTL)
	if err := s.Auth.CreateSession(ctx, domain.HashSessionToken(token), userID, expires); err != nil {
		s.Logger.Error("session creation failed")
		writeError(w, http.StatusInternalServerError, "auth_failed", "could not sign in")
		return false
	}
	// Rotation: keep only the fresh session (logout-everywhere-on-login is
	// the simple, safe default for v1).
	if err := s.Auth.DeleteUserSessions(ctx, userID, domain.HashSessionToken(token)); err != nil {
		s.Logger.Error("session rotation failed", "user_id", userID)
	}
	s.setSessionCookie(w, token, expires)
	return true
}

// authUnavailable guards routes when the server was built without an auth
// store (tests for unrelated handlers).
func (s *Server) authUnavailable(w http.ResponseWriter) bool {
	if s.Auth == nil {
		writeError(w, http.StatusInternalServerError, "auth_unavailable", "accounts are not configured")
		return true
	}
	return false
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// POST /api/auth/register — username + password.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if !requireJSONContentType(w, r) || s.authUnavailable(w) {
		return
	}
	var req registerRequest
	if !decodeStrict(w, r, &req) {
		return
	}
	username := domain.NormalizeUsername(req.Username)
	if err := domain.ValidateUsername(username); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_username", "username must be 3-20 letters, digits or underscores")
		return
	}
	if err := domain.ValidatePassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, "weak_password", "password must be at least 10 characters and not a common password")
		return
	}
	if !s.authIPLimiter.Allow("register:ip:"+clientIP(r)) || !s.authNameLimiter.Allow("register:username:"+username) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many attempts, try again later")
		return
	}

	hash, err := domain.HashPassword(req.Password)
	if err != nil {
		s.Logger.Error("password hashing failed")
		writeError(w, http.StatusInternalServerError, "auth_failed", "could not create account")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	user, err := s.Auth.CreateUser(ctx, username, hash)
	if err != nil {
		if errors.Is(err, store.ErrUsernameTaken) {
			writeError(w, http.StatusConflict, "username_taken", "an account with this username already exists")
			return
		}
		s.Logger.Error("user creation failed")
		writeError(w, http.StatusInternalServerError, "auth_failed", "could not create account")
		return
	}
	if !s.issueSession(ctx, w, user.ID) {
		return
	}
	if err := s.Auth.TouchLastLogin(ctx, user.ID, s.Clock.Now()); err != nil {
		s.Logger.Error("last login stamp failed", "user_id", user.ID)
	}
	s.maybePromote(ctx, &user)
	writeJSON(w, http.StatusCreated, toUserResponse(user))
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// POST /api/auth/login — errors never reveal whether the username exists.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if !requireJSONContentType(w, r) || s.authUnavailable(w) {
		return
	}
	var req loginRequest
	if !decodeStrict(w, r, &req) {
		return
	}
	username := domain.NormalizeUsername(req.Username)
	if !s.authIPLimiter.Allow("login:ip:"+clientIP(r)) || !s.authNameLimiter.Allow("login:username:"+username) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many attempts, try again later")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	user, err := s.Auth.FindUserByUsername(ctx, username)
	if err != nil {
		// Absorb timing so unknown usernames cost ~one password check, then
		// answer exactly like a wrong password. Nothing distinguishing is
		// logged or returned.
		_ = domain.VerifyPassword(dummyPasswordHash, req.Password)
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "username or password is incorrect")
		return
	}
	if err := domain.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "username or password is incorrect")
		return
	}
	if !user.IsActive {
		// Deactivated accounts answer exactly like a wrong password: the
		// lockout itself is never advertised.
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "username or password is incorrect")
		return
	}
	// Drop the presented session, if any, before issuing the fresh one.
	if cookie, cerr := r.Cookie(sessionCookieName); cerr == nil && cookie.Value != "" {
		_ = s.Auth.DeleteSession(ctx, domain.HashSessionToken(cookie.Value))
	}
	if !s.issueSession(ctx, w, user.ID) {
		return
	}
	if err := s.Auth.TouchLastLogin(ctx, user.ID, s.Clock.Now()); err != nil {
		s.Logger.Error("last login stamp failed", "user_id", user.ID)
	}
	s.maybePromote(ctx, &user)
	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// POST /api/auth/logout — idempotent; always clears the cookie.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if !requireJSONContentType(w, r) || s.authUnavailable(w) {
		return
	}
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		_ = s.Auth.DeleteSession(ctx, domain.HashSessionToken(cookie.Value))
	}
	s.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /api/auth/me — the signed-in account, or 401 for guests.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	if s.authUnavailable(w) {
		return
	}
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// myGuessesDate resolves the optional ?date= (default: today) and refuses
// future dates against the server clock, never the client's.
func (s *Server) myGuessesDate(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("date"))
	if raw == "" {
		return s.gameDate(), true
	}
	day, err := time.Parse("2006-01-02", raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_date", "date must be YYYY-MM-DD")
		return time.Time{}, false
	}
	today := s.gameDate().UTC().Format("2006-01-02")
	if raw > today {
		writeError(w, http.StatusBadRequest, "invalid_date", "date must not be in the future")
		return time.Time{}, false
	}
	return day, true
}

// evaluateStoredGuesses replays stored fighter ids through the live game logic
// so client-supplied outcomes are never trusted and attribute updates apply.
func (s *Server) evaluateStoredGuesses(ctx context.Context, pool domain.Pool, day time.Time, stored []store.GameGuess) ([]domain.GuessOutcome, error) {
	ids, err := s.Store.GameFighterIDs(ctx, pool)
	if err != nil {
		return nil, err
	}
	targetID, err := s.Selector.Select(day, ids)
	if err != nil {
		return nil, err
	}
	target, err := s.Store.FighterView(ctx, targetID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.GuessOutcome, 0, len(stored))
	for _, g := range stored {
		guess, err := s.Store.FighterView(ctx, g.FighterID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				continue // fighter removed since: skip, keep history readable
			}
			return nil, err
		}
		out = append(out, domain.EvaluateGuess(target, guess, day))
	}
	return out, nil
}

// GET /api/me/guesses?pool=&date= — signed-in history, restored on any device.
func (s *Server) handleMyGuesses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	if s.authUnavailable(w) {
		return
	}
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	pool, ok := parsePool(w, r.URL.Query().Get("pool"))
	if !ok {
		return
	}
	day, ok := s.myGuessesDate(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	stored, err := s.Auth.ListGuesses(ctx, user.ID, pool, day)
	if err != nil {
		s.Logger.Error("list guesses failed", "user_id", user.ID)
		writeError(w, http.StatusInternalServerError, "guesses_failed", "could not load guesses")
		return
	}
	outcomes, err := s.evaluateStoredGuesses(ctx, pool, day, stored)
	if err != nil {
		s.Logger.Error("guess replay failed", "user_id", user.ID)
		writeError(w, http.StatusInternalServerError, "game_unavailable", "daily game unavailable")
		return
	}
	if outcomes == nil {
		outcomes = []domain.GuessOutcome{}
	}
	writeJSON(w, http.StatusOK, outcomes)
}

type importRequest struct {
	Pool       string `json:"pool"`
	Date       string `json:"date"`
	FighterIDs []int  `json:"fighter_ids"`
}

// POST /api/me/import — migrate local (guest) guesses after first sign-in.
// Each id is re-evaluated server-side; already-stored ids are skipped.
func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if !requireJSONContentType(w, r) || s.authUnavailable(w) {
		return
	}
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req importRequest
	if !decodeStrict(w, r, &req) {
		return
	}
	pool, ok := parsePool(w, req.Pool)
	if !ok {
		return
	}
	day := s.gameDate()
	if strings.TrimSpace(req.Date) != "" {
		raw := strings.TrimSpace(req.Date)
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_date", "date must be YYYY-MM-DD")
			return
		}
		if raw > s.gameDate().UTC().Format("2006-01-02") {
			writeError(w, http.StatusBadRequest, "invalid_date", "date must not be in the future")
			return
		}
		day = parsed
	}
	if len(req.FighterIDs) == 0 || len(req.FighterIDs) > 100 {
		writeError(w, http.StatusBadRequest, "invalid_fighter_ids", "fighter_ids must hold 1-100 ids")
		return
	}
	for _, id := range req.FighterIDs {
		if id <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_fighter_ids", "fighter ids must be positive integers")
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	ids, err := s.Store.GameFighterIDs(ctx, pool)
	if err != nil {
		s.Logger.Error("import target selection failed", "user_id", user.ID)
		writeError(w, http.StatusInternalServerError, "game_unavailable", "daily game unavailable")
		return
	}
	targetID, err := s.Selector.Select(day, ids)
	if err != nil {
		s.Logger.Error("import target selection failed", "user_id", user.ID)
		writeError(w, http.StatusInternalServerError, "game_unavailable", "daily game unavailable")
		return
	}
	target, err := s.Store.FighterView(ctx, targetID)
	if err != nil {
		s.Logger.Error("import target lookup failed", "user_id", user.ID)
		writeError(w, http.StatusInternalServerError, "game_unavailable", "daily game unavailable")
		return
	}
	outcomes := make([]domain.GuessOutcome, 0, len(req.FighterIDs))
	seen := map[int]bool{}
	for _, id := range req.FighterIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		guess, err := s.Store.FighterView(ctx, id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "fighter_not_found", "unknown fighter id in import")
				return
			}
			s.Logger.Error("import lookup failed", "user_id", user.ID)
			writeError(w, http.StatusInternalServerError, "import_failed", "could not import guesses")
			return
		}
		outcome := domain.EvaluateGuess(target, guess, day)
		if _, err := s.Auth.RecordGuess(ctx, user.ID, pool, day, id, outcome.Correct); err != nil {
			s.Logger.Error("import record failed", "user_id", user.ID)
			writeError(w, http.StatusInternalServerError, "import_failed", "could not import guesses")
			return
		}
		if outcome.Correct {
			// Imported solves count too, under the account identity; the
			// counter stays best-effort and never fails the import.
			if err := s.Store.RecordDailySolve(ctx, pool, day, fmt.Sprintf("u:%d", user.ID)); err != nil {
				s.Logger.Error("import solve record failed", "user_id", user.ID)
			}
		}
		outcomes = append(outcomes, outcome)
	}
	writeJSON(w, http.StatusOK, map[string]any{"guesses": outcomes, "imported": len(outcomes)})
}

// recordGuessBestEffort persists a signed-in guess without ever failing the
// guess response: guests and gameplay stay up even if the write fails.
func (s *Server) recordGuessBestEffort(r *http.Request, pool domain.Pool, day time.Time, outcome domain.GuessOutcome) {
	user, ok := s.currentUser(r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := s.Auth.RecordGuess(ctx, user.ID, pool, day, outcome.FighterID, outcome.Correct); err != nil {
		s.Logger.Error("guess record failed", "user_id", user.ID)
	}
}
