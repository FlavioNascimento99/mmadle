package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mmadle/backend/internal/domain"
)

// seedSolves records count solves for a pool through the store, like the
// guess flow would (identities dedupe on repeat).
func seedSolves(t *testing.T, srv *Server, pool domain.Pool, day time.Time, identities ...string) {
	t.Helper()
	ctx := context.Background()
	for _, id := range identities {
		if err := srv.Store.RecordDailySolve(ctx, pool, day, id); err != nil {
			t.Fatalf("RecordDailySolve: %v", err)
		}
	}
}

func getStats(t *testing.T, srv *Server, path string, cookies ...*http.Cookie) (*httptest.ResponseRecorder, dailyStatsResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: status=%d body=%s", path, rec.Code, rec.Body.String())
	}
	var payload dailyStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("%s: decode: %v", path, err)
	}
	return rec, payload
}

func TestDailyStatsReturnsPerPoolSolvers(t *testing.T) {
	srv, day := testServer()
	seedSolves(t, srv, domain.PoolAll, day, "u:1", "u:2", "a:guest-1", "a:guest-1") // repeat dedupes
	seedSolves(t, srv, domain.PoolMen, day, "u:1")

	for path, want := range map[string]dailyStatsResponse{
		"/api/game/stats":          {Date: "2026-09-18", Pool: "all", Solvers: 3},
		"/api/game/stats?pool=all": {Date: "2026-09-18", Pool: "all", Solvers: 3},
		"/api/game/stats?pool=men": {Date: "2026-09-18", Pool: "men", Solvers: 1},
	} {
		_, payload := getStats(t, srv, path)
		if payload != want {
			t.Fatalf("%s: got %+v want %+v", path, payload, want)
		}
	}
}

func TestDailyStatsRejectsUnknownPool(t *testing.T) {
	srv, _ := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/game/stats?pool=women", nil)
	rec := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
}

func TestDailyStatsRejectsPost(t *testing.T) {
	srv, _ := testServer()
	req := httptest.NewRequest(http.MethodPost, "/api/game/stats", nil)
	rec := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want 405", rec.Code)
	}
}

func TestDailyStatsNeverLeaksTarget(t *testing.T) {
	srv, _ := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/game/stats", nil)
	rec := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, leak := range []string{"Target Fighter", "fighter_id", "target"} {
		if strings.Contains(body, leak) {
			t.Fatalf("stats response must not reveal target; contains %q: %s", leak, body)
		}
	}
}

// anonCookie extracts the anon identity cookie from a guess response.
func anonCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == anonCookieName {
			return c
		}
	}
	return nil
}

func postGuessWithCookies(t *testing.T, srv *Server, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := postGuess(t, "/api/game/guess", body)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(rec, req)
	return rec
}

func TestGuestCorrectGuessIssuesAnonIdentity(t *testing.T) {
	srv, _ := testServer()

	rec := postGuessWithCookies(t, srv, `{"fighter_id": 1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	cookie := anonCookie(rec)
	if cookie == nil || !validAnonToken(cookie.Value) {
		t.Fatalf("correct guest guess must issue an anon cookie, got %v", rec.Result().Cookies())
	}
	if !cookie.HttpOnly {
		t.Fatal("anon cookie must be HttpOnly")
	}
	if _, payload := getStats(t, srv, "/api/game/stats"); payload.Solvers != 1 {
		t.Fatalf("solvers=%d want 1", payload.Solvers)
	}

	// Same browser solving again (repeat correct submission) dedupes.
	rec2 := postGuessWithCookies(t, srv, `{"fighter_id": 1}`, cookie)
	if anonCookie(rec2) != nil {
		t.Fatal("established anon identity must not be re-issued")
	}
	if _, payload := getStats(t, srv, "/api/game/stats"); payload.Solvers != 1 {
		t.Fatalf("repeat solve dedupes: solvers=%d want 1", payload.Solvers)
	}

	// A different browser counts separately.
	rec3 := postGuessWithCookies(t, srv, `{"fighter_id": 1}`)
	if anonCookie(rec3) == nil {
		t.Fatal("new guest must get its own anon cookie")
	}
	if _, payload := getStats(t, srv, "/api/game/stats"); payload.Solvers != 2 {
		t.Fatalf("solvers=%d want 2", payload.Solvers)
	}
}

func TestGuestIncorrectGuessRecordsNothing(t *testing.T) {
	srv, _ := testServer()
	rec := postGuessWithCookies(t, srv, `{"fighter_id": 2}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if c := anonCookie(rec); c != nil {
		t.Fatalf("misses must not issue anon cookies, got %v", c)
	}
	if _, payload := getStats(t, srv, "/api/game/stats"); payload.Solvers != 0 {
		t.Fatalf("solvers=%d want 0", payload.Solvers)
	}
}

func TestForeignAnonCookieIsReplaced(t *testing.T) {
	srv, _ := testServer()
	forged := &http.Cookie{Name: anonCookieName, Value: "attacker-chosen-id"}
	rec := postGuessWithCookies(t, srv, `{"fighter_id": 1}`, forged)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	cookie := anonCookie(rec)
	if cookie == nil || cookie.Value == forged.Value || !validAnonToken(cookie.Value) {
		t.Fatalf("foreign anon value must be replaced with an issued token, got %v", rec.Result().Cookies())
	}
	if _, payload := getStats(t, srv, "/api/game/stats"); payload.Solvers != 1 {
		t.Fatalf("solvers=%d want 1", payload.Solvers)
	}
}

// testServerWithAuth mirrors testServer but signs "solver" in, returning the
// session cookie for authenticated requests.
func testServerWithAuth(t *testing.T) (*Server, *http.Cookie) {
	t.Helper()
	srv, day := testServer()
	auth := newFakeAuthStore()
	ctx := context.Background()
	user, err := auth.CreateUser(ctx, "solver", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	const rawToken = "solver-session-token"
	if err := auth.CreateSession(ctx, domain.HashSessionToken(rawToken), user.ID, day.Add(48*time.Hour)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	srv.Auth = auth
	return srv, &http.Cookie{Name: sessionCookieName, Value: rawToken}
}

func TestSignedInCorrectGuessUsesAccountIdentity(t *testing.T) {
	srv, session := testServerWithAuth(t)
	rec := postGuessWithCookies(t, srv, `{"fighter_id": 1}`, session)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if c := anonCookie(rec); c != nil {
		t.Fatal("signed-in solves must not issue anon cookies")
	}
	if _, payload := getStats(t, srv, "/api/game/stats"); payload.Solvers != 1 {
		t.Fatalf("solvers=%d want 1", payload.Solvers)
	}
	// Solving the other pool counts there, still once per account+pool.
	rec = postGuessWithCookies(t, srv, `{"fighter_id": 3, "pool": "men"}`, session)
	if rec.Code != http.StatusOK {
		t.Fatalf("men guess status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, payload := getStats(t, srv, "/api/game/stats?pool=men"); payload.Solvers != 1 {
		t.Fatalf("men solvers=%d want 1", payload.Solvers)
	}
	if _, payload := getStats(t, srv, "/api/game/stats"); payload.Solvers != 1 {
		t.Fatalf("all solvers=%d want 1", payload.Solvers)
	}
}
