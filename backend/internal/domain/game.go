// Package domain holds the pure game logic for MMAdle.
//
// Design rules:
//   - No HTTP, no SQL, no I/O here: handlers call services which call this
//     package, so the comparison system can grow new attributes without
//     touching transport or persistence code.
//   - The frontend never computes comparisons; every attribute returns a
//     semantic Comparison value computed here.
//   - Derived values (age, last event, formatted record) are calculated from
//     canonical data, never stored.
package domain

import (
	"errors"
	"fmt"
	"hash/fnv"
	"time"
)

// Comparison is the semantic result of comparing one guessed attribute
// against the target's attribute, from the player's perspective:
//
//   - "correct" / "incorrect": exact attributes (division, record,
//     nationality, last event).
//   - "higher" / "lower": ordered attributes (age, height). "higher" means
//     the TARGET is higher than the guess (display an up-arrow: guess higher!),
//     "lower" means the target is lower than the guess.
type Comparison string

const (
	ComparisonCorrect   Comparison = "correct"
	ComparisonIncorrect Comparison = "incorrect"
	ComparisonHigher    Comparison = "higher"
	ComparisonLower     Comparison = "lower"
)

// FighterView is the backend's full picture of one fighter for one game date.
// Age is derived from DateOfBirth relative to the game date; LastEvent is
// derived from the fighter's fights. Neither is stored on the fighter row.
type FighterView struct {
	ID            int
	Name          string
	Nickname      *string
	DateOfBirth   time.Time
	HeightCm      int
	Nationality   string
	Wins          int
	Losses        int
	Draws         int
	NoContests    int
	Stance        *string
	PhotoURL      *string
	PhotoCredit   *string
	Division      string
	LastEvent     string
	LastEventDate time.Time
}

// CalculateAge returns the whole years lived at ref date (the game date, so
// daily attributes are deterministic). Standard calendar aging: the age
// increments on the birthday anniversary.
func CalculateAge(dob, ref time.Time) int {
	dob = dob.UTC()
	ref = ref.UTC()
	age := ref.Year() - dob.Year()
	if ref.Month() < dob.Month() ||
		(ref.Month() == dob.Month() && ref.Day() < dob.Day()) {
		age--
	}
	if age < 0 {
		return 0
	}
	return age
}

// FormatRecord renders wins-losses-draws, appending " NC" info when the
// fighter has no-contests, e.g. "23-5-0" or "23-5-0 (1 NC)".
func FormatRecord(wins, losses, draws, noContests int) string {
	base := fmt.Sprintf("%d-%d-%d", wins, losses, draws)
	if noContests > 0 {
		return fmt.Sprintf("%s-%d NC", base, noContests)
	}
	return base
}

// CompareOrdered compares an ordered numeric attribute (age, height).
// It returns ComparisonHigher when target > guess, ComparisonLower when
// target < guess, and ComparisonCorrect on equality.
func CompareOrdered(guess, target int) Comparison {
	switch {
	case target > guess:
		return ComparisonHigher
	case target < guess:
		return ComparisonLower
	default:
		return ComparisonCorrect
	}
}

// CompareExact compares an exact-match attribute (division, nationality,
// last event, record identity).
func CompareExact(guess, target string) Comparison {
	if guess == target {
		return ComparisonCorrect
	}
	return ComparisonIncorrect
}

// Result is one attribute's outcome: the GUESSED fighter's display value
// plus the semantic comparison against the target.
type Result[T any] struct {
	Value      T          `json:"value"`
	Comparison Comparison `json:"comparison"`
}

// GuessResults carries the six MVP attributes. To add an attribute later,
// add a field here plus one comparator below; handlers and SQL are untouched
// apart from providing the new view field.
type GuessResults struct {
	Age         Result[int]    `json:"age"`
	Division    Result[string] `json:"division"`
	Height      Result[int]    `json:"height"`
	HeightArrow string         `json:"height_arrow,omitempty"`
	Record      Result[string] `json:"record"`
	Nationality Result[string] `json:"nationality"`
	LastEvent   Result[string] `json:"last_event"`
}

// GuessOutcome is the full evaluated guess returned to the frontend.
// It contains NO target data: only the guessed fighter's display values
// (name, photo) and semantic comparisons.
type GuessOutcome struct {
	FighterID   int          `json:"fighter_id"`
	FighterName string       `json:"fighter_name"`
	PhotoURL    *string      `json:"photo_url"`
	PhotoCredit *string      `json:"photo_credit"`
	Results     GuessResults `json:"results"`
	Correct     bool         `json:"correct"`
}

// RecordsEqual reports whether two records are identical across all four
// structured fields.
func RecordsEqual(a, b FighterView) bool {
	return a.Wins == b.Wins && a.Losses == b.Losses &&
		a.Draws == b.Draws && a.NoContests == b.NoContests
}

// EvaluateGuess compares guess against target for the given game date and
// returns the structured outcome for the frontend.
func EvaluateGuess(target, guess FighterView, gameDate time.Time) GuessOutcome {
	targetAge := CalculateAge(target.DateOfBirth, gameDate)
	guessAge := CalculateAge(guess.DateOfBirth, gameDate)

	ageCmp := CompareOrdered(guessAge, targetAge)
	heightCmp := CompareOrdered(guess.HeightCm, target.HeightCm)
	divisionCmp := CompareExact(guess.Division, target.Division)
	nationalityCmp := CompareExact(guess.Nationality, target.Nationality)
	lastEventCmp := CompareExact(guess.LastEvent, target.LastEvent)

	recordCmp := ComparisonIncorrect
	if RecordsEqual(target, guess) {
		recordCmp = ComparisonCorrect
	}

	correct := ageCmp == ComparisonCorrect &&
		heightCmp == ComparisonCorrect &&
		divisionCmp == ComparisonCorrect &&
		nationalityCmp == ComparisonCorrect &&
		lastEventCmp == ComparisonCorrect &&
		recordCmp == ComparisonCorrect

	arrow := ""
	switch heightCmp {
	case ComparisonHigher:
		arrow = "up"
	case ComparisonLower:
		arrow = "down"
	}

	return GuessOutcome{
		FighterID:   guess.ID,
		FighterName: guess.Name,
		PhotoURL:    guess.PhotoURL,
		PhotoCredit: guess.PhotoCredit,
		Results: GuessResults{
			Age:         Result[int]{Value: guessAge, Comparison: ageCmp},
			Division:    Result[string]{Value: guess.Division, Comparison: divisionCmp},
			Height:      Result[int]{Value: guess.HeightCm, Comparison: heightCmp},
			HeightArrow: arrow,
			Record:      Result[string]{Value: FormatRecord(guess.Wins, guess.Losses, guess.Draws, guess.NoContests), Comparison: recordCmp},
			Nationality: Result[string]{Value: guess.Nationality, Comparison: nationalityCmp},
			LastEvent:   Result[string]{Value: guess.LastEvent, Comparison: lastEventCmp},
		},
		Correct: correct,
	}
}

// ErrNoFighters is returned when the selector has no eligible fighters.
var ErrNoFighters = errors.New("no eligible fighters for daily selection")

// Selector picks the daily target fighter. The HashSelector below is the MVP
// strategy; replace it with another Selector implementation (e.g. curated
// schedule) without changing callers.
type Selector interface {
	Select(gameDate time.Time, ids []int) (int, error)
}

// HashSelector maps game date -> deterministic hash -> fighter index.
// Same date always yields the same fighter; ids must be in a stable
// (sorted) order, which the repository guarantees via ORDER BY id.
type HashSelector struct{}

func (HashSelector) Select(gameDate time.Time, ids []int) (int, error) {
	if len(ids) == 0 {
		return 0, ErrNoFighters
	}
	key := gameDate.UTC().Format("2006-01-02")
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return ids[int(h.Sum64()%uint64(len(ids)))], nil
}
