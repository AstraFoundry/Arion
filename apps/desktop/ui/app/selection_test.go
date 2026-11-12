package app

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"strings"
	"testing"
	"time"

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

// selectionTester leaves room below the bar for the second menu. It
// reports the height the page asks for.
func selectionTester(store runtime.Store) (*ui.Tester, *int) {
	tt, _, height := sizedSelectionTester(store)
	return tt, height
}

// sizedSelectionTester also reports the width the page asks for, and
// takes that width as the window does once the bar has measured itself.
func sizedSelectionTester(store runtime.Store) (*ui.Tester, *int, *int) {
	page := &selectionpage.Page{}
	width, height := new(int), new(int)
	tt := ui.NewTester(func(c *ui.Context) { *width, *height = page.Render(c, store) }, SelectionWidth, 320)
	settle(tt)
	tt.SetSize(*width, 320)
	settle(tt)
	return tt, width, height
}

// barEnd is the right edge of the bar's last control, the X.
func barEnd(t *testing.T, tt *ui.Tester, messages copy.Catalog) ui.Rect {
	t.Helper()
	x, ok := tt.Find(messages.Text(copy.SelectionDisableMenu))
	if !ok {
		t.Fatalf("no X on the bar: %q", tt.Texts())
	}
	return x
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
			tt, width, height := sizedSelectionTester(store)
			expect(t, tt, true, messages.Text(copy.SelectionClaude), messages.Text(copy.SelectionChatGPT), messages.Text(copy.Copy), messages.Text(copy.SelectionVocabulary))
			expect(t, tt, false, append(promptTitles(messages), messages.Text(copy.SelectionAskChatGPT))...)
			if *height != SelectionHeightFor(store) || SelectionHeightFor(store) > 40 || *width > 480 {
				t.Fatalf("bar %dx%d, window height %d", *width, SelectionHeightFor(store), *height)
			}
			claude, _ := tt.Find(messages.Text(copy.SelectionClaude))
			vocabulary, _ := tt.Find(messages.Text(copy.SelectionVocabulary))
			x := barEnd(t, tt, messages)
			if vocabulary.Y > claude.Y+claude.H || x.Y > claude.Y+claude.H || vocabulary.X+vocabulary.W > x.X {
				t.Fatalf("bar actions are not one row ending in the X: %v %v %v", claude, vocabulary, x)
			}
			if gap := float32(*width) - (x.X + x.W); gap < 4 || gap > 6 {
				t.Fatalf("window %d leaves %.1f DIPs after the X at %v", *width, gap, x)
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

func TestSelectionDisableMenu(t *testing.T) {
	for _, locale := range locales {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			offerFor(state, runtime.SelectionClientClaude)
			state.Selection.Source = runtime.SelectionSource{App: "com.apple.Safari", AppName: "Safari", Site: "example.com"}
			messages := store.Copy()
			tt, width, height := sizedSelectionTester(store)
			pause := messages.SelectionPause(runtime.SelectionPause)
			rows := []string{pause, messages.SelectionDisableSite("example.com"), messages.SelectionDisableApp("Safari"), messages.Text(copy.SelectionTurnOff)}
			expect(t, tt, false, rows...)
			x := barEnd(t, tt, messages)
			clicker(t, tt)(messages.Text(copy.SelectionDisableMenu))
			expect(t, tt, true, rows...)
			expect(t, tt, false, messages.Text(copy.SelectionTranslate))
			first, _ := tt.Find(pause)
			last, _ := tt.Find(messages.Text(copy.SelectionTurnOff))
			if first.Y < x.Y+x.H || last.Y <= first.Y || first.X+first.W > float32(*width) || first.X < float32(*width)-selectionmenu.MenuWidth {
				t.Fatalf("disable rows %v %v are not under the X %v in a window %d wide", first, last, x, *width)
			}
			if *height <= SelectionHeightFor(store) {
				t.Fatalf("disable menu left the window %d tall", *height)
			}
			snapshot(t, tt, "selection-disable-"+string(locale))

			tt.Key(0, ui.KeyEscape)
			settle(tt)
			expect(t, tt, false, rows...)
			clicker(t, tt)(messages.Text(copy.SelectionClaude))
			clicker(t, tt)(messages.Text(copy.SelectionDisableMenu))
			expect(t, tt, false, messages.Text(copy.SelectionTranslate))
			for i, scope := range []string{runtime.SelectionDisablePause, runtime.SelectionDisableSite, runtime.SelectionDisableApp, runtime.SelectionDisableAll} {
				if i > 0 {
					clicker(t, tt)(messages.Text(copy.SelectionDisableMenu))
				}
				clicker(t, tt)(rows[i])
				if !actions.called("selection-disable:" + scope) {
					t.Fatalf("%s did not disable %s: %v", rows[i], scope, actions.calls)
				}
				expect(t, tt, false, rows...)
			}
			for _, call := range actions.calls {
				if strings.Contains(call, "prefs") {
					t.Fatal("the bar wrote preferences")
				}
			}

			actions.selectionError = errors.New("preferences not saved")
			clicker(t, tt)(messages.Text(copy.SelectionDisableMenu))
			clicker(t, tt)(pause)
			expect(t, tt, true, messages.Text(copy.SelectionFailed))
		})
	}
}

func TestSelectionDisableMenuWithoutSource(t *testing.T) {
	state, _, store := fixture()
	offerFor(state)
	messages := store.Copy()
	tt, _, _ := sizedSelectionTester(store)
	clicker(t, tt)(messages.Text(copy.SelectionDisableMenu))
	expect(t, tt, true, messages.SelectionPause(runtime.SelectionPause), messages.Text(copy.SelectionTurnOff))
	for _, text := range tt.Texts() {
		if strings.HasPrefix(text, "Disable on") || strings.HasPrefix(text, "Disable in") {
			t.Fatalf("an unnamed source offered %q", text)
		}
	}
	snapshot(t, tt, "selection-disable-local-en")
}

// TestSelectionMenuMotion: a menu comes in over its first frames, and the
// window keeps its height while a closing menu leaves.
func TestSelectionMenuMotion(t *testing.T) {
	state, _, store := fixture()
	offerFor(state, runtime.SelectionClientChatGPT)
	messages := store.Copy()
	tt, _, height := sizedSelectionTester(store)
	if err := tt.Click(messages.Text(copy.SelectionChatGPT)); err != nil {
		t.Fatal(err)
	}
	open := *height
	var entering [][]byte
	for i := 0; i < 4; i++ {
		rest(tt, 40*time.Millisecond)
		entering = append(entering, pixels(t, tt))
		snapshot(t, tt, fmt.Sprintf("selection-menu-enter-%dms", 40*(i+1)))
	}
	settle(tt)
	settled := pixels(t, tt)
	if bytes.Equal(entering[0], settled) || bytes.Equal(entering[0], entering[1]) {
		t.Fatal("the menu appeared at once instead of coming in")
	}
	if *height != open {
		t.Fatalf("the menu did not settle at %d (window %d)", open, *height)
	}

	if err := tt.Click(messages.Text(copy.SelectionChatGPT)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		rest(tt, 40*time.Millisecond)
		if *height != open {
			t.Fatalf("the window shrank to %d after %d ms, cutting off the leaving menu", *height, 40*(i+1))
		}
		snapshot(t, tt, fmt.Sprintf("selection-menu-exit-%dms", 40*(i+1)))
	}
	settle(tt)
	if *height != SelectionHeightFor(store) {
		t.Fatalf("closed menu left height %d", *height)
	}

	if err := tt.Click(messages.Text(copy.SelectionDisableMenu)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		rest(tt, 40*time.Millisecond)
		snapshot(t, tt, fmt.Sprintf("selection-disable-enter-%dms", 40*(i+1)))
	}
}

func pixels(t *testing.T, tt *ui.Tester) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, tt.Image()); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
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
