package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mmadle/backend/internal/domain"
	"mmadle/backend/internal/store"
)

// fakeStore is an in-memory FighterStore for handler tests.
type fakeStore struct {
	views  map[int]domain.FighterView
	ids    []int
	search []store.SearchResult
	pingOK bool
}

func (f *fakeStore) GameFighterIDs(ctx context.Context) ([]int, error) {
	return append([]int{}, f.ids...), nil
}
func (f *fakeStore) FighterView(ctx context.Context, id int) (domain.FighterView, error) {
	v, ok := f.views[id]
	if !ok {
		return domain.FighterView{}, store.ErrNotFound
	}
	return v, nil
}
func (f *fakeStore) SearchFighters(ctx context.Context, q string, limit int) ([]store.SearchResult, error) {
	return f.search, nil
}
func (f *fakeStore) Ping(ctx context.Context) error {
	if !f.pingOK {
		return errors.New("db down")
	}
	return nil
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func dob(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func testServer() (*Server, time.Time) {
	gameDate, _ := time.Parse("2006-01-02", "2026-09-18")
	views := map[int]domain.FighterView{
		1: {ID: 1, Name: "Target Fighter", DateOfBirth: dob("1987-07-07"), HeightCm: 185,
			Nationality: "Brazil", Wins: 23, Losses: 5, Draws: 0, Division: "Lightweight",
			LastEvent: "UFC 320", LastEventDate: dob("2025-10-04")},
		2: {ID: 2, Name: "Guesser Fighter", DateOfBirth: dob("1995-05-05"), HeightCm: 180,
			Nationality: "USA", Wins: 18, Losses: 4, Draws: 0, Division: "Lightweight",
			LastEvent: "UFC 319", LastEventDate: dob("2025-08-16")},
	}
	st := &fakeStore{
		views: views,
		ids:   []int{1, 2},
		search: []store.SearchResult{
			{ID: 2, Name: "Guesser Fighter", Nationality: "USA"},
		},
		pingOK: true,
	}
	srv := New(st, fixedSelector{target: 1}, fixedClock{t: gameDate}, nil)
	return srv, gameDate
}

// fixedSelector pins the daily target for deterministic handler tests.
type fixedSelector struct{ target int }

func (f fixedSelector) Select(_ time.Time, ids []int) (int, error) {
	for _, id := range ids {
		if id == f.target {
			return id, nil
		}
	}
	return 0, domain.ErrNoFighters
}

func TestTodayDoesNotLeakTarget(t *testing.T) {
	srv, _ := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/game/today", nil)
	rec := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, leak := range []string{"Target Fighter", "fighter_id", "target"} {
		if strings.Contains(body, leak) {
			t.Fatalf("today response must not reveal target; contains %q: %s", leak, body)
		}
	}
	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["date"] != "2026-09-18" {
		t.Fatalf("date wrong: %v", payload)
	}
}

func TestSearchValidation(t *testing.T) {
	srv, _ := testServer()
	for path, want := range map[string]int{
		"/api/fighters/search":       http.StatusBadRequest, // missing q
		"/api/fighters/search?q=":    http.StatusBadRequest,
		"/api/fighters/search?q=con": http.StatusOK,
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.Handler("").ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s: status=%d want %d", path, rec.Code, want)
		}
	}
}

func TestGuessIncorrectAndCorrect(t *testing.T) {
	srv, _ := testServer()
	h := srv.Handler("")

	// Incorrect guess (fighter 2 vs target 1).
	req := httptest.NewRequest(http.MethodPost, "/api/game/guess",
		strings.NewReader(`{"fighter_id": 2}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("incorrect guess status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out domain.GuessOutcome
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Correct {
		t.Fatal("must be incorrect")
	}
	if out.Results.Height.Comparison != domain.ComparisonHigher {
		t.Fatalf("height cmp=%v", out.Results.Height.Comparison)
	}
	if out.FighterName != "Guesser Fighter" {
		t.Fatal("response must identify the GUESSED fighter")
	}
	if strings.Contains(rec.Body.String(), "Target Fighter") {
		t.Fatal("response must never contain the target fighter")
	}

	// Correct guess (fighter 1).
	req = httptest.NewRequest(http.MethodPost, "/api/game/guess",
		strings.NewReader(`{"fighter_id": 1}`))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("correct guess status=%d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.Correct {
		t.Fatalf("must be correct: %+v", out)
	}
}

func TestGuessErrors(t *testing.T) {
	srv, _ := testServer()
	h := srv.Handler("")
	cases := []struct {
		name string
		body string
		want int
	}{
		{"malformed json", "{bad", http.StatusBadRequest},
		{"empty body", "", http.StatusBadRequest},
		{"zero id", `{"fighter_id": 0}`, http.StatusBadRequest},
		{"negative id", `{"fighter_id": -5}`, http.StatusBadRequest},
		{"unknown field", `{"fighter_id": 1, "admin": true}`, http.StatusBadRequest},
		{"unknown fighter", `{"fighter_id": 999}`, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/game/guess",
				strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status=%d want %d body=%s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}

	// Wrong method.
	req := httptest.NewRequest(http.MethodGet, "/api/game/guess", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET guess status=%d want 405", rec.Code)
	}
}

func TestHealth(t *testing.T) {
	srv, _ := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status=%d", rec.Code)
	}
}
