package app

import (
	platform "runtime"
	"testing"
	"time"

	holdname "arion.local/desktop/ui/entities/holdkey"
	"arion.local/desktop/ui/features/holdkey"
	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

// Voice translation follows the dictation key in two steps: what it does
// beside a translating mock, with the target language, then its key beside
// the keyboard figure, never the dictation key, with a press to try it.
func TestOnboardingVoiceTranslation(t *testing.T) {
	for _, locale := range []copy.Locale{copy.English, copy.SimplifiedChinese} {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = runtime.UILanguage(locale)
			// This platform's keys where it has them, so the picker has
			// choices; the figure draws either platform's.
			hold, translate := runtime.HoldRightOption, runtime.HoldRightCommand
			if keys := runtime.HoldKeysFor(platform.GOOS); len(keys) > 1 {
				hold, translate = keys[0], keys[1]
			}
			state.Preferences.HoldKey, state.Preferences.TranslateKey = hold, translate
			// Neither name is the interface language picker's beside it.
			from, to := runtime.TargetJapanese, runtime.TargetFrench
			state.Preferences.TranslateLanguage = from
			messages := copy.For(locale)
			page := onboarding.NewPage()
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
			settle(tt)
			click := func(label string) {
				t.Helper()
				if err := tt.Click(label); err != nil {
					t.Fatalf("%s: %v (%q)", label, err, tt.Texts())
				}
				settle(tt)
			}
			for _, label := range []copy.Key{copy.OnbStart, copy.OnbContinueLocally, copy.Continue, copy.Continue, copy.OnbIMELater, copy.Continue, copy.Continue} {
				click(messages.Text(label))
			}
			if !hasText(tt, messages.Text(copy.OnbHoldKeyTitle)) {
				t.Fatalf("hold key step: %q", tt.Texts())
			}
			click(messages.Text(copy.Continue))

			if !hasText(tt, messages.Text(copy.OnbTranslateTitle)) {
				t.Fatalf("translation step does not follow the hold key: %q", tt.Texts())
			}
			if _, ok := tt.Find(messages.Text(copy.ShowcaseLabel)); !ok {
				t.Fatal("translation step has no mock")
			}
			rest(tt, 2*time.Second)
			snapshot(t, tt, "onboarding-translate-"+string(from)+"-"+string(locale))
			click(holdkey.LanguageName(messages, from))
			for _, language := range runtime.TranslationLanguages {
				if !hasText(tt, holdkey.LanguageName(messages, language)) {
					t.Fatalf("%s not offered: %q", language, tt.Texts())
				}
			}
			snapshot(t, tt, "onboarding-translate-languages-"+string(locale))
			click(holdkey.LanguageName(messages, to))
			if state.Preferences.TranslateLanguage != to {
				t.Fatalf("target language not chosen: %q", state.Preferences.TranslateLanguage)
			}
			rest(tt, 4*time.Second)
			snapshot(t, tt, "onboarding-translate-"+string(to)+"-"+string(locale))
			click(messages.Text(copy.Continue))

			if !hasText(tt, messages.Text(copy.OnbTranslateKeyTitle)) {
				t.Fatalf("translation key step: %q", tt.Texts())
			}
			if _, ok := tt.Find(messages.Text(copy.TranslateKeyFigureLabel)); !ok {
				t.Fatal("translation key step does not show the keyboard figure")
			}
			if !hasText(tt, messages.Text(copy.OnbHoldKeyPress)) {
				t.Fatalf("no invitation to try the key: %q", tt.Texts())
			}
			// The dictation key does not count as the translation key.
			state.Hold.Pressed, state.Hold.PressedKey = time.Now().Add(time.Second), hold
			settle(tt)
			if hasText(tt, messages.Text(copy.OnbHoldKeyHeard)) {
				t.Fatal("dictation key press acknowledged as the translation key")
			}
			state.Hold.PressedKey = translate
			settle(tt)
			if !hasText(tt, messages.Text(copy.OnbHoldKeyHeard)) {
				t.Fatalf("translation key press not acknowledged: %q", tt.Texts())
			}
			rest(tt, 2500*time.Millisecond)
			snapshot(t, tt, "onboarding-translate-key-"+string(locale))
			if keys := runtime.TranslateKeysFor(platform.GOOS); len(keys) > 0 {
				click(holdname.Name(messages, translate))
				if hasText(tt, holdname.Name(messages, hold)) {
					t.Fatalf("dictation key offered for translation: %q", tt.Texts())
				}
				for _, key := range keys {
					if key != hold && !hasText(tt, holdname.Name(messages, key)) {
						t.Fatalf("%s not offered: %q", key, tt.Texts())
					}
				}
				snapshot(t, tt, "onboarding-translate-key-choices-"+string(locale))
				click(holdname.Name(messages, runtime.HoldOff))
				if state.Preferences.TranslateKey != runtime.HoldOff || state.Preferences.HoldKey != hold || !actions.called("translate-hold:off") {
					t.Fatalf("off not chosen: %+v", state.Preferences)
				}
				if hasText(tt, messages.Text(copy.OnbHoldKeyPress)) || hasText(tt, messages.Text(copy.OnbHoldKeyHeard)) {
					t.Fatalf("off still asks for a press: %q", tt.Texts())
				}
				snapshot(t, tt, "onboarding-translate-key-off-"+string(locale))
			}
			click(messages.Text(copy.Continue))
			if !hasText(tt, messages.Text(copy.OnbTryTitle)) {
				t.Fatalf("try step does not follow translation: %q", tt.Texts())
			}
		})
	}
}
