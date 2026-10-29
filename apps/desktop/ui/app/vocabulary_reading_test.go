package app

import (
	"testing"

	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func TestVocabularyReadingEditor(t *testing.T) {
	for _, mode := range []struct{ chinese, dark bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
		chinese := mode.chinese
		state, actions, store := fixture()
		name := "en"
		if chinese {
			state.Preferences.UILanguage = runtime.LanguageChinese
			name = "zh-CN"
		}
		if mode.dark {
			name += "-dark"
		}
		messages := store.Copy()
		state.Vocabulary = []runtime.Replacement{{ID: "term", To: "星槎工坊", Hotword: true}}
		page := settings.NewPage()
		page.Section = settings.Vocabulary
		tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
		tt.SetDark(mode.dark)
		settle(tt)
		tt.Scroll(SettingsWidth*.7, SettingsHeight/2, 0, 600)
		settle(tt)
		if err := tt.Click(messages.VocabReadingAction("星槎工坊")); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		tt.Scroll(SettingsWidth*.7, SettingsHeight/2, 0, 600)
		settle(tt)
		tt.Type("xing cha gong fang")
		settle(tt)
		snapshot(t, tt, "vocabulary-reading-"+name)
		if err := tt.Click(messages.Text(copy.VocabReadingSave)); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		if !actions.called("keyboard-reading") || state.Vocabulary[0].KeyboardPinyin != "xing cha gong fang" {
			t.Fatalf("reading edit not saved: %+v", state.Vocabulary)
		}
		if err := tt.Click(messages.VocabReadingAction("星槎工坊")); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		tt.Scroll(SettingsWidth*.7, SettingsHeight/2, 0, 600)
		settle(tt)
		actions.vocabError = runtime.ErrVocabularyInvalid
		if err := tt.Click(messages.Text(copy.VocabReadingSave)); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		if !hasText(tt, messages.Text(copy.VocabReadingInvalid)) {
			t.Fatal("invalid reading failure was hidden")
		}
		snapshot(t, tt, "vocabulary-reading-invalid-"+name)
		actions.vocabError = runtime.ErrVocabularyConflict
		if err := tt.Click(messages.Text(copy.VocabReadingSave)); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		if !hasText(tt, messages.Text(copy.VocabReadingStale)) {
			t.Fatal("stale spelling failure was hidden")
		}
		snapshot(t, tt, "vocabulary-reading-stale-"+name)
		if err := tt.Click(messages.Text(copy.VocabReadingCancel)); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		if state.Vocabulary[0].KeyboardPinyin != "xing cha gong fang" {
			t.Fatal("cancel changed saved reading")
		}
	}
}

func TestVocabularyReadingKeepsSameWordSourcesIndependent(t *testing.T) {
	state, _, store := fixture()
	state.Vocabulary = []runtime.Replacement{
		{ID: "hotword", To: "星槎", Hotword: true, KeyboardPinyin: "xing zha"},
		{ID: "correction", To: "星槎", From: "星茶"},
	}
	page := settings.NewPage()
	page.Section = settings.Vocabulary
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(tt)
	tt.Scroll(SettingsWidth*.7, SettingsHeight/2, 0, 600)
	settle(tt)
	messages := store.Copy()
	if err := tt.Click(messages.VocabReadingVariant("星槎", "星茶")); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	tt.Scroll(SettingsWidth*.7, SettingsHeight/2, 0, 600)
	settle(tt)
	tt.Type("xing cha")
	if err := tt.Click(messages.Text(copy.VocabReadingSave)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if state.Vocabulary[0].KeyboardPinyin != "xing zha" || state.Vocabulary[1].KeyboardPinyin != "xing cha" {
		t.Fatalf("wrong source received reading: %+v", state.Vocabulary)
	}
}

func TestVocabularyKeyboardDeliveryNotice(t *testing.T) {
	for _, chinese := range []bool{false, true} {
		state, _, store := fixture()
		name := "en"
		if chinese {
			state.Preferences.UILanguage = runtime.LanguageChinese
			name = "zh-CN"
		}
		state.Vocabulary = []runtime.Replacement{{ID: "one", To: "星槎", Hotword: true, KeyboardPinyin: "xing cha"}}
		page := settings.NewPage()
		page.Section = settings.Vocabulary
		tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
		for _, notice := range []copy.Key{copy.VocabKeyboardPending, copy.VocabKeyboardFailed} {
			state.VocabularyKeyboardNotice = notice
			settle(tt)
			if !hasText(tt, store.Copy().Text(notice)) {
				t.Fatal("keyboard delivery outcome is hidden")
			}
			snapshot(t, tt, "vocabulary-keyboard-"+string(notice)+"-"+name)
		}
		state.VocabularyKeyboardNotice = ""
		settle(tt)
		if hasText(tt, store.Copy().Text(copy.VocabKeyboardFailed)) || hasText(tt, store.Copy().Text(copy.VocabKeyboardPending)) {
			t.Fatal("acknowledged vocabulary retains old failure/pending notice")
		}
	}
}
