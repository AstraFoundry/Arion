package app

import (
	"testing"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func TestVocabularyReadingNeedsArionKeyboard(t *testing.T) {
	state, _, store := fixture()
	state.InputMethod = runtime.InputMethodAbsent
	state.VocabularyKeyboardNotice = copy.VocabKeyboardPending
	state.Vocabulary = []runtime.Replacement{{ID: "term", To: "星槎", Hotword: true, KeyboardPinyin: "xing cha"}}
	tt := vocabularyPage(store, SettingsHeight)
	if _, ok := tt.Find(store.Copy().VocabReadingAction("星槎")); ok || hasText(tt, "xing cha") || hasText(tt, store.Copy().Text(copy.VocabKeyboardPending)) {
		t.Fatalf("keyboard readings shown without Arion's input method: %q", tt.Texts())
	}
	state.InputMethod = runtime.InputMethodSelected
	settle(tt)
	if !hasText(tt, "xing cha") || !hasText(tt, store.Copy().Text(copy.VocabKeyboardPending)) {
		t.Fatalf("keyboard readings hidden with Arion's input method: %q", tt.Texts())
	}
}

func TestVocabularyReadingEditor(t *testing.T) {
	for _, mode := range []struct{ chinese, dark bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
		state, actions, store := fixture()
		state.InputMethod = runtime.InputMethodInstalled
		name := "en"
		if mode.chinese {
			state.Preferences.UILanguage = runtime.LanguageChinese
			name = "zh-CN"
		}
		if mode.dark {
			name += "-dark"
		}
		messages := store.Copy()
		state.Vocabulary = []runtime.Replacement{
			{ID: "other", To: "Claude Code", Hotword: true, KeyboardPinyin: "ke lao de"},
			{ID: "term", To: "星槎工坊", Hotword: true},
		}
		tt := vocabularyPage(store, SettingsHeight)
		tt.SetDark(mode.dark)
		settle(tt)
		snapshot(t, tt, "vocabulary-reading-chips-"+name)
		if err := tt.Click(messages.VocabReadingAction("星槎工坊")); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		if _, ok := tt.Find(messages.RemoveVocab("星槎工坊")); ok {
			t.Fatal("removing the term beside an open reading invites the wrong action")
		}
		tt.Type("xing cha gong fang")
		settle(tt)
		snapshot(t, tt, "vocabulary-reading-"+name)
		tt.Key(0, ui.KeyEnter)
		settle(tt)
		if !actions.called("keyboard-reading") || state.Vocabulary[1].KeyboardPinyin != "xing cha gong fang" || !hasText(tt, "xing cha gong fang") {
			t.Fatalf("reading edit not saved: %+v", state.Vocabulary)
		}
		if err := tt.Click(messages.VocabReadingAction("星槎工坊")); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		actions.vocabError = runtime.ErrVocabularyInvalid
		if err := tt.Click(messages.Text(copy.VocabReadingSave)); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		// The failure is a callout on the field, not a line in the list.
		failure, ok := tt.Find(messages.Text(copy.VocabReadingInvalid))
		field, _ := tt.Find(messages.Text(copy.VocabReading))
		if !ok || failure.Y < field.Y+field.H || failure.X > field.X+field.W {
			t.Fatalf("invalid reading failure not attached below its field: %+v, field %+v", failure, field)
		}
		snapshot(t, tt, "vocabulary-reading-invalid-"+name)
		// After a failed save the field checks every edit: still invalid
		// keeps the callout, valid again removes it at once.
		actions.vocabError = nil
		if err := tt.Click(messages.Text(copy.VocabReading)); err != nil {
			t.Fatal(err)
		}
		tt.Key(0, ui.KeyEnd)
		tt.Type("1")
		settle(tt)
		if !hasText(tt, messages.Text(copy.VocabReadingInvalid)) {
			t.Fatal("an edit that is still invalid cleared the failure")
		}
		tt.Key(0, ui.KeyBackspace)
		settle(tt)
		if hasText(tt, messages.Text(copy.VocabReadingInvalid)) {
			t.Fatalf("a valid reading kept the failure: %q", tt.Texts())
		}
		actions.vocabError = runtime.ErrVocabularyInvalid
		if err := tt.Click(messages.Text(copy.VocabReadingSave)); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		actions.vocabError = runtime.ErrVocabularyConflict
		if err := tt.Click(messages.Text(copy.VocabReadingSave)); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		if !hasText(tt, messages.Text(copy.VocabReadingStale)) {
			t.Fatal("stale spelling failure was hidden")
		}
		tt.Key(0, ui.KeyEscape)
		settle(tt)
		if state.Vocabulary[1].KeyboardPinyin != "xing cha gong fang" || hasText(tt, messages.Text(copy.VocabReadingStale)) {
			t.Fatal("cancel changed the saved reading or kept its error")
		}
		if _, ok := tt.Find(messages.RemoveVocab("星槎工坊")); !ok {
			t.Fatal("closing the reading should restore the remove action")
		}
	}
}

func TestVocabularyReadingKeepsSameWordSourcesIndependent(t *testing.T) {
	state, _, store := fixture()
	state.InputMethod = runtime.InputMethodInstalled
	state.Vocabulary = []runtime.Replacement{
		{ID: "hotword", To: "星槎", Hotword: true, KeyboardPinyin: "xing zha"},
		{ID: "correction", To: "星槎", From: "星茶", KeyboardPinyin: "xing chai"},
	}
	tt := vocabularyPage(store, SettingsHeight)
	messages := store.Copy()
	if err := tt.Click(messages.VocabReadingVariant("星槎", "星茶")); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	for range "chai" {
		tt.Key(0, ui.KeyBackspace)
	}
	tt.Type("cha")
	tt.Key(0, ui.KeyEnter)
	settle(tt)
	if state.Vocabulary[0].KeyboardPinyin != "xing zha" || state.Vocabulary[1].KeyboardPinyin != "xing cha" {
		t.Fatalf("wrong source received reading: %+v", state.Vocabulary)
	}
}

func TestVocabularyKeyboardDeliveryNotice(t *testing.T) {
	for _, chinese := range []bool{false, true} {
		state, _, store := fixture()
		state.InputMethod = runtime.InputMethodInstalled
		name := "en"
		if chinese {
			state.Preferences.UILanguage = runtime.LanguageChinese
			name = "zh-CN"
		}
		state.Vocabulary = []runtime.Replacement{{ID: "one", To: "星槎", Hotword: true, KeyboardPinyin: "xing cha"}}
		tt := vocabularyPage(store, SettingsHeight)
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
