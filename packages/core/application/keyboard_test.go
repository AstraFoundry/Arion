package application

import "testing"

func TestCandidateHighlight(t *testing.T) {
	black, white := textColors[TextBlack], textColors[TextWhite]
	for _, tc := range []struct {
		accent CandidateAccent
		dark   bool
		want   Highlight
	}{
		{AccentYellow, false, Highlight{"#ffcc00", black}},
		{AccentYellow, true, Highlight{"#ffcc00", black}},
		{AccentOrange, false, Highlight{"#ff9500", black}},
		{AccentRed, false, Highlight{"#ff3b30", black}},
		{AccentBlue, false, Highlight{"#0088ff", black}},
		{AccentBlue, true, Highlight{"#0099f2", black}},
		{AccentGraphite, false, Highlight{"#8e8e93", black}},
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
	// A custom color is the same in both appearances.
	custom := KeyboardPreferences{Accent: AccentCustom, AccentColor: "#1a1a2e"}
	if got := custom.Highlight(true); got != (Highlight{"#1a1a2e", black}) || custom.Highlight(false) != got {
		t.Errorf("custom color: %+v", got)
	}
	// White text applies to every fill and both appearances.
	for _, k := range []KeyboardPreferences{{Accent: AccentYellow, TextColor: TextWhite}, {Accent: AccentCustom, AccentColor: "#e0f7fa", TextColor: TextWhite}} {
		for _, dark := range []bool{false, true} {
			if got := k.Highlight(dark); got.Text != white {
				t.Errorf("%s dark=%v white text: %+v", k.Accent, dark, got)
			}
		}
	}
}

func TestCandidateTextColorNormalization(t *testing.T) {
	if got := DefaultKeyboardPreferences().ActiveTextColor(); got != TextBlack {
		t.Fatalf("default text color %q", got)
	}
	k := DefaultKeyboardPreferences()
	k.TextColor = TextWhite
	if got, repaired := k.normalize(); repaired || got.TextColor != TextWhite {
		t.Errorf("white: %+v repaired=%v", got, repaired)
	}
	// Profiles written before the choice keep an empty value; unknown values
	// are cleared. Both resolve to black.
	for _, tc := range []struct {
		color    CandidateTextColor
		repaired bool
	}{{"", false}, {"red", true}} {
		k.TextColor = tc.color
		got, repaired := k.normalize()
		if repaired != tc.repaired || got.TextColor != "" || got.ActiveTextColor() != TextBlack {
			t.Errorf("%q: %+v repaired=%v", tc.color, got, repaired)
		}
		if text := got.Highlight(false).Text; text != textColors[TextBlack] {
			t.Errorf("%q resolves to %s", tc.color, text)
		}
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
