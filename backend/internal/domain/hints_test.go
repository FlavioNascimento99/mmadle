package domain

import "testing"

func hintTarget() FighterView {
	nick := "El Matador"
	return FighterView{Name: "Ilia Topuria", Nickname: &nick, Nationality: "Georgia", Division: "Lightweight", LastEvent: "UFC 320"}
}

func TestHintsUnlockProgressively(t *testing.T) {
	cases := []struct {
		guesses int
		kinds   []HintKind
		nextAt  int
	}{
		{0, nil, 3},
		{2, nil, 3},
		{3, []HintKind{HintNationality}, 5},
		{5, []HintKind{HintNationality, HintDivision}, 6},
		{6, []HintKind{HintNationality, HintDivision, HintLastEvent}, 7},
		{7, []HintKind{HintNationality, HintDivision, HintLastEvent, HintNickname}, 9},
		{9, []HintKind{HintNationality, HintDivision, HintLastEvent, HintNickname, HintInitials}, 0},
		{50, []HintKind{HintNationality, HintDivision, HintLastEvent, HintNickname, HintInitials}, 0},
	}
	for _, tc := range cases {
		got := Hints(hintTarget(), tc.guesses)
		if len(got.Hints) != len(tc.kinds) {
			t.Fatalf("guesses=%d: got %d hints, want %d", tc.guesses, len(got.Hints), len(tc.kinds))
		}
		for i, k := range tc.kinds {
			if got.Hints[i].Kind != k {
				t.Fatalf("guesses=%d: hint %d kind=%q want %q", tc.guesses, i, got.Hints[i].Kind, k)
			}
		}
		if tc.nextAt == 0 {
			if got.NextAt != nil {
				t.Fatalf("guesses=%d: all hints unlocked, NextAt must be nil, got %d", tc.guesses, *got.NextAt)
			}
		} else if got.NextAt == nil || *got.NextAt != tc.nextAt {
			t.Fatalf("guesses=%d: NextAt=%v want %d", tc.guesses, got.NextAt, tc.nextAt)
		}
	}
}

func TestHintValues(t *testing.T) {
	got := Hints(hintTarget(), 9).Hints
	want := []string{"Georgia", "Lightweight", "UFC 320", "El Matador", "I. T."}
	for i, w := range want {
		if got[i].Value != w || got[i].Label == "" {
			t.Fatalf("hint %d = %+v, want value %q with a label", i, got[i], w)
		}
	}
}

func TestHintsHandleMissingNicknameAndUnicodeNames(t *testing.T) {
	target := FighterView{Name: "Jiří Procházka", Nationality: "Czech Republic", Division: "Light Heavyweight"}
	got := Hints(target, 9).Hints
	if got[3].Value != "No nickname" {
		t.Fatalf("nickname hint = %q", got[3].Value)
	}
	if got[4].Value != "J. P." {
		t.Fatalf("initials hint = %q", got[4].Value)
	}
}
