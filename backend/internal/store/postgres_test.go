package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"mmadle/backend/internal/domain"
)

// Integration tests against a real PostgreSQL. Run with:
//   TEST_DATABASE_URL=postgres://... go test -run TestPostgres -v ./internal/store/
// Skipped otherwise (unit tests + CI service run them).

func testPool(t *testing.T) *Postgres {
	t.Helper()
	url := postgresTestURL()
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, err := NewPostgres(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(p.Close)
	return p
}

func postgresTestURL() string {
	// env lookup without os import aliasing issues; see below.
	return lookupTestDBURL()
}

func TestPostgres_GameFighterIDs(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	ids, err := p.GameFighterIDs(ctx, domain.PoolAll)
	if err != nil {
		t.Fatalf("GameFighterIDs: %v", err)
	}
	if len(ids) == 0 {
		t.Fatal("expected seeded fighters")
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] < ids[i-1] {
			t.Fatal("ids must be in stable ascending order for deterministic selection")
		}
	}
}

func TestPostgres_SearchCaseInsensitive(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	lower, err := p.SearchFighters(ctx, domain.PoolAll, "conor", 8)
	if err != nil {
		t.Fatal(err)
	}
	upper, err := p.SearchFighters(ctx, domain.PoolAll, "CONOR", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(lower) == 0 || len(upper) == 0 {
		t.Fatal("expected search hits for seeded fighter")
	}
	if len(lower) != len(upper) {
		t.Fatal("search must be case-insensitive")
	}
	partial, err := p.SearchFighters(ctx, domain.PoolAll, "mcg", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(partial) == 0 {
		t.Fatal("expected partial-name match")
	}
}

func TestPostgres_FighterViewDerivesDivisionAndEvent(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	ids, err := p.GameFighterIDs(ctx, domain.PoolAll)
	if err != nil {
		t.Fatal(err)
	}
	v, err := p.FighterView(ctx, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	if v.Division == "" {
		t.Fatal("division must be derived from current fighter_divisions row")
	}
	if v.LastEvent == "" {
		t.Fatal("last event must be derived from fights/events")
	}
}

func TestPostgres_ListFightersSortedByName(t *testing.T) {
	p := testPool(t)
	roster, err := p.ListFighters(context.Background(), domain.PoolAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(roster) == 0 {
		t.Fatal("expected seeded fighters")
	}
	for i := 1; i < len(roster); i++ {
		if roster[i].Name < roster[i-1].Name {
			t.Fatalf("roster must be sorted by name: %q before %q", roster[i-1].Name, roster[i].Name)
		}
	}
}

func TestPostgres_SeededPhotosCarryCredit(t *testing.T) {
	p := testPool(t)
	roster, err := p.ListFighters(context.Background(), domain.PoolAll)
	if err != nil {
		t.Fatal(err)
	}
	withPhoto := 0
	for _, f := range roster {
		if f.PhotoURL == nil {
			continue
		}
		withPhoto++
		if f.PhotoCredit == nil || *f.PhotoCredit == "" {
			t.Fatalf("%s has a photo without attribution", f.Name)
		}
	}
	if withPhoto == 0 {
		t.Fatal("expected seeded fighters with photos")
	}
}

func TestPostgres_WholeRosterIsGameEligible(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	roster, err := p.ListFighters(ctx, domain.PoolAll)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := p.GameFighterIDs(ctx, domain.PoolAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != len(roster) {
		t.Fatalf("every seeded fighter needs a current division and a fight: %d eligible of %d", len(ids), len(roster))
	}
}

func TestPostgres_MenPoolExcludesWomen(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	men, err := p.ListFighters(ctx, domain.PoolMen)
	if err != nil {
		t.Fatal(err)
	}
	all, err := p.ListFighters(ctx, domain.PoolAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(men) == 0 || len(men) >= len(all) {
		t.Fatalf("men pool must be a non-empty strict subset: %d of %d", len(men), len(all))
	}
	for _, f := range men {
		if f.Division == nil || strings.HasPrefix(*f.Division, "Women") {
			t.Fatalf("%s is not in a men's division", f.Name)
		}
	}
	ids, err := p.GameFighterIDs(ctx, domain.PoolMen)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != len(men) {
		t.Fatalf("every men's fighter must be game-eligible: %d of %d", len(ids), len(men))
	}
	hits, err := p.SearchFighters(ctx, domain.PoolMen, "zhang", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("men search must not return women: %+v", hits)
	}
}
