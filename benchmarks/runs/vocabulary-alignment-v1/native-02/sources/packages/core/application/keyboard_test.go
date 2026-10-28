package application

import "testing"

func TestCandidateHighlight(t *testing.T) {
	ink, white := highlightInk, highlightWhite
	for _, tc := range []struct {
		accent CandidateAccent
		dark   bool
		want   Highlight
	}{
		{AccentYellow, false, Highlight{"#ffcc00", ink}},
		{AccentYellow, true, Highlight{"#ffcc00", ink}},
		{AccentOrange, false, Highlight{"#ff9500", ink}},
		{AccentRed, false, Highlight{"#ff3b30", white}},
		{AccentPink, false, Highlight{"#ff2d55", white}},
		{AccentPurple, true, Highlight{"#bf5af2", white}},
		{AccentBlue, false, Highlight{"#0088ff", white}},
		{AccentBlue, true, Highlight{"#0099f2", white}},
		{AccentGreen, false, Highlight{"#34c759", ink}},
		{AccentGraphite, false, Highlight{"#8e8e93", white}},
	} {
		if got := (KeyboardPreferences{Accent: tc.accent}).Highlight(tc.dark); got != tc.want {
			t.Errorf("%s dark=%v: %+v, want %+v", tc.accent, tc.dark, got, tc.want)
		}
	}
	for _, accent := range CandidateAccents {
		if _, ok := accentFills[accent]; !ok && accent != AccentCustom {
			t.Errorf("%s has no fill", accent)
		}
	}
	// A custom color is the same in both appearances, with legible text.
	custom := KeyboardPreferences{Accent: AccentCustom, AccentColor: "#1a1a2e"}
	if got := custom.Highlight(true); got != (Highlight{"#1a1a2e", white}) || custom.Highlight(false) != got {
		t.Errorf("dark custom color: %+v", got)
	}
	custom.AccentColor = "#e0f7fa"
	if got := custom.Highlight(false); got.Text != ink {
		t.Errorf("light custom color: %+v", got)
	}
}

func TestCustomAccentNormalization(t *testing.T) {
	k := DefaultKeyboardPreferences()
	k.Accent, k.AccentColor = AccentCustom, "#12abef"
	if got, repaired := k.normalize(); repaired || got != k {
		t.Fatalf("valid custom accent changed: %+v", got)
	}
	// A preset keeps the custom color for a later return to Custom.
	k.Accent = AccentGreen
	if got, repaired := k.normalize(); repaired || got.AccentColor != "#12abef" {
		t.Fatalf("preset dropped the custom color: %+v", got)
	}
	for _, color := range []string{"", "12abef", "#12abe", "#12abefff", "#zzzzzz"} {
		k.Accent, k.AccentColor = AccentCustom, color
		if got, repaired := k.normalize(); !repaired || got.Accent != AccentYellow || got.AccentColor != "" {
			t.Errorf("custom accent %q not repaired: %+v", color, got)
		}
	}
}
