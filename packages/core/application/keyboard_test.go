package application

import "testing"

func TestCandidateHighlight(t *testing.T) {
	black, white := textColors[TextBlack], textColors[TextWhite]
	// Without a choice, the text follows the contrast recommendation.
	for _, tc := range []struct {
		accent CandidateAccent
		dark   bool
		want   Highlight
	}{
		{AccentYellow, false, Highlight{"#ffcc00", black}},
		{AccentYellow, true, Highlight{"#ffcc00", black}},
		{AccentOrange, false, Highlight{"#ff9500", black}},
		{AccentRed, false, Highlight{"#ff3b30", white}},
		{AccentPink, false, Highlight{"#ff2d55", white}},
		{AccentPurple, true, Highlight{"#bf5af2", white}},
		{AccentBlue, false, Highlight{"#0088ff", white}},
		{AccentBlue, true, Highlight{"#0099f2", white}},
		{AccentGreen, false, Highlight{"#34c759", black}},
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
	if got := custom.Highlight(false); got.Text != black || custom.RecommendedTextColor(false) != TextBlack {
		t.Errorf("light custom color: %+v", got)
	}
	// A choice overrides the recommendation on every fill and appearance.
	for _, k := range []KeyboardPreferences{
		{Accent: AccentYellow, TextColor: TextWhite}, {Accent: AccentBlue, TextColor: TextBlack},
		{Accent: AccentCustom, AccentColor: "#ffb800", TextColor: TextWhite},
	} {
		for _, dark := range []bool{false, true} {
			if got := k.Highlight(dark); got.Text != textColors[k.TextColor] {
				t.Errorf("%s dark=%v %s text: %+v", k.Accent, dark, k.TextColor, got)
			}
		}
	}
}

func TestCandidateTextColorNormalization(t *testing.T) {
	k := DefaultKeyboardPreferences()
	if k.TextColor != "" || k.ActiveTextColor(false) != TextBlack {
		t.Fatalf("default text color %+v", k)
	}
	k.TextColor = TextWhite
	if got, repaired := k.normalize(); repaired || got.TextColor != TextWhite {
		t.Errorf("white: %+v repaired=%v", got, repaired)
	}
	// Profiles written before the choice keep an empty value; unknown values
	// are cleared. Both follow the recommendation.
	for _, tc := range []struct {
		color    CandidateTextColor
		repaired bool
	}{{"", false}, {"red", true}} {
		k.TextColor = tc.color
		got, repaired := k.normalize()
		if repaired != tc.repaired || got.TextColor != "" || got.ActiveTextColor(false) != got.RecommendedTextColor(false) {
			t.Errorf("%q: %+v repaired=%v", tc.color, got, repaired)
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
