package app

import (
	"errors"
	"testing"

	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func vocabularyPage(store runtime.Store, height int) *ui.Tester {
	page := settings.NewPage()
	page.Section = settings.Vocabulary
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, height)
	settle(tt)
	return tt
}

func TestVocabularyTermsGroupTheirForms(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			state.Vocabulary = []runtime.Replacement{
				{ID: "first", From: "cloud code", To: "Claude Code"},
				{ID: "other", From: "a rion", To: "Arion"},
				{ID: "word", To: "Arion", Hotword: true},
				{ID: "second", From: "clawed code", To: "Claude Code"},
				{ID: "learned", From: "补补充", To: "不重", Hotword: true},
			}
			tt := vocabularyPage(store, SettingsHeight+220)
			canonicalCount := 0
			for _, text := range tt.Texts() {
				if text == "Claude Code" {
					canonicalCount++
				}
			}
			if canonicalCount != 1 || !hasText(tt, "cloud code") || !hasText(tt, "clawed code") {
				t.Fatalf("terms not grouped: %q", tt.Texts())
			}
			if first, ok := tt.Find("不重"); !ok {
				t.Fatal("learned term missing")
			} else if second, _ := tt.Find("Claude Code"); first.Y > second.Y {
				t.Fatal("the newest term should lead the list")
			}
			snapshot(t, tt, "settings-vocabulary-terms-"+string(locale))
			tt.SetDark(true)
			settle(tt)
			snapshot(t, tt, "settings-vocabulary-terms-dark-"+string(locale))
			tt.SetDark(false)
			settle(tt)
			if err := tt.Click(store.Copy().Text(copy.VocabSearch)); err != nil {
				t.Fatal(err)
			}
			tt.Type("CLOUD")
			settle(tt)
			if !hasText(tt, "clawed code") || hasText(tt, "Arion") {
				t.Fatalf("variant search should show its complete term: %q", tt.Texts())
			}
			if err := tt.Click(store.Copy().RemoveVocab("cloud code")); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if !actions.called("remove:first") || len(state.Vocabulary) != 4 || hasText(tt, "cloud code") {
				t.Fatalf("wrong form removed: %v %v", actions.calls, state.Vocabulary)
			}
			if err := tt.Click(store.Copy().Text(copy.VocabSearch)); err != nil {
				t.Fatal(err)
			}
			for range "CLOUD" {
				tt.Key(0, ui.KeyBackspace)
			}
			settle(tt)
			if err := tt.Click(store.Copy().RemoveVocab("Arion")); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if len(state.Vocabulary) != 2 || state.Vocabulary[0].ID != "second" || state.Vocabulary[1].ID != "learned" {
				t.Fatalf("removing a term should remove each of its records only: %v", state.Vocabulary)
			}
		})
	}
}

func TestVocabularyAddTermThenForms(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			messages := store.Copy()
			tt := vocabularyPage(store, SettingsHeight)
			if hasText(tt, messages.Text(copy.VocabSearch)) {
				t.Fatal("search shown without terms")
			}
			snapshot(t, tt, "settings-vocabulary-empty-"+string(locale))
			if err := tt.Click(messages.Text(copy.VocabToLabel)); err != nil {
				t.Fatal(err)
			}
			tt.Type("Arion")
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if !actions.called("add-hotword") || len(state.Vocabulary) != 1 || state.Vocabulary[0].To != "Arion" || state.Vocabulary[0].From != "" {
				t.Fatalf("term not saved: %v %v", actions.calls, state.Vocabulary)
			}
			if err := tt.Click(messages.VocabVariantAction("Arion")); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			tt.Type("阿里昂")
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			tt.Type("clawed, code")
			settle(tt)
			snapshot(t, tt, "settings-vocabulary-adding-form-"+string(locale))
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if len(state.Vocabulary) != 3 || state.Vocabulary[1].From != "阿里昂" || state.Vocabulary[2].From != "clawed, code" || state.Vocabulary[2].To != "Arion" {
				t.Fatalf("forms not saved intact: %v", state.Vocabulary)
			}
			tt.Key(0, ui.KeyEscape)
			settle(tt)
			if !hasText(tt, messages.Text(copy.VocabFromLabel)) {
				t.Fatal("escape should restore the add control")
			}
			snapshot(t, tt, "settings-vocabulary-added-"+string(locale))
		})
	}
}

func TestVocabularyFormFailuresKeepTheDraft(t *testing.T) {
	for _, failure := range []error{runtime.ErrVocabularyInvalid, runtime.ErrVocabularyConflict, errors.New("storage unavailable")} {
		t.Run(testErrorName(failure), func(t *testing.T) {
			state, actions, store := fixture()
			state.Vocabulary = []runtime.Replacement{{ID: "word", To: "Claude Code", Hotword: true}}
			tt := vocabularyPage(store, SettingsHeight)
			if err := tt.Click(store.Copy().VocabVariantAction("Claude Code")); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			tt.Type("cloud code")
			actions.vocabError = failure
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			want := copy.VocabSaveFailed
			if errors.Is(failure, runtime.ErrVocabularyInvalid) {
				want = copy.VocabInvalid
			}
			if errors.Is(failure, runtime.ErrVocabularyConflict) {
				want = copy.VocabConflict
			}
			if len(state.Vocabulary) != 1 || !hasText(tt, store.Copy().Text(want)) {
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
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if len(state.Vocabulary) != 2 || state.Vocabulary[1].From != "cloud code" || hasText(tt, store.Copy().Text(want)) {
				t.Fatalf("failed save lost the draft: %v", state.Vocabulary)
			}
		})
	}
}

func TestVocabularyTermFailures(t *testing.T) {
	for _, failure := range []error{runtime.ErrVocabularyInvalid, errors.New("storage unavailable")} {
		t.Run(testErrorName(failure), func(t *testing.T) {
			state, actions, store := fixture()
			actions.vocabError = failure
			tt := vocabularyPage(store, SettingsHeight)
			if err := tt.Click(store.Copy().Text(copy.VocabToLabel)); err != nil {
				t.Fatal(err)
			}
			tt.Type("<|system|>")
			if err := tt.Click(store.Copy().Text(copy.Add)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			want := copy.VocabSaveFailed
			if errors.Is(failure, runtime.ErrVocabularyInvalid) {
				want = copy.VocabTermInvalid
			}
			if len(state.Vocabulary) != 0 || !hasText(tt, store.Copy().Text(want)) {
				t.Fatalf("term failure hidden: %v %q", state.Vocabulary, tt.Texts())
			}
			tt.Type("x")
			settle(tt)
			if hasText(tt, store.Copy().Text(want)) {
				t.Fatal("editing the term should clear its error")
			}
			actions.vocabError = nil
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if len(state.Vocabulary) != 1 || state.Vocabulary[0].To != "<|system|>x" {
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
