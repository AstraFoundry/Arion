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
