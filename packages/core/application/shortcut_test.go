package application

import (
	"encoding/json"
	"testing"
)

func TestCustomShortcutsPersistAndNormalize(t *testing.T) {
	for _, platform := range []string{"darwin", "windows"} {
		for _, key := range []HoldKey{"f8", "ctrl+space", "ctrl+period", "ctrl+alt+shift+super+k"} {
			p := DefaultPreferences(platform)
			p.ChooseTranslateKey(platform, HoldOff)
			p.ChooseHoldKey(key)
			data, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			q := DefaultPreferences(platform)
			if err := json.Unmarshal(data, &q); err != nil {
				t.Fatal(err)
			}
			q, repairs := q.Normalize(platform)
			if q.HoldKey != key || len(repairs) != 0 {
				t.Fatalf("%s %s: %+v %v", platform, key, q, repairs)
			}
		}
		for _, key := range []HoldKey{"a", "space", "ctrl+ctrl+a", "alt+ctrl+space", "ctrl+fn", "super+", "ctrl+caps_lock", "f25"} {
			if _, ok := ParseHoldShortcut(platform, key); ok {
				t.Fatalf("accepted %s on %s", key, platform)
			}
		}
	}
	if _, ok := ParseHoldShortcut("darwin", "f24"); ok {
		t.Fatal("macOS accepted F24")
	}
	if _, ok := ParseHoldShortcut("windows", "f24"); !ok {
		t.Fatal("Windows rejected F24")
	}
}

func TestShortcutModifierPrefixesCoexist(t *testing.T) {
	for _, platform := range []string{"darwin", "windows"} {
		p := DefaultPreferences(platform)
		p.ChooseHoldKey("ctrl+space")
		p.ChooseTranslateKey(platform, HoldRightCtrl)
		if HoldKeysConflict(p.HoldKey, p.TranslateKey) {
			t.Fatalf("overlapping shortcuts: %+v", p)
		}
		p.HoldKey = "ctrl+space"
		p.TranslateKey = HoldRightCtrl
		p, _ = p.Normalize(platform)
		if p.HoldKey != "ctrl+space" || p.TranslateKey != HoldRightCtrl {
			t.Fatalf("modifier prefix lost: %+v", p)
		}
		p.HoldKey = "ctrl+space"
		p.TranslateKey = "ctrl+enter"
		p, _ = p.Normalize(platform)
		if p.TranslateKey != "ctrl+enter" {
			t.Fatal("distinct triggers sharing modifiers rejected")
		}
	}
}

func TestShortcutSelectionsRejectConflictsWithoutChangingPreferences(t *testing.T) {
	for _, platform := range []string{"darwin", "windows"} {
		for _, other := range []HoldKey{"ctrl+space", HoldRightCtrl} {
			for _, translate := range []bool{false, true} {
				p := DefaultPreferences(platform)
				p.HoldKey, p.TranslateKey, p.PreviousTranslateKey = "f8", other, "f10"
				if translate {
					p.HoldKey, p.TranslateKey = other, "f8"
				}
				before := p
				if translate {
					p.ChooseTranslateKey(platform, other)
				} else {
					p.ChooseHoldKey(other)
				}
				if p.HoldKey != before.HoldKey || p.TranslateKey != before.TranslateKey || p.PreviousTranslateKey != before.PreviousTranslateKey {
					t.Fatalf("%s translation=%v other=%s: rejected shortcut changed preferences", platform, translate, other)
				}
			}
		}
	}
}

func TestTranslationShortcutDisableRetainsAssignment(t *testing.T) {
	for _, platform := range []string{"darwin", "windows"} {
		prefs := DefaultPreferences(platform)
		prefs.ChooseTranslateKey(platform, "shift+f8")
		prefs.EnableTranslationShortcut(platform, false)
		if prefs.TranslateKey != HoldOff || prefs.PreviousTranslateKey != "shift+f8" {
			t.Fatal("disabled key not retained")
		}
		data, err := json.Marshal(prefs)
		if err != nil {
			t.Fatal(err)
		}
		var restored Preferences
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatal(err)
		}
		restored.EnableTranslationShortcut(platform, true)
		if restored.TranslateKey != "shift+f8" {
			t.Fatal("disabled assignment not restored after restart")
		}
		restored.EnableTranslationShortcut(platform, false)
		restored.ChooseHoldKey("shift+f8")
		restored.EnableTranslationShortcut(platform, true)
		if restored.HoldKey != "shift+f8" || HoldKeysConflict(restored.HoldKey, restored.TranslateKey) {
			t.Fatal("enabling translation moved dictation or retained conflict")
		}
	}
}

func TestSecondHoldKeyListensBesideThePrimaryShortcut(t *testing.T) {
	for _, platform := range []string{"darwin", "windows", "linux"} {
		prefs := DefaultPreferences(platform)
		if prefs.SecondHoldKey != DefaultSecondHoldKey(platform) {
			t.Fatalf("%s default second shortcut: %q", platform, prefs.SecondHoldKey)
		}
		normalized, repairs := prefs.Normalize(platform)
		if normalized.SecondHoldKey != prefs.SecondHoldKey || len(repairs) != 0 {
			t.Fatalf("%s absent second shortcut repaired: %q %v", platform, normalized.SecondHoldKey, repairs)
		}
		if DefaultHoldKey(platform) == "" {
			continue
		}
		beforeDefault := prefs
		prefs.ChooseSecondHoldKey(platform, "caps_lock")
		if prefs.SecondHoldKey != beforeDefault.SecondHoldKey {
			t.Fatal("unknown second shortcut replaced the stored value")
		}
		prefs.ChooseSecondHoldKey(platform, "f8")
		data, err := json.Marshal(prefs)
		if err != nil {
			t.Fatal(err)
		}
		restored := DefaultPreferences(platform)
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatal(err)
		}
		restored, repairs = restored.Normalize(platform)
		if restored.SecondHoldKey != "f8" || restored.HoldKey != prefs.HoldKey || len(repairs) != 0 {
			t.Fatalf("%s did not keep both dictation shortcuts: %+v %v", platform, restored, repairs)
		}
		assigned := restored.HoldAssignments()
		if len(assigned) != 3 || assigned[0] != restored.HoldKey || assigned[1] != "f8" || assigned[2] != restored.TranslateKey {
			t.Fatalf("%s assignments: %v", platform, assigned)
		}
		if restored.Translates("f8") || restored.Translates(restored.HoldKey) || !restored.Translates(restored.TranslateKey) {
			t.Fatal("second dictation shortcut was treated as translation")
		}
		before := restored
		restored.ChooseSecondHoldKey(platform, restored.HoldKey)
		restored.ChooseHoldKey("f8")
		restored.ChooseTranslateKey(platform, "f8")
		if restored.HoldKey != before.HoldKey || restored.SecondHoldKey != before.SecondHoldKey || restored.TranslateKey != before.TranslateKey {
			t.Fatal("conflicting selection changed a shortcut")
		}
		stored := restored
		stored.SecondHoldKey = stored.HoldKey
		stored, repairs = stored.Normalize(platform)
		if stored.SecondHoldKey != HoldOff || stored.HoldKey != before.HoldKey || len(repairs) != 0 {
			t.Fatalf("stored collision moved the primary shortcut: %+v %v", stored, repairs)
		}
		invalid := DefaultPreferences(platform)
		invalid.SecondHoldKey = "caps_lock"
		invalid, repairs = invalid.Normalize(platform)
		if invalid.SecondHoldKey != HoldOff || len(repairs) != 1 || repairs[0] != "secondHoldKey" {
			t.Fatalf("invalid second shortcut: %q %v", invalid.SecondHoldKey, repairs)
		}
		disabled := DefaultPreferences(platform)
		disabled.ChooseSecondHoldKey(platform, HoldOff)
		if got := disabled.HoldAssignments(); len(got) != 2 || got[1] != disabled.TranslateKey {
			t.Fatalf("disabled second shortcut stayed registered: %v", got)
		}
	}
}

func TestEnablingTranslationSkipsSecondDictationShortcut(t *testing.T) {
	prefs := DefaultPreferences("darwin")
	prefs.ChooseTranslateKey("darwin", HoldOff)
	prefs.ChooseSecondHoldKey("darwin", HoldRightCommand)
	prefs.EnableTranslationShortcut("darwin", true)
	if prefs.SecondHoldKey != HoldRightCommand || prefs.TranslateKey == HoldOff || HoldKeysConflict(prefs.TranslateKey, prefs.HoldKey) || HoldKeysConflict(prefs.TranslateKey, prefs.SecondHoldKey) {
		t.Fatalf("translation enablement collided with dictation: %+v", prefs)
	}
}
