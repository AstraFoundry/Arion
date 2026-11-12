// Package selectionmenu is the bar shown over text selected in another app:
// one row of the enabled assistants, copy and vocabulary, ending in an X
// that opens the disable menu. Resting the pointer on an assistant opens a
// second, smaller menu beside the bar with the prompt rows, the question
// field and that assistant's own actions; on the X it opens the disable
// menu. A click on the control that opened a menu, Escape, or a click
// elsewhere closes it. Each menu, and a failure line, shows in a popup
// window of its own beside the bar, so the bar's window never resizes as
// menus come and go. It renders the actions it is given; the backend
// filters them, fills and opens URLs, writes the vocabulary and saves how
// the bar is disabled.
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

// Width is the bar window's width before any bar has been laid out; from
// then on it is as wide as the bar. BarHeight is the bar window's height,
// which menus do not change. Bar buttons do not shrink. MenuWidth is both
// menus' width. Gap is the space between the bar and a popup beside it.
const (
	Width     = 400
	BarHeight = 37
	MenuWidth = 224
	Gap       = 4

	barPadding = 4
	barBorder  = 1
	barEdge    = barPadding + barBorder
	toolHeight = BarHeight - 2*barPadding
	menuRow    = 26
	menuPad    = 4
	menuGap    = 1
	notice     = 24
	noticePad  = 10
)

// menuExit is how long a closing popup takes to leave; its window closes
// after it.
const menuExit = 160 * time.Millisecond

// hoverDelay is how long the pointer rests on a control before its menu
// opens, so a pointer passing over the bar opens nothing.
const hoverDelay = 120 * time.Millisecond

// Popup keys: an assistant's menu is popupClient and its id.
const (
	PopupDisable = "disable"
	PopupNotice  = "notice:"
	popupClient  = "client-"
)

// Popup is the menu or failure line the bar shows beside it, in a window
// of its own. Key is empty when none shows. Anchor is the span of the bar
// the popup lines up with, in the bar window's DIPs: the control that
// opened it, widened by the bar's padding, across the bar's height. End
// lines the popup up with the anchor's end rather than its start. A
// Focusable popup takes the keyboard when it is clicked.
type Popup struct {
	Key           string
	Anchor        ui.Rect
	End           bool
	Width, Height int
	Focusable     bool
}

// Surface is one popup window's view of the menu: the popup it shows and
// whether its window opened above the bar, as it does where the screen has
// no room below.
type Surface struct {
	Key     string
	Above   bool
	built   bool
	leaving time.Time
}

// Menu keeps the open menu, the highlight, the question, the pointer's
// rest and the measured bar between frames of the bar and its popups. A
// new offer resets the menus.
type Menu struct {
	Question  string
	open      string
	highlight int
	refocus   bool
	offer     string
	failure   copy.Key

	barWidth int
	anchors  map[string]ui.Rect
	// resting is the control the pointer rests on, since restingSince.
	resting      string
	restingSince time.Time
	// spent is the control whose menu a click just closed: it opens none
	// by hover until the pointer leaves it.
	spent    string
	want     Popup
	revision int
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

// prepare resets the menus for a new offer and drops a menu whose
// assistant the offer no longer has.
func (m *Menu) prepare(store runtime.Store) (layout, []string, []disableScope) {
	offer := store.State.Selection
	if offer.Text != m.offer {
		m.offer, m.Question, m.open, m.highlight, m.failure, m.resting, m.spent = offer.Text, "", "", 0, "", "", ""
		m.revision++
	}
	l := split(offer.Actions)
	var clients []string
	for _, id := range offer.Clients {
		if clientName(id) != "" && l.opens(id) {
			clients = append(clients, id)
		}
	}
	if id, ok := strings.CutPrefix(m.open, popupClient); ok && !contains(clients, id) {
		m.set("")
	}
	m.highlight = max(0, min(m.highlight, len(l.prompts)-1))
	return l, clients, disableScopes(store.Copy(), offer.Source)
}

// client is the assistant whose menu is open, or "".
func (m *Menu) client() string {
	if id, ok := strings.CutPrefix(m.open, popupClient); ok {
		return id
	}
	return ""
}

// set opens the menu key, or closes the open one for "". Opening a menu
// clears a failure line.
func (m *Menu) set(key string) {
	if key == m.open {
		return
	}
	m.open, m.resting = key, ""
	if key != "" {
		m.highlight, m.failure = 0, ""
	}
	m.revision++
}

// shut closes the open menu as a click, Escape or a choice in it does: the
// control that opened it opens it again by hover only once the pointer has
// left it.
func (m *Menu) shut() {
	if m.open != "" {
		m.spent = m.open
		m.set("")
	}
}

// Dismiss closes the popup key that the system closed, as on a press on
// the control that opened it or on Escape.
func (m *Menu) Dismiss(key string) {
	switch {
	case key == m.open:
		m.shut()
	case strings.HasPrefix(key, PopupNotice) && m.failure != "":
		m.failure = ""
		m.revision++
	}
}

// Width is the bar's width as last laid out, or 0 before its first frame.
func (m *Menu) Width() int { return m.barWidth }

// Want is the popup the bar shows now.
func (m *Menu) Want() Popup { return m.want }

// Revision changes whenever what the bar or a popup shows changes, so the
// host redraws the other windows.
func (m *Menu) Revision() int { return m.revision }

// scope draws in the bar's dark colors, in either appearance, until the
// returned function restores the theme.
func scope(c *ui.Context) (theme.Tokens, func()) {
	tokens := theme.Apply(c)
	outer := c.Theme()
	scoped := *outer
	scoped.Dark = true
	scoped.Text, scoped.TextMuted, scoped.Border = tokens.PillText, tokens.PillMuted, tokens.PillBorder
	scoped.Surface, scoped.SurfaceHover, scoped.SurfacePressed = tokens.PillBorder, tokens.PillBorder, tokens.PillBorder
	c.SetTheme(&scoped)
	return tokens, func() { c.SetTheme(outer) }
}

// Render draws the bar for the current offer and returns its width, or 0
// in the frame before the bar has been laid out, when the window keeps its
// width. The window is BarHeight tall whatever menu is open.
func (m *Menu) Render(c *ui.Context, store runtime.Store) int {
	messages := store.Copy()
	l, clients, scopes := m.prepare(store)
	tokens, restore := scope(c)
	ui.Column(c).Fill().AlignItems(ui.Start).TextColor(tokens.PillText).Children(func() {
		m.bar(c, store, tokens, messages, l, clients)
	})
	restore()
	c.Root().Background(ui.Transparent)
	switch {
	case m.open != "" && c.Shortcut(0, ui.KeyEscape):
		m.shut()
	case m.open == "" && l.copy != nil && c.Shortcut(0, ui.KeyEnter):
		m.run(store, *l.copy)
	default:
		m.keys(c, store, l)
	}
	m.want = m.popup(c, l, scopes, messages)
	return m.barWidth
}

// keys moves the highlight of an open assistant's prompts with the arrows
// and runs it with Return, in whichever window has the keyboard.
func (m *Menu) keys(c *ui.Context, store runtime.Store, l layout) {
	if m.client() == "" || len(l.prompts) == 0 {
		return
	}
	switch {
	case c.Shortcut(0, ui.KeyDown):
		m.highlight, m.refocus = min(m.highlight+1, len(l.prompts)-1), true
		m.revision++
	case c.Shortcut(0, ui.KeyUp):
		m.highlight, m.refocus = max(m.highlight-1, 0), true
		m.revision++
	case c.Shortcut(0, ui.KeyEnter):
		m.run(store, l.prompts[m.highlight])
	}
}

// popup describes the window the open menu or the failure line needs.
func (m *Menu) popup(c *ui.Context, l layout, scopes []disableScope, messages copy.Catalog) Popup {
	anchor, ok := m.anchors[m.open]
	switch {
	case m.client() != "" && ok:
		return Popup{Key: m.open, Anchor: anchor, Width: MenuWidth, Height: l.menuHeight(m.client()), Focusable: true}
	case m.open == PopupDisable && ok:
		return Popup{Key: PopupDisable, Anchor: anchor, End: true, Width: MenuWidth, Height: menuHeight(len(scopes)), Focusable: true}
	case m.open == "" && m.failure != "" && m.barWidth > 0:
		w, _ := c.MeasureText(0, ui.Span{Text: messages.Text(m.failure), Size: theme.Caption})
		return Popup{Key: PopupNotice + string(m.failure), Anchor: ui.Rect{W: float32(m.barWidth), H: BarHeight},
			Width: int(math.Ceil(float64(w))) + 2*noticePad + 2, Height: notice}
	}
	return Popup{}
}

// RenderPopup draws what s shows while the bar wants it, and lets it leave
// once the bar wants another: it reports true when that exit has finished
// and the window may close.
func (m *Menu) RenderPopup(c *ui.Context, store runtime.Store, s *Surface) (gone bool) {
	messages := store.Copy()
	l, _, scopes := m.prepare(store)
	showing := m.key() == s.Key
	// A new window's first frame builds the popup's container alone: what
	// appears in a parent the last frame showed comes in with its motion.
	entering := !s.built
	s.built = true
	if entering {
		c.AnimationFrame()
	}
	tokens, restore := scope(c)
	ui.Column(c).Fill().TextColor(tokens.PillText).Children(func() {
		if !showing || entering {
			return
		}
		motion := surfaceMotion()
		switch {
		case s.Key == PopupDisable:
			m.disableMenu(c, store, tokens, scopes, motion, s.Above)
		case strings.HasPrefix(s.Key, PopupNotice):
			m.notice(c, tokens, messages, s.Above)
		default:
			m.menu(c, store, tokens, messages, l, motion, s.Above)
		}
	})
	restore()
	c.Root().Background(ui.Transparent)
	if showing {
		s.leaving = time.Time{}
		if m.open != "" && c.Shortcut(0, ui.KeyEscape) {
			m.shut()
		} else {
			m.keys(c, store, l)
		}
	} else {
		now := c.Now()
		if s.leaving.IsZero() {
			s.leaving = now
		}
		if left := menuExit - now.Sub(s.leaving); left > 0 {
			c.After(left)
		} else {
			gone = true
		}
	}
	m.want = m.popup(c, l, scopes, messages)
	return gone
}

// key is the popup the bar shows, without its geometry.
func (m *Menu) key() string {
	switch {
	case m.open != "":
		return m.open
	case m.failure != "":
		return PopupNotice + string(m.failure)
	}
	return ""
}

// surfaceMotion fades a popup in, growing from the corner beside the
// control that opened it, and back out as it leaves. It neither moves nor
// overshoots: its window is its size, and would cut off what went past.
func surfaceMotion() ui.ElementTransition {
	return ui.ElementTransition{
		Animation:    &theme.Disclosure,
		Enter:        &ui.Motion{Scale: 0.96},
		Exit:         &ui.Motion{Scale: 0.98},
		ExitDuration: menuExit,
		ExitEase:     ui.EaseIn,
	}
}

// edge is the transform origin's y for a popup below or above the bar:
// the side that faces it.
func edge(above bool) float32 {
	if above {
		return 1
	}
	return 0
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
	if m.anchors == nil {
		m.anchors = map[string]ui.Rect{}
	}
	ui.Row(c).Height(BarHeight).Padding(barPadding).Gap(2).AlignItems(ui.Center).Radius(theme.RadiusLG).
		Background(tokens.PillSurface).Border(barBorder, tokens.PillBorder).Shrink(0).Children(func() {
		for _, id := range clients {
			m.trigger(c, popupClient+id, m.clientButton(c, tokens, messages, id))
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
		m.trigger(c, PopupDisable, x)
		if end := rightEdge(x.LayoutBounds()); end > 0 {
			m.barWidth = end + barEdge
		}
	})
}

// trigger opens key's menu when the pointer has rested on button for
// hoverDelay, or at once on a click, and closes it on a click while it is
// open. It records the span of the bar the menu lines up with.
func (m *Menu) trigger(c *ui.Context, key string, button ui.Element) {
	if r := button.LayoutBounds(); r.W > 0 {
		m.anchors[key] = ui.Rect{X: r.X - barEdge, W: r.W + 2*barEdge, H: BarHeight}
	}
	if button.Clicked() {
		if m.open == key {
			m.shut()
		} else {
			m.set(key)
		}
		return
	}
	switch {
	case !button.Hovered():
		if m.spent == key {
			m.spent = ""
		}
		if m.resting == key {
			m.resting = ""
		}
	case m.open == key || m.spent == key:
	default:
		now := c.Now()
		if m.resting != key {
			m.resting, m.restingSince = key, now
		}
		if wait := hoverDelay - now.Sub(m.restingSince); wait > 0 {
			c.After(wait)
		} else {
			m.set(key)
		}
	}
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

func (m *Menu) clientButton(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, id string) ui.Element {
	name := messages.Text(clientName(id))
	chosen := m.client() == id
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
	return button
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
	disabling := m.open == PopupDisable
	button := ui.ButtonBase(c.Key("disable")).Size(toolHeight, toolHeight).Radius(theme.RadiusMD).Justify(ui.Center).
		AlignItems(ui.Center).Shrink(0).Label(messages.Text(copy.SelectionDisableMenu)).Selected(disabling)
	switch {
	case disabling:
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
func (m *Menu) menu(c *ui.Context, store runtime.Store, tokens theme.Tokens, messages copy.Catalog, l layout, motion ui.ElementTransition, above bool) {
	ui.Column(c.Key("menu-"+m.open)).Width(MenuWidth).Padding(menuPad).Gap(menuGap).Radius(theme.RadiusLG).Background(tokens.PillSurface).
		Border(1, tokens.PillBorder).TransformOrigin(0, edge(above)).Transition(motion).Children(func() {
		for i, action := range l.prompts {
			if m.row(c, tokens, messages, action, i == m.highlight) {
				m.highlight = i
				m.run(store, action)
			}
		}
		if l.ask != nil {
			m.askField(c, store, tokens, messages, *l.ask)
		}
		for _, action := range l.own(m.client()) {
			if m.row(c, tokens, messages, action, false) {
				m.run(store, action)
			}
		}
	})
}

// disableMenu lines up with the X, at the bar's trailing edge. Choosing a
// row disables the bar as it says; the backend then dismisses it.
func (m *Menu) disableMenu(c *ui.Context, store runtime.Store, tokens theme.Tokens, scopes []disableScope, motion ui.ElementTransition, above bool) {
	ui.Column(c.Key("menu-disable")).Width(MenuWidth).Padding(menuPad).Gap(menuGap).Radius(theme.RadiusLG).
		Background(tokens.PillSurface).Border(1, tokens.PillBorder).TransformOrigin(1, edge(above)).Transition(motion).Children(func() {
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

// notice is the failure line beside the bar.
func (m *Menu) notice(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, above bool) {
	ui.Row(c.Key("failure")).TransformOrigin(0, edge(above)).Transition(surfaceMotion()).Children(func() {
		ui.Text(c, messages.Text(m.failure)).FontSize(theme.Caption).TextColor(tokens.PillMuted).Height(notice).Padding(4, noticePad).
			Radius(theme.RadiusMD).Background(tokens.PillSurface).SingleLine().Shrink(0)
	})
}

// row is one menu entry; the highlighted prompt carries the Return glyph.
// Rows are keyed by the open assistant, so each menu appears anew and its
// highlighted row takes the keyboard once the menu has it, and keeps it as
// the arrows move the highlight: Return runs that row.
func (m *Menu) row(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, action runtime.SelectionAction, highlighted bool) bool {
	name := messages.Text(title(action.ID))
	button := ui.ButtonBase(c.Key("action-"+m.open+"-"+action.ID)).Height(menuRow).Padding(0, 8).Gap(8).Radius(theme.RadiusMD).
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
	field := ui.TextInput(c.Key("action-"+m.open+"-ask"), &m.Question).Label(name).Placeholder(name).Height(menuRow).FontSize(theme.Caption).
		Background(tokens.PillText.Alpha(0.06)).Border(0, ui.Transparent).TextColor(tokens.PillText)
	if field.Submitted() && strings.TrimSpace(m.Question) != "" {
		m.run(store, action)
	}
}

// run performs action. A failure closes the open menu and shows its line
// beside the bar instead.
func (m *Menu) run(store runtime.Store, action runtime.SelectionAction) {
	var err error
	switch action.Kind {
	case runtime.SelectionKindPrompt, runtime.SelectionKindClient:
		question := ""
		if action.ID == runtime.SelectionActionAsk {
			question = strings.TrimSpace(m.Question)
		}
		err = store.Actions.OpenSelection(action.ID, m.client(), question)
	case runtime.SelectionKindCopy:
		err = store.Actions.CopySelection()
	case runtime.SelectionKindVocabulary:
		err = store.Actions.AddSelectionVocabulary()
	}
	m.fail(failure(action, err))
}

func (m *Menu) disable(store runtime.Store, scope string) {
	if err := store.Actions.DisableSelection(scope); err != nil {
		m.fail(copy.SelectionFailed)
		return
	}
	m.shut()
}

// fail shows the failure line key, closing the open menu, or clears it.
func (m *Menu) fail(key copy.Key) {
	if key != "" {
		m.shut()
	}
	if key != m.failure {
		m.failure = key
		m.revision++
	}
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
