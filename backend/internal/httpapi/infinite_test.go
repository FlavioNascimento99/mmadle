package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"mmadle/backend/internal/domain"
	"mmadle/backend/internal/store"
)

// fakeInfiniteStore is an in-memory InfiniteStore mirroring the SQL
// transition rules (lives, solve/death, expiry classification).
type fakeInfiniteStore struct {
	mu      sync.Mutex
	rounds  map[string]*store.InfiniteRound
	records map[string]*store.InfiniteRecord
}

func newFakeInfiniteStore() *fakeInfiniteStore {
	return &fakeInfiniteStore{rounds: map[string]*store.InfiniteRound{}, records: map[string]*store.InfiniteRecord{}}
}

func (f *fakeInfiniteStore) CreateRound(ctx context.Context, id string, userID *int64, pool domain.Pool, targetID int, expiresAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rounds[id] = &store.InfiniteRound{ID: id, UserID: userID, Pool: pool, TargetID: targetID,
		LivesLeft: domain.MaxLives, Status: "playing", ExpiresAt: expiresAt}
	return nil
}

func (f *fakeInfiniteStore) FindRound(ctx context.Context, id string, now time.Time) (store.InfiniteRound, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rounds[id]
	if !ok || !r.ExpiresAt.After(now) {
		return store.InfiniteRound{}, store.ErrNoRound
	}
	return *r, nil
}

func (f *fakeInfiniteStore) ApplyRoundGuess(ctx context.Context, id string, correct bool, now time.Time) (store.InfiniteRound, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rounds[id]
	if !ok || !r.ExpiresAt.After(now) {
		return store.InfiniteRound{}, store.ErrNoRound
	}
	if r.Status != "playing" {
		return store.InfiniteRound{}, store.ErrRoundOver
	}
	tr := domain.ApplyInfiniteGuess(r.LivesLeft, correct)
	r.LivesLeft, r.Guesses = tr.LivesLeft, r.Guesses+1
	switch {
	case tr.Solved:
		r.Status = "solved"
	case tr.Dead:
		r.Status = "dead"
	}
	return *r, nil
}

func recordKey(userID int64, pool domain.Pool) string {
	return strings.Join([]string{itoa64(userID), string(pool)}, "|")
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}

func (f *fakeInfiniteStore) FetchRecord(ctx context.Context, userID int64, pool domain.Pool) (store.InfiniteRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if rec, ok := f.records[recordKey(userID, pool)]; ok {
		return *rec, nil
	}
	return store.InfiniteRecord{}, nil
}

func (f *fakeInfiniteStore) NoteSolve(ctx context.Context, userID int64, pool domain.Pool) (store.InfiniteRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rec, ok := f.records[recordKey(userID, pool)]
	if !ok {
		rec = &store.InfiniteRecord{}
		f.records[recordKey(userID, pool)] = rec
	}
	rec.CurrentStreak++
	if rec.CurrentStreak > rec.BestStreak {
		rec.BestStreak = rec.CurrentStreak
	}
	return *rec, nil
}

func (f *fakeInfiniteStore) NoteDeath(ctx context.Context, userID int64, pool domain.Pool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	rec, ok := f.records[recordKey(userID, pool)]
	if !ok {
		rec = &store.InfiniteRecord{}
		f.records[recordKey(userID, pool)] = rec
	}
	rec.CurrentStreak = 0
	return nil
}

// infiniteTestServer wires game fixtures + in-memory auth + infinite stores.
func infiniteTestServer() (*Server, *fakeAuthStore, *fakeInfiniteStore) {
	srv, auth := authTestServer()
	inf := newFakeInfiniteStore()
	srv.Infinite = inf
	return srv, auth, inf
}

// seedRound inserts a live round with a pinned target for deterministic tests.
func seedRound(t *testing.T, inf *fakeInfiniteStore, id string, target int) {
	t.Helper()
	if err := inf.CreateRound(context.Background(), id, nil, domain.PoolAll, target, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
}

const (
	testRoundA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testRoundB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func TestCreateRound(t *testing.T) {
	srv, _, _ := infiniteTestServer()

	rec := postAuth(srv, "/api/infinite/rounds", `{"pool":"men"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	id, _ := out["round_id"].(string)
	if err := domain.ValidateRoundID(id); err != nil {
		t.Fatalf("round_id %q invalid", id)
	}
	if out["lives"] != float64(domain.MaxLives) || out["pool"] != "men" {
		t.Fatalf("round = %v", out)
	}
	for _, leak := range []string{"Target Fighter", "Guesser Fighter", "target"} {
		if strings.Contains(rec.Body.String(), leak) {
			t.Fatalf("round response must not leak the target: %q", leak)
		}
	}

	if rec := postAuth(srv, "/api/infinite/rounds", `{"pool":"women"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad pool status=%d want 400", rec.Code)
	}
}

func TestInfiniteGuessLivesAndDeath(t *testing.T) {
	srv, _, inf := infiniteTestServer()
	seedRound(t, inf, testRoundA, 2) // target: Guesser Fighter

	guess := func(fighter int) (int, map[string]any) {
		t.Helper()
		rec := postAuth(srv, "/api/infinite/guess",
			`{"round_id":"`+testRoundA+`","fighter_id":`+itoa64(int64(fighter))+`}`)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}

	// Four misses cost four lives; nothing about the streak for guests.
	for i, want := range []float64{4, 3, 2, 1} {
		code, out := guess(1)
		if code != http.StatusOK {
			t.Fatalf("miss %d: status=%d", i, code)
		}
		if out["lives_left"] != want || out["solved"] != false || out["round_over"] != false {
			t.Fatalf("miss %d = %v, want lives=%v", i, out, want)
		}
		if _, hasStreak := out["streak"]; hasStreak {
			t.Fatal("guests must not receive streak fields")
		}
		if _, hasAnswer := out["answer"]; hasAnswer {
			t.Fatal("live rounds must not reveal the answer")
		}
	}

	// Fifth miss kills the round and reveals the answer.
	code, out := guess(1)
	if code != http.StatusOK {
		t.Fatalf("killing miss: status=%d", code)
	}
	if out["lives_left"] != float64(0) || out["round_over"] != true {
		t.Fatalf("death = %v", out)
	}
	answer, _ := out["answer"].(map[string]any)
	if answer == nil || answer["name"] != "Guesser Fighter" {
		t.Fatalf("death must reveal the answer, got %v", out["answer"])
	}

	// Replaying a finished round is gone.
	if code, _ := guess(1); code != http.StatusGone {
		t.Fatalf("replay status=%d want 410", code)
	}
}

func TestInfiniteSolveAndStreak(t *testing.T) {
	srv, _, inf := infiniteTestServer()
	cookie := registerAs(t, srv, "survivor")

	seedRound(t, inf, testRoundA, 2)
	// Bind the round to the player like handleCreateRound would.
	inf.mu.Lock()
	inf.rounds[testRoundA].UserID = userIDOf(t, srv, cookie)
	inf.mu.Unlock()

	rec := postAuth(srv, "/api/infinite/guess",
		`{"round_id":"`+testRoundA+`","fighter_id":2}`, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("solve status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["solved"] != true || out["lives_left"] != float64(domain.MaxLives) {
		t.Fatalf("solve must not cost a life: %v", out)
	}
	if out["streak"] != float64(1) || out["best"] != float64(1) || out["new_best"] != true {
		t.Fatalf("first solve streak = %v", out)
	}

	// Second solve extends the run.
	seedRound(t, inf, testRoundB, 1)
	inf.mu.Lock()
	inf.rounds[testRoundB].UserID = userIDOf(t, srv, cookie)
	inf.mu.Unlock()
	rec = postAuth(srv, "/api/infinite/guess",
		`{"round_id":"`+testRoundB+`","fighter_id":1}`, cookie)
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["streak"] != float64(2) || out["best"] != float64(2) {
		t.Fatalf("second solve streak = %v", out)
	}

	// Record endpoint agrees; a death resets current but keeps best.
	got := getAuth(srv, "/api/me/infinite/record?pool=all", cookie)
	var record map[string]any
	_ = json.Unmarshal(got.Body.Bytes(), &record)
	if record["best_streak"] != float64(2) || record["current_streak"] != float64(2) {
		t.Fatalf("record = %v", record)
	}
}

func TestInfiniteDeathResetsRun(t *testing.T) {
	srv, _, inf := infiniteTestServer()
	cookie := registerAs(t, srv, "doomed")

	seedRound(t, inf, testRoundA, 2)
	inf.mu.Lock()
	inf.rounds[testRoundA].UserID = userIDOf(t, srv, cookie)
	inf.mu.Unlock()
	// One solve then a fresh lethal round.
	if rec := postAuth(srv, "/api/infinite/guess", `{"round_id":"`+testRoundA+`","fighter_id":2}`, cookie); rec.Code != http.StatusOK {
		t.Fatalf("solve: %d", rec.Code)
	}
	seedRound(t, inf, testRoundB, 2)
	inf.mu.Lock()
	inf.rounds[testRoundB].UserID = userIDOf(t, srv, cookie)
	inf.mu.Unlock()
	for i := 0; i < domain.MaxLives; i++ {
		rec := postAuth(srv, "/api/infinite/guess", `{"round_id":"`+testRoundB+`","fighter_id":1}`, cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("miss %d: %d %s", i, rec.Code, rec.Body.String())
		}
	}
	got := getAuth(srv, "/api/me/infinite/record?pool=all", cookie)
	var record map[string]any
	_ = json.Unmarshal(got.Body.Bytes(), &record)
	if record["best_streak"] != float64(1) || record["current_streak"] != float64(0) {
		t.Fatalf("after death = %v, want best kept, current reset", record)
	}
}

func userIDOf(t *testing.T, srv *Server, cookie *http.Cookie) *int64 {
	t.Helper()
	rec := getAuth(srv, "/api/auth/me", cookie)
	var me struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	return &me.ID
}

func TestInfiniteErrors(t *testing.T) {
	srv, _, inf := infiniteTestServer()
	seedRound(t, inf, testRoundA, 2)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"malformed round", `{"round_id":"xyz","fighter_id":1}`, http.StatusBadRequest},
		{"short round", `{"round_id":"abc","fighter_id":1}`, http.StatusBadRequest},
		{"unknown round", `{"round_id":"cccccccccccccccccccccccccccccccc","fighter_id":1}`, http.StatusNotFound},
		{"zero fighter", `{"round_id":"` + testRoundA + `","fighter_id":0}`, http.StatusBadRequest},
		{"unknown fighter", `{"round_id":"` + testRoundA + `","fighter_id":999}`, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rec := postAuth(srv, "/api/infinite/guess", tc.body); rec.Code != tc.want {
				t.Fatalf("status=%d want %d body=%s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}

	// No content type: rejected like every other state-changing route.
	req := httptest.NewRequest(http.MethodPost, "/api/infinite/guess",
		strings.NewReader(`{"round_id":"`+testRoundA+`","fighter_id":1}`))
	rec := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d want 415", rec.Code)
	}

	// Record needs auth and a valid pool.
	if rec := getAuth(srv, "/api/me/infinite/record?pool=all"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("guest record status=%d want 401", rec.Code)
	}
	cookie := registerAs(t, srv, "rec_check")
	if rec := getAuth(srv, "/api/me/infinite/record?pool=women", cookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad pool status=%d want 400", rec.Code)
	}
	if rec := getAuth(srv, "/api/me/infinite/record?pool=men", cookie); rec.Code != http.StatusOK {
		t.Fatalf("empty record status=%d", rec.Code)
	}
}

func TestInfiniteNoLeakBeforeDeath(t *testing.T) {
	srv, _, inf := infiniteTestServer()
	seedRound(t, inf, testRoundA, 1) // target: Target Fighter
	rec := postAuth(srv, "/api/infinite/guess", `{"round_id":"`+testRoundA+`","fighter_id":2}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "Target Fighter") {
		t.Fatal("live guess response must never contain the target")
	}
}
