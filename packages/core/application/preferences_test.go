package application

import (
	"math"
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

func TestMicrophoneGainPreference(t *testing.T) {
	if DefaultPreferences().MicrophoneGain != 0 {
		t.Fatal("first-run gain must be the device's own level")
	}
	for i, stop := range GainStops {
		if i > 0 && stop.DB-GainStops[i-1].DB != GainStep {
			t.Fatalf("stops not %d dB apart: %v", GainStep, GainStops)
		}
		// Each stop compensates the inverse-distance loss from 30 cm.
		want := 30 * math.Pow(10, float64(stop.DB)/20)
		if math.Abs(float64(stop.DistanceCM)-want) > 1 {
			t.Errorf("%d dB suggests %d cm, want about %.0f", stop.DB, stop.DistanceCM, want)
		}
		prefs := DefaultPreferences()
		prefs.MicrophoneGain = stop.DB
		if normalized, repairs := prefs.Normalize(); normalized.MicrophoneGain != stop.DB || len(repairs) != 0 {
			t.Fatalf("supported gain changed: %v %v", normalized, repairs)
		}
	}
	for _, db := range []int{-9, 1, 15} {
		prefs := DefaultPreferences()
		prefs.MicrophoneGain = db
		normalized, repairs := prefs.Normalize()
		if normalized.MicrophoneGain != 0 || !slices.Contains(repairs, "microphoneGain") {
			t.Fatalf("invalid gain %d not explicitly repaired: %v %v", db, normalized, repairs)
		}
	}
	if f := GainFactor(6); math.Abs(float64(f)-1.995) > 0.01 || GainFactor(0) != 1 {
		t.Fatalf("gain factors: %v %v", f, GainFactor(0))
	}
}
