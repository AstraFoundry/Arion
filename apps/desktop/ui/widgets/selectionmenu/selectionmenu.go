// Package selectionmenu is the bar shown over text selected in another app:
// one row of the enabled assistants, copy and vocabulary, ending in an X
// that opens the disable menu. Choosing an assistant opens a second,
// smaller menu beneath the bar with the prompt rows, the question field and
// that assistant's own actions. It renders the actions it is given; the
// backend filters them, fills and opens URLs, writes the vocabulary and
// saves how the bar is disabled.
package selectionmenu

import (
	"errors"
	"math"
	"strings"
	"time"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Width is the window's width before any bar has been laid out; from then
// on the window is as wide as the bar, an open menu or a failure line,
// whichever is widest. BarHeight is the bar's height, and the window's while
// no menu is open. Bar buttons do not shrink. MenuWidth is both menus' width.
const (
	Width     = 400
	BarHeight = 37
	MenuWidth = 224

	barPadding = 4
	barBorder  = 1
	toolHeight = BarHeight - 2*barPadding
	menuRow    = 26
	menuPad    = 4
	menuGap    = 1
	stackGap   = 4
	notice     = 24
)

// menuExit is how long a closing menu takes to leave. The window keeps the
// menu's size meanwhile, so the menu is not cut off as it goes.
const menuExit = 160 * time.Millisecond

// menuMotion brings a menu down from the bar, scaling up from the corner
// beneath the control that opened it, and sends it back up as it closes.
var menuMotion = ui.ElementTransition{
	Animation:    &theme.Pop,
	Enter:        &ui.Motion{Y: -6, Scale: 0.96},
	Exit:         &ui.Motion{Y: -4, Scale: 0.98},
	ExitDuration: menuExit,
	ExitEase:     ui.EaseIn,
}

// Menu keeps the open assistant or disable menu, the highlight, the
// question and the measured widths between frames. A new offer resets the
// menus.
type Menu struct {
	Question  string
	client    string
	disabling bool
	highlight int
	refocus   bool
	offer     string
	failure   copy.Key

	barWidth    int
	noticeWidth int
	heldWidth   int
	heldHeight  int
	release     time.Time
}

type layout struct {
	prompts []runtime.SelectionAction
	ask     *runtime.SelectionAction
	clients []runtime.SelectionAction
	copy    *runtime.SelectionAction
	vocab   *runtime.SelectionAction
}

func split(actions []runtime.SelectionAction) layout {
	var l layout
	for i := range actions {
		action := &actions[i]
		switch action.Kind {
		case runtime.SelectionKindPrompt:
			if action.ID == runtime.SelectionActionAsk {
				l.ask = action
			} else {
				l.prompts = append(l.prompts, *action)
			}
		case runtime.SelectionKindClient:
			l.clients = append(l.clients, *action)
		case runtime.SelectionKindCopy:
			l.copy = action
		case runtime.SelectionKindVocabulary:
			l.vocab = action
		}
	}
	return l
}

// own is the client's actions that the second menu lists after the prompts.
func (l layout) own(client string) []runtime.SelectionAction {
	var out []runtime.SelectionAction
	for _, action := range l.clients {
		if action.ClientID == client {
			out = append(out, action)
		}
	}
	return out
}

// opens reports whether choosing client has a second menu to show.
func (l layout) opens(client string) bool {
	return len(l.prompts) > 0 || l.ask != nil || len(l.own(client)) > 0
}

func (l layout) menuHeight(client string) int {
	rows := len(l.prompts) + len(l.own(client))
	if l.ask != nil {
		rows++
	}
	return menuHeight(rows)
}

func menuHeight(rows int) int {
	return 2*menuPad + rows*menuRow + max(0, rows-1)*menuGap
}

// disableScope is one row of the disable menu.
type disableScope struct {
	scope string
	label string
}

// disableScopes lists the disable menu: the pause, the page's site and the
// app when the reader named them, then the master switch.
func disableScopes(messages copy.Catalog, source runtime.SelectionSource) []disableScope {
	scopes := []disableScope{{runtime.SelectionDisablePause, messages.SelectionPause(runtime.SelectionPause)}}
	if source.Site != "" {
		scopes = append(scopes, disableScope{runtime.SelectionDisableSite, messages.SelectionDisableSite(source.Site)})
	}
	if source.App != "" {
		name := source.AppName
		if name == "" {
			name = source.App
		}
		scopes = append(scopes, disableScope{runtime.SelectionDisableApp, messages.SelectionDisableApp(name)})
	}
	return append(scopes, disableScope{runtime.SelectionDisableAll, messages.Text(copy.SelectionTurnOff)})
}

// title names a built-in action; unknown ids have no title and are not drawn.
func title(id string) copy.Key {
	switch id {
	case runtime.SelectionActionTranslate:
		return copy.SelectionTranslate
	case runtime.SelectionActionExplain:
		return copy.SelectionExplain
	case runtime.SelectionActionSummarize:
		return copy.SelectionSummarize
	case runtime.SelectionActionRewrite:
		return copy.SelectionRewrite
	case runtime.SelectionActionAsk:
		return copy.SelectionAsk
	case runtime.SelectionActionAskChatGPT:
		return copy.SelectionAskChatGPT
	case runtime.SelectionActionCopy:
		return copy.Copy
	case runtime.SelectionActionVocabulary:
		return copy.SelectionVocabulary
	}
	return ""
}

func clientName(id string) copy.Key {
	switch id {
	case runtime.SelectionClientClaude:
		return copy.SelectionClaude
	case runtime.SelectionClientChatGPT:
		return copy.SelectionChatGPT
	}
	return ""
}

func clientIcon(id string) *ui.SVG {
	if id == runtime.SelectionClientClaude {
		return icons.Anthropic
	}
	return icons.OpenAI
}

// Render draws the current offer, dark in either appearance, and returns
// the window size it needs: the bar, the open menu and a failure line. The
// width is 0 in the frame before the bar has been laid out, when the
// window keeps its width.
func (m *Menu) Render(c *ui.Context, store runtime.Store) (width, height int) {
	messages := store.Copy()
	offer := store.State.Selection
	if offer.Text != m.offer {
		m.offer, m.Question, m.client, m.disabling, m.highlight, m.failure = offer.Text, "", "", false, 0, ""
	}
	l := split(offer.Actions)
	var clients []string
	for _, id := range offer.Clients {
		if clientName(id) != "" && l.opens(id) {
			clients = append(clients, id)
		}
	}
	if !contains(clients, m.client) {
		m.client = ""
	}
	open := m.client != ""
	scopes := disableScopes(messages, offer.Source)
	m.highlight = max(0, min(m.highlight, len(l.prompts)-1))
	tokens := theme.Apply(c)
	outer := c.Theme()
	scoped := *outer
	scoped.Dark = true
	scoped.Text, scoped.TextMuted, scoped.Border = tokens.PillText, tokens.PillMuted, tokens.PillBorder
	scoped.Surface, scoped.SurfaceHover, scoped.SurfacePressed = tokens.PillBorder, tokens.PillBorder, tokens.PillBorder
	c.SetTheme(&scoped)
	ui.Column(c).Fill().Gap(stackGap).AlignItems(ui.Start).TextColor(tokens.PillText).Children(func() {
		m.bar(c, store, tokens, messages, l, clients)
		switch {
		case open:
			m.menu(c, store, tokens, messages, l)
		case m.disabling:
			m.disableMenu(c, store, tokens, scopes)
		}
		if m.failure != "" {
			ui.Row(c.Key("failure")).Transition(ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Y: -4}}).Children(func() {
				line := ui.Text(c, messages.Text(m.failure)).FontSize(theme.Caption).TextColor(tokens.PillMuted).Height(notice).Padding(4, 10).
					Radius(theme.RadiusMD).Background(tokens.PillSurface).SingleLine().Shrink(0)
				m.noticeWidth = rightEdge(line.LayoutBounds())
			})
		}
	})
	c.SetTheme(outer)
	c.Root().Background(ui.Transparent)
	switch {
	case (open || m.disabling) && c.Shortcut(0, ui.KeyEscape):
		m.client, m.disabling = "", false
	case open && len(l.prompts) > 0 && c.Shortcut(0, ui.KeyDown):
		m.highlight, m.refocus = min(m.highlight+1, len(l.prompts)-1), true
	case open && len(l.prompts) > 0 && c.Shortcut(0, ui.KeyUp):
		m.highlight, m.refocus = max(m.highlight-1, 0), true
	case c.Shortcut(0, ui.KeyEnter):
		switch {
		case open && len(l.prompts) > 0:
			m.run(store, l.prompts[m.highlight])
		case !open && !m.disabling && l.copy != nil:
			m.run(store, *l.copy)
		}
	}
	width, height = m.barWidth, BarHeight
	switch {
	case m.client != "":
		width, height = max(width, MenuWidth), height+stackGap+l.menuHeight(m.client)
	case m.disabling:
		width, height = max(width, MenuWidth), height+stackGap+menuHeight(len(scopes))
	}
	if m.failure != "" {
		width, height = max(width, m.noticeWidth), height+stackGap+notice
	}
	return m.hold(c, width, height)
}

// hold keeps the window at the size of a menu that is closing until its
// exit has finished, then lets it shrink to size.
func (m *Menu) hold(c *ui.Context, width, height int) (int, int) {
	if width >= m.heldWidth && height >= m.heldHeight {
		m.heldWidth, m.heldHeight, m.release = width, height, time.Time{}
		return width, height
	}
	now := c.Now()
	if m.release.IsZero() {
		m.release = now.Add(menuExit)
	}
	if !now.Before(m.release) {
		m.heldWidth, m.heldHeight, m.release = width, height, time.Time{}
		return width, height
	}
	c.AnimationFrame()
	return max(width, m.heldWidth), max(height, m.heldHeight)
}

// rightEdge is the window width that shows all of r, a box laid out from
// the window's left edge.
func rightEdge(r ui.Rect) int {
	if r.W <= 0 {
		return 0
	}
	return int(math.Ceil(float64(r.X + r.W)))
}

func contains(list []string, id string) bool {
	for _, item := range list {
		if item == id {
			return true
		}
	}
	return false
}

// bar is the first surface: the assistants, which open the second menu,
// copy and vocabulary, which act at once, and the X, which opens the
// disable menu. It is as wide as its buttons.
func (m *Menu) bar(c *ui.Context, store runtime.Store, tokens theme.Tokens, messages copy.Catalog, l layout, clients []string) {
	ui.Row(c).Height(BarHeight).Padding(barPadding).Gap(2).AlignItems(ui.Center).Radius(theme.RadiusLG).
		Background(tokens.PillSurface).Border(barBorder, tokens.PillBorder).Children(func() {
		for _, id := range clients {
			if m.clientButton(c, tokens, messages, id) {
				if m.client == id {
					m.client = ""
				} else {
					m.client, m.disabling, m.highlight, m.failure = id, false, 0, ""
				}
			}
		}
		if len(clients) > 0 && (l.copy != nil || l.vocab != nil) {
			divider(c, tokens)
		}
		if l.copy != nil && m.tool(c, tokens, messages, *l.copy, icons.Copy) {
			m.run(store, *l.copy)
		}
		if l.vocab != nil && m.tool(c, tokens, messages, *l.vocab, icons.BookOpen) {
			m.run(store, *l.vocab)
		}
		divider(c, tokens)
		x := m.closeButton(c, tokens, messages)
		if x.Clicked() {
			m.disabling = !m.disabling
			if m.disabling {
				m.client, m.failure = "", ""
			}
		}
		if end := rightEdge(x.LayoutBounds()); end > 0 {
			m.barWidth = end + barPadding + barBorder
		}
	})
}

func divider(c *ui.Context, tokens theme.Tokens) {
	ui.Box(c).Width(1).Height(16).Margin(0, 3).Shrink(0).Background(tokens.PillBorder)
}

// feel gives a bar or menu button its motion: the hover and chosen fills
// fade, and the button sinks a little while pressed.
func feel(button ui.Element) {
	button.Transition(ui.ElementTransition{Colors: true, Animation: &theme.Quick})
	kit.Press(button)
}

func (m *Menu) clientButton(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, id string) bool {
	name := messages.Text(clientName(id))
	chosen := m.client == id
	button := ui.ButtonBase(c.Key("client-"+id)).Height(toolHeight).Padding(0, 7).Gap(5).Radius(theme.RadiusMD).
		AlignItems(ui.Center).Shrink(0).Label(name).Selected(chosen)
	switch {
	case chosen:
		button.Background(tokens.PillText.Alpha(0.14))
	case button.Hovered():
		button.Background(tokens.PillText.Alpha(0.08))
	}
	feel(button)
	button.Children(func() {
		ui.Icon(c, clientIcon(id)).FontSize(14).TextColor(tokens.PillText)
		ui.Text(c, name).FontSize(theme.Caption).FontWeight(500).TextColor(tokens.PillText).SingleLine()
		turn := float32(0)
		if chosen {
			turn = 180
		}
		ui.Icon(c, icons.ChevronDown).FontSize(12).TextColor(tokens.PillMuted).AnimatePose(ui.Pose{Rotate: turn}, theme.Disclosure)
	})
	return button.Clicked()
}

// tool is a bar action that runs on click.
func (m *Menu) tool(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, action runtime.SelectionAction, icon *ui.SVG) bool {
	name := messages.Text(title(action.ID))
	button := ui.ButtonBase(c.Key("action-"+action.ID)).Height(toolHeight).Padding(0, 7).Gap(5).Radius(theme.RadiusMD).
		AlignItems(ui.Center).Shrink(0).Label(name)
	if button.Hovered() {
		button.Background(tokens.PillText.Alpha(0.08))
	}
	feel(button)
	button.Children(func() {
		ui.Icon(c, icon).FontSize(14).TextColor(tokens.PillMuted)
		ui.Text(c, name).FontSize(theme.Caption).FontWeight(500).TextColor(tokens.PillText).SingleLine()
	})
	return button.Clicked()
}

// closeButton is the X at the bar's trailing edge.
func (m *Menu) closeButton(c *ui.Context, tokens theme.Tokens, messages copy.Catalog) ui.Element {
	button := ui.ButtonBase(c.Key("disable")).Size(toolHeight, toolHeight).Radius(theme.RadiusMD).Justify(ui.Center).
		AlignItems(ui.Center).Shrink(0).Label(messages.Text(copy.SelectionDisableMenu)).Selected(m.disabling)
	switch {
	case m.disabling:
		button.Background(tokens.PillText.Alpha(0.14))
	case button.Hovered():
		button.Background(tokens.PillText.Alpha(0.08))
	}
	feel(button)
	button.Children(func() {
		ui.Icon(c, icons.X).FontSize(13).TextColor(tokens.PillMuted)
	})
	return button
}

// menu is the second surface for the chosen assistant. Clicking the
// question field only focuses it; submitting it asks.
func (m *Menu) menu(c *ui.Context, store runtime.Store, tokens theme.Tokens, messages copy.Catalog, l layout) {
	ui.Column(c.Key("menu-"+m.client)).Width(MenuWidth).Padding(menuPad).Gap(menuGap).Radius(theme.RadiusLG).Background(tokens.PillSurface).
		Border(1, tokens.PillBorder).TransformOrigin(0, 0).Transition(menuMotion).Children(func() {
		for i, action := range l.prompts {
			if m.row(c, tokens, messages, action, i == m.highlight) {
				m.highlight = i
				m.run(store, action)
			}
		}
		if l.ask != nil {
			m.askField(c, store, tokens, messages, *l.ask)
		}
		for _, action := range l.own(m.client) {
			if m.row(c, tokens, messages, action, false) {
				m.run(store, action)
			}
		}
	})
}

// disableMenu opens under the X, at the bar's trailing edge. Choosing a
// row disables the bar as it says; the backend then dismisses it.
func (m *Menu) disableMenu(c *ui.Context, store runtime.Store, tokens theme.Tokens, scopes []disableScope) {
	ui.Column(c.Key("menu-disable")).Width(MenuWidth).AlignSelf(ui.End).Padding(menuPad).Gap(menuGap).Radius(theme.RadiusLG).
		Background(tokens.PillSurface).Border(1, tokens.PillBorder).TransformOrigin(1, 0).Transition(menuMotion).Children(func() {
		for _, scope := range scopes {
			button := ui.ButtonBase(c.Key("disable-"+scope.scope)).Height(menuRow).Padding(0, 8).Radius(theme.RadiusMD).
				Justify(ui.Start).AlignItems(ui.Center).Label(scope.label)
			if button.Hovered() {
				button.Background(tokens.PillText.Alpha(0.08))
			}
			feel(button)
			button.Children(func() {
				ui.Text(c, scope.label).FontSize(theme.Caption).TextColor(tokens.PillText).SingleLine().Grow(1).Shrink(1)
			})
			if button.Clicked() {
				m.disable(store, scope.scope)
			}
		}
	})
}

// row is one menu entry; the highlighted prompt carries the Return glyph.
// Rows are keyed by the open assistant, so each menu appears anew and its
// highlighted row takes the keyboard from the assistant's button, and
// keeps it as the arrows move the highlight: Return runs that row.
func (m *Menu) row(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, action runtime.SelectionAction, highlighted bool) bool {
	name := messages.Text(title(action.ID))
	button := ui.ButtonBase(c.Key("action-"+m.client+"-"+action.ID)).Height(menuRow).Padding(0, 8).Gap(8).Radius(theme.RadiusMD).
		Justify(ui.Start).AlignItems(ui.Center).Label(name).Selected(highlighted)
	ink := tokens.PillText
	switch {
	case highlighted:
		if m.refocus {
			button.Focus()
			m.refocus = false
		}
		button.AutoFocus().Background(tokens.Accent)
		ink = tokens.AccentText
	case button.Hovered():
		button.Background(tokens.PillText.Alpha(0.08))
	}
	feel(button)
	button.Children(func() {
		ui.Text(c, name).FontSize(theme.Caption).TextColor(ink).SingleLine().Grow(1).Shrink(1)
		if highlighted {
			ui.Icon(c, icons.CornerDownLeft).FontSize(12).TextColor(ink).Label(messages.Text(copy.SelectionReturn))
		}
	})
	return button.Clicked()
}

func (m *Menu) askField(c *ui.Context, store runtime.Store, tokens theme.Tokens, messages copy.Catalog, action runtime.SelectionAction) {
	name := messages.Text(title(action.ID))
	field := ui.TextInput(c.Key("action-"+m.client+"-ask"), &m.Question).Label(name).Placeholder(name).Height(menuRow).FontSize(theme.Caption).
		Background(tokens.PillText.Alpha(0.06)).Border(0, ui.Transparent).TextColor(tokens.PillText)
	if field.Submitted() && strings.TrimSpace(m.Question) != "" {
		m.run(store, action)
	}
}

func (m *Menu) run(store runtime.Store, action runtime.SelectionAction) {
	var err error
	switch action.Kind {
	case runtime.SelectionKindPrompt, runtime.SelectionKindClient:
		question := ""
		if action.ID == runtime.SelectionActionAsk {
			question = strings.TrimSpace(m.Question)
		}
		err = store.Actions.OpenSelection(action.ID, m.client, question)
	case runtime.SelectionKindCopy:
		err = store.Actions.CopySelection()
	case runtime.SelectionKindVocabulary:
		err = store.Actions.AddSelectionVocabulary()
	}
	m.failure = failure(action, err)
}

func (m *Menu) disable(store runtime.Store, scope string) {
	if err := store.Actions.DisableSelection(scope); err != nil {
		m.failure = copy.SelectionFailed
		return
	}
	m.disabling = false
}

func failure(action runtime.SelectionAction, err error) copy.Key {
	switch {
	case err == nil:
		return ""
	case runtime.CodeOf(err) == runtime.CodeSecureFieldBlocked:
		return copy.FailSecureTitle
	case action.Kind == runtime.SelectionKindVocabulary && errors.Is(err, runtime.ErrVocabularyInvalid):
		return copy.VocabTermInvalid
	case action.Kind == runtime.SelectionKindVocabulary:
		return copy.VocabSaveFailed
	case action.Kind == runtime.SelectionKindPrompt || action.Kind == runtime.SelectionKindClient:
		return copy.SelectionOpenFailed
	}
	return copy.SelectionFailed
}
