package app

import (
	platform "runtime"
	"testing"

	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

func TestShortcutFeedbackBelongsToTheEditedSetting(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		for _, dark := range []bool{false, true} {
			name := string(locale) + "-light"
			if dark {
				name = string(locale) + "-dark"
			}
			t.Run(name, func(t *testing.T) {
				state, _, store := fixture()
				state.Preferences.UILanguage = locale
				state.Preferences.HoldKey, state.Preferences.TranslateKey = "ctrl+f8", "alt+f9"
				state.Preferences.TranslateLanguage = runtime.TargetEnglish
				page := settings.NewPage()
				page.Section = settings.Dictation
				tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
				tt.SetDark(dark)
				messages := store.Copy()
				click := func(key copy.Key) {
					t.Helper()
					if err := tt.Click(messages.Text(key)); err != nil {
						t.Fatal(err)
					}
					tt.Frame()
				}
				inTranslation := func(key copy.Key) {
					t.Helper()
					row, rowOK := tt.Find(messages.Text(copy.TranslateKeyLabel))
					feedback, feedbackOK := tt.Find(messages.Text(key))
					language, languageOK := tt.Find(messages.Text(copy.TranslateLanguageLabel))
					if !rowOK || !feedbackOK || !languageOK || feedback.Y <= row.Y || feedback.Y+feedback.H > language.Y {
						t.Fatalf("%s outside translation row: row %+v, feedback %+v, language %+v", key, row, feedback, language)
					}
				}
				if hasText(tt, messages.Text(copy.HoldKeySystemHint)) || hasText(tt, messages.Text(copy.HoldKeyConflict)) || hasText(tt, messages.Text(copy.TranslateKeyConflict)) {
					t.Fatal("custom assignments show an unverified warning")
				}
				snapshot(t, tt, "shortcut-feedback-idle-"+name)
				click(copy.TranslateKeyEdit)
				inTranslation(copy.HoldKeySystemHint)
				snapshot(t, tt, "shortcut-feedback-recording-"+name)
				state.Hold.RecordedKey = state.Preferences.HoldKey
				state.Hold.RecordedSeq++
				tt.Frame()
				inTranslation(copy.TranslateKeyConflict)
				if hasText(tt, messages.Text(copy.HoldKeySystemHint)) || hasText(tt, messages.Text(copy.HoldKeyConflict)) || state.Preferences.TranslateKey != "alt+f9" {
					t.Fatal("duplicate error has the wrong message or changed the assignment")
				}
				snapshot(t, tt, "shortcut-feedback-duplicate-"+name)
				state.Hold.RecordedKey = "ctrl+f10"
				state.Hold.RecordedSeq++
				tt.Frame()
				if state.Preferences.TranslateKey != "ctrl+f10" || hasText(tt, messages.Text(copy.HoldKeySystemHint)) || hasText(tt, messages.Text(copy.TranslateKeyConflict)) {
					t.Fatal("saving did not clear local feedback")
				}
				click(copy.HoldKeyEdit)
				snapshot(t, tt, "shortcut-feedback-dictation-recording-"+name)
				state.Hold.RecordedKey = state.Preferences.TranslateKey
				state.Hold.RecordedSeq++
				tt.Frame()
				if !hasText(tt, messages.Text(copy.HoldKeyConflict)) || hasText(tt, messages.Text(copy.TranslateKeyConflict)) || state.Preferences.HoldKey != "ctrl+f8" {
					t.Fatal("dictation duplicate did not identify translation")
				}
				snapshot(t, tt, "shortcut-feedback-dictation-duplicate-"+name)
				click(copy.HoldKeyCancel)
				if hasText(tt, messages.Text(copy.HoldKeySystemHint)) || hasText(tt, messages.Text(copy.HoldKeyConflict)) {
					t.Fatal("cancel retained local feedback")
				}
				if platform.GOOS == "darwin" {
					state.Preferences.TranslateKey = runtime.HoldFn
					tt.Frame()
					inTranslation(copy.HoldKeyFnHint)
					snapshot(t, tt, "shortcut-feedback-fn-"+name)
				}
				click(copy.TranslateEnabled)
				rest(tt, theme.Disclosure.Duration)
				tt.Frame()
				if hasText(tt, messages.Text(copy.HoldKeyFnHint)) || hasText(tt, messages.Text(copy.TranslateKeyLabel)) {
					t.Fatal("disabled translation retained its notes")
				}
			})
		}
	}
}

func TestShortcutListenerFailureIsSharedAndRepairable(t *testing.T) {
	state, actions, store := fixture()
	state.Hold = runtime.Hold{Failure: runtime.CodePermissionDenied}
	page := settings.NewPage()
	page.Section = settings.Dictation
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	messages := store.Copy()
	status, ok := tt.Find(messages.Text(copy.HoldKeyNeedsMonitoring))
	row, rowOK := tt.Find(messages.Text(copy.HoldKeyLabel))
	if !ok || !rowOK || status.Y+status.H >= row.Y {
		t.Fatal("shared permission state is inside the dictation setting")
	}
	if err := tt.Click(messages.Text(copy.Allow)); err != nil || !actions.called("request-monitoring") {
		t.Fatalf("missing permission repair action: %v", err)
	}
	snapshot(t, tt, "shortcut-feedback-permission")
	state.Hold.Failure = ""
	tt.Frame()
	if !hasText(tt, messages.Text(copy.HoldKeyUnavailable)) {
		t.Fatal("listener failure is not explained")
	}
	snapshot(t, tt, "shortcut-feedback-unavailable")
	if err := tt.Click(messages.Text(copy.Retry)); err != nil || !actions.called("retry-shortcuts") {
		t.Fatalf("missing listener retry action: %v", err)
	}
	state.Hold.Active = true
	tt.Frame()
	if hasText(tt, messages.Text(copy.HoldKeyUnavailable)) || hasText(tt, messages.Text(copy.HoldKeyNeedsMonitoring)) {
		t.Fatal("recovered listener retained a failure")
	}
}
