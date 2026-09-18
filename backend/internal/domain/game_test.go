package domain

import (
	"testing"
	"time"
)

func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func view(id, name string, dob string, height int, division, nat, event string, w, l, d, nc int) FighterView {
	return FighterView{
		ID: 0, Name: name,
		DateOfBirth: date(dob), HeightCm: height,
		Nationality: nat, Wins: w, Losses: l, Draws: d, NoContests: nc,
		Division: division, LastEvent: event,
		LastEventDate: date("2025-10-04"),
	}
}

func TestCalculateAge(t *testing.T) {
	cases := []struct {
		name string
		dob  string
		ref  string
		want int
	}{
		{"birthday today", "1990-09-18", "2026-09-18", 36},
		{"birthday tomorrow", "1990-09-19", "2026-09-18", 35},
		{"birthday yesterday", "1990-09-17", "2026-09-18", 36},
		{"leap day before feb28 non-leap", "2000-02-29", "2026-02-28", 25}, // Feb 28: not yet Mar-drift; day 28 < 29
		{"leap day after mar1 non-leap", "2000-02-29", "2026-03-01", 26},
		{"newborn", "2026-09-18", "2026-09-18", 0},
		{"unborn clamps zero", "2027-01-01", "2026-09-18", 0},
		{"year boundary", "1988-07-14", "2026-09-18", 38},
		{"end of year", "1988-12-31", "2026-01-01", 37},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CalculateAge(date(tc.dob), date(tc.ref)); got != tc.want {
				t.Fatalf("CalculateAge(%s,%s)=%d want %d", tc.dob, tc.ref, got, tc.want)
			}
		})
	}
}

func TestFormatRecord(t *testing.T) {
	if got := FormatRecord(23, 5, 0, 0); got != "23-5-0" {
		t.Fatalf("got %q", got)
	}
	if got := FormatRecord(22, 6, 0, 1); got != "22-6-0-1 NC" {
		t.Fatalf("got %q", got)
	}
}

func TestCompareOrdered(t *testing.T) {
	// Convention: "higher" means the TARGET is higher than the guess.
	if CompareOrdered(180, 185) != ComparisonHigher {
		t.Fatal("guess shorter than target must be higher")
	}
	if CompareOrdered(190, 185) != ComparisonLower {
		t.Fatal("guess taller than target must be lower")
	}
	if CompareOrdered(185, 185) != ComparisonCorrect {
		t.Fatal("equal must be correct")
	}
}

func TestCompareExact(t *testing.T) {
	if CompareExact("Lightweight", "Lightweight") != ComparisonCorrect {
		t.Fatal("equal strings must be correct")
	}
	if CompareExact("Lightweight", "Welterweight") != ComparisonIncorrect {
		t.Fatal("different strings must be incorrect")
	}
}

func TestEvaluateGuess_Correct(t *testing.T) {
	gameDate := date("2026-09-18")
	a := view("", "Fighter A", "1990-01-01", 180, "Lightweight", "Brazil", "UFC 320", 18, 4, 0, 0)
	b := a
	b.Name = "Fighter A"
	out := EvaluateGuess(a, b, gameDate)
	if !out.Correct {
		t.Fatalf("identical fighters must be correct: %+v", out)
	}
	for _, cmp := range []Comparison{
		out.Results.Age.Comparison, out.Results.Division.Comparison,
		out.Results.Height.Comparison, out.Results.Record.Comparison,
		out.Results.Nationality.Comparison, out.Results.LastEvent.Comparison,
	} {
		if cmp != ComparisonCorrect {
			t.Fatalf("all comparisons must be correct, got %v", cmp)
		}
	}
}

func TestEvaluateGuess_Mixed(t *testing.T) {
	gameDate := date("2026-09-18")
	// Target: older (1987), taller (185), Lightweight, Brazil, UFC 320, 23-5-0.
	target := view("", "Target", "1987-07-07", 185, "Lightweight", "Brazil", "UFC 320", 23, 5, 0, 0)
	// Guess: younger (1995), shorter (180), same division, different record/nat/event.
	guess := view("", "Guess", "1995-05-05", 180, "Lightweight", "USA", "UFC 319", 18, 4, 0, 0)
	out := EvaluateGuess(target, guess, gameDate)
	if out.Correct {
		t.Fatal("must not be correct")
	}
	if out.Results.Age.Comparison != ComparisonHigher {
		t.Fatalf("target older -> higher, got %v", out.Results.Age.Comparison)
	}
	if out.Results.Age.Value != 31 {
		t.Fatalf("guess age must be 31, got %d", out.Results.Age.Value)
	}
	if out.Results.Height.Comparison != ComparisonHigher {
		t.Fatalf("target taller -> higher, got %v", out.Results.Height.Comparison)
	}
	if out.Results.Division.Comparison != ComparisonCorrect {
		t.Fatal("same division must be correct")
	}
	if out.Results.Record.Comparison != ComparisonIncorrect {
		t.Fatal("different record must be incorrect")
	}
	if out.Results.Nationality.Comparison != ComparisonIncorrect {
		t.Fatal("different nationality must be incorrect")
	}
	if out.Results.LastEvent.Comparison != ComparisonIncorrect {
		t.Fatal("different event must be incorrect")
	}
	if out.Results.Record.Value != "18-4-0" {
		t.Fatalf("record display wrong: %q", out.Results.Record.Value)
	}
	// Response must carry the GUESS display values, never the target's.
	if out.Results.Height.Value != 180 || out.Results.Nationality.Value != "USA" {
		t.Fatal("response must contain guessed fighter values")
	}
}

func TestEvaluateGuess_NCRecord(t *testing.T) {
	gameDate := date("2026-09-18")
	a := view("", "A", "1990-01-01", 180, "Lightweight", "Brazil", "UFC 320", 22, 6, 0, 1)
	b := view("", "B", "1990-01-01", 180, "Lightweight", "Brazil", "UFC 320", 22, 6, 0, 0)
	out := EvaluateGuess(a, b, gameDate)
	if out.Results.Record.Comparison != ComparisonIncorrect {
		t.Fatal("NC difference must make record incorrect")
	}
	if out.Results.Record.Value != "22-6-0" {
		t.Fatalf("got %q", out.Results.Record.Value)
	}
}

func TestHashSelector_Deterministic(t *testing.T) {
	sel := HashSelector{}
	ids := []int{1, 2, 3, 4, 5, 6, 7, 8}
	d1 := date("2026-09-18")
	a, err := sel.Select(d1, ids)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := sel.Select(d1, ids)
	if a != b {
		t.Fatalf("same date must give same fighter: %d vs %d", a, b)
	}
	// Selected id must always be a member of the eligible set.
	seen := map[int]bool{}
	for i := 0; i < 60; i++ {
		day := date("2026-09-18").AddDate(0, 0, i)
		id, err := sel.Select(day, ids)
		if err != nil {
			t.Fatal(err)
		}
		valid := false
		for _, v := range ids {
			if v == id {
				valid = true
			}
		}
		if !valid {
			t.Fatalf("selected invalid fighter %d", id)
		}
		seen[id] = true
	}
	if len(seen) < 2 {
		t.Fatal("selector should vary across dates (sanity check)")
	}
	// Different dates should (usually) differ; determinism is the requirement.
	if _, err := sel.Select(d1, nil); err != ErrNoFighters {
		t.Fatal("empty ids must return ErrNoFighters")
	}
}
