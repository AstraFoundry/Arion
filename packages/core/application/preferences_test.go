package application

import (
	"slices"
	"testing"
)

func TestUILanguagePreference(t *testing.T) {
	if DefaultPreferences().UILanguage != English {
		t.Fatal("first-run UI must be English")
	}
	for _, language := range []UILanguage{English, SimplifiedChinese} {
		prefs := DefaultPreferences()
		prefs.UILanguage = language
		normalized, repairs := prefs.Normalize()
		if normalized.UILanguage != language || len(repairs) != 0 {
			t.Fatalf("supported language changed: %v %v", normalized, repairs)
		}
	}
	for _, language := range []UILanguage{"", "fr", "zh-TW"} {
		prefs := DefaultPreferences()
		prefs.UILanguage = language
		normalized, repairs := prefs.Normalize()
		if normalized.UILanguage != English || !slices.Contains(repairs, "uiLanguage") {
			t.Fatalf("invalid language not explicitly repaired: %v %v", normalized, repairs)
		}
	}
}
