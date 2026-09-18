package store

import (
	"context"
	"testing"
	"time"
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
	ids, err := p.GameFighterIDs(ctx)
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
	lower, err := p.SearchFighters(ctx, "conor", 8)
	if err != nil {
		t.Fatal(err)
	}
	upper, err := p.SearchFighters(ctx, "CONOR", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(lower) == 0 || len(upper) == 0 {
		t.Fatal("expected search hits for seeded fighter")
	}
	if len(lower) != len(upper) {
		t.Fatal("search must be case-insensitive")
	}
	partial, err := p.SearchFighters(ctx, "mcg", 8)
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
	ids, err := p.GameFighterIDs(ctx)
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
