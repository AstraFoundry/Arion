package app

import (
	"strings"
	"testing"

	"arion.local/desktop/ui/features/selection"
	"arion.local/desktop/ui/pages/onboarding"
	selectionpage "arion.local/desktop/ui/pages/selection"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

var locales = []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese}

// toLastStep continues from Welcome until Finish is the footer action.
func toLastStep(t *testing.T, tt *ui.Tester, messages copy.Catalog) {
	t.Helper()
	clicker(t, tt)(messages.Text(copy.OnbStart))
	for i := 0; i < 8; i++ {
		if _, ok := tt.Find(messages.Text(copy.OnbFinish)); ok {
			return
		}
		clicker(t, tt)(messages.Text(copy.Continue))
	}
	t.Fatalf("finish not reached: %q", tt.Texts())
}

// toggle clicks the switch at the trailing edge, right, of the row titled
// title with detail beneath it.
func toggle(t *testing.T, tt *ui.Tester, right float32, title, detail string) {
	t.Helper()
	top, ok := tt.Find(title)
	bottom, detailOK := tt.Find(detail)
	if !ok || !detailOK {
		t.Fatalf("row %q missing: %q", title, tt.Texts())
	}
	tt.ClickAt(right-36, (top.Y+bottom.Y+bottom.H)/2)
	settle(tt)
}

func TestOnboardingSelectionStep(t *testing.T) {
	for _, locale := range locales {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			state.Account.Disabled = true
			page := onboarding.NewPage()
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
			settle(tt)
			messages := store.Copy()
			toLastStep(t, tt, messages)
			expect(t, tt, true, messages.Text(copy.OnbSelectionTitle), messages.Text(copy.SelectionEnabled), messages.Text(copy.OnbDoneBody))
			expect(t, tt, false, messages.Text(copy.Continue), messages.Text(copy.SelectionClaude), messages.Text(copy.SelectionChatGPT))
			if state.Preferences.Selection.Enabled {
				t.Fatal("a fresh profile starts with selection actions on")
			}
			snapshot(t, tt, "onboarding-selection-"+string(locale))
			right := float32(onboarding.Pane + onboarding.Inset + onboarding.Content)
			master := func() {
				t.Helper()
				toggle(t, tt, right, messages.Text(copy.SelectionEnabled), messages.Text(copy.SelectionEnabledDetail))
			}
			master()
			expect(t, tt, true, messages.Text(copy.SelectionClaude), messages.Text(copy.SelectionChatGPT), messages.Text(copy.SelectionChatGPTDetail))
			toggle(t, tt, right, messages.Text(copy.SelectionChatGPT), messages.Text(copy.SelectionChatGPTDetail))
			if !state.Preferences.Selection.Enabled || !selection.ClientEnabled(state.Preferences, runtime.SelectionClientChatGPT) ||
				selection.ClientEnabled(state.Preferences, runtime.SelectionClientClaude) {
				t.Fatalf("selection preferences = %+v", state.Preferences.Selection)
			}
			snapshot(t, tt, "onboarding-selection-on-"+string(locale))
			master()
			clicker(t, tt)(messages.Text(copy.OnbFinish))
			if state.Preferences.Selection.Enabled || !actions.called("complete") {
				t.Fatalf("leaving selection off did not finish: %+v %v", state.Preferences.Selection, actions.calls)
			}
		})
	}
}

func TestOnboardingOmitsSelectionWithoutReader(t *testing.T) {
	state, actions, store := fixture()
	state.Account.Disabled = true
	state.SelectionReader = false
	page := onboarding.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	settle(tt)
	messages := store.Copy()
	toLastStep(t, tt, messages)
	expect(t, tt, false, messages.Text(copy.OnbSelectionTitle), messages.Text(copy.SelectionEnabled))
	clicker(t, tt)(messages.Text(copy.OnbFinish))
	if !actions.called("complete") {
		t.Fatal("setup without a selection reader did not finish")
	}
}

func offerFor(state *runtime.State, clients ...string) {
	state.Preferences.Selection.Enabled = true
	for _, id := range clients {
		for i := range state.Preferences.Selection.Clients {
			if state.Preferences.Selection.Clients[i].ID == id {
				state.Preferences.Selection.Clients[i].Enabled = true
			}
		}
	}
	state.Selection = runtime.SelectionOffer{Text: "Feature-Sliced Design", Actions: state.Preferences.Selection.VisibleActions()}
}

func selectionTester(store runtime.Store) *ui.Tester {
	page := &selectionpage.Page{}
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SelectionWidth, SelectionHeightFor(store)+40)
	settle(tt)
	return tt
}

func TestSelectionMenu(t *testing.T) {
	for _, locale := range locales {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			offerFor(state, runtime.SelectionClientClaude, runtime.SelectionClientChatGPT)
			messages := store.Copy()
			tt := selectionTester(store)
			expect(t, tt, true, messages.Text(copy.SelectionTranslate), messages.Text(copy.SelectionExplain), messages.Text(copy.SelectionSummarize),
				messages.Text(copy.SelectionRewrite), messages.Text(copy.SelectionAsk), messages.Text(copy.SelectionAskChatGPT),
				messages.Text(copy.Copy), messages.Text(copy.SelectionVocabulary), messages.Text(copy.SelectionReturn))
			snapshot(t, tt, "selection-menu-"+string(locale))
			row, _ := tt.Find(messages.Text(copy.SelectionTranslate))
			glyph, _ := tt.Find(messages.Text(copy.SelectionReturn))
			explain, _ := tt.Find(messages.Text(copy.SelectionExplain))
			if glyph.Y < row.Y || glyph.Y+glyph.H > row.Y+row.H+8 || glyph.Y >= explain.Y || glyph.X <= row.X {
				t.Fatalf("return glyph %v is not on the translate row %v", glyph, row)
			}
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if !actions.called("selection-open:" + runtime.SelectionActionTranslate) {
				t.Fatalf("return did not translate: %v", actions.calls)
			}
			tt.Key(0, ui.KeyDown)
			settle(tt)
			if glyph, _ := tt.Find(messages.Text(copy.SelectionReturn)); glyph.Y < explain.Y-8 {
				t.Fatal("arrow down did not move the highlight")
			}
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if !actions.called("selection-open:" + runtime.SelectionActionExplain) {
				t.Fatalf("return did not run the highlighted row: %v", actions.calls)
			}
			actions.calls = nil
			clicker(t, tt)(messages.Text(copy.SelectionAsk))
			tt.Type("why")
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if !actions.called("selection-open:"+runtime.SelectionActionAsk+":why") || len(actions.calls) != 1 {
				t.Fatalf("question field did not ask: %v", actions.calls)
			}
			clicker(t, tt)(messages.Text(copy.SelectionAskChatGPT))
			clicker(t, tt)(messages.Text(copy.Copy))
			clicker(t, tt)(messages.Text(copy.SelectionVocabulary))
			for _, call := range []string{"selection-open:" + runtime.SelectionActionAskChatGPT, "selection-copy", "selection-vocabulary"} {
				if !actions.called(call) {
					t.Fatalf("%s not called: %v", call, actions.calls)
				}
			}
			for _, call := range actions.calls {
				if strings.Contains(call, "prefs") {
					t.Fatal("the menu wrote preferences")
				}
			}
		})
	}
}

func TestSelectionMenuWithoutClients(t *testing.T) {
	for _, locale := range locales {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			offerFor(state)
			messages := store.Copy()
			tt := selectionTester(store)
			expect(t, tt, true, messages.Text(copy.Copy), messages.Text(copy.SelectionVocabulary))
			expect(t, tt, false, messages.Text(copy.SelectionTranslate), messages.Text(copy.SelectionExplain), messages.Text(copy.SelectionSummarize),
				messages.Text(copy.SelectionRewrite), messages.Text(copy.SelectionAsk), messages.Text(copy.SelectionAskChatGPT), messages.Text(copy.SelectionReturn))
			snapshot(t, tt, "selection-menu-local-"+string(locale))
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if !actions.called("selection-copy") {
				t.Fatalf("return without prompts did not copy: %v", actions.calls)
			}
			actions.selectionError = runtime.ErrVocabularyInvalid
			clicker(t, tt)(messages.Text(copy.SelectionVocabulary))
			expect(t, tt, true, messages.Text(copy.VocabTermInvalid))
		})
	}
}

func TestGeneralSelectionSettings(t *testing.T) {
	for _, locale := range locales {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			page := settings.NewPage()
			page.Section = settings.General
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
			settle(tt)
			messages := store.Copy()
			reveal(t, tt, messages.Text(copy.SelectionEnabledDetail))
			expect(t, tt, false, messages.Text(copy.SelectionChatGPT))
			right := float32(SettingsWidth - 32)
			master := func() {
				t.Helper()
				toggle(t, tt, right, messages.Text(copy.SelectionEnabled), messages.Text(copy.SelectionEnabledDetail))
			}
			master()
			if !state.Preferences.Selection.Enabled || !actions.called("prefs") {
				t.Fatalf("switch did not enable selection actions: %+v", state.Preferences.Selection)
			}
			reveal(t, tt, messages.Text(copy.SelectionChatGPTDetail))
			if !strings.Contains(messages.Text(copy.SelectionChatGPTDetail), "codex://") {
				t.Fatal("ChatGPT row does not name codex://")
			}
			snapshot(t, tt, "settings-general-selection-"+string(locale))
			toggle(t, tt, right, messages.Text(copy.SelectionClaude), messages.Text(copy.SelectionClaudeDetail))
			if !selection.ClientEnabled(state.Preferences, runtime.SelectionClientClaude) || selection.ClientEnabled(state.Preferences, runtime.SelectionClientChatGPT) {
				t.Fatalf("client flags = %+v", state.Preferences.Selection.Clients)
			}
			master()
			if state.Preferences.Selection.Enabled || !selection.ClientEnabled(state.Preferences, runtime.SelectionClientClaude) {
				t.Fatalf("turning the switch off changed the clients: %+v", state.Preferences.Selection)
			}
		})
	}
}
