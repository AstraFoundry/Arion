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
	"arion.local/desktop/ui/widgets/selectionmenu"
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
	prefs := state.Preferences.Selection
	state.Selection = runtime.SelectionOffer{Text: "Feature-Sliced Design", Actions: prefs.VisibleActions(), Clients: prefs.EnabledClients()}
}

// selectionTester leaves room below the bar for the second menu.
func selectionTester(store runtime.Store) (*ui.Tester, *int) {
	page := &selectionpage.Page{}
	height := new(int)
	tt := ui.NewTester(func(c *ui.Context) { *height = page.Render(c, store) }, SelectionWidth, 320)
	settle(tt)
	return tt, height
}

// promptTitles are the rows of the second menu, which the bar never shows.
func promptTitles(messages copy.Catalog) []string {
	return []string{messages.Text(copy.SelectionTranslate), messages.Text(copy.SelectionExplain), messages.Text(copy.SelectionSummarize),
		messages.Text(copy.SelectionRewrite), messages.Text(copy.SelectionAsk), messages.Text(copy.SelectionReturn)}
}

func TestSelectionBar(t *testing.T) {
	for _, locale := range locales {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			offerFor(state, runtime.SelectionClientClaude, runtime.SelectionClientChatGPT)
			messages := store.Copy()
			tt, height := selectionTester(store)
			expect(t, tt, true, messages.Text(copy.SelectionClaude), messages.Text(copy.SelectionChatGPT), messages.Text(copy.Copy), messages.Text(copy.SelectionVocabulary))
			expect(t, tt, false, append(promptTitles(messages), messages.Text(copy.SelectionAskChatGPT))...)
			if *height != SelectionHeightFor(store) || SelectionHeightFor(store) > 40 || SelectionWidth > 420 {
				t.Fatalf("bar %dx%d, window height %d", SelectionWidth, SelectionHeightFor(store), *height)
			}
			claude, _ := tt.Find(messages.Text(copy.SelectionClaude))
			vocabulary, _ := tt.Find(messages.Text(copy.SelectionVocabulary))
			if vocabulary.Y > claude.Y+claude.H || vocabulary.X+vocabulary.W > SelectionWidth {
				t.Fatalf("bar actions are not one row: %v %v", claude, vocabulary)
			}
			snapshot(t, tt, "selection-bar-"+string(locale))

			clicker(t, tt)(messages.Text(copy.SelectionChatGPT))
			expect(t, tt, true, append(promptTitles(messages), messages.Text(copy.SelectionAskChatGPT))...)
			menu := *height - SelectionHeightFor(store)
			if menu <= 0 || menu >= 243 {
				t.Fatalf("second menu adds %d", menu)
			}
			row, _ := tt.Find(messages.Text(copy.SelectionTranslate))
			if row.X+row.W > selectionmenu.MenuWidth || row.Y < claude.Y+claude.H {
				t.Fatalf("translate row %v is not in the menu under the bar", row)
			}
			snapshot(t, tt, "selection-menu-"+string(locale))
			glyph, _ := tt.Find(messages.Text(copy.SelectionReturn))
			explain, _ := tt.Find(messages.Text(copy.SelectionExplain))
			if glyph.Y < row.Y || glyph.Y+glyph.H > row.Y+row.H+8 || glyph.Y >= explain.Y || glyph.X <= row.X {
				t.Fatalf("return glyph %v is not on the translate row %v", glyph, row)
			}
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if !actions.called("selection-open:" + runtime.SelectionClientChatGPT + ":" + runtime.SelectionActionTranslate) {
				t.Fatalf("return did not translate in ChatGPT: %v", actions.calls)
			}
			tt.Key(0, ui.KeyDown)
			settle(tt)
			if glyph, _ := tt.Find(messages.Text(copy.SelectionReturn)); glyph.Y < explain.Y-8 {
				t.Fatal("arrow down did not move the highlight")
			}
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if !actions.called("selection-open:" + runtime.SelectionClientChatGPT + ":" + runtime.SelectionActionExplain) {
				t.Fatalf("return did not run the highlighted row: %v", actions.calls)
			}

			actions.calls = nil
			clicker(t, tt)(messages.Text(copy.SelectionAsk))
			expect(t, tt, true, promptTitles(messages)[:4]...)
			if len(actions.calls) != 0 {
				t.Fatalf("clicking the question field ran %v", actions.calls)
			}
			tt.Type("why")
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if !actions.called("selection-open:"+runtime.SelectionClientChatGPT+":"+runtime.SelectionActionAsk+":why") || len(actions.calls) != 1 {
				t.Fatalf("question field did not ask: %v", actions.calls)
			}
			expect(t, tt, true, messages.Text(copy.SelectionTranslate))
			clicker(t, tt)(messages.Text(copy.SelectionAskChatGPT))
			if !actions.called("selection-open:" + runtime.SelectionClientChatGPT + ":" + runtime.SelectionActionAskChatGPT) {
				t.Fatalf("Ask ChatGPT not called: %v", actions.calls)
			}

			clicker(t, tt)(messages.Text(copy.SelectionClaude))
			expect(t, tt, false, messages.Text(copy.SelectionAskChatGPT))
			clicker(t, tt)(messages.Text(copy.SelectionExplain))
			if !actions.called("selection-open:" + runtime.SelectionClientClaude + ":" + runtime.SelectionActionExplain) {
				t.Fatalf("explain did not go to Claude: %v", actions.calls)
			}
			clicker(t, tt)(messages.Text(copy.SelectionClaude))
			expect(t, tt, false, promptTitles(messages)...)
			if *height != SelectionHeightFor(store) {
				t.Fatalf("closed menu left height %d", *height)
			}

			clicker(t, tt)(messages.Text(copy.Copy))
			clicker(t, tt)(messages.Text(copy.SelectionVocabulary))
			for _, call := range []string{"selection-copy", "selection-vocabulary"} {
				if !actions.called(call) {
					t.Fatalf("%s not called: %v", call, actions.calls)
				}
			}
			for _, call := range actions.calls {
				if strings.Contains(call, "prefs") {
					t.Fatal("the bar wrote preferences")
				}
			}
		})
	}
}

func TestSelectionBarWithoutClients(t *testing.T) {
	for _, locale := range locales {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			offerFor(state)
			messages := store.Copy()
			tt, _ := selectionTester(store)
			expect(t, tt, true, messages.Text(copy.Copy), messages.Text(copy.SelectionVocabulary))
			expect(t, tt, false, append(promptTitles(messages), messages.Text(copy.SelectionAskChatGPT),
				messages.Text(copy.SelectionClaude), messages.Text(copy.SelectionChatGPT))...)
			snapshot(t, tt, "selection-bar-local-"+string(locale))
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if !actions.called("selection-copy") {
				t.Fatalf("return without a menu did not copy: %v", actions.calls)
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
