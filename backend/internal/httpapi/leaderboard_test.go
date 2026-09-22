package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"mmadle/backend/internal/domain"
)

// fakeLeaderboardStore is an in-memory LeaderboardStore for handler tests.
type fakeLeaderboardStore struct {
	mu     sync.Mutex
	games  []domain.LeaderboardGame
	optIns map[int64]bool
	err    error
}

func (f *fakeLeaderboardStore) ListLeaderboardGames(ctx context.Context, pool domain.Pool, today string) ([]domain.LeaderboardGame, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return append([]domain.LeaderboardGame{}, f.games...), nil
}

func (f *fakeLeaderboardStore) SetLeaderboardOptIn(ctx context.Context, userID int64, optIn bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.optIns[userID] = optIn
	return nil
}

func (f *fakeLeaderboardStore) LeaderboardOptIn(ctx context.Context, userID int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return false, f.err
	}
	return f.optIns[userID], nil
}

func leaderboardTestServer(games []domain.LeaderboardGame) (*Server, *fakeLeaderboardStore) {
	srv, _ := authTestServer()
	lb := &fakeLeaderboardStore{games: games, optIns: map[int64]bool{}}
	srv.Leaderboard = lb
	return srv, lb
}

func lbGames() []domain.LeaderboardGame {
	return []domain.LeaderboardGame{
		{UserID: 1, Username: "alpha", Date: "2026-09-16", Guesses: 1, Won: true},
		{UserID: 2, Username: "beta", Date: "2026-09-16", Guesses: 6, Won: true},
		{UserID: 3, Username: "zeta", Date: "2026-09-16", Guesses: 2, Won: false},
	}
}

func TestLeaderboardPublicOrdering(t *testing.T) {
	srv, _ := leaderboardTestServer(lbGames())
	rec := serve(srv, http.MethodGet, "/api/leaderboard?pool=all", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out leaderboardResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Pool != "all" || out.Total != 3 {
		t.Fatalf("pool/total = %s/%d, want all/3", out.Pool, out.Total)
	}
	if len(out.Rankings) != 3 {
		t.Fatalf("rankings = %+v", out.Rankings)
	}
	if out.Rankings[0].Username != "alpha" || out.Rankings[0].Score != 11 || out.Rankings[0].Rank != 1 {
		t.Fatalf("rank1 = %+v", out.Rankings[0])
	}
	if out.Rankings[1].Username != "beta" || out.Rankings[1].Score != 6 {
		t.Fatalf("rank2 = %+v", out.Rankings[1])
	}
	if out.Rankings[2].Username != "zeta" || out.Rankings[2].Score != 0 {
		t.Fatalf("rank3 = %+v", out.Rankings[2])
	}
	// Never leak account ids to the public payload.
	if strings.Contains(rec.Body.String(), `"user_id"`) {
		t.Fatalf("leaderboard must not expose user ids: %s", rec.Body.String())
	}
	if out.My != nil {
		t.Fatalf("guest must not get a my entry: %+v", out.My)
	}
}

func TestLeaderboardPagination(t *testing.T) {
	srv, _ := leaderboardTestServer(lbGames())
	rec := serve(srv, http.MethodGet, "/api/leaderboard?limit=2&offset=1", "")
	var out leaderboardResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Rankings) != 2 || out.Total != 3 {
		t.Fatalf("page = %+v", out.Rankings)
	}
	if out.Rankings[0].Username != "beta" || out.Rankings[1].Username != "zeta" {
		t.Fatalf("offset page = %+v", out.Rankings)
	}
	for path, want := range map[string]int{
		"/api/leaderboard?limit=0":    http.StatusBadRequest,
		"/api/leaderboard?limit=101":  http.StatusBadRequest,
		"/api/leaderboard?offset=-1":  http.StatusBadRequest,
		"/api/leaderboard?pool=champ": http.StatusBadRequest,
	} {
		if rec := serve(srv, http.MethodGet, path, ""); rec.Code != want {
			t.Fatalf("GET %s status=%d want %d", path, rec.Code, want)
		}
	}
	if rec := serve(srv, http.MethodPost, "/api/leaderboard", "{}"); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d want 405", rec.Code)
	}
}

func TestLeaderboardEmptyEncodesArray(t *testing.T) {
	srv, _ := leaderboardTestServer(nil)
	rec := serve(srv, http.MethodGet, "/api/leaderboard", "")
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw["rankings"]) != "[]" {
		t.Fatalf("rankings must encode as [], got %s body=%s", raw["rankings"], rec.Body.String())
	}
	if string(raw["total"]) != "0" {
		t.Fatalf("total = %s, want 0", raw["total"])
	}
}

func TestLeaderboardMyEntry(t *testing.T) {
	srv, _ := leaderboardTestServer(lbGames())
	rec := postAuth(srv, "/api/auth/register", `{"username":"alpha","password":"a-correct-horse-battery9"}`)
	cookie := rec.Result().Cookies()[0]

	// Opted out by default: state visible, entry null.
	got := getAuth(srv, "/api/leaderboard", cookie)
	var before leaderboardResponse
	if err := json.Unmarshal(got.Body.Bytes(), &before); err != nil {
		t.Fatal(err)
	}
	if before.My == nil || before.My.OptIn || before.My.Entry != nil {
		t.Fatalf("default my = %+v, want entry null + opt_in=false", before.My)
	}

	// Toggle on, then the entry appears at the player's rank.
	tog := postAuth(srv, "/api/me/leaderboard", `{"opt_in": true}`, cookie)
	if tog.Code != http.StatusOK {
		t.Fatalf("toggle status=%d body=%s", tog.Code, tog.Body.String())
	}
	got = getAuth(srv, "/api/leaderboard", cookie)
	var after leaderboardResponse
	if err := json.Unmarshal(got.Body.Bytes(), &after); err != nil {
		t.Fatal(err)
	}
	if after.My == nil || !after.My.OptIn || after.My.Entry == nil {
		t.Fatalf("opted-in my = %+v", after.My)
	}
	if after.My.Entry.Username != "alpha" || after.My.Entry.Rank != 1 {
		t.Fatalf("my entry = %+v", after.My.Entry)
	}

	// Opting out hides the entry again.
	postAuth(srv, "/api/me/leaderboard", `{"opt_in": false}`, cookie)
	got = getAuth(srv, "/api/leaderboard", cookie)
	var off leaderboardResponse
	if err := json.Unmarshal(got.Body.Bytes(), &off); err != nil {
		t.Fatal(err)
	}
	if off.My == nil || off.My.OptIn || off.My.Entry != nil {
		t.Fatalf("opted-out my = %+v", off.My)
	}
}

func TestLeaderboardOptInGuardrails(t *testing.T) {
	srv, _ := leaderboardTestServer(lbGames())
	// Guests are refused; toggling needs a session.
	if rec := postAuth(srv, "/api/me/leaderboard", `{"opt_in": true}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("guest toggle status=%d want 401", rec.Code)
	}
	rec := postAuth(srv, "/api/auth/register", `{"username":"toggler","password":"a-correct-horse-battery9"}`)
	cookie := rec.Result().Cookies()[0]
	if rec := postAuth(srv, "/api/me/leaderboard", `{"opt_in": true, "extra": 1}`, cookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status=%d want 400", rec.Code)
	}
	if rec := postAuth(srv, "/api/me/leaderboard", `{"opt_in": "yes"}`, cookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("wrong type status=%d want 400", rec.Code)
	}
	if rec := postAuth(srv, "/api/me/leaderboard", `{}`, cookie); rec.Code != http.StatusOK {
		t.Fatalf("missing opt_in defaults false, status=%d", rec.Code)
	}
	if rec := getAuth(srv, "/api/me/leaderboard", cookie); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET toggle status=%d want 405", rec.Code)
	}
	// Missing JSON content type is rejected (CSRF defense).
	req := httptest.NewRequest(http.MethodPost, "/api/me/leaderboard", strings.NewReader(`{"opt_in": true}`))
	rec2 := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(rec2, req)
	if rec2.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("no content-type status=%d want 415", rec2.Code)
	}
}
