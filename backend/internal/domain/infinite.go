package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
)

// Infinity (survival) mode: each round hides one random fighter. The player
// has MaxLives lives per round: a wrong guess costs one, a correct guess
// solves the round (the next round starts full again). At zero lives the
// round is dead and the answer is revealed. Lives live server-side so
// account streaks stay meaningful; the transition below is pure so the rule
// is unit-testable without a database.
const MaxLives = 5

// roundIDPattern constrains round ids to 128-bit hex (32 chars), matching
// the infinite_rounds.id CHECK constraint.
var roundIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// ErrInvalidRoundID is returned for malformed round ids.
var ErrInvalidRoundID = errors.New("invalid round id")

// NewRoundID mints an opaque 128-bit round id. It carries no information
// about the target: only the id ever travels to the client.
func NewRoundID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate round id: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

// ValidateRoundID rejects malformed ids before they reach the store.
func ValidateRoundID(id string) error {
	if !roundIDPattern.MatchString(id) {
		return ErrInvalidRoundID
	}
	return nil
}

// LivesTransition is the result of applying one evaluated guess to a round.
type LivesTransition struct {
	LivesLeft int
	Solved    bool
	Dead      bool
}

// ApplyInfiniteGuess folds one guess into the round's lives: a correct guess
// solves the round (lives untouched), a wrong guess costs one life, and the
// round dies exactly when lives reach zero.
func ApplyInfiniteGuess(lives int, correct bool) LivesTransition {
	if correct {
		return LivesTransition{LivesLeft: lives, Solved: true}
	}
	left := lives - 1
	if left < 0 {
		left = 0
	}
	return LivesTransition{LivesLeft: left, Dead: left == 0}
}
