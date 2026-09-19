package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"mmadle/backend/internal/domain"
	"mmadle/backend/internal/store"
)

// fakeAuthStore is an in-memory AuthStore for handler tests: no hashing, no
// SQL, just the contract (unique username, session expiry, ordered guesses).
type fakeAuthStore struct {
	mu       sync.Mutex
	users    map[int64]store.User
	byName   map[string]int64
	sessions map[string]sessionRow
	guesses  map[string][]store.GameGuess
	nextID   int64
}

type sessionRow struct {
	userID  int64
	expires time.Time
}

func newFakeAuthStore() *fakeAuthStore {
	return &fakeAuthStore{
		users:    map[int64]store.User{},
		byName:   map[string]int64{},
		sessions: map[string]sessionRow{},
		guesses:  map[string][]store.GameGuess{},
		nextID:   1,
	}
}

func (f *fakeAuthStore) CreateUser(ctx context.Context, username, passwordHash string) (store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, taken := f.byName[username]; taken {
		return store.User{}, store.ErrUsernameTaken
	}
	u := store.User{ID: f.nextID, Username: username, PasswordHash: passwordHash, Role: "player", CreatedAt: time.Now()}
	f.nextID++
	f.users[u.ID] = u
	f.byName[username] = u.ID
	return u, nil
}

func (f *fakeAuthStore) FindUserByUsername(ctx context.Context, username string) (store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id, ok := f.byName[username]; ok {
		return f.users[id], nil
	}
	return store.User{}, store.ErrNotFound
}

func (f *fakeAuthStore) FindUserByID(ctx context.Context, id int64) (store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return store.User{}, store.ErrNotFound
}

func (f *fakeAuthStore) CreateSession(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sessions[tokenHash] = sessionRow{userID: userID, expires: expiresAt}
	return nil
}

func (f *fakeAuthStore) FindSessionUser(ctx context.Context, tokenHash string, now time.Time) (store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.sessions[tokenHash]
	if !ok || !row.expires.After(now) {
		return store.User{}, store.ErrNoSession
	}
	return f.users[row.userID], nil
}

func (f *fakeAuthStore) DeleteSession(ctx context.Context, tokenHash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sessions, tokenHash)
	return nil
}

func (f *fakeAuthStore) DeleteUserSessions(ctx context.Context, userID int64, exceptTokenHash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for hash, row := range f.sessions {
		if row.userID == userID && hash != exceptTokenHash {
			delete(f.sessions, hash)
		}
	}
	return nil
}

func (f *fakeAuthStore) SetUserRole(ctx context.Context, userID int64, role string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u := f.users[userID]
	u.Role = role
	f.users[userID] = u
	return nil
}

func (f *fakeAuthStore) RecordGuess(ctx context.Context, userID int64, pool domain.Pool, gameDate time.Time, fighterID int, correct bool) (store.GameGuess, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := guessKey(userID, pool, gameDate)
	for _, g := range f.guesses[key] {
		if g.FighterID == fighterID {
			return g, nil
		}
	}
	g := store.GameGuess{FighterID: fighterID, GuessIndex: len(f.guesses[key]) + 1, Correct: correct}
	f.guesses[key] = append(f.guesses[key], g)
	return g, nil
}

func (f *fakeAuthStore) ListGuesses(ctx context.Context, userID int64, pool domain.Pool, gameDate time.Time) ([]store.GameGuess, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := append([]store.GameGuess{}, f.guesses[guessKey(userID, pool, gameDate)]...)
	if out == nil {
		out = []store.GameGuess{}
	}
	return out, nil
}

func guessKey(userID int64, pool domain.Pool, day time.Time) string {
	return strconv.FormatInt(userID, 10) + "|" + string(pool) + "|" + day.UTC().Format("2006-01-02")
}

// authTestServer wires the game fixtures with an in-memory auth store.
func authTestServer() (*Server, *fakeAuthStore) {
	srv, _ := testServer()
	auth := newFakeAuthStore()
	srv.Auth = auth
	srv.SessionSecure = false
	return srv, auth
}

func postAuth(srv *Server, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(rec, req)
	return rec
}

func getAuth(srv *Server, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(rec, req)
	return rec
}

func sessionCookies(rec *httptest.ResponseRecorder) []*http.Cookie {
	return rec.Result().Cookies()
}

func TestRegisterLoginLogoutMe(t *testing.T) {
	srv, _ := authTestServer()

	rec := postAuth(srv, "/api/auth/register", `{"username":"octagon_fan","password":"a-correct-horse-battery9"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", rec.Code, rec.Body.String())
	}
	var user map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
		t.Fatal(err)
	}
	if user["username"] != "octagon_fan" {
		t.Fatalf("register response = %v", user)
	}
	if _, ok := user["password_hash"]; ok {
		t.Fatal("password hash must never be serialized")
	}
	cookies := sessionCookies(rec)
	if len(cookies) == 0 {
		t.Fatal("register must set a session cookie")
	}
	c := cookies[0]
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.Value == "" {
		t.Fatalf("session cookie attrs wrong: %+v", c)
	}

	// Duplicate registration conflicts.
	if rec := postAuth(srv, "/api/auth/register", `{"username":"octagon_fan","password":"another-horse-battery9"}`); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d want 409", rec.Code)
	}

	// me works with the cookie, 401 without.
	if rec := getAuth(srv, "/api/auth/me", c); rec.Code != http.StatusOK {
		t.Fatalf("me status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := getAuth(srv, "/api/auth/me"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("guest me status=%d want 401", rec.Code)
	}

	// Logout clears server-side and client-side sessions.
	rec = postAuth(srv, "/api/auth/logout", `{}`, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout status=%d", rec.Code)
	}
	if rec := getAuth(srv, "/api/auth/me", c); rec.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status=%d want 401", rec.Code)
	}

	// Login with the same credentials works; wrong password and unknown
	// username answer identically (no account enumeration).
	rec = postAuth(srv, "/api/auth/login", `{"username":"octagon_fan","password":"a-correct-horse-battery9"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", rec.Code, rec.Body.String())
	}
	loginCookies := sessionCookies(rec)
	if len(loginCookies) == 0 || (len(cookies) > 0 && loginCookies[0].Value == cookies[0].Value) {
		t.Fatal("login must rotate to a fresh session token")
	}
	badPass := postAuth(srv, "/api/auth/login", `{"username":"octagon_fan","password":"wrong-horse-battery9"}`)
	unknown := postAuth(srv, "/api/auth/login", `{"username":"nobody_here","password":"wrong-horse-battery9"}`)
	if badPass.Code != http.StatusUnauthorized || unknown.Code != http.StatusUnauthorized {
		t.Fatalf("login failures: known=%d unknown=%d, want both 401", badPass.Code, unknown.Code)
	}
	if badPass.Body.String() != unknown.Body.String() {
		t.Fatalf("login errors must be identical:\n%q\n%q", badPass.Body.String(), unknown.Body.String())
	}
}

func TestRegisterValidation(t *testing.T) {
	srv, _ := authTestServer()
	cases := []struct {
		name string
		body string
		want int
	}{
		{"bad username", `{"username":"no","password":"a-correct-horse-battery9"}`, http.StatusBadRequest},
		{"short password", `{"username":"abc","password":"short9"}`, http.StatusBadRequest},
		{"common password", `{"username":"abc","password":"password123"}`, http.StatusBadRequest},
		{"bad chars", `{"username":"has space","password":"a-correct-horse-battery9"}`, http.StatusBadRequest},
		{"unknown field", `{"username":"abc","password":"a-correct-horse-battery9","admin":true}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rec := postAuth(srv, "/api/auth/register", tc.body); rec.Code != tc.want {
				t.Fatalf("status=%d want %d body=%s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
	// Missing JSON content type is rejected (CSRF defense).
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"abc","password":"a-correct-horse-battery9"}`))
	rec := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d want 415", rec.Code)
	}
}

func TestAuthRateLimited(t *testing.T) {
	srv, _ := authTestServer()
	srv.authNameLimiter = NewRateLimiter(2, 10*time.Minute, srv.Clock.Now)
	srv.authIPLimiter = NewRateLimiter(100, 10*time.Minute, srv.Clock.Now)
	for i := 0; i < 2; i++ {
		rec := postAuth(srv, "/api/auth/login", `{"username":"rate_limited","password":"wrong-horse-battery9"}`)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status=%d want 401", i, rec.Code)
		}
	}
	if rec := postAuth(srv, "/api/auth/login", `{"username":"rate_limited","password":"wrong-horse-battery9"}`); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("rate-limited status=%d want 429 body=%s", rec.Code, rec.Body.String())
	}
}

func TestSignedInGuessesPersistAndRestore(t *testing.T) {
	srv, _ := authTestServer()
	rec := postAuth(srv, "/api/auth/register", `{"username":"cage_gamer","password":"a-correct-horse-battery9"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	cookie := sessionCookies(rec)[0]

	// Guest history starts empty.
	if rec := getAuth(srv, "/api/me/guesses", cookie); rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("fresh history = %d %q, want 200 []", rec.Code, rec.Body.String())
	}
	if rec := getAuth(srv, "/api/me/guesses"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("guest history status=%d want 401", rec.Code)
	}

	// A signed-in guess is recorded; guessing the same fighter twice stays
	// one row (idempotent).
	req := postGuess(t, "/api/game/guess", `{"fighter_id": 2}`)
	req.AddCookie(cookie)
	guessRec := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(guessRec, req)
	if guessRec.Code != http.StatusOK {
		t.Fatalf("guess: %d %s", guessRec.Code, guessRec.Body.String())
	}
	req = postGuess(t, "/api/game/guess", `{"fighter_id": 2}`)
	req.AddCookie(cookie)
	srv.Handler("").ServeHTTP(httptest.NewRecorder(), req)

	restored := getAuth(srv, "/api/me/guesses", cookie)
	if restored.Code != http.StatusOK {
		t.Fatalf("restore: %d %s", restored.Code, restored.Body.String())
	}
	var outcomes []domain.GuessOutcome
	if err := json.Unmarshal(restored.Body.Bytes(), &outcomes); err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 || outcomes[0].FighterID != 2 || outcomes[0].Correct {
		t.Fatalf("restored = %+v, want one incorrect guess for fighter 2", outcomes)
	}

	// Import merges local guest guesses (fighter 1 is the all-pool target).
	imp := postAuth(srv, "/api/me/import", `{"fighter_ids":[1,2]}`, cookie)
	if imp.Code != http.StatusOK {
		t.Fatalf("import: %d %s", imp.Code, imp.Body.String())
	}
	restored = getAuth(srv, "/api/me/guesses", cookie)
	if err := json.Unmarshal(restored.Body.Bytes(), &outcomes); err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 2 {
		t.Fatalf("after import want 2 guesses, got %+v", outcomes)
	}
	if !outcomes[1].Correct || outcomes[1].FighterID != 1 {
		t.Fatalf("imported order/outcome wrong: %+v", outcomes)
	}

	// Import of an unknown fighter is a 404, not a silent skip.
	if imp := postAuth(srv, "/api/me/import", `{"fighter_ids":[999]}`, cookie); imp.Code != http.StatusNotFound {
		t.Fatalf("unknown import status=%d want 404", imp.Code)
	}
	// Future dates are refused.
	if rec := getAuth(srv, "/api/me/guesses?date=2999-01-01", cookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("future date status=%d want 400", rec.Code)
	}
}

func TestGuessStillWorksForGuests(t *testing.T) {
	srv, _ := authTestServer()
	rec := serve(srv, http.MethodPost, "/api/game/guess", `{"fighter_id": 2}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("guest guess status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSessionCookieNeverLogged(t *testing.T) {
	srv, _ := authTestServer()
	rec := postAuth(srv, "/api/auth/register", `{"username":"quiet_fan","password":"a-correct-horse-battery9"}`)
	token := sessionCookies(rec)[0].Value
	if strings.Contains(rec.Body.String(), token) {
		t.Fatal("session token must never appear in a response body")
	}
	hashed := domain.HashSessionToken(token)
	if strings.Contains(rec.Body.String(), hashed) {
		t.Fatal("session token hash must never appear in a response body")
	}
}
