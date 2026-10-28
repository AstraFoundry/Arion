package app

import (
	"testing"
	"time"

	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

func TestTranslationDisclosureLeavesDictationAndSwitchStationary(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		for _, reduce := range []bool{false, true} {
			name := string(locale) + "-animated"
			if reduce {
				name = string(locale) + "-reduced-motion"
			}
			t.Run(name, func(t *testing.T) {
				state, _, store := fixture()
				state.Preferences.UILanguage = locale
				state.Preferences.TranslateKey = runtime.HoldOff
				state.Preferences.TranslateLanguage = runtime.TargetEnglish
				page := settings.NewPage()
				page.Section = settings.Dictation
				tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
				tt.SetPreferences(ui.Preferences{ReduceMotion: reduce})
				messages := store.Copy()
				fixed := map[copy.Key]ui.Rect{}
				for _, key := range []copy.Key{copy.HoldKeyLabel, copy.HoldKeyEdit, copy.TranslateGroup, copy.TranslateEnabled} {
					r, ok := tt.Find(messages.Text(key))
					if !ok {
						t.Fatalf("missing fixed control %s", key)
					}
					fixed[key] = r
				}
				stationary := func() {
					t.Helper()
					for key, before := range fixed {
						if r, ok := tt.Find(messages.Text(key)); !ok || r != before {
							t.Fatalf("%s moved from %+v to %+v", key, before, r)
						}
					}
				}
				toggle := func() {
					t.Helper()
					if err := tt.Click(messages.Text(copy.TranslateEnabled)); err != nil {
						t.Fatal(err)
					}
					tt.Move(0, 0)
					stationary()
				}
				snapshot(t, tt, "translation-closed-"+name)
				toggle()
				rest(tt, 40*time.Millisecond)
				stationary()
				snapshot(t, tt, "translation-opening-"+name)
				// Reverse while opening, then reopen before closing finishes.
				toggle()
				toggle()
				rest(tt, theme.Disclosure.Duration)
				tt.Frame()
				stationary()
				if !hasText(tt, messages.Text(copy.TranslateLanguageLabel)) {
					t.Fatal("translation settings did not expand")
				}
				snapshot(t, tt, "translation-expanded-"+name)
				toggle()
				if !reduce && !hasText(tt, messages.Text(copy.TranslateKeyLabel)) {
					t.Fatal("closing removed content before the animation")
				}
				snapshot(t, tt, "translation-closing-"+name)
				rest(tt, 40*time.Millisecond)
				stationary()
				rest(tt, theme.Disclosure.Duration)
				tt.Frame()
				stationary()
				if hasText(tt, messages.Text(copy.TranslateKeyLabel)) || hasText(tt, messages.Text(copy.TranslateLanguageLabel)) {
					t.Fatal("collapsed translation retained settings")
				}
				snapshot(t, tt, "translation-collapsed-"+name)
			})
		}
	}
}
