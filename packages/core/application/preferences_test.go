package application

import (
	"arion.local/core/domain"
	"encoding/json"
	"math"
	"slices"
	"testing"
)

func TestUILanguagePreference(t *testing.T) {
	if DefaultPreferences("darwin").UILanguage != English {
		t.Fatal("first-run UI must be English")
	}
	for _, language := range []UILanguage{English, SimplifiedChinese} {
		prefs := DefaultPreferences("darwin")
		prefs.UILanguage = language
		normalized, repairs := prefs.Normalize("darwin")
		if normalized.UILanguage != language || len(repairs) != 0 {
			t.Fatalf("supported language changed: %v %v", normalized, repairs)
		}
	}
	for _, language := range []UILanguage{"", "fr", "zh-TW"} {
		prefs := DefaultPreferences("darwin")
		prefs.UILanguage = language
		normalized, repairs := prefs.Normalize("darwin")
		if normalized.UILanguage != English || !slices.Contains(repairs, "uiLanguage") {
			t.Fatalf("invalid language not explicitly repaired: %v %v", normalized, repairs)
		}
	}
}

func TestMicrophoneGainPreference(t *testing.T) {
	if DefaultPreferences("darwin").MicrophoneGain != 0 {
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
		prefs := DefaultPreferences("darwin")
		prefs.MicrophoneGain = stop.DB
		if normalized, repairs := prefs.Normalize("darwin"); normalized.MicrophoneGain != stop.DB || len(repairs) != 0 {
			t.Fatalf("supported gain changed: %v %v", normalized, repairs)
		}
	}
	for _, db := range []int{-9, 1, 15} {
		prefs := DefaultPreferences("darwin")
		prefs.MicrophoneGain = db
		normalized, repairs := prefs.Normalize("darwin")
		if normalized.MicrophoneGain != 0 || !slices.Contains(repairs, "microphoneGain") {
			t.Fatalf("invalid gain %d not explicitly repaired: %v %v", db, normalized, repairs)
		}
	}
	if f := GainFactor(6); math.Abs(float64(f)-1.995) > 0.01 || GainFactor(0) != 1 {
		t.Fatalf("gain factors: %v %v", f, GainFactor(0))
	}
}

func TestSceneOverridesRoundTripAndRepair(t *testing.T) {
	prefs := DefaultPreferences("darwin")
	prefs.SceneOverrides = domain.SceneOverrides{"mail.google.com": domain.SceneWork, "search@com.apple.mail": domain.SceneChat}
	data, err := json.Marshal(prefs)
	if err != nil {
		t.Fatal(err)
	}
	var loaded Preferences
	if err := json.Unmarshal(data, &loaded); err != nil || loaded.SceneOverrides["mail.google.com"] != domain.SceneWork {
		t.Fatalf("round trip lost overrides: %+v %v", loaded.SceneOverrides, err)
	}
	loaded.SceneOverrides = loaded.SceneOverrides.WithScene("poem.example", "poem").WithScene("list.example", domain.SceneList)
	stored := loaded.SceneOverrides
	normalized, repairs := loaded.Normalize("darwin")
	if len(normalized.SceneOverrides) != 2 || len(repairs) == 0 {
		t.Fatalf("invalid overrides kept: %+v %v", normalized.SceneOverrides, repairs)
	}
	if len(stored) != 4 {
		t.Fatal("Normalize mutated the stored map")
	}
}

func TestKeyboardPreferences(t *testing.T) {
	var old Preferences
	prefs := DefaultPreferences("darwin")
	if err := json.Unmarshal([]byte(`{"version":1,"uiLanguage":"en","mode":"shortcut","shortcut":"Alt+Space"}`), &prefs); err != nil {
		t.Fatal(err)
	}
	if prefs.Keyboard != DefaultKeyboardPreferences() || !prefs.Keyboard.ContextRanking {
		t.Fatalf("profile without keyboard preferences did not load defaults: %+v", prefs.Keyboard)
	}
	old = DefaultPreferences("darwin")
	old.Keyboard = KeyboardPreferences{PageSize: 6, CandidateSize: CandidateLarge, Appearance: AppearanceDark}
	if normalized, repairs := old.Normalize("darwin"); normalized.Keyboard != old.Keyboard || len(repairs) != 0 {
		t.Fatalf("valid keyboard choice changed: %+v %v", normalized.Keyboard, repairs)
	}
	old.Keyboard = KeyboardPreferences{PageSize: 12, CandidateSize: "huge", Appearance: "sepia", ContextRanking: true}
	normalized, repairs := old.Normalize("darwin")
	if normalized.Keyboard != DefaultKeyboardPreferences() || !slices.Contains(repairs, "keyboard") {
		t.Fatalf("invalid keyboard choice not explicitly repaired: %+v %v", normalized.Keyboard, repairs)
	}
}

func TestPolishEnginePreference(t *testing.T) {
	if got := DefaultPreferences("darwin").PolishEngine; got != PolishLocal {
		t.Fatalf("default engine %q", got)
	}
	p := DefaultPreferences("darwin")
	p.PolishEngine = PolishCloud
	if got, repaired := p.Normalize("darwin"); got.PolishEngine != PolishCloud || len(repaired) != 0 {
		t.Fatalf("cloud engine changed: %q %v", got.PolishEngine, repaired)
	}
	p.PolishEngine = "remote"
	got, repaired := p.Normalize("darwin")
	if got.PolishEngine != PolishLocal || len(repaired) != 1 || repaired[0] != "polishEngine" {
		t.Fatalf("unknown engine not repaired: %q %v", got.PolishEngine, repaired)
	}
}
