package domain

import (
	"testing"
)

func TestNewRoundID(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		id, err := NewRoundID()
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateRoundID(id); err != nil {
			t.Fatalf("fresh id %q rejected: %v", id, err)
		}
		if seen[id] {
			t.Fatal("round ids must be unique")
		}
		seen[id] = true
	}
	for _, bad := range []string{"", "xyz", "1", "gg", "0123456789abcdef0123456789abcde", "0123456789ABCDEF0123456789ABCDEF"} {
		if err := ValidateRoundID(bad); err == nil {
			t.Fatalf("round id %q must be rejected", bad)
		}
	}
}

func TestApplyInfiniteGuess(t *testing.T) {
	// Correct guesses solve without touching lives.
	if got := ApplyInfiniteGuess(5, true); !got.Solved || got.Dead || got.LivesLeft != 5 {
		t.Fatalf("solve = %+v", got)
	}
	if got := ApplyInfiniteGuess(1, true); !got.Solved || got.LivesLeft != 1 {
		t.Fatalf("last-life solve = %+v", got)
	}
	// Wrong guesses cost exactly one life; death lands exactly on zero.
	cases := []struct {
		lives int
		left  int
		dead  bool
	}{
		{5, 4, false},
		{2, 1, false},
		{1, 0, true},
		{0, 0, true},
	}
	for _, tc := range cases {
		got := ApplyInfiniteGuess(tc.lives, false)
		if got.Solved || got.LivesLeft != tc.left || got.Dead != tc.dead {
			t.Fatalf("lives=%d miss = %+v, want left=%d dead=%v", tc.lives, got, tc.left, tc.dead)
		}
	}
}
