package domain

import (
	"strings"
	"unicode/utf8"
)

// HintKind identifies which target attribute a hint reveals.
type HintKind string

const (
	HintNationality HintKind = "nationality"
	HintDivision    HintKind = "division"
	HintNickname    HintKind = "nickname"
	HintInitials    HintKind = "initials"
)

// Hint is one revealed fact about the target. It never carries the full name.
type Hint struct {
	Kind  HintKind `json:"kind"`
	Label string   `json:"label"`
	Value string   `json:"value"`
}

// HintsResult lists the unlocked hints and the guess count that unlocks the
// next one (nil once every hint is unlocked).
type HintsResult struct {
	Hints  []Hint `json:"hints"`
	NextAt *int   `json:"next_at"`
}

type hintRule struct {
	after  int
	kind   HintKind
	label  string
	reveal func(FighterView) string
}

// hintRules is ordered by unlock threshold (guesses made).
var hintRules = []hintRule{
	{3, HintNationality, "Nationality", func(f FighterView) string { return f.Nationality }},
	{5, HintDivision, "Division", func(f FighterView) string { return f.Division }},
	{7, HintNickname, "Nickname", nicknameHint},
	{9, HintInitials, "Initials", initials},
}

// Hints returns the target hints unlocked after the given number of guesses.
func Hints(target FighterView, guesses int) HintsResult {
	out := HintsResult{Hints: []Hint{}}
	for _, r := range hintRules {
		if guesses < r.after {
			next := r.after
			out.NextAt = &next
			break
		}
		out.Hints = append(out.Hints, Hint{Kind: r.kind, Label: r.label, Value: r.reveal(target)})
	}
	return out
}

func nicknameHint(f FighterView) string {
	if f.Nickname == nil || *f.Nickname == "" {
		return "No nickname"
	}
	return *f.Nickname
}

// initials renders "Ilia Topuria" as "I. T.", rune-aware for accented names.
func initials(f FighterView) string {
	words := strings.Fields(f.Name)
	parts := make([]string, 0, len(words))
	for _, w := range words {
		r, _ := utf8.DecodeRuneInString(w)
		parts = append(parts, string(r)+".")
	}
	return strings.Join(parts, " ")
}
