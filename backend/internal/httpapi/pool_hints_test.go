package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mmadle/backend/internal/domain"
)

func serve(srv *Server, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler("").ServeHTTP(rec, req)
	return rec
}

func TestUnknownPoolIsRejected(t *testing.T) {
	srv, _ := testServer()
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/fighters?pool=women", ""},
		{http.MethodGet, "/api/fighters/search?q=con&pool=x", ""},
		{http.MethodGet, "/api/game/hints?guesses=3&pool=x", ""},
		{http.MethodPost, "/api/game/guess", `{"fighter_id": 2, "pool": "x"}`},
	} {
		if rec := serve(srv, tc.method, tc.path, tc.body); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s %s: status=%d want 400", tc.method, tc.path, rec.Code)
		}
	}
}

func TestRosterAndSearchForwardPool(t *testing.T) {
	for _, tc := range []struct {
		path string
		want domain.Pool
	}{
		{"/api/fighters", domain.PoolAll},
		{"/api/fighters?pool=men", domain.PoolMen},
		{"/api/fighters/search?q=con&pool=men", domain.PoolMen},
	} {
		srv, _ := testServer()
		if rec := serve(srv, http.MethodGet, tc.path, ""); rec.Code != http.StatusOK {
			t.Fatalf("%s: status=%d", tc.path, rec.Code)
		}
		if got := srv.Store.(*fakeStore).gotPool; got != tc.want {
			t.Fatalf("%s: store got pool %q want %q", tc.path, got, tc.want)
		}
	}
}

func TestGuessUsesPoolTarget(t *testing.T) {
	srv, _ := testServer()
	decode := func(rec *httptest.ResponseRecorder) domain.GuessOutcome {
		t.Helper()
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var out domain.GuessOutcome
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	if !decode(serve(srv, http.MethodPost, "/api/game/guess", `{"fighter_id": 3, "pool": "men"}`)).Correct {
		t.Fatal("fighter 3 is the men's target")
	}
	if decode(serve(srv, http.MethodPost, "/api/game/guess", `{"fighter_id": 3}`)).Correct {
		t.Fatal("fighter 3 is not the all-fighters target")
	}
}

func TestHints(t *testing.T) {
	srv, _ := testServer()
	rec := serve(srv, http.MethodGet, "/api/game/hints?guesses=3", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out domain.HintsResult
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Hints) != 1 || out.Hints[0].Value != "Brazil" {
		t.Fatalf("want the all-fighters target's nationality, got %+v", out.Hints)
	}
	if out.NextAt == nil || *out.NextAt != 5 {
		t.Fatalf("next_at=%v want 5", out.NextAt)
	}

	rec = serve(srv, http.MethodGet, "/api/game/hints?guesses=9&pool=men", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Hints[0].Value != "Georgia" {
		t.Fatalf("men pool must hint the men's target, got %+v", out.Hints)
	}
	if strings.Contains(rec.Body.String(), "Men Target") {
		t.Fatal("hints must never reveal the target's full name")
	}
}

func TestHintsValidation(t *testing.T) {
	srv, _ := testServer()
	for _, q := range []string{"", "?guesses=", "?guesses=abc", "?guesses=-1", "?guesses=101"} {
		if rec := serve(srv, http.MethodGet, "/api/game/hints"+q, ""); rec.Code != http.StatusBadRequest {
			t.Fatalf("hints%s: status=%d want 400", q, rec.Code)
		}
	}
	if rec := serve(srv, http.MethodPost, "/api/game/hints?guesses=3", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST hints: status=%d want 405", rec.Code)
	}
}
