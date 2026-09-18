package domain

import (
	"errors"
	"testing"
)

func TestParsePool(t *testing.T) {
	for in, want := range map[string]Pool{"": PoolAll, "all": PoolAll, "men": PoolMen} {
		got, err := ParsePool(in)
		if err != nil || got != want {
			t.Fatalf("ParsePool(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"women", "MEN", "x"} {
		if _, err := ParsePool(in); !errors.Is(err, ErrUnknownPool) {
			t.Fatalf("ParsePool(%q) must fail with ErrUnknownPool, got %v", in, err)
		}
	}
}
