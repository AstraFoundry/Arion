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
	if prefs.Keyboard != DefaultKeyboardPreferences() || prefs.Keyboard.Accent != AccentYellow {
		t.Fatalf("profile without keyboard preferences did not load defaults: %+v", prefs.Keyboard)
	}
	// Keyboard preferences saved before the accent existed keep their
	// choices and take the default accent without a repair.
	prefs = DefaultPreferences("darwin")
	if err := json.Unmarshal([]byte(`{"keyboard":{"pageSize":6,"candidateSize":"large","appearance":"dark"}}`), &prefs); err != nil {
		t.Fatal(err)
	}
	if normalized, repairs := prefs.Normalize("darwin"); normalized.Keyboard.Accent != AccentYellow || normalized.Keyboard.PageSize != 6 || !normalized.Keyboard.EmojiSuggestions || len(repairs) != 0 {
		t.Fatalf("keyboard preferences without an accent: %+v %v", normalized.Keyboard, repairs)
	}
	prefs = DefaultPreferences("darwin")
	if err := json.Unmarshal([]byte(`{"keyboard":{"emojiSuggestions":false}}`), &prefs); err != nil || prefs.Keyboard.EmojiSuggestions {
		t.Fatalf("explicit Emoji preference was not retained: %+v %v", prefs.Keyboard, err)
	}
	old = DefaultPreferences("darwin")
	old.Keyboard = KeyboardPreferences{PageSize: 6, CandidateSize: CandidateLarge, Appearance: AppearanceDark, Accent: AccentBlue}
	if normalized, repairs := old.Normalize("darwin"); normalized.Keyboard != old.Keyboard || len(repairs) != 0 {
		t.Fatalf("valid keyboard choice changed: %+v %v", normalized.Keyboard, repairs)
	}
	old.Keyboard = KeyboardPreferences{PageSize: 12, CandidateSize: "huge", Appearance: "sepia", Accent: "teal", EmojiSuggestions: true}
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

func TestKeyboardSpellingPreferences(t *testing.T) {
	p := DefaultPreferences("darwin")
	if p.Keyboard.TypoCorrection || p.Keyboard.FuzzyPinyin != 0 {
		t.Fatal("spelling expansion must be explicit")
	}
	p.Keyboard.TypoCorrection = true
	p.Keyboard.FuzzyPinyin = FuzzyZZH | FuzzyIANIANG
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	restored := DefaultPreferences("darwin")
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	normalized, repairs := restored.Normalize("darwin")
	if normalized.Keyboard != p.Keyboard || len(repairs) != 0 {
		t.Fatalf("spelling preferences changed after roundtrip: %+v %v", normalized.Keyboard, repairs)
	}
	restored.Keyboard.FuzzyPinyin |= 1 << 31
	normalized, repairs = restored.Normalize("darwin")
	if normalized.Keyboard.FuzzyPinyin != p.Keyboard.FuzzyPinyin || !slices.Contains(repairs, "keyboard") {
		t.Fatalf("unknown fuzzy bits were not explicitly repaired: %+v %v", normalized.Keyboard, repairs)
	}
}

func TestPunctuationPreference(t *testing.T) {
	if got := DefaultPreferences("darwin").Punctuation; got != domain.PunctuationSmart {
		t.Fatalf("default punctuation %q", got)
	}
	prefs := DefaultPreferences("darwin")
	if err := json.Unmarshal([]byte(`{"version":1,"uiLanguage":"en"}`), &prefs); err != nil {
		t.Fatal(err)
	}
	if normalized, repairs := prefs.Normalize("darwin"); normalized.Punctuation != domain.PunctuationSmart || len(repairs) != 0 {
		t.Fatalf("older profile did not keep smart: %q %v", normalized.Punctuation, repairs)
	}
	for _, mode := range []domain.Punctuation{domain.PunctuationSmart, domain.PunctuationFull, domain.PunctuationOmit, domain.PunctuationSpaces} {
		prefs = DefaultPreferences("darwin")
		prefs.Punctuation = mode
		if normalized, repairs := prefs.Normalize("darwin"); normalized.Punctuation != mode || len(repairs) != 0 {
			t.Fatalf("mode %q changed: %q %v", mode, normalized.Punctuation, repairs)
		}
	}
	prefs = DefaultPreferences("darwin")
	prefs.Punctuation = "comma"
	normalized, repairs := prefs.Normalize("darwin")
	if normalized.Punctuation != domain.PunctuationSmart || !slices.Contains(repairs, "punctuation") {
		t.Fatalf("invalid punctuation not repaired: %q %v", normalized.Punctuation, repairs)
	}
}

func TestTranslationPreferences(t *testing.T) {
	if got := DefaultPreferences("darwin"); got.TranslateKey != HoldRightCommand || got.TranslateLanguage != domain.TargetEnglish {
		t.Fatalf("darwin translation defaults: key=%q lang=%q", got.TranslateKey, got.TranslateLanguage)
	}
	if got := DefaultPreferences("windows"); got.TranslateKey != HoldRightCtrl || got.TranslateLanguage != domain.TargetEnglish {
		t.Fatalf("windows translation defaults: key=%q lang=%q", got.TranslateKey, got.TranslateLanguage)
	}
	for _, platform := range []string{"darwin", "windows"} {
		for _, key := range TranslateKeysFor(platform) {
			prefs := DefaultPreferences(platform)
			prefs.TranslateKey = key
			normalized, repairs := prefs.Normalize(platform)
			if len(repairs) != 0 {
				t.Fatalf("%s key %q caused repairs: %v", platform, key, repairs)
			}
			wantKey := key
			if key == prefs.HoldKey {
				wantKey = HoldOff
			}
			if normalized.TranslateKey != wantKey {
				t.Fatalf("%s key %q normalized to %q, want %q", platform, key, normalized.TranslateKey, wantKey)
			}
		}
		for _, lang := range domain.TranslationLanguages {
			prefs := DefaultPreferences(platform)
			prefs.TranslateLanguage = lang
			if normalized, repairs := prefs.Normalize(platform); normalized.TranslateLanguage != lang || len(repairs) != 0 {
				t.Fatalf("%s language %q changed: %q %v", platform, lang, normalized.TranslateLanguage, repairs)
			}
		}
		prefs := DefaultPreferences(platform)
		prefs.TranslateKey = "caps_lock"
		prefs.TranslateLanguage = "fr-CA"
		normalized, repairs := prefs.Normalize(platform)
		if normalized.TranslateKey != DefaultTranslateKey(platform) || !slices.Contains(repairs, "translateKey") {
			t.Fatalf("%s invalid translateKey not repaired: %q %v", platform, normalized.TranslateKey, repairs)
		}
		if normalized.TranslateLanguage != domain.TargetEnglish || !slices.Contains(repairs, "translateLanguage") {
			t.Fatalf("%s invalid translateLanguage not repaired: %q %v", platform, normalized.TranslateLanguage, repairs)
		}
	}
}

func TestAccountDisplayPreference(t *testing.T) {
	prefs := DefaultPreferences("darwin")
	if name, email, ok := prefs.DisplayForAccount("https://issuer.example", "user-1"); ok || name != "" || email != "" {
		t.Fatalf("empty profile matched: %q %q %v", name, email, ok)
	}
	prefs = prefs.WithAccountDisplay("https://issuer.example", " user-1 ", " Ada ", " ada@example.com ")
	if prefs.AccountSubject != "user-1" || prefs.AccountName != "Ada" || prefs.AccountEmail != "ada@example.com" {
		t.Fatalf("display not trimmed: %+v", prefs)
	}
	name, email, ok := prefs.DisplayForAccount("https://issuer.example", "user-1")
	if !ok || name != "Ada" || email != "ada@example.com" {
		t.Fatalf("same subject: %q %q %v", name, email, ok)
	}
	if _, _, ok := prefs.DisplayForAccount("https://issuer.example", "user-2"); ok {
		t.Fatal("other subject matched")
	}
	cleared := prefs.WithAccountDisplay("https://issuer.example", "", "Ada", "ada@example.com")
	if cleared.AccountSubject != "" || cleared.AccountName != "" || cleared.AccountEmail != "" {
		t.Fatalf("empty subject kept a name: %+v", cleared)
	}
	orphan := DefaultPreferences("darwin")
	orphan.AccountName = "Ada"
	normalized, repairs := orphan.Normalize("darwin")
	if normalized.AccountName != "" || !slices.Contains(repairs, "accountDisplay") {
		t.Fatalf("name without subject not repaired: %+v %v", normalized, repairs)
	}
}

func TestAccountDisplayIsIssuerScoped(t *testing.T) {
	prefs := DefaultPreferences("darwin").WithAccountDisplay("https://one.example", "same-subject", "Alice", "alice@example.com")
	if _, _, ok := prefs.DisplayForAccount("https://two.example", "same-subject"); ok {
		t.Fatal("profile crossed issuer boundary")
	}
	legacy := DefaultPreferences("darwin")
	legacy.AccountSubject = "same-subject"
	legacy.AccountName = "Legacy"
	repaired, _ := legacy.Normalize("darwin")
	if repaired.AccountName != "" {
		t.Fatal("unscoped legacy profile retained")
	}
}
