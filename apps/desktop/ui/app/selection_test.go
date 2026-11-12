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

// selectionHarness hosts the bar in one tester and the popup it wants in
// another, as bootstrap hosts them in separate windows. The bar's tester
// takes the width the bar measures and keeps SelectionHeight.
type selectionHarness struct {
	t     *testing.T
	store runtime.Store
	page  *selectionpage.Page
	bar   *ui.Tester
	// popup shows want; leaving is the popup it replaced, until it has
	// gone.
	want        SelectionPopup
	popup       *ui.Tester
	surface     *selectionpage.Surface
	leaving     *ui.Tester
	leavingFrom *selectionpage.Surface
	// gone records the popups that reported their exit finished.
	gone  map[*selectionpage.Surface]bool
	above bool
}

func newSelectionHarness(t *testing.T, store runtime.Store) *selectionHarness {
	t.Helper()
	h := &selectionHarness{t: t, store: store, page: &selectionpage.Page{}, gone: map[*selectionpage.Surface]bool{}}
	h.bar = ui.NewTester(func(c *ui.Context) { h.page.Render(c, store) }, SelectionWidth, SelectionHeight)
	settle(h.bar)
	h.bar.SetSize(h.page.Width(), SelectionHeight)
	h.settle()
	return h
}

// sync opens a tester for the popup the bar wants, as bootstrap opens a
// window, and keeps the one it replaces until it has left.
func (h *selectionHarness) sync() {
	want := h.page.Want()
	if want == h.want {
		return
	}
	if h.popup != nil && want.Key != h.want.Key {
		h.leaving, h.leavingFrom = h.popup, h.surface
	}
	h.want, h.popup, h.surface = want, nil, nil
	if want.Key == "" {
		return
	}
	surface := &selectionpage.Surface{Key: want.Key, Above: h.above}
	h.popup = ui.NewTester(func(c *ui.Context) {
		if h.page.RenderPopup(c, h.store, surface) {
			if h.page.Want().Key == surface.Key {
				h.t.Errorf("the popup %q left while the bar wants it", surface.Key)
			}
			h.gone[surface] = true
		}
	}, want.Width, want.Height)
	h.surface = surface
}

// settle runs the bar and its popups until they rest.
func (h *selectionHarness) settle() {
	for i := 0; i < 3; i++ {
		settle(h.bar)
		h.sync()
		if h.popup != nil {
			settle(h.popup)
			h.sync()
		}
	}
}

// hover rests the pointer on the bar's control labeled label.
func (h *selectionHarness) hover(label string) {
	h.t.Helper()
	r, ok := h.bar.Find(label)
	if !ok {
		h.t.Fatalf("%q not on the bar: %q", label, h.bar.Texts())
	}
	h.bar.Move(r.X+r.W/2, r.Y+r.H/2)
	h.settle()
}

// away moves the pointer off the bar.
func (h *selectionHarness) away() {
	h.bar.Move(-10, -10)
	h.settle()
}

// click clicks label in the popup if it shows it, else on the bar.
func (h *selectionHarness) click(label string) {
	h.t.Helper()
	tt := h.bar
	if h.popup != nil {
		if _, ok := h.popup.Find(label); ok {
			tt = h.popup
		}
	}
	if err := tt.Click(label); err != nil {
		h.t.Fatalf("%s: %v (bar %q, popup %q)", label, err, h.bar.Texts(), h.popupTexts())
	}
	h.settle()
}

func (h *selectionHarness) popupTexts() []string {
	if h.popup == nil {
		return nil
	}
	return h.popup.Texts()
}

// key presses key in the popup, which has the keyboard once clicked.
func (h *selectionHarness) key(key ui.Key) {
	h.t.Helper()
	if h.popup == nil {
		h.t.Fatal("no popup has the keyboard")
	}
	h.popup.Key(0, key)
	h.settle()
}

// expectPopup checks the popup the bar wants and that it shows texts.
func (h *selectionHarness) expectPopup(key string, texts ...string) {
	h.t.Helper()
	if h.want.Key != key {
		h.t.Fatalf("popup %q, want %q (bar %q)", h.want.Key, key, h.bar.Texts())
	}
	if key != "" {
		expect(h.t, h.popup, true, texts...)
	}
}

// promptTitles are the rows of the second menu, which the bar never shows.
func promptTitles(messages copy.Catalog) []string {
	return []string{messages.Text(copy.SelectionTranslate), messages.Text(copy.SelectionExplain), messages.Text(copy.SelectionSummarize),
		messages.Text(copy.SelectionRewrite), messages.Text(copy.SelectionAsk), messages.Text(copy.SelectionReturn)}
}

const chatGPTMenu = "client-" + runtime.SelectionClientChatGPT
const claudeMenu = "client-" + runtime.SelectionClientClaude

func TestSelectionBar(t *testing.T) {
	for _, locale := range locales {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			offerFor(state, runtime.SelectionClientClaude, runtime.SelectionClientChatGPT)
			messages := store.Copy()
			h := newSelectionHarness(t, store)
			expect(t, h.bar, true, messages.Text(copy.SelectionClaude), messages.Text(copy.SelectionChatGPT), messages.Text(copy.Copy), messages.Text(copy.SelectionVocabulary))
			expect(t, h.bar, false, append(promptTitles(messages), messages.Text(copy.SelectionAskChatGPT))...)
			width := h.page.Width()
			if SelectionHeight > 40 || width <= 0 || width > 480 {
				t.Fatalf("bar %dx%d", width, SelectionHeight)
			}
			claude, _ := h.bar.Find(messages.Text(copy.SelectionClaude))
			chatGPT, _ := h.bar.Find(messages.Text(copy.SelectionChatGPT))
			vocabulary, _ := h.bar.Find(messages.Text(copy.SelectionVocabulary))
			x, _ := h.bar.Find(messages.Text(copy.SelectionDisableMenu))
			if vocabulary.Y > claude.Y+claude.H || x.Y > claude.Y+claude.H || vocabulary.X+vocabulary.W > x.X {
				t.Fatalf("bar actions are not one row ending in the X: %v %v %v", claude, vocabulary, x)
			}
			if gap := float32(width) - (x.X + x.W); gap < 4 || gap > 6 {
				t.Fatalf("window %d leaves %.1f DIPs after the X at %v", width, gap, x)
			}
			snapshot(t, h.bar, "selection-bar-"+string(locale))

			h.hover(messages.Text(copy.SelectionChatGPT))
			h.expectPopup(chatGPTMenu, append(promptTitles(messages), messages.Text(copy.SelectionAskChatGPT))...)
			if h.want.Width != selectionmenu.MenuWidth || h.want.Height <= 0 || h.want.Height >= 243 || !h.want.Focusable || h.want.End {
				t.Fatalf("second menu %+v", h.want)
			}
			if h.want.Anchor.H != SelectionHeight || h.want.Anchor.X > chatGPT.X || h.want.Anchor.X < chatGPT.X-8 {
				t.Fatalf("menu anchor %v is not the ChatGPT button %v across the bar", h.want.Anchor, chatGPT)
			}
			snapshot(t, h.bar, "selection-bar-open-"+string(locale))
			snapshot(t, h.popup, "selection-menu-"+string(locale))
			row, _ := h.popup.Find(messages.Text(copy.SelectionTranslate))
			glyph, _ := h.popup.Find(messages.Text(copy.SelectionReturn))
			explain, _ := h.popup.Find(messages.Text(copy.SelectionExplain))
			if glyph.Y < row.Y || glyph.Y+glyph.H > row.Y+row.H+8 || glyph.Y >= explain.Y || glyph.X <= row.X {
				t.Fatalf("return glyph %v is not on the translate row %v", glyph, row)
			}
			h.key(ui.KeyEnter)
			if !actions.called("selection-open:" + runtime.SelectionClientChatGPT + ":" + runtime.SelectionActionTranslate) {
				t.Fatalf("return did not translate in ChatGPT: %v", actions.calls)
			}
			h.key(ui.KeyDown)
			if glyph, _ := h.popup.Find(messages.Text(copy.SelectionReturn)); glyph.Y < explain.Y-8 {
				t.Fatal("arrow down did not move the highlight")
			}
			h.key(ui.KeyEnter)
			if !actions.called("selection-open:" + runtime.SelectionClientChatGPT + ":" + runtime.SelectionActionExplain) {
				t.Fatalf("return did not run the highlighted row: %v", actions.calls)
			}

			actions.calls = nil
			h.click(messages.Text(copy.SelectionAsk))
			h.expectPopup(chatGPTMenu, promptTitles(messages)[:4]...)
			if len(actions.calls) != 0 {
				t.Fatalf("clicking the question field ran %v", actions.calls)
			}
			h.popup.Type("why")
			h.key(ui.KeyEnter)
			if !actions.called("selection-open:"+runtime.SelectionClientChatGPT+":"+runtime.SelectionActionAsk+":why") || len(actions.calls) != 1 {
				t.Fatalf("question field did not ask: %v", actions.calls)
			}
			h.click(messages.Text(copy.SelectionAskChatGPT))
			if !actions.called("selection-open:" + runtime.SelectionClientChatGPT + ":" + runtime.SelectionActionAskChatGPT) {
				t.Fatalf("Ask ChatGPT not called: %v", actions.calls)
			}

			h.hover(messages.Text(copy.SelectionClaude))
			h.expectPopup(claudeMenu, messages.Text(copy.SelectionExplain))
			expect(t, h.popup, false, messages.Text(copy.SelectionAskChatGPT))
			if h.want.Anchor.X != 0 {
				t.Fatalf("the first assistant's menu does not line up with the bar: %v", h.want.Anchor)
			}
			h.click(messages.Text(copy.SelectionExplain))
			if !actions.called("selection-open:" + runtime.SelectionClientClaude + ":" + runtime.SelectionActionExplain) {
				t.Fatalf("explain did not go to Claude: %v", actions.calls)
			}

			// A click closes the menu; resting on the control does not open
			// it again until the pointer has left it.
			h.click(messages.Text(copy.SelectionClaude))
			h.expectPopup("")
			h.settle()
			h.expectPopup("")
			h.away()
			h.hover(messages.Text(copy.SelectionClaude))
			h.expectPopup(claudeMenu, messages.Text(copy.SelectionExplain))
			h.click(messages.Text(copy.SelectionClaude))
			h.expectPopup("")

			h.click(messages.Text(copy.Copy))
			h.click(messages.Text(copy.SelectionVocabulary))
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

// TestSelectionHoverIntent: a pointer passing over a control opens
// nothing, one resting there opens its menu, and moving to another
// control switches menus without a click.
func TestSelectionHoverIntent(t *testing.T) {
	state, _, store := fixture()
	offerFor(state, runtime.SelectionClientClaude, runtime.SelectionClientChatGPT)
	messages := store.Copy()
	h := newSelectionHarness(t, store)
	r, _ := h.bar.Find(messages.Text(copy.SelectionChatGPT))
	h.bar.Move(r.X+r.W/2, r.Y+r.H/2)
	rest(h.bar, 60*time.Millisecond)
	h.away()
	h.expectPopup("")

	h.hover(messages.Text(copy.SelectionChatGPT))
	h.expectPopup(chatGPTMenu, messages.Text(copy.SelectionAskChatGPT))
	h.away()
	h.expectPopup(chatGPTMenu, messages.Text(copy.SelectionAskChatGPT))
	h.hover(messages.Text(copy.SelectionDisableMenu))
	h.expectPopup(selectionmenu.PopupDisable, messages.Text(copy.SelectionTurnOff))

	h.page.Dismiss(selectionmenu.PopupDisable)
	h.settle()
	h.expectPopup("")
	h.hover(messages.Text(copy.SelectionDisableMenu))
	h.expectPopup("")
}

func TestSelectionDisableMenu(t *testing.T) {
	for _, locale := range locales {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			offerFor(state, runtime.SelectionClientClaude)
			state.Selection.Source = runtime.SelectionSource{App: "com.apple.Safari", AppName: "Safari", Site: "example.com"}
			messages := store.Copy()
			h := newSelectionHarness(t, store)
			pause := messages.SelectionPause(runtime.SelectionPause)
			rows := []string{pause, messages.SelectionDisableSite("example.com"), messages.SelectionDisableApp("Safari"), messages.Text(copy.SelectionTurnOff)}
			expect(t, h.bar, false, rows...)
			h.hover(messages.Text(copy.SelectionDisableMenu))
			h.expectPopup(selectionmenu.PopupDisable, rows...)
			expect(t, h.popup, false, messages.Text(copy.SelectionTranslate))
			if end := h.want.Anchor.X + h.want.Anchor.W; !h.want.End || end < float32(h.page.Width())-1 || end > float32(h.page.Width())+1 {
				t.Fatalf("disable menu %+v does not line up with the bar's end %d", h.want, h.page.Width())
			}
			first, _ := h.popup.Find(pause)
			last, _ := h.popup.Find(messages.Text(copy.SelectionTurnOff))
			if last.Y <= first.Y || last.Y+last.H > float32(h.want.Height) {
				t.Fatalf("disable rows %v %v do not fit the popup %+v", first, last, h.want)
			}
			snapshot(t, h.popup, "selection-disable-"+string(locale))

			h.key(ui.KeyEscape)
			h.expectPopup("")
			h.away()
			h.click(messages.Text(copy.SelectionClaude))
			h.expectPopup(claudeMenu)
			for i, scope := range []string{runtime.SelectionDisablePause, runtime.SelectionDisableSite, runtime.SelectionDisableApp, runtime.SelectionDisableAll} {
				h.away()
				h.hover(messages.Text(copy.SelectionDisableMenu))
				h.expectPopup(selectionmenu.PopupDisable, rows...)
				h.click(rows[i])
				if !actions.called("selection-disable:" + scope) {
					t.Fatalf("%s did not disable %s: %v", rows[i], scope, actions.calls)
				}
				h.expectPopup("")
			}
			for _, call := range actions.calls {
				if strings.Contains(call, "prefs") {
					t.Fatal("the bar wrote preferences")
				}
			}

			actions.selectionError = errors.New("preferences not saved")
			h.away()
			h.hover(messages.Text(copy.SelectionDisableMenu))
			h.click(pause)
			h.expectPopup(selectionmenu.PopupNotice+string(copy.SelectionFailed), messages.Text(copy.SelectionFailed))
			if h.want.Focusable || h.want.Height != 24 {
				t.Fatalf("failure line %+v", h.want)
			}
		})
	}
}

func TestSelectionDisableMenuWithoutSource(t *testing.T) {
	state, _, store := fixture()
	offerFor(state)
	messages := store.Copy()
	h := newSelectionHarness(t, store)
	h.click(messages.Text(copy.SelectionDisableMenu))
	h.expectPopup(selectionmenu.PopupDisable, messages.SelectionPause(runtime.SelectionPause), messages.Text(copy.SelectionTurnOff))
	for _, text := range h.popup.Texts() {
		if strings.HasPrefix(text, "Disable on") || strings.HasPrefix(text, "Disable in") {
			t.Fatalf("an unnamed source offered %q", text)
		}
	}
	snapshot(t, h.popup, "selection-disable-local-en")
}

// TestSelectionMenuMotion: a menu comes in over its first frames in its
// own window, the bar's window never changes size, and a closing menu
// plays its exit before its window may close.
func TestSelectionMenuMotion(t *testing.T) {
	for _, above := range []bool{false, true} {
		t.Run(fmt.Sprintf("above=%v", above), func(t *testing.T) {
			state, _, store := fixture()
			offerFor(state, runtime.SelectionClientChatGPT)
			messages := store.Copy()
			h := newSelectionHarness(t, store)
			h.above = above
			width := h.page.Width()
			if err := h.bar.Click(messages.Text(copy.SelectionChatGPT)); err != nil {
				t.Fatal(err)
			}
			h.bar.Frame()
			h.sync()
			if h.popup == nil {
				t.Fatal("the click opened no popup")
			}
			var entering [][]byte
			for i := 0; i < 4; i++ {
				rest(h.popup, 40*time.Millisecond)
				entering = append(entering, pixels(t, h.popup))
				snapshot(t, h.popup, fmt.Sprintf("selection-menu-enter-%dms-above-%v", 40*(i+1), above))
			}
			h.settle()
			settled := pixels(t, h.popup)
			if bytes.Equal(entering[0], settled) || bytes.Equal(entering[0], entering[1]) {
				t.Fatal("the menu appeared at once instead of coming in")
			}

			if err := h.bar.Click(messages.Text(copy.SelectionChatGPT)); err != nil {
				t.Fatal(err)
			}
			h.bar.Frame()
			h.sync()
			if h.leaving == nil || h.want.Key != "" {
				t.Fatalf("the click left popup %q", h.want.Key)
			}
			leaving, surface := h.leaving, h.leavingFrom
			for i := 0; i < 3; i++ {
				rest(leaving, 40*time.Millisecond)
				if h.gone[surface] {
					t.Fatalf("the menu's window may close after %d ms, before its exit", 40*(i+1))
				}
				snapshot(t, leaving, fmt.Sprintf("selection-menu-exit-%dms-above-%v", 40*(i+1), above))
			}
			settle(leaving)
			if !h.gone[surface] {
				t.Fatal("the closed menu never let its window close")
			}
			if h.page.Width() != width {
				t.Fatalf("the bar went from %d to %d wide", width, h.page.Width())
			}
		})
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
			h := newSelectionHarness(t, store)
			expect(t, h.bar, true, messages.Text(copy.Copy), messages.Text(copy.SelectionVocabulary))
			expect(t, h.bar, false, append(promptTitles(messages), messages.Text(copy.SelectionAskChatGPT),
				messages.Text(copy.SelectionClaude), messages.Text(copy.SelectionChatGPT))...)
			snapshot(t, h.bar, "selection-bar-local-"+string(locale))
			h.bar.Key(0, ui.KeyEnter)
			h.settle()
			if !actions.called("selection-copy") {
				t.Fatalf("return without a menu did not copy: %v", actions.calls)
			}
			actions.selectionError = runtime.ErrVocabularyInvalid
			h.click(messages.Text(copy.SelectionVocabulary))
			h.expectPopup(selectionmenu.PopupNotice+string(copy.VocabTermInvalid), messages.Text(copy.VocabTermInvalid))
			line, _ := h.popup.Find(messages.Text(copy.VocabTermInvalid))
			if line.X+line.W > float32(h.want.Width) || h.want.Anchor.W != float32(h.page.Width()) {
				t.Fatalf("failure line %v does not fit its popup %+v", line, h.want)
			}
			snapshot(t, h.popup, "selection-failure-"+string(locale))
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
