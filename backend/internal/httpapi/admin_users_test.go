package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"mmadle/backend/internal/store"
)

// adminUsersServer registers an admin plus a player and returns their
// session cookies, with a canned two-account listing.
func adminUsersServer(t *testing.T) (*Server, *fakeAuthStore, *http.Cookie, *http.Cookie) {
	t.Helper()
	srv, auth := adminTestServer()
	srv.AdminUsernames = map[string]bool{"cage_boss": true}
	srv.AdminMetrics.(*fakeAdminStore).users = store.UsersPage{
		Total: 2, Active: 2,
		Users: []store.AdminUser{
			{ID: 2, Username: "octagon_fan", Role: "player", IsActive: true, CreatedAt: "2026-09-17T10:00:00Z", Games: 3, GamesWon: 1},
			{ID: 1, Username: "cage_boss", Role: "admin", IsActive: true, CreatedAt: "2026-09-16T10:00:00Z", LastLoginAt: strPtr("2026-09-18T10:00:00Z")},
		},
	}
	admin := registerAs(t, srv, "cage_boss")
	player := registerAs(t, srv, "octagon_fan")
	return srv, auth, admin, player
}

func strPtr(s string) *string { return &s }

func TestAdminUsersGating(t *testing.T) {
	srv, _, admin, player := adminUsersServer(t)

	if rec := getAuth(srv, "/api/admin/users"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("guest list status=%d want 401", rec.Code)
	}
	if rec := getAuth(srv, "/api/admin/users", player); rec.Code != http.StatusNotFound {
		t.Fatalf("player list status=%d want 404", rec.Code)
	}
	if rec := postAuth(srv, "/api/admin/users/active", `{"user_id":2,"active":false}`, player); rec.Code != http.StatusNotFound {
		t.Fatalf("player toggle status=%d want 404", rec.Code)
	}

	rec := getAuth(srv, "/api/admin/users", admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page store.UsersPage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || page.Active != 2 || len(page.Users) != 2 {
		t.Fatalf("page = %+v", page)
	}
	if page.Users[0].Username != "octagon_fan" || page.Users[0].Games != 3 {
		t.Fatalf("first row = %+v", page.Users[0])
	}
	if page.Users[1].LastLoginAt == nil {
		t.Fatalf("admin row must carry last_login_at: %+v", page.Users[1])
	}
}

func TestAdminUsersValidation(t *testing.T) {
	srv, _, admin, _ := adminUsersServer(t)

	for path, want := range map[string]int{
		"/api/admin/users?limit=0":    http.StatusBadRequest,
		"/api/admin/users?limit=101":  http.StatusBadRequest,
		"/api/admin/users?limit=many": http.StatusBadRequest,
		"/api/admin/users?offset=-1":  http.StatusBadRequest,
	} {
		if rec := getAuth(srv, path, admin); rec.Code != want {
			t.Fatalf("%s: status=%d want %d body=%s", path, rec.Code, want, rec.Body.String())
		}
	}
	longQ := ""
	for i := 0; i < 65; i++ {
		longQ += "x"
	}
	if rec := getAuth(srv, "/api/admin/users?q="+longQ, admin); rec.Code != http.StatusBadRequest {
		t.Fatalf("long q status=%d want 400", rec.Code)
	}

	// Toggle validation: wrong method, bad bodies.
	if rec := getAuth(srv, "/api/admin/users/active", admin); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET toggle status=%d want 405", rec.Code)
	}
	for body, want := range map[string]int{
		`{"user_id":0,"active":false}`:   http.StatusBadRequest,
		`{"user_id":-3,"active":false}`:  http.StatusBadRequest,
		`{"active":false}`:               http.StatusBadRequest, // user_id 0
		`{"user_id":2}`:                  http.StatusOK,         // active defaults false
		`{"user_id":2,"admin":true}`:     http.StatusBadRequest, // unknown field
		`{bad`:                           http.StatusBadRequest,
	} {
		rec := postAuth(srv, "/api/admin/users/active", body, admin)
		if rec.Code != want {
			t.Fatalf("%s: status=%d want %d body=%s", body, rec.Code, want, rec.Body.String())
		}
	}
}

func TestAdminSetActiveFlow(t *testing.T) {
	srv, auth, admin, player := adminUsersServer(t)

	// Unknown account.
	if rec := postAuth(srv, "/api/admin/users/active", `{"user_id":999,"active":false}`, admin); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown user status=%d want 404", rec.Code)
	}
	// Self-deactivation is refused.
	if rec := postAuth(srv, "/api/admin/users/active", `{"user_id":1,"active":false}`, admin); rec.Code != http.StatusBadRequest {
		t.Fatalf("self toggle status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
	// Deactivate the player: their session dies immediately.
	if rec := postAuth(srv, "/api/admin/users/active", `{"user_id":2,"active":false}`, admin); rec.Code != http.StatusOK {
		t.Fatalf("toggle status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := getAuth(srv, "/api/auth/me", player); rec.Code != http.StatusUnauthorized {
		t.Fatalf("deactivated session status=%d want 401", rec.Code)
	}
	// ...and the login door stays shut without revealing why.
	if rec := postAuth(srv, "/api/auth/login", `{"username":"octagon_fan","password":"a-correct-horse-battery9"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("deactivated login status=%d want 401", rec.Code)
	}
	if auth.users[2].IsActive {
		t.Fatal("fake user must be inactive after deactivation")
	}
	// Reactivate: the account works again (fresh login required).
	if rec := postAuth(srv, "/api/admin/users/active", `{"user_id":2,"active":true}`, admin); rec.Code != http.StatusOK {
		t.Fatalf("reactivate status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := postAuth(srv, "/api/auth/login", `{"username":"octagon_fan","password":"a-correct-horse-battery9"}`); rec.Code != http.StatusOK {
		t.Fatalf("reactivated login status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestLoginStampsLastLogin(t *testing.T) {
	srv, auth := authTestServer()
	registerAs(t, srv, "octagon_fan")
	if auth.users[1].LastLoginAt == nil {
		t.Fatal("register must stamp last login")
	}
	u := auth.users[1]
	u.LastLoginAt = nil
	auth.users[1] = u
	if rec := postAuth(srv, "/api/auth/login", `{"username":"octagon_fan","password":"a-correct-horse-battery9"}`); rec.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", rec.Code, rec.Body.String())
	}
	if auth.users[1].LastLoginAt == nil {
		t.Fatal("login must stamp last login")
	}
}
