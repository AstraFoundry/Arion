//go:build arion_e2e && (darwin || windows)

package e2e

import (
	"testing"
	"time"

	"arion.local/desktop/ui/pages/settings"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

// MEM-A22: render approved alias groups in the production native window.
// Storage/actions are tested separately with isolated synthetic encryption keys.
func TestVocabularyNativeGroups(t *testing.T) {
	app.OpenSettings()
	var window *mygo.Window
	eventually(t, "vocabulary settings opened", func() bool { window = app.Windows()["settings"]; return window != nil && window.IsVisible() })
	var before []arion.Replacement
	var language arion.UILanguage
	app.Update(func(s *arion.State) { before, language = s.Vocabulary, s.Preferences.UILanguage })
	defer func() {
		app.Update(func(s *arion.State) {
			s.Vocabulary = before
			app.Store().Actions.UpdatePreferences(func(p *arion.Preferences) { p.UILanguage = language })
		})
		window.Close()
	}()
	height := min(850, mygo.Screen.PrimaryDisplay().WorkArea.Height)
	window.SetSize(820, height)
	eventually(t, "vocabulary settings resized", func() bool { w, h := window.Size(); return w == 820 && h == height })
	for _, locale := range []arion.UILanguage{arion.LanguageEnglish, arion.LanguageChinese} {
		app.Update(func(s *arion.State) {
			app.Store().Actions.UpdatePreferences(func(p *arion.Preferences) { p.UILanguage = locale })
			s.Vocabulary = []arion.Replacement{{ID: "first", From: "cloud code", To: "Claude Code"}, {ID: "second", From: "克劳德", To: "Claude Code"}, {ID: "third", From: "阿里昂", To: "Arion"}}
			s.Focus = arion.SettingsFocus{Section: settings.Vocabulary, Seq: s.Focus.Seq + 1}
		})
		time.Sleep(350 * time.Millisecond)
		if !nonblank(capture(t, window, "settings-vocabulary-"+string(locale), 760, height-64)) {
			t.Fatalf("%s: vocabulary content blank", locale)
		}
	}
}

// MEM-A24: exercise production vocabulary actions while the native Settings
// window is alive. Headless form tests separately dispatch control clicks.
func TestVocabularyNativeHotwordActions(t *testing.T) {
	app.OpenSettings()
	var window *mygo.Window
	eventually(t, "hotword settings opened", func() bool { window = app.Windows()["settings"]; return window != nil && window.IsVisible() })
	var language arion.UILanguage
	app.Update(func(s *arion.State) { language = s.Preferences.UILanguage })
	defer func() {
		app.Update(func(s *arion.State) {
			app.Store().Actions.UpdatePreferences(func(p *arion.Preferences) { p.UILanguage = language })
		})
		window.Close()
	}()
	height := min(850, mygo.Screen.PrimaryDisplay().WorkArea.Height)
	window.SetSize(820, height)
	eventually(t, "hotword settings resized", func() bool { w, h := window.Size(); return w == 820 && h == height })
	var actionErr error
	app.Update(func(s *arion.State) {
		actionErr = app.Store().Actions.AddReplacement("native heard", "NativeTerm")
		if actionErr == nil {
			actionErr = app.Store().Actions.AddHotword("NativeTerm")
		}
		s.Focus = arion.SettingsFocus{Section: settings.Vocabulary, Seq: s.Focus.Seq + 1}
	})
	if actionErr != nil {
		t.Fatal(actionErr)
	}
	var hotwordID, aliasID string
	app.Update(func(s *arion.State) {
		for _, entry := range s.Vocabulary {
			if entry.To != "NativeTerm" {
				continue
			}
			if entry.From == "" && entry.Hotword {
				hotwordID = entry.ID
			} else if entry.From == "native heard" {
				aliasID = entry.ID
			}
		}
	})
	if hotwordID == "" || aliasID == "" || hotwordID == aliasID {
		t.Fatal("native actions did not create independent records")
	}
	for _, locale := range []arion.UILanguage{arion.LanguageEnglish, arion.LanguageChinese} {
		app.Update(func(s *arion.State) {
			app.Store().Actions.UpdatePreferences(func(p *arion.Preferences) { p.UILanguage = locale })
		})
		time.Sleep(350 * time.Millisecond)
		if !nonblank(capture(t, window, "settings-hotword-actions-"+string(locale), 760, height-64)) {
			t.Fatal("native hotword content blank")
		}
	}
	app.Update(func(s *arion.State) { app.Store().Actions.RemoveReplacement(hotwordID) })
	var keptAlias bool
	app.Update(func(s *arion.State) {
		for _, entry := range s.Vocabulary {
			if entry.ID == hotwordID {
				t.Error("removed native hotword remains")
			}
			keptAlias = keptAlias || entry.ID == aliasID
		}
		app.Store().Actions.RemoveReplacement(aliasID)
	})
	if !keptAlias {
		t.Fatal("hotword removal removed its independent correction")
	}
}
