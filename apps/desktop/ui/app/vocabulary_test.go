package app

import (
	"errors"
	"testing"

	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func TestVocabularyGroupedVariants(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			state.Vocabulary = []runtime.Replacement{
				{ID: "first", From: "cloud code", To: "Claude Code"},
				{ID: "other", From: "a rion", To: "Arion"},
				{ID: "second", From: "clawed code", To: "Claude Code"},
			}
			page := settings.NewPage()
			page.Section = settings.Vocabulary
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight+220)
			settle(tt)
			canonicalCount := 0
			for _, text := range tt.Texts() {
				if text == "Claude Code" {
					canonicalCount++
				}
			}
			if canonicalCount != 1 || !hasText(tt, "cloud code") || !hasText(tt, "clawed code") {
				t.Fatalf("terms not grouped: %q", tt.Texts())
			}
			snapshot(t, tt, "settings-vocabulary-grouped-"+string(locale))
			tt.SetDark(true)
			settle(tt)
			snapshot(t, tt, "settings-vocabulary-grouped-dark-"+string(locale))
			tt.SetDark(false)
			settle(tt)
			if err := tt.Click(store.Copy().Text(copy.VocabSearch)); err != nil {
				t.Fatal(err)
			}
			tt.Type("CLOUD")
			settle(tt)
			if !hasText(tt, "clawed code") || hasText(tt, "Arion") {
				t.Fatalf("variant search should show its complete group: %q", tt.Texts())
			}
			if err := tt.Click(store.Copy().RemoveVocabVariant("cloud code")); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if !actions.called("remove:first") || len(state.Vocabulary) != 2 || state.Vocabulary[1].ID != "second" {
				t.Fatalf("wrong variant removed: %v %v", actions.calls, state.Vocabulary)
			}
		})
	}
}

func TestVocabularyMultipleVariantsAndFailures(t *testing.T) {
	for _, failure := range []error{nil, runtime.ErrVocabularyInvalid, runtime.ErrVocabularyConflict, errors.New("storage unavailable")} {
		t.Run(testErrorName(failure), func(t *testing.T) {
			state, actions, store := fixture()
			actions.vocabError = failure
			page := settings.NewPage()
			page.Section = settings.Vocabulary
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
			settle(tt)
			if err := tt.Click(store.Copy().Text(copy.VocabToLabel)); err != nil {
				t.Fatal(err)
			}
			tt.Type("Claude Code")
			if err := tt.Click(store.Copy().Text(copy.VocabFromLabel)); err != nil {
				t.Fatal(err)
			}
			tt.Type("cloud code\nclawed, code")
			if err := tt.Click(store.Copy().Text(copy.Add)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if failure == nil {
				if len(state.Vocabulary) != 2 || state.Vocabulary[0].From != "cloud code" || state.Vocabulary[1].From != "clawed, code" {
					t.Fatalf("aliases were not saved intact: %v", state.Vocabulary)
				}
				return
			}
			want := copy.VocabSaveFailed
			if errors.Is(failure, runtime.ErrVocabularyInvalid) {
				want = copy.VocabInvalid
			}
			if errors.Is(failure, runtime.ErrVocabularyConflict) {
				want = copy.VocabConflict
			}
			if len(state.Vocabulary) != 0 || !hasText(tt, store.Copy().Text(want)) {
				t.Fatalf("failure lost draft or lacked explanation: %v %q", state.Vocabulary, tt.Texts())
			}
			snapshot(t, tt, "settings-vocabulary-"+string(want)+"-en")
			state.Preferences.UILanguage = runtime.LanguageChinese
			settle(tt)
			if !hasText(tt, store.Copy().Text(want)) {
				t.Fatalf("stale error locale: %q", tt.Texts())
			}
			snapshot(t, tt, "settings-vocabulary-"+string(want)+"-zh-CN")
			actions.vocabError = nil
			if err := tt.Click(store.Copy().Text(copy.Add)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if len(state.Vocabulary) != 2 || state.Vocabulary[1].From != "clawed, code" {
				t.Fatalf("failed save lost the draft: %v", state.Vocabulary)
			}
		})
	}
}

func testErrorName(err error) string {
	if err == nil {
		return "success"
	}
	return err.Error()
}

func TestStandaloneHotwordFormAndRemoval(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			state.Vocabulary = []runtime.Replacement{{ID: "pair", From: "ary on", To: "Arion", Hotword: true}}
			page := settings.NewPage()
			page.Section = settings.Vocabulary
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
			settle(tt)
			if err := tt.Click(store.Copy().Text(copy.VocabHotwordMode)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			for _, text := range tt.Texts() {
				if text == store.Copy().Text(copy.VocabFromLabel) {
					t.Fatal("hotword form requires an alias")
				}
			}
			if err := tt.Click(store.Copy().Text(copy.VocabToLabel)); err != nil {
				t.Fatal(err)
			}
			tt.Type("Arion")
			if err := tt.Click(store.Copy().Text(copy.Add)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if !actions.called("add-hotword") || len(state.Vocabulary) != 2 || state.Vocabulary[1].From != "" {
				t.Fatalf("standalone save %v %v", actions.calls, state.Vocabulary)
			}
			snapshot(t, tt, "settings-vocabulary-hotword-"+string(locale))
			tt.SetDark(true)
			settle(tt)
			snapshot(t, tt, "settings-vocabulary-hotword-dark-"+string(locale))
			tt.SetDark(false)
			settle(tt)
			if err := tt.Click(store.Copy().RemoveHotword("Arion")); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if len(state.Vocabulary) != 1 || state.Vocabulary[0].ID != "pair" || !hasText(tt, "ary on") {
				t.Fatalf("removed correction with hotword %v", state.Vocabulary)
			}
		})
	}
}
