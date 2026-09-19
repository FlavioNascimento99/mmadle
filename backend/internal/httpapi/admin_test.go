package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mmadle/backend/internal/store"
)

// fakeAdminStore returns canned admin payloads for handler tests.
type fakeAdminStore struct {
	overview store.MetricsOverview
	users    store.UsersPage
}

func (f *fakeAdminStore) MetricsOverview(ctx context.Context, days int) (store.MetricsOverview, error) {
	out := f.overview
	out.Days = days
	return out, nil
}

func (f *fakeAdminStore) ListUsers(ctx context.Context, q string, limit, offset int) (store.UsersPage, error) {
	return f.users, nil
}

// adminTestServer wires game fixtures + in-memory auth + canned metrics.
func adminTestServer() (*Server, *fakeAuthStore) {
	srv, auth := authTestServer()
	srv.AdminMetrics = &fakeAdminStore{overview: store.MetricsOverview{
		Days: 30, GamesTotal: 10, GamesWon: 4, WinRate: 0.4,
		SignupsByDay: []store.DayCount{}, PlayersByDay: []store.DayCount{},
		GuessesByDay: []store.DayCount{}, ByPool: []store.PoolSplit{},
		TopFighters: []store.TopFighter{},
	}}
	return srv, auth
}

func registerAs(t *testing.T, srv *Server, username string) *http.Cookie {
	t.Helper()
	rec := postAuth(srv, "/api/auth/register",
		`{"username":`+strconvQuote(username)+`,"password":"a-correct-horse-battery9"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %s: %d %s", username, rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no session cookie")
	}
	return cookies[0]
}

func strconvQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func TestAdminPromotionOnRegister(t *testing.T) {
	srv, _ := adminTestServer()
	srv.AdminUsernames = map[string]bool{"cage_boss": true}

	cookie := registerAs(t, srv, "Cage_Boss") // allowlist matches case-insensitively
	rec := getAuth(srv, "/api/auth/me", cookie)
	var me map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me["role"] != "admin" {
		t.Fatalf("allowlisted account role = %v, want admin", me["role"])
	}

	// A regular player stays a player.
	plain := registerAs(t, srv, "octagon_fan")
	rec = getAuth(srv, "/api/auth/me", plain)
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me["role"] != "player" {
		t.Fatalf("regular account role = %v, want player", me["role"])
	}
}

func TestAdminOverviewGating(t *testing.T) {
	srv, _ := adminTestServer()
	srv.AdminUsernames = map[string]bool{"cage_boss": true}
	admin := registerAs(t, srv, "cage_boss")
	player := registerAs(t, srv, "octagon_fan")

	// Guests: 401 like /api/me/*.
	if rec := getAuth(srv, "/api/admin/metrics/overview"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("guest status=%d want 401", rec.Code)
	}
	// Players: 404 — the interface is not advertised to them.
	if rec := getAuth(srv, "/api/admin/metrics/overview", player); rec.Code != http.StatusNotFound {
		t.Fatalf("player status=%d want 404 body=%s", rec.Code, rec.Body.String())
	}
	// Admins: 200 with the overview payload.
	rec := getAuth(srv, "/api/admin/metrics/overview", admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out store.MetricsOverview
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.GamesTotal != 10 || out.WinRate != 0.4 {
		t.Fatalf("overview = %+v", out)
	}
	if rec := getAuth(srv, "/api/admin/metrics/overview?days=0", admin); rec.Code != http.StatusBadRequest {
		t.Fatalf("days=0 status=%d want 400", rec.Code)
	}
}

func TestAdminGatingPrecedesConfig(t *testing.T) {
	srv, _ := authTestServer()
	srv.AdminMetrics = nil // unconfigured backend
	srv.AdminUsernames = map[string]bool{"cage_boss": true}
	player := registerAs(t, srv, "octagon_fan")

	// Gating runs before the config check: players see 404, guests 401 —
	// never a 500 leaking that metrics are unconfigured.
	if rec := getAuth(srv, "/api/admin/metrics/overview", player); rec.Code != http.StatusNotFound {
		t.Fatalf("player status=%d want 404", rec.Code)
	}
	if rec := getAuth(srv, "/api/admin/metrics/overview"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("guest status=%d want 401", rec.Code)
	}
}

func TestCloudflareWorkersNeedsConfig(t *testing.T) {
	srv, _ := adminTestServer()
	srv.AdminUsernames = map[string]bool{"cage_boss": true}
	admin := registerAs(t, srv, "cage_boss")

	// Without secrets the endpoint says so explicitly (501), even for admins.
	if rec := getAuth(srv, "/api/admin/cloudflare/workers", admin); rec.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d want 501", rec.Code)
	}
	// Players still get 404 before config is even considered.
	player := registerAs(t, srv, "octagon_fan")
	if rec := getAuth(srv, "/api/admin/cloudflare/workers", player); rec.Code != http.StatusNotFound {
		t.Fatalf("player status=%d want 404", rec.Code)
	}
}

func TestCloudflareWorkersProxy(t *testing.T) {
	var sawAuth string
	var sawScript string
	graphql := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		var payload struct {
			Variables map[string]any `json:"variables"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if s, ok := payload.Variables["script"].(string); ok {
			sawScript = s
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"viewer": map[string]any{"accounts": []any{
			map[string]any{"workersInvocationsAdaptive": []any{
				map[string]any{
					"dimensions": map[string]any{"date": "2026-09-18"},
					"sum":        map[string]any{"requests": 100, "errors": 2, "subrequests": 5},
					"quantiles":  map[string]any{"cpuTimeP50": 1.5, "cpuTimeP99": 9.25},
				},
			}},
		}}}})
	}))
	defer graphql.Close()

	srv, _ := adminTestServer()
	srv.AdminUsernames = map[string]bool{"cage_boss": true}
	srv.CloudflareToken = "test-token"
	srv.CloudflareAccount = "test-account"
	srv.cfGraphQLURL = graphql.URL
	admin := registerAs(t, srv, "cage_boss")

	rec := getAuth(srv, "/api/admin/cloudflare/workers?days=7", admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if sawAuth != "Bearer test-token" {
		t.Fatalf("GraphQL Authorization = %q, want Bearer token", sawAuth)
	}
	if sawScript != "mmadle" {
		t.Fatalf("script filter = %q, want mmadle", sawScript)
	}
	var report struct {
		Requests int64 `json:"requests"`
		Errors   int64 `json:"errors"`
		ByDay    []struct {
			Date       string  `json:"date"`
			Requests   int64   `json:"requests"`
			CPUTimeP99 float64 `json:"cpu_time_p99_us"`
		} `json:"by_day"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Requests != 100 || report.Errors != 2 || len(report.ByDay) != 1 {
		t.Fatalf("report = %+v", report)
	}
	if report.ByDay[0].CPUTimeP99 != 9.25 {
		t.Fatalf("cpu p99 = %v", report.ByDay[0].CPUTimeP99)
	}
	if strings.Contains(rec.Body.String(), "test-token") {
		t.Fatal("API token must never appear in a response body")
	}
}
